package agent

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"

	models "github.com/faust8888/go-musthave-metrics/internal/model"
)

type Collector struct {
	mu        sync.Mutex
	gauges    map[string]float64
	pollCount int64
}

func NewCollector() *Collector {
	return &Collector{
		gauges: make(map[string]float64),
	}
}

func (c *Collector) Collect() {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.gauges["Alloc"] = float64(ms.Alloc)
	c.gauges["BuckHashSys"] = float64(ms.BuckHashSys)
	c.gauges["Frees"] = float64(ms.Frees)
	c.gauges["GCCPUFraction"] = ms.GCCPUFraction
	c.gauges["GCSys"] = float64(ms.GCSys)
	c.gauges["HeapAlloc"] = float64(ms.HeapAlloc)
	c.gauges["HeapIdle"] = float64(ms.HeapIdle)
	c.gauges["HeapInuse"] = float64(ms.HeapInuse)
	c.gauges["HeapObjects"] = float64(ms.HeapObjects)
	c.gauges["HeapReleased"] = float64(ms.HeapReleased)
	c.gauges["HeapSys"] = float64(ms.HeapSys)
	c.gauges["LastGC"] = float64(ms.LastGC)
	c.gauges["Lookups"] = float64(ms.Lookups)
	c.gauges["MCacheInuse"] = float64(ms.MCacheInuse)
	c.gauges["MCacheSys"] = float64(ms.MCacheSys)
	c.gauges["MSpanInuse"] = float64(ms.MSpanInuse)
	c.gauges["MSpanSys"] = float64(ms.MSpanSys)
	c.gauges["Mallocs"] = float64(ms.Mallocs)
	c.gauges["NextGC"] = float64(ms.NextGC)
	c.gauges["NumForcedGC"] = float64(ms.NumForcedGC)
	c.gauges["NumGC"] = float64(ms.NumGC)
	c.gauges["OtherSys"] = float64(ms.OtherSys)
	c.gauges["PauseTotalNs"] = float64(ms.PauseTotalNs)
	c.gauges["StackInuse"] = float64(ms.StackInuse)
	c.gauges["StackSys"] = float64(ms.StackSys)
	c.gauges["Sys"] = float64(ms.Sys)
	c.gauges["TotalAlloc"] = float64(ms.TotalAlloc)
	c.gauges["RandomValue"] = rand.Float64()

	c.pollCount++
}

// CollectGopsutil reads extra system gauges via gopsutil.
// CPUutilizationN is emitted once per logical CPU (1-based).
func (c *Collector) CollectGopsutil() {
	vm, vmErr := mem.VirtualMemory()
	percents, cpuErr := cpu.Percent(0, true)
	if cpuErr != nil || len(percents) == 0 {
		percents = make([]float64, runtime.NumCPU())
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if vmErr == nil {
		c.gauges["TotalMemory"] = float64(vm.Total)
		c.gauges["FreeMemory"] = float64(vm.Free)
	}
	for i, p := range percents {
		c.gauges[fmt.Sprintf("CPUutilization%d", i+1)] = p
	}
}

// Snapshot copies current gauges and the accumulated PollCount, then resets the counter.
func (c *Collector) Snapshot() []models.Metrics {
	return metricsFrom(c)
}

func (c *Collector) Gauges() map[string]float64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	snapshot := make(map[string]float64, len(c.gauges))
	for k, v := range c.gauges {
		snapshot[k] = v
	}
	return snapshot
}

func (c *Collector) PollCount() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.pollCount
}

func (c *Collector) TakeAndResetPollCount() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	v := c.pollCount
	c.pollCount = 0
	return v
}

func metricsFrom(c MetricsProvider) []models.Metrics {
	gauges := c.Gauges()
	pollCount := c.TakeAndResetPollCount()

	metrics := make([]models.Metrics, 0, len(gauges)+1)
	for name, value := range gauges {
		v := value
		metrics = append(metrics, models.Metrics{ID: name, MType: models.Gauge, Value: &v})
	}
	metrics = append(metrics, models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &pollCount})
	return metrics
}
