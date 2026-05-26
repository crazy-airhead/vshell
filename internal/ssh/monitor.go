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
	prevCPUCores map[int]CPUSample
	osType       string
	osDetected   bool
	hostname     string
	ipAddresses  []string
}

type CPUCoreStat struct {
	Core    int     `json:"core"`
	Percent float64 `json:"percent"`
}

type ProcessInfo struct {
	PID        int     `json:"pid"`
	User       string  `json:"user"`
	Name       string  `json:"name"`
	CPUPercent float64 `json:"cpu_percent"`
	MemPercent float64 `json:"mem_percent"`
	MemBytes   uint64  `json:"mem_bytes"`
	Command    string  `json:"command"`
	ExePath    string  `json:"exe_path"`
}

type SystemStats struct {
	CPUPercent    float64          `json:"cpu_percent"`
	MemPercent    float64          `json:"mem_percent"`
	MemTotal      uint64           `json:"mem_total"`
	MemUsed       uint64           `json:"mem_used"`
	SwapTotal     uint64           `json:"swap_total"`
	SwapUsed      uint64           `json:"swap_used"`
	NetInterfaces map[string]NetIO `json:"net_interfaces"`
	LoadAvg       [3]float64       `json:"load_avg"`
	DiskStats     []DiskStat       `json:"disk_stats"`
	UptimeSeconds float64          `json:"uptime_seconds"`
	OS            string           `json:"os"`
	CPUCores      []CPUCoreStat    `json:"cpu_cores"`
	TopProcesses  []ProcessInfo    `json:"top_processes"`
	Hostname      string           `json:"hostname"`
	IPAddresses   []string         `json:"ip_addresses"`
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

type NetConnProcess struct {
	PID         int      `json:"pid"`
	Name        string   `json:"name"`
	ListenAddrs []string `json:"listen_addrs"`
	Ports       []string `json:"ports"`
	ConnCount   int      `json:"conn_count"`
}

func NewMonitor(manager *Manager, connectionID string) *Monitor {
	return &Monitor{
		manager:      manager,
		connectionID: connectionID,
		stopCh:       make(chan struct{}),
		prevCPUCores: make(map[int]CPUSample),
	}
}

func (m *Monitor) Start() {
	// Detect OS
	out, err := m.manager.ExecOnConnection(m.connectionID, "uname -s")
	if err == nil {
		m.osType = strings.TrimSpace(out)
	}
	m.osDetected = true

	// Collect host info once
	m.sampleHostInfo()

	// Start sampling goroutines
	go m.sampleLoop("cpu_mem_net", 2*time.Second, m.sampleCPUMemNet)
	go m.sampleLoop("disk", 10*time.Second, m.sampleDisk)
	go m.sampleLoop("load", 5*time.Second, m.sampleLoad)
	go m.sampleLoop("processes", 5*time.Second, m.sampleProcesses)
	go m.sampleLoop("netconns", 10*time.Second, m.sampleNetConns)
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

func (m *Monitor) sampleHostInfo() {
	// Hostname
	out, err := m.manager.ExecOnConnection(m.connectionID, "hostname 2>/dev/null")
	if err == nil {
		m.hostname = strings.TrimSpace(out)
	}

	// IP addresses
	out, err = m.manager.ExecOnConnection(m.connectionID,
		"hostname -I 2>/dev/null")
	if err == nil {
		for _, ip := range strings.Fields(strings.TrimSpace(out)) {
			if ip != "" {
				m.ipAddresses = append(m.ipAddresses, ip)
			}
		}
	}
	if len(m.ipAddresses) == 0 {
		// Fallback: try ip addr
		out, err = m.manager.ExecOnConnection(m.connectionID,
			"ip -4 addr show 2>/dev/null | grep inet | awk '{print $2}' | cut -d/ -f1")
		if err == nil {
			for _, ip := range strings.Split(out, "\n") {
				ip = strings.TrimSpace(ip)
				if ip != "" && ip != "127.0.0.1" {
					m.ipAddresses = append(m.ipAddresses, ip)
				}
			}
		}
	}
}

func (m *Monitor) sampleCPUMemNet() {
	stats := &SystemStats{
		OS:          m.osType,
		Hostname:    m.hostname,
		IPAddresses: m.ipAddresses,
	}

	// CPU
	out, err := m.manager.ExecOnConnection(m.connectionID, "cat /proc/stat")
	if err == nil {
		stats.CPUPercent, stats.CPUCores = m.parseCPU(out)
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

func (m *Monitor) parseCPU(procStat string) (float64, []CPUCoreStat) {
	m.mu.Lock()
	defer m.mu.Unlock()

	lines := strings.Split(procStat, "\n")
	if len(lines) == 0 {
		return 0, nil
	}

	// Parse aggregate cpu line
	fields := strings.Fields(lines[0])
	if len(fields) < 8 || fields[0] != "cpu" {
		return 0, nil
	}

	vals := make([]uint64, len(fields)-1)
	var total uint64
	for i := 1; i < len(fields); i++ {
		v, _ := strconv.ParseUint(fields[i], 10, 64)
		vals[i-1] = v
		total += v
	}

	idle := vals[3] // idle field

	var cpuPct float64
	if m.prevCPU.Total != 0 {
		totalDiff := float64(total - m.prevCPU.Total)
		idleDiff := float64(idle - m.prevCPU.Idle)
		if totalDiff > 0 {
			cpuPct = (1 - idleDiff/totalDiff) * 100
		}
	}
	m.prevCPU = CPUSample{Total: total, Idle: idle}

	// Parse per-core lines (cpu0, cpu1, ...)
	var cores []CPUCoreStat
	for _, line := range lines[1:] {
		f := strings.Fields(line)
		if len(f) < 8 {
			continue
		}
		if !strings.HasPrefix(f[0], "cpu") {
			break
		}
		coreNum, err := strconv.Atoi(f[0][3:])
		if err != nil {
			break
		}

		v := make([]uint64, len(f)-1)
		var t uint64
		for i := 1; i < len(f); i++ {
			n, _ := strconv.ParseUint(f[i], 10, 64)
			v[i-1] = n
			t += n
		}
		coreIdle := v[3]

		var pct float64
		if prev, ok := m.prevCPUCores[coreNum]; ok && prev.Total > 0 {
			td := float64(t - prev.Total)
			id := float64(coreIdle - prev.Idle)
			if td > 0 {
				pct = (1 - id/td) * 100
			}
		}
		m.prevCPUCores[coreNum] = CPUSample{Total: t, Idle: coreIdle}
		cores = append(cores, CPUCoreStat{Core: coreNum, Percent: pct})
	}

	return cpuPct, cores
}

func (m *Monitor) parseMem(procMeminfo string, stats *SystemStats) {
	var memTotal, memAvailable, swapTotal, swapFree uint64
	for _, line := range strings.Split(procMeminfo, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, _ := strconv.ParseUint(fields[1], 10, 64)
		switch fields[0] {
		case "MemTotal:":
			memTotal = val
		case "MemAvailable:":
			memAvailable = val
		case "SwapTotal:":
			swapTotal = val
		case "SwapFree:":
			swapFree = val
		}
	}

	if memTotal > 0 {
		stats.MemTotal = memTotal
		stats.MemUsed = memTotal - memAvailable
		stats.MemPercent = float64(memTotal-memAvailable) / float64(memTotal) * 100
	}
	if swapTotal > 0 {
		stats.SwapTotal = swapTotal
		stats.SwapUsed = swapTotal - swapFree
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

func (m *Monitor) sampleProcesses() {
	if m.osType != "Linux" {
		return
	}

	out, err := m.manager.ExecOnConnection(m.connectionID,
		"ps -eo pid,user,%cpu,%mem,rss,args --sort=-%cpu --no-headers 2>/dev/null | head -30")
	if err != nil {
		return
	}

	// Collect PIDs for exe resolution
	type parsedProc struct {
		pid      int
		user     string
		cpuPct   float64
		memPct   float64
		memBytes uint64
		command  string
		exePath  string
	}
	var parsed []parsedProc
	pidOrder := []int{}

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		pid, _ := strconv.Atoi(fields[0])
		user := fields[1]
		cpuPct, _ := strconv.ParseFloat(fields[2], 64)
		memPct, _ := strconv.ParseFloat(fields[3], 64)
		memBytes, _ := strconv.ParseUint(fields[4], 10, 64)
		memBytes *= 1024 // KB to bytes
		command := strings.Join(fields[5:], " ")

		parsed = append(parsed, parsedProc{
			pid:      pid,
			user:     user,
			cpuPct:   cpuPct,
			memPct:   memPct,
			memBytes: memBytes,
			command:  command,
		})
		pidOrder = append(pidOrder, pid)
	}

	// Resolve executable paths in one batch
	exeMap := map[int]string{}
	if len(pidOrder) > 0 {
		var reads []string
		for _, pid := range pidOrder {
			reads = append(reads, fmt.Sprintf("readlink /proc/%d/exe 2>/dev/null || echo -", pid))
		}
		exeOut, err := m.manager.ExecOnConnection(m.connectionID,
			strings.Join(reads, " && "))
		if err == nil {
			exeLines := strings.Split(strings.TrimSpace(exeOut), "\n")
			for i, p := range exeLines {
				p = strings.TrimSpace(p)
				if p != "" && p != "-" && i < len(pidOrder) {
					exeMap[pidOrder[i]] = p
				}
			}
		}
	}

	var procs []ProcessInfo
	for _, pp := range parsed {
		exePath := exeMap[pp.pid]
		// Extract process name from exe path
		name := pp.command
		if idx := strings.Index(pp.command, " "); idx > 0 {
			name = pp.command[:idx]
		}
		procs = append(procs, ProcessInfo{
			PID:        pp.pid,
			User:       pp.user,
			Name:       name,
			CPUPercent: pp.cpuPct,
			MemPercent: pp.memPct,
			MemBytes:   pp.memBytes,
			Command:    pp.command,
			ExePath:    exePath,
		})
	}

	m.manager.onEvent("monitor:processes", map[string]any{
		"connectionID": m.connectionID,
		"processes":    procs,
	})
}

func (m *Monitor) sampleNetConns() {
	if m.osType != "Linux" {
		return
	}

	// Get listening processes
	type listenInfo struct {
		pid   int
		name  string
		addrs []string
		ports []string
	}
	listenMap := make(map[int]*listenInfo) // keyed by PID

	out, err := m.manager.ExecOnConnection(m.connectionID,
		"ss -tlnp 2>/dev/null | tail -n +2")
	if err == nil {
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) < 5 {
				continue
			}

			// Parse local address (field 3, format: ip:port or [ip]:port)
			localAddr := fields[3]
			host, port := splitHostPort(localAddr)

			// Parse PID from users field (format: users:(("name",pid=123,fd=4))
			pid := parsePIDFromUsers(fields[len(fields)-1])
			if pid == 0 {
				continue
			}
			name := parseNameFromUsers(fields[len(fields)-1])

			info, ok := listenMap[pid]
			if !ok {
				info = &listenInfo{pid: pid, name: name}
				listenMap[pid] = info
			}
			if host != "" {
				found := false
				for _, a := range info.addrs {
					if a == host {
						found = true
						break
					}
				}
				if !found {
					info.addrs = append(info.addrs, host)
				}
			}
			if port != "" {
				found := false
				for _, p := range info.ports {
					if p == port {
						found = true
						break
					}
				}
				if !found {
					info.ports = append(info.ports, port)
				}
			}
		}
	}

	// Get connection counts per PID
	connCounts := make(map[int]int)
	out, err = m.manager.ExecOnConnection(m.connectionID,
		"ss -tnp state established 2>/dev/null | tail -n +2")
	if err == nil {
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) < 5 {
				continue
			}
			pid := parsePIDFromUsers(fields[len(fields)-1])
			if pid > 0 {
				connCounts[pid]++
			}
		}
	}

	// Build result
	var result []NetConnProcess
	for pid, info := range listenMap {
		result = append(result, NetConnProcess{
			PID:         pid,
			Name:        info.name,
			ListenAddrs: info.addrs,
			Ports:       info.ports,
			ConnCount:   connCounts[pid],
		})
	}
	// Also add processes with connections but not listening
	for pid, count := range connCounts {
		if _, ok := listenMap[pid]; !ok {
			result = append(result, NetConnProcess{
				PID:       pid,
				ConnCount: count,
			})
		}
	}

	m.manager.onEvent("monitor:netconns", map[string]any{
		"connectionID": m.connectionID,
		"netconns":     result,
	})
}

// splitHostPort splits "ip:port" or "[ipv6]:port" into host and port.
func splitHostPort(addr string) (string, string) {
	// IPv6: [::1]:80
	if strings.HasPrefix(addr, "[") {
		idx := strings.Index(addr, "]:")
		if idx < 0 {
			return addr, ""
		}
		return addr[1:idx], addr[idx+2:]
	}
	// IPv4: 0.0.0.0:80
	idx := strings.LastIndex(addr, ":")
	if idx < 0 {
		return addr, ""
	}
	return addr[:idx], addr[idx+1:]
}

// parsePIDFromUsers parses pid from "users:(("nginx",pid=1234,fd=6))"
func parsePIDFromUsers(field string) int {
	idx := strings.Index(field, "pid=")
	if idx < 0 {
		return 0
	}
	rest := field[idx+4:]
	end := strings.IndexAny(rest, ",)")
	if end < 0 {
		return 0
	}
	pid, err := strconv.Atoi(rest[:end])
	if err != nil {
		return 0
	}
	return pid
}

// parseNameFromUsers parses process name from "users:(("nginx",pid=1234,fd=6))"
func parseNameFromUsers(field string) string {
	idx := strings.Index(field, "((\"")
	if idx < 0 {
		return ""
	}
	rest := field[idx+3:]
	end := strings.Index(rest, "\"")
	if end < 0 {
		return ""
	}
	return rest[:end]
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
