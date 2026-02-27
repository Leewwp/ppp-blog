package collector

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

const (
	DefaultHistorySize = 60
)

type SystemSnapshot struct {
	Timestamp time.Time `json:"timestamp"`

	CPUUsagePercent float64 `json:"cpu_usage_percent"`

	MemoryUsedBytes    uint64  `json:"memory_used_bytes"`
	MemoryTotalBytes   uint64  `json:"memory_total_bytes"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`

	DiskUsedBytes    uint64  `json:"disk_used_bytes"`
	DiskTotalBytes   uint64  `json:"disk_total_bytes"`
	DiskUsagePercent float64 `json:"disk_usage_percent"`

	NetworkBytesSentPerSec float64 `json:"network_bytes_sent_per_sec"`
	NetworkBytesRecvPerSec float64 `json:"network_bytes_recv_per_sec"`

	Load1  float64 `json:"load_1"`
	Load5  float64 `json:"load_5"`
	Load15 float64 `json:"load_15"`

	ProcessCount int `json:"process_count"`
}

type SystemCollector struct {
	mu       sync.RWMutex
	history  []SystemSnapshot
	capacity int
	interval time.Duration
	logger   *slog.Logger

	lastNetworkSample *networkSample

	startOnce sync.Once
	updates   chan SystemSnapshot
}

type networkSample struct {
	timestamp time.Time
	sent      uint64
	recv      uint64
}

func NewSystemCollector(interval time.Duration, capacity int, logger *slog.Logger) *SystemCollector {
	if logger == nil {
		logger = slog.Default()
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}
	if capacity <= 0 {
		capacity = DefaultHistorySize
	}

	return &SystemCollector{
		history:  make([]SystemSnapshot, 0, capacity),
		capacity: capacity,
		interval: interval,
		logger:   logger,
		updates:  make(chan SystemSnapshot, 1),
	}
}

// Start runs background collection until ctx is canceled.
// The returned channel emits snapshots whenever a new sample is collected.
func (c *SystemCollector) Start(ctx context.Context) <-chan SystemSnapshot {
	c.startOnce.Do(func() {
		go c.run(ctx)
	})
	return c.updates
}

func (c *SystemCollector) run(ctx context.Context) {
	defer close(c.updates)

	c.collectStoreAndPublish()

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("system collector stopped")
			return
		case <-ticker.C:
			c.collectStoreAndPublish()
		}
	}
}

func (c *SystemCollector) collectStoreAndPublish() {
	snapshot, err := c.collectOnce()
	if err != nil {
		c.logger.Warn("collect system metrics with partial errors", "error", err)
	}
	c.appendSnapshot(snapshot)
	c.publish(snapshot)
}

func (c *SystemCollector) publish(snapshot SystemSnapshot) {
	select {
	case c.updates <- snapshot:
	default:
		select {
		case <-c.updates:
		default:
		}
		select {
		case c.updates <- snapshot:
		default:
		}
	}
}

func (c *SystemCollector) Current() (SystemSnapshot, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.history) == 0 {
		return SystemSnapshot{}, false
	}
	return c.history[len(c.history)-1], true
}

func (c *SystemCollector) History(limit int) []SystemSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if limit <= 0 || limit > len(c.history) {
		limit = len(c.history)
	}
	start := len(c.history) - limit

	out := make([]SystemSnapshot, limit)
	copy(out, c.history[start:])
	return out
}

func (c *SystemCollector) HistorySince(duration time.Duration) []SystemSnapshot {
	if duration <= 0 {
		return c.History(c.capacity)
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.history) == 0 {
		return []SystemSnapshot{}
	}

	cutoff := time.Now().Add(-duration)
	filtered := make([]SystemSnapshot, 0, len(c.history))
	for _, s := range c.history {
		if s.Timestamp.After(cutoff) || s.Timestamp.Equal(cutoff) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func (c *SystemCollector) appendSnapshot(snapshot SystemSnapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.history) >= c.capacity {
		copy(c.history, c.history[1:])
		c.history[len(c.history)-1] = snapshot
		return
	}
	c.history = append(c.history, snapshot)
}

func (c *SystemCollector) collectOnce() (SystemSnapshot, error) {
	now := time.Now()
	snapshot := SystemSnapshot{
		Timestamp: now,
	}
	errs := make([]string, 0)

	if cpuPercent, err := cpu.Percent(0, false); err == nil && len(cpuPercent) > 0 {
		snapshot.CPUUsagePercent = cpuPercent[0]
	} else if err != nil {
		errs = append(errs, fmt.Sprintf("cpu: %v", err))
	}

	if vm, err := mem.VirtualMemory(); err == nil {
		snapshot.MemoryUsedBytes = vm.Used
		snapshot.MemoryTotalBytes = vm.Total
		snapshot.MemoryUsagePercent = vm.UsedPercent
	} else {
		errs = append(errs, fmt.Sprintf("memory: %v", err))
	}

	root := rootPartitionPath()
	if du, err := disk.Usage(root); err == nil {
		snapshot.DiskUsedBytes = du.Used
		snapshot.DiskTotalBytes = du.Total
		snapshot.DiskUsagePercent = du.UsedPercent
	} else {
		errs = append(errs, fmt.Sprintf("disk(%s): %v", root, err))
	}

	if ioCounters, err := net.IOCounters(false); err == nil && len(ioCounters) > 0 {
		sentPerSec, recvPerSec := c.computeNetworkRate(now, ioCounters[0].BytesSent, ioCounters[0].BytesRecv)
		snapshot.NetworkBytesSentPerSec = sentPerSec
		snapshot.NetworkBytesRecvPerSec = recvPerSec
	} else if err != nil {
		errs = append(errs, fmt.Sprintf("network: %v", err))
	}

	if avg, err := load.Avg(); err == nil {
		snapshot.Load1 = avg.Load1
		snapshot.Load5 = avg.Load5
		snapshot.Load15 = avg.Load15
	} else {
		errs = append(errs, fmt.Sprintf("load average: %v", err))
	}

	if pids, err := process.Pids(); err == nil {
		snapshot.ProcessCount = len(pids)
	} else {
		errs = append(errs, fmt.Sprintf("process count: %v", err))
	}

	if len(errs) > 0 {
		return snapshot, errors.New(strings.Join(errs, "; "))
	}
	return snapshot, nil
}

func (c *SystemCollector) computeNetworkRate(now time.Time, sent, recv uint64) (float64, float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	curr := &networkSample{
		timestamp: now,
		sent:      sent,
		recv:      recv,
	}
	if c.lastNetworkSample == nil {
		c.lastNetworkSample = curr
		return 0, 0
	}

	prev := c.lastNetworkSample
	c.lastNetworkSample = curr

	deltaSec := curr.timestamp.Sub(prev.timestamp).Seconds()
	if deltaSec <= 0 {
		return 0, 0
	}

	var sentPerSec float64
	if curr.sent >= prev.sent {
		sentPerSec = float64(curr.sent-prev.sent) / deltaSec
	}
	var recvPerSec float64
	if curr.recv >= prev.recv {
		recvPerSec = float64(curr.recv-prev.recv) / deltaSec
	}

	return sentPerSec, recvPerSec
}

func rootPartitionPath() string {
	if runtime.GOOS == "windows" {
		return `C:\`
	}
	return "/"
}
