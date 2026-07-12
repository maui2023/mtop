package ui

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/maui/mtop/internal/stats"
)

// Curated dark mode color palette
var (
	colorCyan      = lipgloss.Color("#00f5d4") // Neon Cyan
	colorBlue      = lipgloss.Color("#00bbf9") // Bright Blue
	colorPurple    = lipgloss.Color("#9b5de5") // Vivid Purple
	colorPink      = lipgloss.Color("#f15bb5") // Hot Pink
	colorGold      = lipgloss.Color("#fee440") // Gold
	colorText      = lipgloss.Color("#f8fafc") // Off-white
	colorDim       = lipgloss.Color("#64748b") // Dim gray
	colorBorder    = lipgloss.Color("#334155") // Slate-700
	colorHighlight = lipgloss.Color("#1e293b") // Dark Slate (cursor bg)
	colorWarn      = lipgloss.Color("#ef4444") // Coral Red

	// Styles
	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorCyan).
			Padding(0, 1)

	styleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder)

	styleBoxTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorBlue)

	styleHeaderLabel = lipgloss.NewStyle().
				Foreground(colorDim)

	styleHeaderVal = lipgloss.NewStyle().
				Foreground(colorText).
				Bold(true)
)

func (m Model) View() string {
	if !m.ready {
		return "\n  Gathering system statistics..."
	}

	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n  Press 'r' to retry, 'q' to quit.", m.err)
	}

	// Safety check for terminal sizes
	if m.width < 70 || m.height < 20 {
		return fmt.Sprintf(
			"\n  Terminal size too small (%dx%d).\n  Please resize to at least 70x20.\n  Press 'q' to exit.",
			m.width, m.height,
		)
	}

	// 1. Render Header
	header := m.renderHeader()

	// 2. Render CPU Box (Row 1 - Full Width)
	cpuHeight := 12
	cpuBox := m.renderCPUBox(m.width, cpuHeight)

	// 3. Render GPU & Memory (RAM) Box (Row 2 - Balanced 50%/50%)
	gpuRamHeight := 7
	gpuWidth := m.width / 2
	ramWidth := m.width - gpuWidth

	gpuBox := m.renderGPUBox(gpuWidth, gpuRamHeight)
	ramBox := m.renderMemoryBox(ramWidth, gpuRamHeight)

	row2 := lipgloss.JoinHorizontal(lipgloss.Top, gpuBox, ramBox)

	// 4. Render Disk & Net Box (Row 3 - Balanced 50%/50%)
	sec2Height := 9
	diskWidth := m.width / 2
	netWidth := m.width - diskWidth

	diskBox := m.renderDiskBox(diskWidth, sec2Height)
	netBox := m.renderNetBox(netWidth, sec2Height)

	row3 := lipgloss.JoinHorizontal(lipgloss.Top, diskBox, netBox)

	// 5. Render Processes Box
	procHeight := m.height - (headerHeight() + cpuHeight + gpuRamHeight + sec2Height)
	if procHeight < 6 {
		procHeight = 6
	}
	procBox := m.renderProcessBox(m.width, procHeight)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		cpuBox,
		row2,
		row3,
		procBox,
	)
}

func headerHeight() int {
	return 2 // Height of our header row + border
}

func (m Model) renderHeader() string {
	// Detect if we have a cgroup limit
	envType := "Bare Metal / Host"
	cgroupMaxPath, _ := stats.FindCgroupFile("cpu.max")
	if cgroupMaxPath != "" && stats.ParseCPUMax(cgroupMaxPath) > 0 {
		envType = "Proxmox LXC Container (Cgroup Limited)"
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "lxc-container"
	}

	logo := styleTitle.Render("MTOP")
	hostInfo := fmt.Sprintf("%s %s %s %s %s %s",
		styleHeaderLabel.Render("Host:"), styleHeaderVal.Render(hostname),
		styleHeaderLabel.Render("| Env:"), styleHeaderVal.Render(envType),
		styleHeaderLabel.Render("| CPU:"), styleHeaderVal.Render(m.stats.CPU.ModelName),
	)

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(colorBorder).
		Padding(0, 1).
		Width(m.width).
		Render(lipgloss.JoinHorizontal(lipgloss.Left, logo, "  ", hostInfo))
}

func (m Model) renderCPUBox(width, height int) string {
	title := styleBoxTitle.Render(" CPU ")

	// Split 50% left (cores and info) / 50% right (graph)
	innerAvailableWidth := width - 4
	graphWidth := innerAvailableWidth / 2
	leftWidth := innerAvailableWidth - graphWidth

	graphHeight := height - 4
	if graphHeight < 3 {
		graphHeight = 3
	}
	graph := m.renderCPUGraph(graphWidth, graphHeight)

	// Total usage bar with a neat, smaller size
	totalPct := m.stats.CPU.UsageTotal
	progressBar := makeProgressBar(totalPct, 12, colorCyan)
	totalStr := fmt.Sprintf("Usage: %5.1f%% %s", totalPct, progressBar)

	// Core bars
	var coresStr []string
	coresPerLine := 3
	if leftWidth < 55 {
		coresPerLine = 2
	}
	if leftWidth < 38 {
		coresPerLine = 1
	}

	line := ""
	for i, core := range m.stats.CPU.UsagePerCore {
		colWidth := leftWidth / coresPerLine
		coreBarWidth := colWidth - 14
		if coreBarWidth < 4 {
			coreBarWidth = 4
		}
		if coreBarWidth > 10 {
			coreBarWidth = 10 // keep CPU core bars compact
		}
		coreBar := makeProgressBar(core.Usage, coreBarWidth, colorBlue)
		coreStr := fmt.Sprintf("C%d:%4.0f%% %s", core.ID, core.Usage, coreBar)

		if line == "" {
			line = coreStr
		} else {
			padLen := colWidth - lipgloss.Width(line) - 1
			if padLen > 0 {
				line += strings.Repeat(" ", padLen)
			}
			line += " " + coreStr
		}

		if (i+1)%coresPerLine == 0 || i == len(m.stats.CPU.UsagePerCore)-1 {
			coresStr = append(coresStr, line)
			line = ""
		}
	}

	// Limit to inner height
	maxCoresLines := height - 5
	if maxCoresLines < 1 {
		maxCoresLines = 1
	}
	if len(coresStr) > maxCoresLines {
		coresStr = coresStr[:maxCoresLines]
	}

	// Left column content
	leftLines := []string{totalStr, ""}
	leftLines = append(leftLines, coresStr...)
	leftContent := title + "\n\n" + strings.Join(leftLines, "\n")

	// Right column content
	graphLabel := styleHeaderLabel.Render("CPU History")
	graphContent := graphLabel + "\n" + graph

	leftCol := lipgloss.NewStyle().Width(leftWidth).Render(leftContent)
	rightCol := lipgloss.NewStyle().
		Width(graphWidth).
		Align(lipgloss.Right).
		Render(graphContent)

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol)

	return styleBorder.
		Width(width - 2).
		Height(height - 2).
		Render(content)
}

// renderCPUGraph draws a sparkline (bar graph) of CPU history using block characters.
// Each column represents one historical sample. Height is the number of terminal rows.
func (m Model) renderCPUGraph(width, height int) string {
	if height < 1 {
		height = 1
	}

	// The 8 vertical block chars from bottom to top
	blockChars := []string{" ", "▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

	// Take the last `width` samples
	history := m.cpuHistory
	if len(history) > width {
		history = history[len(history)-width:]
	}

	// Pad with zeros if we don't have enough history yet
	padded := make([]float64, width)
	offset := width - len(history)
	for i, v := range history {
		padded[offset+i] = v
	}

	// Build the graph row by row (top row = height-1, bottom = 0)
	rows := make([]string, height)
	for row := 0; row < height; row++ {
		// Row 0 is the bottom, row (height-1) is the top
		bottomRow := height - 1 - row
		var sb strings.Builder
		for _, pct := range padded {
			// Map 0-100% to 0..(height*8-1) granularity
			totalUnits := height * 8
			units := int(math.Round(pct / 100.0 * float64(totalUnits)))
			// For this row, what sub-block level?
			rowBase := bottomRow * 8
			rowTop := rowBase + 8

			var ch string
			if units >= rowTop {
				// Full block
				ch = blockChars[8]
			} else if units > rowBase {
				// Partial block
				ch = blockChars[units-rowBase]
			} else {
				// Empty
				ch = blockChars[0]
			}
			sb.WriteString(ch)
		}
		// Color: green for low, yellow for mid, red for high
		// Use last sample to determine color
		var graphColor lipgloss.Color
		if len(history) > 0 {
			last := history[len(history)-1]
			switch {
			case last >= 80:
				graphColor = colorWarn
			case last >= 50:
				graphColor = colorGold
			default:
				graphColor = colorCyan
			}
		} else {
			graphColor = colorCyan
		}
		rows[row] = lipgloss.NewStyle().Foreground(graphColor).Render(sb.String())
	}

	return strings.Join(rows, "\n")
}

func (m Model) renderMemoryBox(width, height int) string {
	title := styleBoxTitle.Render(" Memory ")

	mem := m.stats.Memory
	ramPct := mem.UsagePct

	// Calculate bar width dynamically to fit inside the box content area (width - 4) and cap at 20
	barWidth := (width - 4) - 33
	if barWidth < 5 {
		barWidth = 5
	}
	if barWidth > 20 {
		barWidth = 20
	}

	ramBar := makeProgressBar(ramPct, barWidth, colorPurple)
	ramDetail := fmt.Sprintf("RAM:  %5.1f%% %s  %s / %s",
		ramPct,
		ramBar,
		formatBytes(mem.Used),
		formatBytes(mem.Total),
	)

	var swapDetail string
	if mem.SwapTotal > 0 {
		swapPct := float64(mem.SwapUsed) / float64(mem.SwapTotal) * 100
		swapBar := makeProgressBar(swapPct, barWidth, colorPink)
		swapDetail = fmt.Sprintf("SWAP: %5.1f%% %s  %s / %s",
			swapPct,
			swapBar,
			formatBytes(mem.SwapUsed),
			formatBytes(mem.SwapTotal),
		)
	} else {
		swapDetail = "SWAP: Not Configured"
	}

	content := title + "\n\n" + ramDetail + "\n\n" + swapDetail

	return styleBorder.
		Width(width - 2).
		Height(height - 2).
		Render(content)
}

func (m Model) renderDiskBox(width, height int) string {
	title := styleBoxTitle.Render(" Storage ")

	if len(m.stats.Disks) == 0 {
		return styleBorder.
			Width(width - 2).
			Height(height - 2).
			Render(title + "\n\nNo Storage Devices Found")
	}

	lines := []string{
		fmt.Sprintf("%-15s %-10s %-8s %-5s %s", 
			styleHeaderLabel.Render("Mount"), 
			styleHeaderLabel.Render("Used"), 
			styleHeaderLabel.Render("Total"), 
			styleHeaderLabel.Render("Usage"), 
			styleHeaderLabel.Render("Bar")),
	}

	for _, d := range m.stats.Disks {
		mount := d.MountPoint
		if len(mount) > 15 {
			mount = mount[:12] + "..."
		}

		// Calculate dynamic bar width to stretch all the way to the right edge
		// non-bar text length is 45 (including brackets). Fit inside width - 4
		barW := (width - 4) - 45
		if barW < 5 {
			barW = 5
		}
		bar := makeProgressBar(d.UsagePct, barW, colorGold)

		lines = append(lines, fmt.Sprintf("%-15s %-10s %-8s %4.1f%%  %s",
			mount,
			formatBytes(d.Used),
			formatBytes(d.Total),
			d.UsagePct,
			bar,
		))
	}

	// Limit to inner height (height - 4 for borders, title, headers)
	maxLines := height - 4
	if maxLines < 1 {
		maxLines = 1
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}

	content := title + "\n\n" + strings.Join(lines, "\n")

	return styleBorder.
		Width(width - 2).
		Height(height - 2).
		Render(content)
}

func (m Model) renderNetBox(width, height int) string {
	title := styleBoxTitle.Render(" Network ")

	if len(m.stats.Networks) == 0 {
		return styleBorder.
			Width(width - 2).
			Height(height - 2).
			Render(title + "\n\nNo Network Interfaces Found")
	}

	var lines []string
	for _, n := range m.stats.Networks {
		lines = append(lines, fmt.Sprintf("%s\n  %s %-12s | %s %-12s\n  %s %-12s | %s %-12s",
			styleHeaderVal.Render(n.Name),
			styleHeaderLabel.Render("▲ Rx Rate:"), formatRate(n.RxRate),
			styleHeaderLabel.Render("▼ Tx Rate:"), formatRate(n.TxRate),
			styleHeaderLabel.Render("  Total Rx:"), formatBytes(n.RxBytes),
			styleHeaderLabel.Render("  Total Tx:"), formatBytes(n.TxBytes),
		))
	}

	// Inner height constraints
	maxLines := (height - 3) / 3
	if maxLines < 1 {
		maxLines = 1
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}

	content := title + "\n\n" + strings.Join(lines, "\n\n")

	return styleBorder.
		Width(width - 2).
		Height(height - 2).
		Render(content)
}

func (m Model) renderProcessBox(width, height int) string {
	sortLabels := map[SortField]string{
		SortCPU:  "[CPU]",
		SortMem:  "[MEM]",
		SortPID:  "[PID]",
		SortName: "[NAME]",
	}

	title := styleBoxTitle.Render(fmt.Sprintf(" Processes (Sorted by: %s) ", sortLabels[m.sortBy]))

	// Header row
	header := fmt.Sprintf("%-8s %-12s %-6s %-6s %-10s %s",
		styleHeaderLabel.Render("PID"),
		styleHeaderLabel.Render("User"),
		styleHeaderLabel.Render("CPU%"),
		styleHeaderLabel.Render("MEM%"),
		styleHeaderLabel.Render("Size"),
		styleHeaderLabel.Render("Command"),
	)

	var lines []string
	lines = append(lines, header)

	// Available rows for process display (height - 5 for borders, title, header row, and footer help)
	visibleHeight := height - 5
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	procs := m.stats.Processes
	if len(procs) == 0 {
		return styleBorder.
			Width(width - 2).
			Height(height - 2).
			Render(title + "\n\nNo processes found.")
	}

	// Make sure scrollOffset is bounded
	if m.scrollOffset > len(procs)-1 {
		m.scrollOffset = 0
	}

	endIdx := m.scrollOffset + visibleHeight
	if endIdx > len(procs) {
		endIdx = len(procs)
	}

	for i := m.scrollOffset; i < endIdx; i++ {
		p := procs[i]
		cmd := p.Name
		// Truncate command/name if too long
		maxCmdWidth := width - 50
		if maxCmdWidth > 0 && len(cmd) > maxCmdWidth {
			cmd = cmd[:maxCmdWidth-3] + "..."
		}

		line := fmt.Sprintf("%-8d %-12s %-6.1f %-6.1f %-10s %s",
			p.PID,
			p.User,
			p.CPU,
			p.Memory,
			formatBytes(p.MemSize),
			cmd,
		)

		if i == m.cursor {
			// Highlight current selected row
			line = lipgloss.NewStyle().
				Background(colorHighlight).
				Foreground(colorCyan).
				Bold(true).
				Render(line)
		}

		lines = append(lines, line)
	}

	// Pad with empty lines if needed
	for len(lines) < visibleHeight+1 {
		lines = append(lines, "")
	}

	// Footer help instructions
	helpStr := styleHeaderLabel.Render(" [q] Quit  |  [s] Toggle Sort  |  [j/k] Scroll Processes  |  [9] Kill Proc  |  [r] Refresh ")
	content := title + "\n\n" + strings.Join(lines, "\n") + "\n\n" + helpStr

	return styleBorder.
		Width(width - 2).
		Height(height - 2).
		Render(content)
}

// renderGPUBox renders GPU info for all detected GPUs (Intel/Nvidia/AMD).
func (m Model) renderGPUBox(width, height int) string {
	title := styleBoxTitle.Render(" GPU ")

	if len(m.stats.GPUs) == 0 {
		return styleBorder.
			Width(width - 2).
			Height(height - 2).
			Render(title + "\n\nNo GPU Detected")
	}

	var lines []string
	for _, g := range m.stats.GPUs {
		// Vendor badge color
		var vendorColor lipgloss.Color
		switch g.Vendor {
		case stats.GPUVendorNvidia:
			vendorColor = lipgloss.Color("#76b900") // NVIDIA green
		case stats.GPUVendorAMD:
			vendorColor = lipgloss.Color("#ed1c24") // AMD red
		case stats.GPUVendorIntel:
			vendorColor = colorBlue
		default:
			vendorColor = colorDim
		}

		vendorBadge := lipgloss.NewStyle().Bold(true).Foreground(vendorColor).Render(string(g.Vendor))
		gpuName := g.Name
		if len(gpuName) > 40 {
			gpuName = gpuName[:37] + "..."
		}
		nameStr := fmt.Sprintf("%s  %s", vendorBadge, styleHeaderVal.Render(gpuName))

		barWidth := width - 28
		if barWidth < 5 {
			barWidth = 5
		}
		if barWidth > 20 {
			barWidth = 20
		}

		// GPU core utilization bar
		var usageColor lipgloss.Color
		switch {
		case g.UsagePct >= 80:
			usageColor = colorWarn
		case g.UsagePct >= 50:
			usageColor = colorGold
		default:
			usageColor = lipgloss.Color("#76b900")
		}
		usageBar := makeProgressBar(g.UsagePct, barWidth, usageColor)
		usageStr := fmt.Sprintf("  GPU: %5.1f%% %s", g.UsagePct, usageBar)

		// Memory info
		var memStr string
		if g.MemTotalMB > 0 {
			memPct := float64(g.MemUsedMB) / float64(g.MemTotalMB) * 100
			memBar := makeProgressBar(memPct, barWidth, colorPurple)
			memStr = fmt.Sprintf("  MEM: %5.1f%% %s  (%d/%d MB)",
				memPct, memBar, g.MemUsedMB, g.MemTotalMB)
		}

		// Temperature
		var tempStr string
		if g.TemperatureC > 0 {
			tempColor := colorCyan
			if g.TemperatureC >= 80 {
				tempColor = colorWarn
			} else if g.TemperatureC >= 60 {
				tempColor = colorGold
			}
			tempStr = "  " + lipgloss.NewStyle().Foreground(tempColor).
				Render(fmt.Sprintf("Temp: %.0f°C", g.TemperatureC))
		}

		lines = append(lines, nameStr)
		lines = append(lines, usageStr)
		if memStr != "" {
			lines = append(lines, memStr)
		}
		if tempStr != "" {
			lines = append(lines, tempStr)
		}
		lines = append(lines, "") // spacer between GPUs
	}

	content := title + "\n\n" + strings.Join(lines, "\n")

	return styleBorder.
		Width(width - 2).
		Height(height - 2).
		Render(content)
}

// makeProgressBar draws a smooth Unicode progress bar
func makeProgressBar(pct float64, width int, color lipgloss.Color) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}

	filledLength := float64(width) * (pct / 100.0)
	fullBlocks := int(math.Floor(filledLength))
	
	// Characters for smooth sub-block rendering (cgroups/btop visual fidelity)
	unicodeBlocks := []string{" ", "▏", "▎", "▍", "▌", "▋", "▊", "▉", "█"}
	
	var sb strings.Builder
	sb.WriteString(strings.Repeat("█", fullBlocks))
	
	if fullBlocks < width {
		remainder := filledLength - float64(fullBlocks)
		blockIndex := int(math.Round(remainder * 8))
		if blockIndex > 0 && blockIndex < len(unicodeBlocks) {
			sb.WriteString(unicodeBlocks[blockIndex])
		} else {
			sb.WriteString(" ")
		}
		
		// Fill remaining empty space
		emptyLength := width - fullBlocks - 1
		if emptyLength > 0 {
			sb.WriteString(strings.Repeat(" ", emptyLength))
		}
	}

	barStr := sb.String()
	// Apply style color
	return lipgloss.NewStyle().Foreground(color).Render("[" + barStr + "]")
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func formatRate(bps float64) string {
	if bps < 1024 {
		return fmt.Sprintf("%.0f B/s", bps)
	} else if bps < 1024*1024 {
		return fmt.Sprintf("%.1f KB/s", bps/1024)
	}
	return fmt.Sprintf("%.1f MB/s", bps/(1024*1024))
}
