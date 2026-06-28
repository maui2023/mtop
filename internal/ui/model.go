package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/maui/mtop/internal/stats"
)

type SortField int

const (
	SortCPU SortField = iota
	SortMem
	SortPID
	SortName
)

type statsMsg struct {
	stats stats.SystemStats
	err   error
}

type Model struct {
	collector    *stats.Collector
	stats        stats.SystemStats
	err          error
	width        int
	height       int
	cursor       int
	scrollOffset int
	sortBy       SortField
	hostname     string
	ready        bool
}

func NewModel(collector *stats.Collector) Model {
	return Model{
		collector: collector,
		sortBy:    SortCPU,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchStatsCmd(),
		tickCmd(),
	)
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) fetchStatsCmd() tea.Cmd {
	return func() tea.Msg {
		s, err := m.collector.GetSystemStats()
		return statsMsg{stats: s, err: err}
	}
}
