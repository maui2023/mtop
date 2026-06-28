package stats

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

// GetNetStats parses /proc/net/dev to retrieve network interface stats and calculate rx/tx rates.
func (c *Collector) GetNetStats() ([]NetInterface, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	now := time.Now()
	var elapsed float64
	if !c.prevNetTime.IsZero() {
		elapsed = now.Sub(c.prevNetTime).Seconds()
	}

	var interfaces []NetInterface
	scanner := bufio.NewScanner(f)
	
	// Skip the first 2 header lines
	if scanner.Scan() { // Header 1
		_ = scanner.Text()
	}
	if scanner.Scan() { // Header 2
		_ = scanner.Text()
	}

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		
		// Filter out local loopback and common virtual interfaces to keep it clean
		if name == "lo" || strings.HasPrefix(name, "veth") || strings.HasPrefix(name, "docker") || strings.HasPrefix(name, "br-") {
			continue
		}

		fields := strings.Fields(parts[1])
		if len(fields) < 9 {
			continue
		}

		rxBytes, err1 := strconv.ParseUint(fields[0], 10, 64)
		txBytes, err2 := strconv.ParseUint(fields[8], 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}

		var rxRate, txRate float64
		prev, exists := c.prevNetBytes[name]
		if exists && elapsed > 0 {
			if rxBytes >= prev.rx {
				rxRate = float64(rxBytes-prev.rx) / elapsed
			}
			if txBytes >= prev.tx {
				txRate = float64(txBytes-prev.tx) / elapsed
			}
		}

		c.prevNetBytes[name] = netBytes{rx: rxBytes, tx: txBytes}

		interfaces = append(interfaces, NetInterface{
			Name:    name,
			RxBytes: rxBytes,
			TxBytes: txBytes,
			RxRate:  rxRate,
			TxRate:  txRate,
		})
	}

	c.prevNetTime = now
	return interfaces, nil
}
