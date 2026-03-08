# 故障排查手册

## 1. 日常巡检 SOP

1. 查看容器状态。
```bash
docker-compose ps
```

2. 检查服务健康。
```bash
curl http://localhost:8094/health
curl http://localhost:8090/actuator/health/readiness
```

3. 查看资源使用。
```bash
docker stats --no-stream
```

4. 检查日志。
```bash
docker-compose logs --tail=100
```

## 2. CPU 飙升排查

1. 定位高 CPU 容器。
```bash
docker stats --no-stream | sort -k3 -rh
```

2. 采集 Go 服务 CPU Profile。
```bash
curl -o cpu.pprof "http://localhost:8094/debug/pprof/profile?seconds=30"
```

3. 常见原因。
- Kafka 消费者重试风暴（消息处理失败且持续重试）。
- Redis 连接池耗尽导致请求阻塞与自旋重试。
- Lua 限流脚本执行过长（大 Key 或热点 Key）。

## 3. 内存飙升排查

1. 采集 Go 服务 Heap Profile。
```bash
curl -o heap.pprof "http://localhost:8094/debug/pprof/heap"
```

2. 检查 MySQL 线程状态。
```sql
SHOW PROCESSLIST;
SHOW STATUS LIKE 'Threads%';
```

3. 常见原因。
- Kafka 批量消息过大，消费者堆积内存。
- Redis 缓存 Key 无 TTL，内存持续增长。
- MySQL 连接泄漏，连接数长期不回落。

## 4. Kafka 问题排查

1. 查看 Topic 状态。
```bash
kafka-topics.sh --bootstrap-server localhost:9092 --describe --topic comment-events
```

2. 查看消费组 lag。
```bash
kafka-consumer-groups.sh --bootstrap-server localhost:9092 --describe --group comment-service-group
```

3. 积压处理建议。
- 短期：提高 comment-service 消费并发或扩实例。
- 长期：排查 MySQL 慢 SQL 与索引命中情况。

## 5. Redis 问题排查

1. 查看连接数。
```bash
redis-cli -a "$REDIS_PASSWORD" INFO clients
```

2. 查看内存使用。
```bash
redis-cli -a "$REDIS_PASSWORD" INFO memory
```

3. 查看慢查询。
```bash
redis-cli -a "$REDIS_PASSWORD" SLOWLOG GET 10
```

## 6. 数据一致性排查

1. Redis 与 MySQL 不一致。
- 删除对应 Redis Key 触发回源刷新。
- 检查 Kafka 消费组 lag 是否持续增长。

2. 点赞数不一致。
- 检查 `like:sync:pending` 是否持续堆积。
- 手动触发同步：
```bash
curl -X POST http://localhost:8094/api/v1/admin/sync-likes
```
