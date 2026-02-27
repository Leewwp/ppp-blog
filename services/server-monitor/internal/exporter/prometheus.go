package exporter

import (
	"github.com/gin-gonic/gin"
	"github.com/ppp-blog/server-monitor/internal/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PrometheusExporter struct {
	cpuUsagePercent prometheus.Gauge

	memoryUsedBytes    prometheus.Gauge
	memoryTotalBytes   prometheus.Gauge
	memoryUsagePercent prometheus.Gauge

	diskUsedBytes    prometheus.Gauge
	diskTotalBytes   prometheus.Gauge
	diskUsagePercent prometheus.Gauge

	networkSentPerSec prometheus.Gauge
	networkRecvPerSec prometheus.Gauge

	load1  prometheus.Gauge
	load5  prometheus.Gauge
	load15 prometheus.Gauge

	processCount prometheus.Gauge

	gatherer prometheus.Gatherer
}

func NewPrometheusExporter(registry *prometheus.Registry) (*PrometheusExporter, error) {
	e := &PrometheusExporter{
		cpuUsagePercent: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "server_cpu_usage_percent",
			Help: "Current CPU usage percentage.",
		}),
		memoryUsedBytes: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "server_memory_used_bytes",
			Help: "Current memory used in bytes.",
		}),
		memoryTotalBytes: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "server_memory_total_bytes",
			Help: "Current total memory in bytes.",
		}),
		memoryUsagePercent: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "server_memory_usage_percent",
			Help: "Current memory usage percentage.",
		}),
		diskUsedBytes: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "server_disk_used_bytes",
			Help: "Current disk used bytes for root partition.",
		}),
		diskTotalBytes: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "server_disk_total_bytes",
			Help: "Current disk total bytes for root partition.",
		}),
		diskUsagePercent: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "server_disk_usage_percent",
			Help: "Current disk usage percentage for root partition.",
		}),
		networkSentPerSec: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "server_network_bytes_sent_per_second",
			Help: "Current network sent throughput in bytes/s.",
		}),
		networkRecvPerSec: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "server_network_bytes_recv_per_second",
			Help: "Current network recv throughput in bytes/s.",
		}),
		load1: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "server_load_1",
			Help: "Load average in 1 minute.",
		}),
		load5: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "server_load_5",
			Help: "Load average in 5 minutes.",
		}),
		load15: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "server_load_15",
			Help: "Load average in 15 minutes.",
		}),
		processCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "server_process_count",
			Help: "Current number of processes.",
		}),
		gatherer: registry,
	}

	collectors := []prometheus.Collector{
		e.cpuUsagePercent,
		e.memoryUsedBytes, e.memoryTotalBytes, e.memoryUsagePercent,
		e.diskUsedBytes, e.diskTotalBytes, e.diskUsagePercent,
		e.networkSentPerSec, e.networkRecvPerSec,
		e.load1, e.load5, e.load15,
		e.processCount,
	}
	for _, collector := range collectors {
		if err := registry.Register(collector); err != nil {
			return nil, err
		}
	}
	return e, nil
}

func (e *PrometheusExporter) Update(snapshot collector.SystemSnapshot) {
	e.cpuUsagePercent.Set(snapshot.CPUUsagePercent)

	e.memoryUsedBytes.Set(float64(snapshot.MemoryUsedBytes))
	e.memoryTotalBytes.Set(float64(snapshot.MemoryTotalBytes))
	e.memoryUsagePercent.Set(snapshot.MemoryUsagePercent)

	e.diskUsedBytes.Set(float64(snapshot.DiskUsedBytes))
	e.diskTotalBytes.Set(float64(snapshot.DiskTotalBytes))
	e.diskUsagePercent.Set(snapshot.DiskUsagePercent)

	e.networkSentPerSec.Set(snapshot.NetworkBytesSentPerSec)
	e.networkRecvPerSec.Set(snapshot.NetworkBytesRecvPerSec)

	e.load1.Set(snapshot.Load1)
	e.load5.Set(snapshot.Load5)
	e.load15.Set(snapshot.Load15)

	e.processCount.Set(float64(snapshot.ProcessCount))
}

func (e *PrometheusExporter) Handler() gin.HandlerFunc {
	handler := promhttp.HandlerFor(e.gatherer, promhttp.HandlerOpts{})
	return func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
	}
}
