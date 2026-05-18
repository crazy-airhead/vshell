package ssh

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

type CPUSample struct {
	Total uint64
	Idle  uint64
}

type Monitor struct {
	manager      *Manager
	connectionID string
	mu           sync.Mutex
	stopCh       chan struct{}
	prevCPU      CPUSample
	osType       string
	osDetected   bool
}

type SystemStats struct {
	CPUPercent    float64          `json:"cpu_percent"`
	MemPercent    float64          `json:"mem_percent"`
	MemTotal      uint64           `json:"mem_total"`
	MemUsed       uint64           `json:"mem_used"`
	NetInterfaces map[string]NetIO `json:"net_interfaces"`
	LoadAvg       [3]float64       `json:"load_avg"`
	DiskStats     []DiskStat       `json:"disk_stats"`
	UptimeSeconds float64          `json:"uptime_seconds"`
	OS            string           `json:"os"`
}

type NetIO struct {
	ReceiveBytes  uint64  `json:"receive_bytes"`
	TransmitBytes uint64  `json:"transmit_bytes"`
	ReceiveKBps   float64 `json:"receive_kbps"`
	TransmitKBps  float64 `json:"transmit_kbps"`
}

type DiskStat struct {
	Device     string  `json:"device"`
	MountPoint string  `json:"mount_point"`
	Total      uint64  `json:"total"`
	Used       uint64  `json:"used"`
	Percent    float64 `json:"percent"`
}

func NewMonitor(manager *Manager, connectionID string) *Monitor {
	return &Monitor{
		manager:      manager,
		connectionID: connectionID,
		stopCh:       make(chan struct{}),
	}
}

func (m *Monitor) Start() {
	// Detect OS
	out, err := m.manager.ExecOnConnection(m.connectionID, "uname -s")
	if err == nil {
		m.osType = strings.TrimSpace(out)
	}
	m.osDetected = true

	// Start sampling goroutines
	go m.sampleLoop("cpu_mem_net", 2*time.Second, m.sampleCPUMemNet)
	go m.sampleLoop("disk", 10*time.Second, m.sampleDisk)
	go m.sampleLoop("load", 5*time.Second, m.sampleLoad)
}

func (m *Monitor) Stop() {
	close(m.stopCh)
}

func (m *Monitor) sampleLoop(name string, interval time.Duration, fn func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fn() // initial sample

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			fn()
		}
	}
}

func (m *Monitor) sampleCPUMemNet() {
	stats := &SystemStats{OS: m.osType}

	// CPU
	out, err := m.manager.ExecOnConnection(m.connectionID, "cat /proc/stat")
	if err == nil {
		stats.CPUPercent = m.parseCPU(out)
	}

	// Memory
	out, err = m.manager.ExecOnConnection(m.connectionID, "cat /proc/meminfo")
	if err == nil {
		m.parseMem(out, stats)
	}

	// Network
	out, err = m.manager.ExecOnConnection(m.connectionID, "cat /proc/net/dev")
	if err == nil {
		m.parseNet(out, stats)
	}

	// Uptime
	out, err = m.manager.ExecOnConnection(m.connectionID, "cat /proc/uptime")
	if err == nil {
		fields := strings.Fields(out)
		if len(fields) >= 1 {
			stats.UptimeSeconds, _ = strconv.ParseFloat(fields[0], 64)
		}
	}

	m.manager.onEvent("monitor:stats", map[string]any{
		"connectionID": m.connectionID,
		"stats":        stats,
	})
}

func (m *Monitor) sampleDisk() {
	if m.osType != "Linux" {
		return
	}

	out, err := m.manager.ExecOnConnection(m.connectionID, "df -B1 -x tmpfs -x devtmpfs 2>/dev/null | tail -n +2")
	if err != nil {
		return
	}

	var disks []DiskStat
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		total, _ := strconv.ParseUint(fields[1], 10, 64)
		used, _ := strconv.ParseUint(fields[2], 10, 64)
		pctStr := strings.TrimSuffix(fields[4], "%")
		pct, _ := strconv.ParseFloat(pctStr, 64)

		disks = append(disks, DiskStat{
			Device:     fields[0],
			MountPoint: fields[5],
			Total:      total,
			Used:       used,
			Percent:    pct,
		})
	}

	m.manager.onEvent("monitor:disk", map[string]any{
		"connectionID": m.connectionID,
		"disks":        disks,
	})
}

func (m *Monitor) sampleLoad() {
	out, err := m.manager.ExecOnConnection(m.connectionID, "cat /proc/loadavg")
	if err != nil {
		return
	}

	fields := strings.Fields(out)
	var loads [3]float64
	if len(fields) >= 3 {
		loads[0], _ = strconv.ParseFloat(fields[0], 64)
		loads[1], _ = strconv.ParseFloat(fields[1], 64)
		loads[2], _ = strconv.ParseFloat(fields[2], 64)
	}

	m.manager.onEvent("monitor:load", map[string]any{
		"connectionID": m.connectionID,
		"load":         loads,
	})
}

func (m *Monitor) parseCPU(procStat string) float64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	lines := strings.Split(procStat, "\n")
	if len(lines) == 0 {
		return 0
	}

	fields := strings.Fields(lines[0])
	if len(fields) < 8 || fields[0] != "cpu" {
		return 0
	}

	vals := make([]uint64, len(fields)-1)
	var total uint64
	for i := 1; i < len(fields); i++ {
		v, _ := strconv.ParseUint(fields[i], 10, 64)
		vals[i-1] = v
		total += v
	}

	idle := vals[3] // idle field

	if m.prevCPU.Total == 0 {
		m.prevCPU = CPUSample{Total: total, Idle: idle}
		return 0
	}

	totalDiff := float64(total - m.prevCPU.Total)
	idleDiff := float64(idle - m.prevCPU.Idle)

	m.prevCPU = CPUSample{Total: total, Idle: idle}

	if totalDiff == 0 {
		return 0
	}

	return (1 - idleDiff/totalDiff) * 100
}

func (m *Monitor) parseMem(procMeminfo string, stats *SystemStats) {
	var total, available uint64
	for _, line := range strings.Split(procMeminfo, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, _ := strconv.ParseUint(fields[1], 10, 64)
		switch fields[0] {
		case "MemTotal:":
			total = val
		case "MemAvailable:":
			available = val
		}
	}

	if total > 0 {
		stats.MemTotal = total
		stats.MemUsed = total - available
		stats.MemPercent = float64(total-available) / float64(total) * 100
	}
}

func (m *Monitor) parseNet(procNetDev string, stats *SystemStats) {
	stats.NetInterfaces = make(map[string]NetIO)

	for _, line := range strings.Split(procNetDev, "\n") {
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}

		iface := strings.TrimSpace(line[:idx])
		fields := strings.Fields(strings.TrimSpace(line[idx+1:]))
		if len(fields) < 10 {
			continue
		}

		rxBytes, _ := strconv.ParseUint(fields[0], 10, 64)
		txBytes, _ := strconv.ParseUint(fields[8], 10, 64)

		stats.NetInterfaces[iface] = NetIO{
			ReceiveBytes:  rxBytes,
			TransmitBytes: txBytes,
		}
	}
}

// FormatBytes formats bytes into human-readable string.
func FormatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
