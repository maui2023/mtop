package stats

// GetSystemStats aggregates all statistical data from CPU, Memory, Disks, Networks, and Processes.
func (c *Collector) GetSystemStats() (SystemStats, error) {
	var stats SystemStats
	var err error

	// 1. CPU
	stats.CPU, err = c.GetCPUStats()
	if err != nil {
		return stats, err
	}

	// 2. Memory
	stats.Memory, err = c.GetMemoryStats()
	if err != nil {
		return stats, err
	}

	// 3. Disks
	stats.Disks, err = c.GetDiskStats()
	if err != nil {
		return stats, err
	}

	// 4. Networks
	stats.Networks, err = c.GetNetStats()
	if err != nil {
		return stats, err
	}

	// 5. Processes
	stats.Processes, err = c.GetProcessStats(stats.CPU.Cores, stats.Memory.Total)
	if err != nil {
		return stats, err
	}

	return stats, nil
}
