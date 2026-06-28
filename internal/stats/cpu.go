package stats

import (
	"bufio"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

// ReadCPUModel returns the CPU model name from /proc/cpuinfo.
func (c *Collector) ReadCPUModel() string {
	if c.cpuModel != "" {
		return c.cpuModel
	}

	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return "Unknown"
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") || strings.HasPrefix(line, "Model") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				c.cpuModel = strings.TrimSpace(parts[1])
				return c.cpuModel
			}
		}
	}

	c.cpuModel = "Generic CPU"
	return c.cpuModel
}

// readProcStat parses /proc/stat and returns the per-core cpu times and total cpu times.
func readProcStat() ([]cpuTimes, cpuTimes, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return nil, cpuTimes{}, err
	}
	defer f.Close()

	var perCore []cpuTimes
	var total cpuTimes

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		if fields[0] == "cpu" {
			total = parseCpuFields(fields[1:])
			continue
		}

		if strings.HasPrefix(fields[0], "cpu") && len(fields[0]) > 3 {
			if _, err := strconv.Atoi(fields[0][3:]); err == nil {
				coreTimes := parseCpuFields(fields[1:])
				perCore = append(perCore, coreTimes)
			}
		}
	}

	return perCore, total, nil
}

func parseCpuFields(fields []string) cpuTimes {
	var total uint64
	var idle uint64

	for i, field := range fields {
		val, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			continue
		}
		total += val
		if i == 3 || i == 4 { // idle and iowait
			idle += val
		}
	}

	return cpuTimes{total: total, idle: idle}
}

// GetCPUStats collects CPU info, using cgroups where restricted, with bare-metal fallback.
func (c *Collector) GetCPUStats() (CPUStats, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	stats := CPUStats{
		ModelName: c.ReadCPUModel(),
	}

	// 1. Determine cores from cgroups first
	cgroupMaxPath, _ := FindCgroupFile("cpu.max")
	cgroupLimit := float64(-1)
	if cgroupMaxPath != "" {
		cgroupLimit = ParseCPUMax(cgroupMaxPath)
	}

	// Read /proc/stat for raw hardware cores & stats
	procPerCore, procTotal, err := readProcStat()
	if err != nil {
		return stats, err
	}

	rawCoreCount := len(procPerCore)
	if rawCoreCount == 0 {
		rawCoreCount = 1
	}

	// If cgroup CPU limit is active, use it as core count.
	// We round it to nearest integer.
	isCgroupLimited := cgroupLimit > 0
	var coreCount int
	if isCgroupLimited {
		coreCount = int(math.Ceil(cgroupLimit))
	} else {
		coreCount = rawCoreCount
	}
	stats.Cores = coreCount

	// 2. Calculate CPU usage
	timeNow := time.Now()
	var elapsed time.Duration
	if !c.prevTime.IsZero() {
		elapsed = timeNow.Sub(c.prevTime)
	}

	// CPU usage total percentage
	var usageTotal float64
	perCoreUsages := make([]float64, coreCount)

	if isCgroupLimited {
		// Calculate usage via cgroups cpu.stat
		cgroupStatPath, _ := FindCgroupFile("cpu.stat")
		var currentCPUTime int64
		if cgroupStatPath != "" {
			currentCPUTime = parseCPUStat(cgroupStatPath)
		}

		if c.prevCgroupCPUTime > 0 && elapsed > 0 {
			deltaCPUTime := currentCPUTime - c.prevCgroupCPUTime // in microseconds
			elapsedMicro := elapsed.Microseconds()
			if elapsedMicro > 0 {
				// total CPU usage percentage across all allocated capacity (0-100%)
				usageTotal = (float64(deltaCPUTime) / float64(elapsedMicro)) * 100 / cgroupLimit
				if usageTotal > 100 {
					usageTotal = 100
				}
				if usageTotal < 0 {
					usageTotal = 0
				}
			}
		}

		c.prevCgroupCPUTime = currentCPUTime

		// For per-core display in LXC:
		// If lxcfs has virtualized /proc/stat, the number of cores in /proc/stat matches cgroup core count.
		if len(procPerCore) == coreCount && len(c.prevProcPerCore) == coreCount {
			for i := 0; i < coreCount; i++ {
				dt := procPerCore[i].total - c.prevProcPerCore[i].total
				di := procPerCore[i].idle - c.prevProcPerCore[i].idle
				if dt > 0 {
					perCoreUsages[i] = float64(dt-di) / float64(dt) * 100
				}
			}
		} else {
			// If not virtualized, we distribute the total usage evenly
			for i := 0; i < coreCount; i++ {
				perCoreUsages[i] = usageTotal
			}
		}
	} else {
		// Bare metal calculation using /proc/stat
		if c.prevProcCPUTotal > 0 {
			dt := procTotal.total - c.prevProcCPUTotal
			di := procTotal.idle - c.prevProcCPUIdle
			if dt > 0 {
				usageTotal = float64(dt-di) / float64(dt) * 100
			}
		}

		// Per-core calculation
		if len(c.prevProcPerCore) == len(procPerCore) && len(procPerCore) > 0 {
			for i := 0; i < len(procPerCore); i++ {
				dt := procPerCore[i].total - c.prevProcPerCore[i].total
				di := procPerCore[i].idle - c.prevProcPerCore[i].idle
				if dt > 0 {
					perCoreUsages[i] = float64(dt-di) / float64(dt) * 100
				}
			}
		}
	}

	// Update previous states
	c.prevProcCPUTotal = procTotal.total
	c.prevProcCPUIdle = procTotal.idle
	c.prevProcPerCore = procPerCore
	c.prevTime = timeNow

	stats.UsageTotal = usageTotal
	stats.UsagePerCore = make([]CPUCoreInfo, coreCount)
	for i := 0; i < coreCount; i++ {
		stats.UsagePerCore[i] = CPUCoreInfo{
			ID:    i,
			Usage: perCoreUsages[i],
		}
	}

	return stats, nil
}
