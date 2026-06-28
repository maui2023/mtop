package stats

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// getSelfCgroupPath reads /proc/self/cgroup and returns the relative path.
func getSelfCgroupPath() string {
	f, err := os.Open("/proc/self/cgroup")
	if err != nil {
		return "/"
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) >= 3 && parts[1] == "" { // cgroup v2 has empty controller name (0::/path)
			return parts[2]
		}
	}
	return "/"
}

// FindCgroupFile searches for the specified file starting from the process's leaf cgroup up to the root.
func FindCgroupFile(filename string) (string, string) {
	relPath := getSelfCgroupPath()
	baseDir := "/sys/fs/cgroup"

	curr := relPath
	for {
		fullPath := filepath.Join(baseDir, curr, filename)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath, filepath.Join(baseDir, curr)
		}

		if curr == "/" || curr == "." || curr == "" {
			break
		}
		curr = filepath.Dir(curr)
	}

	// Fallback to root sysfs path if not found in traversal
	fullPath := filepath.Join(baseDir, filename)
	if _, err := os.Stat(fullPath); err == nil {
		return fullPath, baseDir
	}

	return "", ""
}

// ParseCPUMax parses a cpu.max file and returns the core limit.
// If limit is not set (max) or fails, returns -1.
func ParseCPUMax(path string) float64 {
	content, err := os.ReadFile(path)
	if err != nil {
		return -1
	}

	fields := strings.Fields(string(content))
	if len(fields) < 2 {
		return -1
	}

	quotaStr := fields[0]
	periodStr := fields[1]

	if quotaStr == "max" {
		return -1
	}

	quota, err := strconv.ParseFloat(quotaStr, 64)
	if err != nil {
		return -1
	}

	period, err := strconv.ParseFloat(periodStr, 64)
	if err != nil {
		return -1
	}

	if period == 0 {
		return -1
	}

	return quota / period
}

// parseMemoryMax parses memory.max file and returns limit in bytes.
// If not set (max) or fails, returns 0.
func parseMemoryMax(path string) uint64 {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0
	}

	valStr := strings.TrimSpace(string(content))
	if valStr == "max" {
		return 0
	}

	val, err := strconv.ParseUint(valStr, 10, 64)
	if err != nil {
		return 0
	}

	return val
}

// readCgroupMetric reads a single uint64 value from a file.
func readCgroupMetric(path string) uint64 {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	val, err := strconv.ParseUint(strings.TrimSpace(string(content)), 10, 64)
	if err != nil {
		return 0
	}
	return val
}

// parseCPUStat reads usage_usec from a cpu.stat file.
func parseCPUStat(path string) int64 {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	return parseCPUStatFromReader(f)
}

func parseCPUStatFromReader(r io.Reader) int64 {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "usage_usec" {
			val, err := strconv.ParseInt(fields[1], 10, 64)
			if err == nil {
				return val
			}
		}
	}
	return 0
}
