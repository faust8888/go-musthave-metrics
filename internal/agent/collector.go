package agent

import (
	"math/rand"
	"runtime"
	"sync"
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

func (c *Collector) ResetPollCount() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pollCount = 0
}
