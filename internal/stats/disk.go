package stats

import (
	"bufio"
	"os"
	"strings"
	"golang.org/x/sys/unix"
)

var ignoredFSTypes = map[string]bool{
	"proc":          true,
	"sysfs":         true,
	"devtmpfs":      true,
	"devpts":        true,
	"tmpfs":         true,
	"cgroup":        true,
	"cgroup2":       true,
	"pstore":        true,
	"configfs":      true,
	"securityfs":    true,
	"autofs":        true,
	"hugetlbfs":     true,
	"mqueue":        true,
	"debugfs":       true,
	"tracefs":       true,
	"binfmt_misc":   true,
	"fusectl":       true,
	"nsfs":          true,
	"bpf":           true,
	"selinuxfs":     true,
	"devfs":         true,
}

// GetDiskStats collects disk size and usage metrics for mounted filesystems.
func (c *Collector) GetDiskStats() ([]DiskInfo, error) {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var disks []DiskInfo
	seenMounts := make(map[string]bool)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		device := fields[0]
		mountPoint := fields[1]
		fsType := fields[2]

		// Skip ignored filesystems
		if ignoredFSTypes[fsType] {
			continue
		}

		// Avoid duplicates
		if seenMounts[mountPoint] {
			continue
		}

		// Ignore typical Docker/LXC internal mounts unless it's root or a real mount
		if strings.HasPrefix(mountPoint, "/sys/") ||
			strings.HasPrefix(mountPoint, "/proc/") ||
			strings.HasPrefix(mountPoint, "/dev/") && mountPoint != "/dev" {
			continue
		}

		// Perform statfs system call
		var stat unix.Statfs_t
		err := unix.Statfs(mountPoint, &stat)
		if err != nil {
			continue
		}

		bsize := uint64(stat.Bsize)
		// On some systems stat.Frsize is used for block size instead of Bsize.
		// If Frsize is available and non-zero, it is the fundamental block size.
		// Go's unix.Statfs_t on Linux has Frsize.
		if stat.Frsize > 0 {
			bsize = uint64(stat.Frsize)
		}

		total := stat.Blocks * bsize
		if total == 0 {
			continue
		}

		free := stat.Bfree * bsize
		avail := stat.Bavail * bsize
		used := total - free

		usagePct := float64(used) / float64(total) * 100

		disks = append(disks, DiskInfo{
			MountPoint: mountPoint,
			Device:     device,
			Total:      total,
			Used:       used,
			Free:       avail, // Bavail is what's actually usable by normal users
			UsagePct:   usagePct,
		})

		seenMounts[mountPoint] = true
	}

	return disks, nil
}
