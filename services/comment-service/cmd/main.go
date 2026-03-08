package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/ppp-blog/comment-service/internal/cache"
	"github.com/ppp-blog/comment-service/internal/config"
	"github.com/ppp-blog/comment-service/internal/handler"
	"github.com/ppp-blog/comment-service/internal/middleware"
	"github.com/ppp-blog/comment-service/internal/queue"
	"github.com/ppp-blog/comment-service/internal/ratelimit"
	"github.com/ppp-blog/comment-service/internal/repository"
	"github.com/ppp-blog/comment-service/internal/service"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/redis/go-redis/v9"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("comment-service exited with error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	gin.SetMode(gin.ReleaseMode)

	runtime, err := initRuntime(cfg, logger)
	if err != nil {
		return err
	}
	defer runtime.Close(logger)

	server := newHTTPServer(cfg.Port, runtime.engine)
	waitCtx, stopSignal := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignal()

	consumerCtx, cancelConsumer := context.WithCancel(context.Background())
	syncCtx, cancelSync := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	startBackgroundWorkers(&wg, consumerCtx, syncCtx, runtime, logger)
	serverErrCh := startHTTPServer(server, cfg.Port, logger)

	if err := waitForShutdown(waitCtx, serverErrCh); err != nil {
		logger.Error("http server terminated unexpectedly", "error", err)
	}
	shutdownServer(server, logger)

	cancelConsumer()
	cancelSync()
	closeConsumer(runtime.consumer, logger)
	wg.Wait()
	logger.Info("comment-service stopped")
	return nil
}

type appRuntime struct {
	db          *sql.DB
	rdb         *redis.Client
	producer    *queue.Producer
	consumer    *queue.Consumer
	repository  *repository.CommentRepo
	likeService *service.LikeService
	engine      *gin.Engine
}

func initRuntime(cfg config.Config, logger *slog.Logger) (*appRuntime, error) {
	db, err := initMySQL(cfg.MySQLDSN)
	if err != nil {
		return nil, err
	}
	rdb, err := initRedis(cfg)
	if err != nil {
		db.Close()
		return nil, err
	}
	metrics, err := initMetrics()
	if err != nil {
		rdb.Close()
		db.Close()
		return nil, err
	}

	runtime := assembleRuntime(cfg, logger, db, rdb, metrics)
	return runtime, nil
}

func assembleRuntime(
	cfg config.Config,
	logger *slog.Logger,
	db *sql.DB,
	rdb *redis.Client,
	metrics *middleware.Metrics,
) *appRuntime {
	router := repository.NewShardRouter(cfg.ShardCount)
	repo := repository.NewCommentRepo(db, router)
	commentCache := cache.NewCommentCache(rdb, logger)
	hotCache := cache.NewHotCommentCache(rdb)
	likeCache := cache.NewLikeCache(rdb)

	producer := queue.NewProducer(cfg.KafkaBrokers, cfg.KafkaTopic, logger)
	consumer := queue.NewConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.ConsumerGroup, repo, commentCache, hotCache, metrics, logger)
	commentSvc := service.NewCommentService(repo, commentCache, hotCache, producer, metrics, logger)
	likeSvc := service.NewLikeService(repo, commentCache, likeCache, hotCache, producer, metrics, logger)

	engine := newRouter(logger, cfg, db, rdb, metrics, producer, repo, commentSvc, likeSvc)
	return &appRuntime{
		db:          db,
		rdb:         rdb,
		producer:    producer,
		consumer:    consumer,
		repository:  repo,
		likeService: likeSvc,
		engine:      engine,
	}
}

func (r *appRuntime) Close(logger *slog.Logger) {
	if err := r.producer.Close(); err != nil {
		logger.Error("close kafka producer failed", "error", err)
	}
	if err := r.db.Close(); err != nil {
		logger.Error("close mysql failed", "error", err)
	}
	if err := r.rdb.Close(); err != nil {
		logger.Error("close redis failed", "error", err)
	}
}

func newRouter(
	logger *slog.Logger,
	cfg config.Config,
	db *sql.DB,
	rdb *redis.Client,
	metrics *middleware.Metrics,
	producer *queue.Producer,
	repo *repository.CommentRepo,
	commentSvc *service.CommentService,
	likeSvc *service.LikeService,
) *gin.Engine {
	commentHandler := handler.NewCommentHandler(commentSvc)
	likeHandler := handler.NewLikeHandler(likeSvc)
	adminHandler := handler.NewAdminHandler(producer, likeSvc, repo, metrics)
	healthHandler := handler.NewHealthHandler(db, rdb, cfg.KafkaBrokers)

	limiter := ratelimit.NewSlidingWindowLimiter(rdb)
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(traceIDMiddleware())
	engine.Use(requestLogger(logger))
	engine.Use(middleware.CORS())
	engine.Use(metrics.HTTPMiddleware())

	registerBaseRoutes(engine, healthHandler, metrics)
	registerCommentRoutes(engine, commentHandler, likeHandler, limiter, cfg.RateLimit, metrics, logger)
	registerAdminRoutes(engine, adminHandler)
	return engine
}

func registerBaseRoutes(
	engine *gin.Engine,
	healthHandler *handler.HealthHandler,
	metrics *middleware.Metrics,
) {
	engine.GET("/health", healthHandler.Health)
	engine.GET("/metrics", metrics.MetricsHandler())
	registerPprofRoutes(engine)
}

func registerCommentRoutes(
	engine *gin.Engine,
	commentHandler *handler.CommentHandler,
	likeHandler *handler.LikeHandler,
	limiter *ratelimit.SlidingWindowLimiter,
	rateCfg config.RateLimitConfig,
	metrics *middleware.Metrics,
	logger *slog.Logger,
) {
	group := engine.Group("/api/v1/comments")
	group.Use(middleware.RateLimit(limiter, rateCfg, metrics, logger))
	group.POST("", commentHandler.Submit)
	group.GET("", commentHandler.List)
	group.GET("/hot", commentHandler.Hot)
	group.GET("/count", commentHandler.Count)
	group.POST("/:comment_id/like", likeHandler.Like)
	group.DELETE("/:comment_id/like", likeHandler.Unlike)
	group.GET("/:comment_id/like", likeHandler.HasLiked)
}

func registerAdminRoutes(engine *gin.Engine, adminHandler *handler.AdminHandler) {
	group := engine.Group("/api/v1/admin")
	group.POST("/comments/:comment_id/approve", adminHandler.Approve)
	group.POST("/comments/:comment_id/reject", adminHandler.Reject)
	group.POST("/sync-likes", adminHandler.SyncLikes)
}

func startBackgroundWorkers(
	wg *sync.WaitGroup,
	consumerCtx context.Context,
	syncCtx context.Context,
	runtime *appRuntime,
	logger *slog.Logger,
) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := runtime.consumer.Start(consumerCtx); err != nil {
			logger.Error("kafka consumer exited", "error", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runtime.likeService.StartSyncLoop(syncCtx, runtime.repository)
	}()
}

func startHTTPServer(server *http.Server, port string, logger *slog.Logger) chan error {
	errCh := make(chan error, 1)
	go func() {
		logger.Info("comment-service started", "port", port)
		errCh <- server.ListenAndServe()
	}()
	return errCh
}

func waitForShutdown(ctx context.Context, serverErrCh <-chan error) error {
	select {
	case <-ctx.Done():
		return nil
	case err := <-serverErrCh:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func shutdownServer(server *http.Server, logger *slog.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("shutdown http server failed", "error", err)
	}
}

func closeConsumer(consumer *queue.Consumer, logger *slog.Logger) {
	if err := consumer.Close(); err != nil {
		logger.Error("close kafka consumer failed", "error", err)
	}
}

func initMySQL(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	return db, nil
}

func initRedis(cfg config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		PoolSize:     32,
		MinIdleConns: 4,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return client, nil
}

func initMetrics() (*middleware.Metrics, error) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	metrics, err := middleware.NewMetrics(registry)
	if err != nil {
		return nil, fmt.Errorf("new metrics: %w", err)
	}
	return metrics, nil
}

func newHTTPServer(port string, router *gin.Engine) *http.Server {
	return &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func registerPprofRoutes(engine *gin.Engine) {
	group := engine.Group("/debug/pprof")
	group.GET("/", gin.WrapF(pprof.Index))
	group.GET("/cmdline", gin.WrapF(pprof.Cmdline))
	group.GET("/profile", gin.WrapF(pprof.Profile))
	group.POST("/symbol", gin.WrapF(pprof.Symbol))
	group.GET("/symbol", gin.WrapF(pprof.Symbol))
	group.GET("/trace", gin.WrapF(pprof.Trace))
	group.GET("/allocs", gin.WrapH(pprof.Handler("allocs")))
	group.GET("/block", gin.WrapH(pprof.Handler("block")))
	group.GET("/goroutine", gin.WrapH(pprof.Handler("goroutine")))
	group.GET("/heap", gin.WrapH(pprof.Handler("heap")))
	group.GET("/mutex", gin.WrapH(pprof.Handler("mutex")))
	group.GET("/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))
}
