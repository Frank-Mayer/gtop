package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shirou/gopsutil/process"
)

// command line arguments for sorting and filtering
var (
	order = flag.String("order", "cpu", "sort by cpu, mem, pid, name, user, time, status")
	count = flag.Int("count", 32, "number of processes to show")
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type model struct {
	table table.Model
}

func orderByCPU(plist *[]*process.Process) {
	cache := make(map[int32]float64)
	for _, p := range *plist {
		if p == nil {
			continue
		}
		cpu, err := p.CPUPercent()
		if err != nil {
			cpu = -1
		}
		cache[p.Pid] = cpu
	}

	sort.Slice(*plist, func(i, j int) bool {
		iCpu, _ := cache[(*plist)[i].Pid]
		jCpu, _ := cache[(*plist)[j].Pid]
		return iCpu > jCpu
	})
}

func orderByMem(plist *[]*process.Process) {
	cache := make(map[int32]float32)
	for _, p := range *plist {
		mem, err := p.MemoryPercent()
		if err != nil {
			mem = -1
		}
		cache[p.Pid] = mem
	}

	sort.Slice(*plist, func(i, j int) bool {
		iMem, _ := cache[(*plist)[i].Pid]
		jMem, _ := cache[(*plist)[j].Pid]
		return iMem > jMem
	})
}

func plist() ([]table.Row, error) {
	plist, err := process.Processes()
	if err != nil {
		return nil, err
	}

	switch *order {
	case "cpu":
		orderByCPU(&plist)
	case "mem":
		orderByMem(&plist)
	}

	// new process list for table ui
	rows := make([]table.Row, len(plist))

	// iterate over processes
	for i, p := range plist {
		if p == nil {
			continue
		}

		name, err := p.Name()
		if err != nil {
			name = "<unknown>"
		}
		username, err := p.Username()
		if err != nil {
			username = "<unknown>"
		}
		cpu, err := p.CPUPercent()
		if err != nil {
			cpu = -1
		}
		mem, err := p.MemoryPercent()
		if err != nil {
			mem = -1
		}
		t, err := p.CreateTime()
		if err != nil {
			t = -1
		}
		create_time := time.Unix(t/1000, 0).Format(time.RFC3339)

		status, err := p.Status()
		if err != nil {
			status = "<unknown>"
		}

		if i >= *count {
			break
		}

		rows[i] = table.Row{
			fmt.Sprintf("%d", p.Pid),
			name,
			username,
			fmt.Sprintf("%.2f", cpu),
			fmt.Sprintf("%.2f", mem),
			create_time,
			status,
		}
	}

	return rows, nil
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			newRows, err := plist()
			if err != nil {
				return m, nil
			}
			m.table.SetRows(newRows)
		case "d":
			pid, err := strconv.Atoi(m.table.SelectedRow()[0])
			if err != nil {
				return m, nil
			}
			p, err := process.NewProcess(int32(pid))
			if err != nil {
				return m, nil
			}
			err = p.Kill()
			if err != nil {
				return m, nil
			}
			newRows, err := plist()
			if err != nil {
				return m, nil
			}
			m.table.SetRows(newRows)
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return baseStyle.Render(m.table.View()) + "\n"
}

func main() {
	flag.Parse()

	columns := []table.Column{
		{Title: "PID", Width: 10},
		{Title: "Name", Width: 20},
		{Title: "User", Width: 10},
		{Title: "CPU%", Width: 6},
		{Title: "Mem%", Width: 6},
		{Title: "Time", Width: 25},
		{Title: "Status", Width: 6},
	}

	rows, err := plist()
	if err != nil {
		panic(err)
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	m := model{t}
	prog := tea.NewProgram(m)
	if _, err := prog.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

