package stats

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var (
	passwdCache = make(map[string]string)
	passwdOnce  sync.Once
)

const clkTck = 100

// loadPasswdCache loads usernames from /etc/passwd.
func loadPasswdCache() {
	f, err := os.Open("/etc/passwd")
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) >= 3 {
			username := parts[0]
			uid := parts[2]
			passwdCache[uid] = username
		}
	}
}

// resolveUID converts UID to username using the cached /etc/passwd file.
func resolveUID(uid string) string {
	passwdOnce.Do(loadPasswdCache)
	if name, exists := passwdCache[uid]; exists {
		return name
	}
	return uid
}

type procSample struct {
	ticks       uint64
	systemTotal uint64
}

var (
	procHistory = make(map[int]procSample)
	historyMu   sync.Mutex
)

// GetProcessStats collects process stats for all running processes.
func (c *Collector) GetProcessStats(cores int, totalMem uint64) ([]ProcessInfo, error) {
	historyMu.Lock()
	defer historyMu.Unlock()

	// Read total system ticks from /proc/stat
	_, systemTotalTimes, err := readProcStat()
	if err != nil {
		return nil, err
	}
	systemTotal := systemTotalTimes.total

	files, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	pageSize := uint64(os.Getpagesize())
	var processes []ProcessInfo
	newHistory := make(map[int]procSample)

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(file.Name())
		if err != nil {
			continue // Skip non-numeric directories
		}

		procDir := filepath.Join("/proc", file.Name())

		// Read /proc/<PID>/stat
		statBytes, err := os.ReadFile(filepath.Join(procDir, "stat"))
		if err != nil {
			continue // Process might have terminated
		}

		statStr := string(statBytes)
		lastCloseParen := strings.LastIndex(statStr, ")")
		if lastCloseParen == -1 {
			continue
		}

		// Field 1 is PID, Field 2 is (name) which can contain spaces, so split after last )
		name := statStr[strings.Index(statStr, "(")+1 : lastCloseParen]
		afterParen := strings.Fields(statStr[lastCloseParen+1:])
		if len(afterParen) < 20 {
			continue
		}

		state := afterParen[0]
		utime, _ := strconv.ParseUint(afterParen[11], 10, 64) // field 14 in stat
		stime, _ := strconv.ParseUint(afterParen[12], 10, 64) // field 15 in stat
		rss, _ := strconv.ParseUint(afterParen[21], 10, 64)   // field 24 in stat (pages)

		procTicks := utime + stime

		// Read /proc/<PID>/status to get UID
		statusFile, err := os.Open(filepath.Join(procDir, "status"))
		uidStr := "unknown"
		if err == nil {
			scanner := bufio.NewScanner(statusFile)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "Uid:") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						uidStr = fields[1]
						break
					}
				}
			}
			statusFile.Close()
		}

		// Calculate CPU usage
		var cpuUsage float64
		prevSample, exists := procHistory[pid]
		if exists {
			deltaProc := procTicks - prevSample.ticks
			deltaSys := systemTotal - prevSample.systemTotal
			if deltaSys > 0 {
				// CPU usage scaled to represent 0-100% of a single core
				cpuUsage = (float64(deltaProc) / float64(deltaSys)) * 100 * float64(cores)
				// Cap it at cores * 100
				if cpuUsage > float64(cores)*100 {
					cpuUsage = float64(cores) * 100
				}
			}
		}

		newHistory[pid] = procSample{ticks: procTicks, systemTotal: systemTotal}

		memSize := rss * pageSize
		var memUsagePct float64
		if totalMem > 0 {
			memUsagePct = float64(memSize) / float64(totalMem) * 100
		}

		processes = append(processes, ProcessInfo{
			PID:     pid,
			Name:    name,
			CPU:     cpuUsage,
			Memory:  memUsagePct,
			MemSize: memSize,
			State:   state,
			User:    resolveUID(uidStr),
		})
	}

	// Keep history only for running processes
	procHistory = newHistory

	return processes, nil
}
