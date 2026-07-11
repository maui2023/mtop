package stats

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// GPUVendor identifies the GPU manufacturer.
type GPUVendor string

const (
	GPUVendorNvidia  GPUVendor = "NVIDIA"
	GPUVendorAMD     GPUVendor = "AMD"
	GPUVendorIntel   GPUVendor = "Intel"
	GPUVendorUnknown GPUVendor = "Unknown"
)

// GPUStats holds information about a detected GPU.
type GPUStats struct {
	Vendor      GPUVendor
	Name        string
	UsagePct    float64 // GPU core utilization 0-100
	MemUsedMB   uint64
	MemTotalMB  uint64
	TemperatureC float64
	Available   bool
}

// GetGPUStats detects and returns stats for the primary GPU found.
// Detection order: NVIDIA → AMD → Intel
func (c *Collector) GetGPUStats() []GPUStats {
	var gpus []GPUStats

	if nv := getNvidiaGPU(); nv.Available {
		gpus = append(gpus, nv)
	}
	if amd := getAMDGPU(); amd.Available {
		gpus = append(gpus, amd)
	}
	if intel := getIntelGPU(); intel.Available {
		gpus = append(gpus, intel)
	}

	return gpus
}

// getNvidiaGPU queries nvidia-smi for GPU info.
func getNvidiaGPU() GPUStats {
	// nvidia-smi --query-gpu=name,utilization.gpu,memory.used,memory.total,temperature.gpu --format=csv,noheader,nounits
	cmd := exec.Command("nvidia-smi",
		"--query-gpu=name,utilization.gpu,memory.used,memory.total,temperature.gpu",
		"--format=csv,noheader,nounits",
	)
	out, err := cmd.Output()
	if err != nil {
		return GPUStats{Vendor: GPUVendorNvidia}
	}

	line := strings.TrimSpace(string(out))
	// Take first GPU only
	if idx := strings.Index(line, "\n"); idx != -1 {
		line = line[:idx]
	}

	fields := strings.Split(line, ",")
	if len(fields) < 5 {
		return GPUStats{Vendor: GPUVendorNvidia}
	}

	g := GPUStats{
		Vendor:    GPUVendorNvidia,
		Name:      strings.TrimSpace(fields[0]),
		Available: true,
	}
	if v, err := strconv.ParseFloat(strings.TrimSpace(fields[1]), 64); err == nil {
		g.UsagePct = v
	}
	if v, err := strconv.ParseUint(strings.TrimSpace(fields[2]), 10, 64); err == nil {
		g.MemUsedMB = v
	}
	if v, err := strconv.ParseUint(strings.TrimSpace(fields[3]), 10, 64); err == nil {
		g.MemTotalMB = v
	}
	if v, err := strconv.ParseFloat(strings.TrimSpace(fields[4]), 64); err == nil {
		g.TemperatureC = v
	}
	return g
}

// getAMDGPU reads AMD GPU stats from sysfs (amdgpu driver) or rocm-smi.
func getAMDGPU() GPUStats {
	// First try rocm-smi
	if g := amdViaRocmSMI(); g.Available {
		return g
	}
	// Fallback to sysfs
	return amdViaSysfs()
}

func amdViaRocmSMI() GPUStats {
	// rocm-smi --showuse --showmeminfo vram --showtemp --csv
	cmd := exec.Command("rocm-smi", "--showuse", "--showmeminfo", "vram", "--showtemp", "--csv")
	out, err := cmd.Output()
	if err != nil {
		return GPUStats{Vendor: GPUVendorAMD}
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	// Find the data line after headers
	for _, line := range lines {
		if strings.HasPrefix(line, "card") || strings.HasPrefix(line, "0") {
			fields := strings.Split(line, ",")
			if len(fields) < 2 {
				continue
			}
			g := GPUStats{Vendor: GPUVendorAMD, Name: "AMD GPU (rocm-smi)", Available: true}
			// Try to parse usage from fields
			for _, f := range fields {
				f = strings.TrimSpace(f)
				if v, err := strconv.ParseFloat(strings.TrimSuffix(f, "%"), 64); err == nil && v >= 0 && v <= 100 {
					g.UsagePct = v
					break
				}
			}
			return g
		}
	}
	return GPUStats{Vendor: GPUVendorAMD}
}

func amdViaSysfs() GPUStats {
	// Look for amdgpu devices under /sys/class/drm/card*/device
	matches, err := filepath.Glob("/sys/class/drm/card*/device/gpu_busy_percent")
	if err != nil || len(matches) == 0 {
		return GPUStats{Vendor: GPUVendorAMD}
	}

	g := GPUStats{Vendor: GPUVendorAMD, Available: true}

	// Read GPU usage
	busyPath := matches[0]
	if data, err := os.ReadFile(busyPath); err == nil {
		if v, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64); err == nil {
			g.UsagePct = v
		}
	}

	// GPU name from uevent
	deviceDir := filepath.Dir(busyPath)
	if data, err := os.ReadFile(filepath.Join(deviceDir, "product_name")); err == nil {
		g.Name = strings.TrimSpace(string(data))
	} else {
		// Try reading vendor/device from uevent
		g.Name = amdNameFromUevent(filepath.Join(deviceDir, "uevent"))
	}
	if g.Name == "" {
		g.Name = "AMD GPU"
	}

	// VRAM usage from mem_info_vram_used / mem_info_vram_total
	baseDir := filepath.Dir(busyPath)
	if data, err := os.ReadFile(filepath.Join(baseDir, "mem_info_vram_used")); err == nil {
		if v, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64); err == nil {
			g.MemUsedMB = v / 1024 / 1024
		}
	}
	if data, err := os.ReadFile(filepath.Join(baseDir, "mem_info_vram_total")); err == nil {
		if v, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64); err == nil {
			g.MemTotalMB = v / 1024 / 1024
		}
	}

	// Temperature from hwmon
	hwmonGlob, _ := filepath.Glob(filepath.Join(deviceDir, "hwmon/hwmon*/temp1_input"))
	if len(hwmonGlob) > 0 {
		if data, err := os.ReadFile(hwmonGlob[0]); err == nil {
			if v, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64); err == nil {
				g.TemperatureC = v / 1000 // millidegrees → degrees
			}
		}
	}

	return g
}

func amdNameFromUevent(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "DRIVER=") {
			return "AMD GPU (" + strings.TrimPrefix(line, "DRIVER=") + ")"
		}
	}
	return ""
}

// getIntelGPU reads Intel GPU busy stats from sysfs (i915/xe driver).
func getIntelGPU() GPUStats {
	// Intel GPU engine utilization is in:
	// /sys/class/drm/card*/device/drm/card*/engine/render/busy
	// or newer xe driver paths.

	// Try i915 render engine busy path
	patterns := []string{
		"/sys/class/drm/card*/device/drm/card*/engine/render/busy",
		"/sys/class/drm/card*/gt/gt*/engines/render/busy",
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil || len(matches) == 0 {
			continue
		}
		return intelFromSysfs(matches[0])
	}

	// Also check /sys/class/drm/card*/device/vendor to verify Intel is present
	vendorMatches, _ := filepath.Glob("/sys/class/drm/card*/device/vendor")
	for _, vendorPath := range vendorMatches {
		data, err := os.ReadFile(vendorPath)
		if err != nil {
			continue
		}
		vendor := strings.TrimSpace(string(data))
		// Intel vendor ID is 0x8086
		if vendor == "0x8086" {
			return GPUStats{
				Vendor:    GPUVendorIntel,
				Name:      intelGPUName(filepath.Dir(vendorPath)),
				Available: true,
				// Usage not readable without kernel counter access
			}
		}
	}

	return GPUStats{Vendor: GPUVendorIntel}
}

func intelFromSysfs(busyPath string) GPUStats {
	g := GPUStats{
		Vendor:    GPUVendorIntel,
		Available: true,
	}
	deviceDir := filepath.Dir(filepath.Dir(filepath.Dir(busyPath)))
	g.Name = intelGPUName(deviceDir)
	if g.Name == "" {
		g.Name = "Intel GPU"
	}

	// Read GPU usage percentage directly from sysfs
	if data, err := os.ReadFile(busyPath); err == nil {
		if v, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64); err == nil {
			g.UsagePct = v
		}
	}

	return g
}

func intelGPUName(deviceDir string) string {
	// Try to read product name
	if data, err := os.ReadFile(filepath.Join(deviceDir, "product_name")); err == nil {
		return strings.TrimSpace(string(data))
	}
	// Read from /proc/cpuinfo for integrated GPU name
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return "Intel GPU"
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				model := strings.TrimSpace(parts[1])
				return "Intel GPU (" + model + ")"
			}
		}
	}
	return "Intel GPU"
}
