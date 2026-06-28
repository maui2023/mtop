package stats

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// parseMeminfo parses /proc/meminfo and returns a map of keys to bytes.
func parseMeminfo() (map[string]uint64, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	metrics := make(map[string]uint64)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		valFields := strings.Fields(parts[1])
		if len(valFields) == 0 {
			continue
		}
		val, err := strconv.ParseUint(valFields[0], 10, 64)
		if err != nil {
			continue
		}

		// Values in /proc/meminfo are in kB, convert to bytes
		metrics[key] = val * 1024
	}

	return metrics, nil
}

// parseMemoryStat parses a memory.stat file under cgroups v2.
func parseMemoryStat(path string) map[string]uint64 {
	stats := make(map[string]uint64)
	f, err := os.Open(path)
	if err != nil {
		return stats
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 {
			val, err := strconv.ParseUint(fields[1], 10, 64)
			if err == nil {
				stats[fields[0]] = val
			}
		}
	}
	return stats
}

// GetMemoryStats gathers Memory & Swap stats, using cgroups if restricted, with bare-metal fallback.
func (c *Collector) GetMemoryStats() (MemoryStats, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var stats MemoryStats

	// 1. Try to find cgroup v2 memory limit and current usage
	cgroupMaxPath, cgroupDir := FindCgroupFile("memory.max")
	var cgroupLimit uint64
	if cgroupMaxPath != "" {
		cgroupLimit = parseMemoryMax(cgroupMaxPath)
	}

	meminfo, err := parseMeminfo()
	if err != nil {
		return stats, err
	}

	isCgroupLimited := cgroupLimit > 0

	if isCgroupLimited {
		// Memory Limit is determined by cgroup limit
		stats.Total = cgroupLimit

		// Memory Usage is determined by memory.current
		memCurrentPath := findCgroupFileInDir(cgroupDir, "memory.current")
		memCurrent := readCgroupMetric(memCurrentPath)

		// Page cache subtraction to get "Working Set" memory
		memStatPath := findCgroupFileInDir(cgroupDir, "memory.stat")
		var cacheSize uint64
		if memStatPath != "" {
			memStats := parseMemoryStat(memStatPath)
			// inactive_file is commonly considered reclaimable cache
			cacheSize = memStats["inactive_file"]
		}

		if memCurrent > cacheSize {
			stats.Used = memCurrent - cacheSize
		} else {
			stats.Used = memCurrent
		}

		if stats.Used > stats.Total {
			stats.Used = stats.Total
		}
		stats.Free = stats.Total - stats.Used

		// Swap stats under cgroup
		swapMaxPath := findCgroupFileInDir(cgroupDir, "memory.swap.max")
		swapCurrentPath := findCgroupFileInDir(cgroupDir, "memory.swap.current")

		swapLimit := parseMemoryMax(swapMaxPath)
		swapCurrent := readCgroupMetric(swapCurrentPath)

		if swapLimit > 0 {
			stats.SwapTotal = swapLimit
			stats.SwapUsed = swapCurrent
			if stats.SwapUsed > stats.SwapTotal {
				stats.SwapUsed = stats.SwapTotal
			}
			stats.SwapFree = stats.SwapTotal - stats.SwapUsed
		} else {
			// Fallback to proc meminfo swap if cgroups swap limit isn't configured/available
			stats.SwapTotal = meminfo["SwapTotal"]
			stats.SwapFree = meminfo["SwapFree"]
			stats.SwapUsed = stats.SwapTotal - stats.SwapFree
		}

	} else {
		// Bare metal memory stats from /proc/meminfo
		stats.Total = meminfo["MemTotal"]
		stats.Free = meminfo["MemAvailable"]
		if stats.Free > stats.Total {
			stats.Free = stats.Total
		}
		stats.Used = stats.Total - stats.Free

		stats.SwapTotal = meminfo["SwapTotal"]
		stats.SwapFree = meminfo["SwapFree"]
		stats.SwapUsed = stats.SwapTotal - stats.SwapFree
	}

	if stats.Total > 0 {
		stats.UsagePct = float64(stats.Used) / float64(stats.Total) * 100
	}

	return stats, nil
}

// findCgroupFileInDir is a small helper to find a file in the cgroup directory, or fallback to the root cgroup dir.
func findCgroupFileInDir(dir, filename string) string {
	if dir != "" {
		p := dir + "/" + filename
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	p := "/sys/fs/cgroup/" + filename
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}
