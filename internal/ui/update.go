package ui

import (
	"sort"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case statsMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.stats = msg.stats
			m.err = nil
			m.ready = true
			m.sortProcesses()
		}
		return m, nil

	case tickMsg:
		// Trigger a stats fetch on every tick
		return m, tea.Batch(m.fetchStatsCmd(), tickCmd())

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.scrollOffset {
					m.scrollOffset = m.cursor
				}
			}

		case "down", "j":
			maxIdx := len(m.stats.Processes) - 1
			if m.cursor < maxIdx {
				m.cursor++
				// Calculate visible items in process box based on window height
				visibleHeight := m.height - 18 // Estimate remaining space for processes
				if visibleHeight < 3 {
					visibleHeight = 3
				}
				if m.cursor >= m.scrollOffset+visibleHeight {
					m.scrollOffset = m.cursor - visibleHeight + 1
				}
			}

		case "s":
			m.sortBy = (m.sortBy + 1) % 4
			m.sortProcesses()
			m.cursor = 0
			m.scrollOffset = 0

		case "r":
			return m, m.fetchStatsCmd()
		}
	}

	return m, nil
}

func (m *Model) sortProcesses() {
	if len(m.stats.Processes) == 0 {
		return
	}

	sort.Slice(m.stats.Processes, func(i, j int) bool {
		pi := m.stats.Processes[i]
		pj := m.stats.Processes[j]

		switch m.sortBy {
		case SortCPU:
			return pi.CPU > pj.CPU
		case SortMem:
			return pi.MemSize > pj.MemSize
		case SortPID:
			return pi.PID < pj.PID
		case SortName:
			return pi.Name < pj.Name
		default:
			return pi.CPU > pj.CPU
		}
	})
}
