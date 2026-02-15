package serverInfo

import (
	"context"
	"encoding/json"
	"io"
	"math"
	stdnet "net"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	gonet "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// API models

type SCServerStatusResponse struct {
	Name               string                `json:"name"`
	OsInfo             string                `json:"osInfo"`
	CPUUsagePercent    float64               `json:"cpuUsagePercent"`
	SystemUsagePercent float64               `json:"systemUsagePercent"`
	UserUsagePercent   float64               `json:"userUsagePercent"`
	IOWaitPercent      float64               `json:"ioWaitPercent"`
	StealPercent       float64               `json:"stealPercent"`
	IdlePercent        float64               `json:"idlePercent"`
	UptimeSeconds      int                   `json:"uptimeSeconds"`
	Load1              float64               `json:"load1"`
	Load5              float64               `json:"load5"`
	Load15             float64               `json:"load15"`
	Cores              []SCCoreUsageResponse `json:"cores"`
	Memory             SCMemoryResponse      `json:"memory"`
	Network            SCNetworkResponse     `json:"network"`
	Disks              []SCDiskResponse      `json:"disks"`
	Containers         []SCContainerResponse `json:"containers"`
	Processes          []SCProcessResponse   `json:"processes"`
}

type SCCoreUsageResponse struct {
	System float64 `json:"system"`
	User   float64 `json:"user"`
	IOWait float64 `json:"iowait"`
	Steal  float64 `json:"steal"`
	Idle   float64 `json:"idle"`
}

type SCMemoryResponse struct {
	AvailableBytes float64 `json:"availableBytes"`
	UsedBytes      float64 `json:"usedBytes"`
	CacheBytes     float64 `json:"cacheBytes"`
	UsagePercent   float64 `json:"usagePercent"`
}

type SCNetworkResponse struct {
	TotalUploadBps     float64                      `json:"totalUploadBps"`
	TotalDownloadBps   float64                      `json:"totalDownloadBps"`
	TotalUploadBytes   float64                      `json:"totalUploadBytes"`
	TotalDownloadBytes float64                      `json:"totalDownloadBytes"`
	RetransRatePercent float64                      `json:"retransRatePercent"`
	ActiveConn         int                          `json:"activeConn"`
	PassiveConn        int                          `json:"passiveConn"`
	FailConn           int                          `json:"failConn"`
	Established        int                          `json:"established"`
	TimeWait           int                          `json:"timeWait"`
	CloseWait          int                          `json:"closeWait"`
	SynRecv            int                          `json:"synRecv"`
	Interfaces         []SCNetworkInterfaceResponse `json:"interfaces"`
	PrimaryInterface   *SCNetworkInterfaceResponse  `json:"primaryInterface,omitempty"`
}

type SCNetworkInterfaceResponse struct {
	Name               string  `json:"name"`
	IP                 *string `json:"ip"`
	IsVirtual          bool    `json:"isVirtual"`
	UploadBps          float64 `json:"uploadBps"`
	DownloadBps        float64 `json:"downloadBps"`
	TotalUploadBytes   float64 `json:"totalUploadBytes"`
	TotalDownloadBytes float64 `json:"totalDownloadBytes"`
}

type SCDiskResponse struct {
	MountPoint   string  `json:"mountPoint"`
	Device       string  `json:"device"`
	FileSystem   string  `json:"fileSystem"`
	UsedBytes    float64 `json:"usedBytes"`
	TotalBytes   float64 `json:"totalBytes"`
	ReadBps      float64 `json:"readBps"`
	ReadBytes    float64 `json:"readBytes"`
	ReadIOPS     float64 `json:"readIOPS"`
	ReadDelayMs  float64 `json:"readDelayMs"`
	WriteBps     float64 `json:"writeBps"`
	WriteBytes   float64 `json:"writeBytes"`
	WriteIOPS    float64 `json:"writeIOPS"`
	WriteDelayMs float64 `json:"writeDelayMs"`
}

type SCContainerResponse struct {
	Name             string  `json:"name"`
	CPUUsagePercent  float64 `json:"cpuUsagePercent"`
	MemoryBytes      float64 `json:"memoryBytes"`
	NetUploadBytes   float64 `json:"netUploadBytes"`
	NetDownloadBytes float64 `json:"netDownloadBytes"`
	BlockReadBytes   float64 `json:"blockReadBytes"`
	BlockWriteBytes  float64 `json:"blockWriteBytes"`
}

type SCProcessResponse struct {
	PID        int32   `json:"pid"`
	Name       string  `json:"name"`
	CPUPercent float64 `json:"cpuPercent"`
	MemBytes   uint64  `json:"memBytes"`
	MemPercent float64 `json:"memPercent"`
}

// Collector

type collector struct {
	mu         sync.RWMutex
	lastSample time.Time
	prevNet    map[string]gonet.IOCountersStat
	prevDisk   map[string]disk.IOCountersStat
	prevCPU    []cpu.TimesStat
	prevPerCPU []cpu.TimesStat

	dockerClient *client.Client
}

func newCollector() *collector {
	var dockerClient *client.Client
	if cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation()); err == nil {
		dockerClient = cli
	}
	return &collector{
		prevNet:      map[string]gonet.IOCountersStat{},
		prevDisk:     map[string]disk.IOCountersStat{},
		prevCPU:      nil,
		prevPerCPU:   nil,
		dockerClient: dockerClient,
	}
}

func (c *collector) snapshot(ctx context.Context) SCServerStatusResponse {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(c.lastSample)
	if c.lastSample.IsZero() {
		elapsed = 2 * time.Second
	}
	c.lastSample = now

	name, _ := os.Hostname()
	osInfo := runtime.GOOS + " " + runtime.GOARCH
	if hi, err := host.InfoWithContext(ctx); err == nil {
		osInfo = hi.Platform + " " + hi.PlatformVersion
	}

	uptimeSeconds := 0
	if up, err := host.UptimeWithContext(ctx); err == nil {
		uptimeSeconds = int(up)
	}

	load1, load5, load15 := 0.0, 0.0, 0.0
	if la, err := load.AvgWithContext(ctx); err == nil {
		load1, load5, load15 = la.Load1, la.Load5, la.Load15
	}

	currCPU, _ := cpu.TimesWithContext(ctx, false)
	currPerCPU, _ := cpu.TimesWithContext(ctx, true)
	cpuTotal, cpuSystem, cpuUser, cpuIOWait, cpuSteal, cpuIdle := c.computeCPU(currCPU)
	cores := c.computePerCore(currPerCPU)

	memory := SCMemoryResponse{}
	if vm, err := mem.VirtualMemoryWithContext(ctx); err == nil {
		memory.AvailableBytes = float64(vm.Available)
		memory.UsedBytes = float64(vm.Used)
		memory.CacheBytes = float64(vm.Cached)
		memory.UsagePercent = vm.UsedPercent / 100
	}

	netResp := c.collectNetwork(ctx, elapsed)
	disks := c.collectDisks(ctx, elapsed)
	containers := c.collectDocker(ctx, elapsed)
	processes := topProcesses(ctx, 100)

	return SCServerStatusResponse{
		Name:               name,
		OsInfo:             osInfo,
		CPUUsagePercent:    cpuTotal,
		SystemUsagePercent: cpuSystem,
		UserUsagePercent:   cpuUser,
		IOWaitPercent:      cpuIOWait,
		StealPercent:       cpuSteal,
		IdlePercent:        cpuIdle,
		UptimeSeconds:      uptimeSeconds,
		Load1:              load1,
		Load5:              load5,
		Load15:             load15,
		Cores:              cores,
		Memory:             memory,
		Network:            netResp,
		Disks:              disks,
		Containers:         containers,
		Processes:          processes,
	}
}

func (c *collector) collectNetwork(ctx context.Context, elapsed time.Duration) SCNetworkResponse {
	ioStats, _ := gonet.IOCountersWithContext(ctx, true)
	ifaceStats := make([]SCNetworkInterfaceResponse, 0, len(ioStats))
	var primary *SCNetworkInterfaceResponse

	totalUploadBps := 0.0
	totalDownloadBps := 0.0
	totalUploadBytes := 0.0
	totalDownloadBytes := 0.0

	for _, stat := range ioStats {
		prev := c.prevNet[stat.Name]
		uploadBps := rate(stat.BytesSent, prev.BytesSent, elapsed)
		downloadBps := rate(stat.BytesRecv, prev.BytesRecv, elapsed)
		c.prevNet[stat.Name] = stat

		// skip interfaces whose total traffic is under 1 MB ever (likely noise/virtual)
		if stat.BytesSent+stat.BytesRecv < 1_000_000 {
			continue
		}

		ifaceStats = append(ifaceStats, SCNetworkInterfaceResponse{
			Name:               stat.Name,
			IP:                 ipForInterface(stat.Name),
			IsVirtual:          isVirtualInterface(stat.Name),
			UploadBps:          uploadBps,
			DownloadBps:        downloadBps,
			TotalUploadBytes:   float64(stat.BytesSent),
			TotalDownloadBytes: float64(stat.BytesRecv),
		})

		totalUploadBps += uploadBps
		totalDownloadBps += downloadBps
		totalUploadBytes += float64(stat.BytesSent)
		totalDownloadBytes += float64(stat.BytesRecv)

		// pick primary by highest combined bps
		if primary == nil || (uploadBps+downloadBps) > (primary.UploadBps+primary.DownloadBps) {
			cp := ifaceStats[len(ifaceStats)-1]
			primary = &cp
		}
	}

	active, passive, fail := tcpStats(ctx)
	established, timeWait, closeWait, synRecv := tcpStateCounts(ctx)
	retransRate := 0.0
	if tcp, err := gonet.ProtoCountersWithContext(ctx, []string{"tcp"}); err == nil && len(tcp) > 0 {
		retr := tcp[0].Stats["RetransSegs"]
		out := tcp[0].Stats["OutSegs"]
		if out > 0 {
			retransRate = (float64(retr) / float64(out)) * 100
		}
	}

	return SCNetworkResponse{
		TotalUploadBps:     totalUploadBps,
		TotalDownloadBps:   totalDownloadBps,
		TotalUploadBytes:   totalUploadBytes,
		TotalDownloadBytes: totalDownloadBytes,
		RetransRatePercent: retransRate,
		ActiveConn:         active,
		PassiveConn:        passive,
		FailConn:           fail,
		Established:        established,
		TimeWait:           timeWait,
		CloseWait:          closeWait,
		SynRecv:            synRecv,
		Interfaces:         ifaceStats,
		PrimaryInterface:   primary,
	}
}

func (c *collector) collectDisks(ctx context.Context, elapsed time.Duration) []SCDiskResponse {
	parts, _ := disk.PartitionsWithContext(ctx, false)
	ioStats, _ := disk.IOCountersWithContext(ctx)
	results := make([]SCDiskResponse, 0, len(parts))

	rootDevice := detectRootDevice(parts)

	for _, part := range parts {
		if skipDisk(part, rootDevice) {
			continue
		}
		usage, err := disk.UsageWithContext(ctx, part.Mountpoint)
		if err != nil {
			continue
		}
		ioStat, key, ok := matchIOStat(ioStats, part.Device)
		if !ok {
			// fallback: use zeroed counters to still surface the disk info
			key = part.Device
		}
		prev := c.prevDisk[key]
		c.prevDisk[key] = ioStat

		readBytesDelta := delta(ioStat.ReadBytes, prev.ReadBytes)
		writeBytesDelta := delta(ioStat.WriteBytes, prev.WriteBytes)
		readCountDelta := delta(ioStat.ReadCount, prev.ReadCount)
		writeCountDelta := delta(ioStat.WriteCount, prev.WriteCount)
		readTimeDelta := delta(ioStat.ReadTime, prev.ReadTime) // milliseconds
		writeTimeDelta := delta(ioStat.WriteTime, prev.WriteTime)

		results = append(results, SCDiskResponse{
			MountPoint:   part.Mountpoint,
			Device:       part.Device,
			FileSystem:   part.Fstype,
			UsedBytes:    float64(usage.Used),
			TotalBytes:   float64(usage.Total),
			ReadBps:      bytesPerSec(readBytesDelta, elapsed),
			ReadBytes:    float64(ioStat.ReadBytes),
			ReadIOPS:     opsPerSec(readCountDelta, elapsed),
			ReadDelayMs:  avgLatencyMs(readTimeDelta, readCountDelta),
			WriteBps:     bytesPerSec(writeBytesDelta, elapsed),
			WriteBytes:   float64(ioStat.WriteBytes),
			WriteIOPS:    opsPerSec(writeCountDelta, elapsed),
			WriteDelayMs: avgLatencyMs(writeTimeDelta, writeCountDelta),
		})
	}

	return results
}

func (c *collector) collectDocker(ctx context.Context, elapsed time.Duration) []SCContainerResponse {
	if c.dockerClient == nil {
		return nil
	}

	list, err := c.dockerClient.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return nil
	}

	results := make([]SCContainerResponse, 0, len(list))
	for _, item := range list {
		stats, err := c.containerStats(ctx, item.ID)
		if err != nil {
			continue
		}

		cpuPercent := dockerCPUPercent(stats)
		mem := float64(stats.MemoryStats.Usage)
		netRx, netTx := dockerNet(stats)
		blkRead, blkWrite := dockerBlockIO(stats)

		name := item.Names
		containerName := item.ID[:12]
		if len(name) > 0 {
			containerName = trimSlash(name[0])
		}

		results = append(results, SCContainerResponse{
			Name:             containerName,
			CPUUsagePercent:  cpuPercent,
			MemoryBytes:      mem,
			NetUploadBytes:   netTx,
			NetDownloadBytes: netRx,
			BlockReadBytes:   blkRead,
			BlockWriteBytes:  blkWrite,
		})
	}

	return results
}

func (c *collector) containerStats(ctx context.Context, id string) (container.StatsResponse, error) {
	stats, err := c.dockerClient.ContainerStats(ctx, id, false)
	if err != nil {
		return container.StatsResponse{}, err
	}
	defer stats.Body.Close()

	var decoded container.StatsResponse
	if err := jsonNewDecoder(stats.Body).Decode(&decoded); err != nil {
		return container.StatsResponse{}, err
	}
	return decoded, nil
}

// compute overall CPU percentages using delta between samples
func (c *collector) computeCPU(curr []cpu.TimesStat) (total, system, user, iowait, steal, idle float64) {
	if len(curr) == 0 {
		return 0, 0, 0, 0, 0, 0
	}

	if c.prevCPU == nil {
		if pct, err := cpu.Percent(0, false); err == nil && len(pct) > 0 {
			total = pct[0]
			idle = 100 - pct[0]
			c.prevCPU = curr
			return
		}
		c.prevCPU = curr
		return 0, 0, 0, 0, 0, 0
	}

	// use first aggregate entry (times=false returns one entry)
	prev := c.prevCPU[0]
	now := curr[0]

	dUser := now.User - prev.User
	dSystem := now.System - prev.System
	dIdle := now.Idle - prev.Idle
	dIowait := now.Iowait - prev.Iowait
	dSteal := now.Steal - prev.Steal
	dTotal := dUser + dSystem + dIdle + dIowait + dSteal + (now.Nice - prev.Nice) + (now.Irq - prev.Irq) + (now.Softirq - prev.Softirq)

	c.prevCPU = curr

	if dTotal <= 0 {
		if pct, err := cpu.Percent(0, false); err == nil && len(pct) > 0 {
			total = pct[0]
			idle = 100 - pct[0]
			// keep prevCPU so next delta works
			return
		}
		return 0, 0, 0, 0, 0, 0
	}

	total = (dTotal - dIdle) / dTotal * 100
	system = dSystem / dTotal * 100
	user = dUser / dTotal * 100
	iowait = dIowait / dTotal * 100
	steal = dSteal / dTotal * 100
	idle = dIdle / dTotal * 100
	return
}

// compute per-core ratios (0-1) using delta between samples
func (c *collector) computePerCore(curr []cpu.TimesStat) []SCCoreUsageResponse {
	if len(curr) == 0 {
		return nil
	}
	if c.prevPerCPU == nil {
		// fallback to cpu.Percent to give immediate data
		if pct, err := cpu.Percent(0, true); err == nil && len(pct) == len(curr) {
			out := make([]SCCoreUsageResponse, len(pct))
			for i, v := range pct {
				out[i] = SCCoreUsageResponse{
					System: v / 100.0,
					User:   0,
					IOWait: 0,
					Steal:  0,
					Idle:   1 - (v / 100.0),
				}
			}
			c.prevPerCPU = curr
			return out
		}
		c.prevPerCPU = curr
		return make([]SCCoreUsageResponse, len(curr))
	}
	n := len(curr)
	out := make([]SCCoreUsageResponse, n)
	for i := 0; i < n; i++ {
		prev := c.prevPerCPU[i]
		now := curr[i]
		dUser := now.User - prev.User
		dSystem := now.System - prev.System
		dIdle := now.Idle - prev.Idle
		dIowait := now.Iowait - prev.Iowait
		dSteal := now.Steal - prev.Steal
		dTotal := dUser + dSystem + dIdle + dIowait + dSteal + (now.Nice - prev.Nice) + (now.Irq - prev.Irq) + (now.Softirq - prev.Softirq)
		if dTotal <= 0 {
			continue
		}
		out[i] = SCCoreUsageResponse{
			System: dSystem / dTotal,
			User:   dUser / dTotal,
			IOWait: dIowait / dTotal,
			Steal:  dSteal / dTotal,
			Idle:   dIdle / dTotal,
		}
	}
	c.prevPerCPU = curr
	return out
}

func rate(current, previous uint64, elapsed time.Duration) float64 {
	if elapsed <= 0 {
		return 0
	}
	if current < previous {
		return 0
	}
	return float64(current-previous) / elapsed.Seconds()
}

func isVirtualInterface(name string) bool {
	virtualPrefixes := []string{"lo", "docker", "br-", "veth", "virbr", "vmnet", "tun", "tap", "gif", "stf", "p2p", "awdl", "utun"}
	for _, prefix := range virtualPrefixes {
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func ipForInterface(name string) *string {
	ifaces, err := stdnet.Interfaces()
	if err != nil {
		return nil
	}
	for _, iface := range ifaces {
		if iface.Name != name {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil
		}
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *stdnet.IPNet:
				if v.IP.To4() != nil {
					ip := v.IP.String()
					return &ip
				}
			case *stdnet.IPAddr:
				if v.IP.To4() != nil {
					ip := v.IP.String()
					return &ip
				}
			}
		}
	}
	return nil
}

func tcpStats(ctx context.Context) (active, passive, fail int) {
	if runtime.GOOS == "linux" {
		// Prefer /proc/net/snmp for accurate counters
		if a, p, f, ok := readProcNetSNMP(); ok {
			return a, p, f
		}
	}

	// Fallback: connection state counts (less accurate, esp. on macOS)
	conns, err := gonet.ConnectionsWithContext(ctx, "tcp")
	if err != nil {
		return 0, 0, 0
	}
	for _, c := range conns {
		switch c.Status {
		case "SYN-SENT":
			active++
		case "SYN-RECEIVED":
			passive++
		case "CLOSE":
			fail++
		}
	}
	return active, passive, fail
}

func tcpStateCounts(ctx context.Context) (established, timeWait, closeWait, synRecv int) {
	conns, err := gonet.ConnectionsWithContext(ctx, "tcp")
	if err != nil {
		return
	}
	for _, c := range conns {
		switch c.Status {
		case "ESTABLISHED":
			established++
		case "TIME_WAIT":
			timeWait++
		case "CLOSE_WAIT":
			closeWait++
		case "SYN_RECV", "SYN-RECEIVED":
			synRecv++
		}
	}
	return
}

func dockerCPUPercent(stats container.StatsResponse) float64 {
	// Docker already provides previous sample in PreCPUStats
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	if cpuDelta <= 0 || systemDelta <= 0 {
		return 0
	}
	online := float64(stats.CPUStats.OnlineCPUs)
	if online == 0 {
		online = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
	}
	return math.Min(100, (cpuDelta/systemDelta)*online*100)
}

func dockerNet(stats container.StatsResponse) (rx float64, tx float64) {
	for _, v := range stats.Networks {
		rx += float64(v.RxBytes)
		tx += float64(v.TxBytes)
	}
	return rx, tx
}

func dockerBlockIO(stats container.StatsResponse) (read float64, write float64) {
	for _, entry := range stats.BlkioStats.IoServiceBytesRecursive {
		switch entry.Op {
		case "Read":
			read += float64(entry.Value)
		case "Write":
			write += float64(entry.Value)
		}
	}
	return read, write
}

// return top N processes sorted by CPU percent
func topProcesses(ctx context.Context, limit int) []SCProcessResponse {
	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil
	}
	result := make([]SCProcessResponse, 0, len(procs))
	for _, p := range procs {
		name, _ := p.NameWithContext(ctx)
		cpuPct, _ := p.CPUPercentWithContext(ctx)
		memInfo, _ := p.MemoryInfoWithContext(ctx)
		memPct, _ := p.MemoryPercentWithContext(ctx)

		result = append(result, SCProcessResponse{
			PID:        p.Pid,
			Name:       name,
			CPUPercent: cpuPct,
			MemBytes: func() uint64 {
				if memInfo != nil {
					return memInfo.RSS
				}
				return 0
			}(),
			MemPercent: float64(memPct),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CPUPercent > result[j].CPUPercent
	})
	if len(result) > limit {
		result = result[:limit]
	}
	return result
}

func trimSlash(name string) string {
	if len(name) > 0 && name[0] == '/' {
		return name[1:]
	}
	return name
}

// match gopsutil IOCounters key with device path/partition
func matchIOStat(ioStats map[string]disk.IOCountersStat, device string) (disk.IOCountersStat, string, bool) {
	dev := trimSlash(strings.TrimPrefix(device, "/dev/"))

	try := func(key string) (disk.IOCountersStat, string, bool) {
		if stat, ok := ioStats[key]; ok {
			return stat, key, true
		}
		return disk.IOCountersStat{}, "", false
	}

	// 1) exact
	if stat, key, ok := try(dev); ok {
		return stat, key, ok
	}

	// 2) repeatedly strip trailing partition segments like disk3s3s1 -> disk3s3 -> disk3
	temp := dev
	for strings.Contains(temp, "s") {
		if idx := strings.LastIndex(temp, "s"); idx != -1 {
			temp = temp[:idx]
			if stat, key, ok := try(temp); ok {
				return stat, key, ok
			}
		} else {
			break
		}
	}

	// 3) linux style: drop trailing digits sda1 -> sda
	base := strings.TrimRightFunc(dev, func(r rune) bool { return r >= '0' && r <= '9' })
	if stat, key, ok := try(base); ok {
		return stat, key, ok
	}

	return disk.IOCountersStat{}, "", false
}

// Small wrapper to keep stdlib json out of import list in docs.
func jsonNewDecoder(r io.Reader) *json.Decoder {
	return json.NewDecoder(r)
}

func delta(curr, prev uint64) float64 {
	if curr < prev {
		return 0
	}
	return float64(curr - prev)
}

func bytesPerSec(deltaBytes float64, elapsed time.Duration) float64 {
	if elapsed <= 0 {
		return 0
	}
	return deltaBytes / elapsed.Seconds()
}

func opsPerSec(deltaOps float64, elapsed time.Duration) float64 {
	if elapsed <= 0 {
		return 0
	}
	return deltaOps / elapsed.Seconds()
}

// avg latency in milliseconds per operation
func avgLatencyMs(deltaTimeMs float64, deltaOps float64) float64 {
	if deltaOps <= 0 {
		return 0
	}
	return deltaTimeMs / deltaOps
}

// Linux: parse /proc/net/snmp for Tcp ActiveOpens, PassiveOpens, AttemptFails
func readProcNetSNMP() (active, passive, fails int, ok bool) {
	const path = "/proc/net/snmp"
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, 0, false
	}
	lines := strings.Split(string(data), "\n")
	for i := 0; i < len(lines)-1; i++ {
		if strings.HasPrefix(lines[i], "Tcp:") && strings.HasPrefix(lines[i+1], "Tcp:") {
			keys := strings.Fields(lines[i])
			vals := strings.Fields(lines[i+1])
			if len(keys) != len(vals) {
				continue
			}
			field := func(name string) (int, bool) {
				for idx, k := range keys {
					if k == name {
						v, err := strconv.Atoi(vals[idx])
						if err != nil {
							return 0, false
						}
						return v, true
					}
				}
				return 0, false
			}
			a, ok1 := field("ActiveOpens")
			p, ok2 := field("PassiveOpens")
			f, ok3 := field("AttemptFails")
			if ok1 && ok2 && ok3 {
				return a, p, f, true
			}
		}
	}
	return 0, 0, 0, false
}

// filter pseudo/virtual filesystems to keep disk list clean
func skipDisk(part disk.PartitionStat, rootDevice string) bool {
	if part.Fstype == "" || part.Mountpoint == "" {
		return true
	}

	excludeFS := map[string]struct{}{
		"devfs": {}, "procfs": {}, "proc": {}, "sysfs": {}, "autofs": {}, "tmpfs": {}, "overlay": {}, "tracefs": {},
		"cgroup": {}, "cgroup2": {}, "fuse.lxcfs": {}, "fdescfs": {}, "configfs": {}, "debugfs": {},
	}
	if _, ok := excludeFS[part.Fstype]; ok {
		return true
	}

	// skip macOS system APFS helper volumes
	if part.Fstype == "apfs" && strings.HasPrefix(part.Mountpoint, "/System/Volumes/") {
		return true
	}

	// skip macOS simulator / TimeMachine / recovery volumes
	if strings.HasPrefix(part.Mountpoint, "/Library/Developer/CoreSimulator") {
		return true
	}
	if strings.HasPrefix(part.Mountpoint, "/Volumes/Time Machine") ||
		strings.HasPrefix(part.Mountpoint, "/Volumes/com.apple.TimeMachine") {
		return true
	}
	if strings.HasPrefix(part.Mountpoint, "/Volumes/Recovery") ||
		strings.HasPrefix(part.Mountpoint, "/Volumes/Update") {
		return true
	}

	// macOS: only keep root disk
	if runtime.GOOS == "darwin" {
		return part.Mountpoint != "/"
	}

	// linux: keep all remaining
	return false
}

func detectRootDevice(parts []disk.PartitionStat) string {
	for _, part := range parts {
		if part.Mountpoint == "/" {
			return part.Device
		}
	}
	return ""
}

var collectorMain = newCollector()

func FetchData() SCServerStatusResponse {
	return collectorMain.snapshot(context.Background())
}

func FetchProcesses() []SCProcessResponse {
	return topProcesses(context.Background(), 100)
}
