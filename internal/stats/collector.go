package stats

import (
	"sync"
	"time"
)

type CPUCoreInfo struct {
	ID    int
	Usage float64
}

type CPUStats struct {
	ModelName    string
	Cores        int
	UsageTotal   float64
	UsagePerCore []CPUCoreInfo
}

type MemoryStats struct {
	Total     uint64
	Used      uint64
	Free      uint64
	UsagePct  float64
	SwapTotal uint64
	SwapUsed  uint64
	SwapFree  uint64
}

type DiskInfo struct {
	MountPoint string
	Device     string
	Total      uint64
	Used       uint64
	Free       uint64
	UsagePct   float64
}

type NetInterface struct {
	Name      string
	RxBytes   uint64
	TxBytes   uint64
	RxRate    float64 // bytes/sec
	TxRate    float64 // bytes/sec
}

type ProcessInfo struct {
	PID     int
	Name    string
	CPU     float64
	Memory  float64 // percentage
	MemSize uint64  // bytes
	State   string
	User    string
}

type SystemStats struct {
	CPU      CPUStats
	Memory   MemoryStats
	Disks    []DiskInfo
	Networks []NetInterface
	Processes []ProcessInfo
}

type Collector struct {
	mu sync.Mutex

	// Previous state for CPU
	prevCgroupCPUTime  int64     // nanoseconds or microseconds
	prevProcCPUTotal   uint64
	prevProcCPUIdle    uint64
	prevProcCPUNum     int
	prevProcPerCore    []cpuTimes // cpu0, cpu1...
	prevTime           time.Time

	// Previous state for Net
	prevNetBytes       map[string]netBytes
	prevNetTime        time.Time

	// CPU Model cache
	cpuModel string
}

type cpuTimes struct {
	total uint64
	idle  uint64
}

type netBytes struct {
	rx uint64
	tx uint64
}

func NewCollector() *Collector {
	return &Collector{
		prevNetBytes: make(map[string]netBytes),
	}
}
