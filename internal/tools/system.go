package tools

import (
	"bufio"
	"context"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

// ---------------------------------------------------------------- memória

// Memory é a RAM e o swap, em MB.
type Memory struct{ TotalMB, AvailMB, SwapTotalMB, SwapFreeMB int64 }

// UsedPct é a porcentagem de RAM em uso.
func (m Memory) UsedPct() int {
	if m.TotalMB == 0 {
		return 0
	}
	return int(100 - m.AvailMB*100/m.TotalMB)
}

// ReadMemory lê /proc/meminfo.
func (t *Toolbox) ReadMemory() (Memory, bool) {
	if t.Fake && t.Proc == "" {
		return Memory{TotalMB: 5800, AvailMB: 2100 + int64(rand.IntN(200)), SwapTotalMB: 2048, SwapFreeMB: 1500}, true
	}
	f, err := os.Open(filepath.Join(t.proc(), "meminfo"))
	if err != nil {
		return Memory{}, false
	}
	defer f.Close()
	var m Memory
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fs := strings.Fields(sc.Text())
		if len(fs) < 2 {
			continue
		}
		kb, _ := strconv.ParseInt(fs[1], 10, 64)
		switch fs[0] {
		case "MemTotal:":
			m.TotalMB = kb / 1024
		case "MemAvailable:":
			m.AvailMB = kb / 1024
		case "SwapTotal:":
			m.SwapTotalMB = kb / 1024
		case "SwapFree:":
			m.SwapFreeMB = kb / 1024
		}
	}
	return m, m.TotalMB > 0
}

// ---------------------------------------------------------------- CPU

// CPUTimes são os contadores acumulados de um núcleo (/proc/stat).
type CPUTimes struct{ Busy, Total uint64 }

// ReadCPU lê os contadores: índice 0 = total, 1.. = cada núcleo.
// ok=false quando o Android bloqueia /proc/stat (comum nas versões novas).
func (t *Toolbox) ReadCPU() ([]CPUTimes, bool) {
	f, err := os.Open(filepath.Join(t.proc(), "stat"))
	if err != nil {
		return nil, false
	}
	defer f.Close()
	var out []CPUTimes
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fs := strings.Fields(sc.Text())
		if len(fs) < 5 || !strings.HasPrefix(fs[0], "cpu") {
			continue
		}
		var vals []uint64
		for _, s := range fs[1:] {
			v, _ := strconv.ParseUint(s, 10, 64)
			vals = append(vals, v)
		}
		var total uint64
		for _, v := range vals {
			total += v
		}
		idle := vals[3]
		if len(vals) > 4 {
			idle += vals[4] // iowait
		}
		out = append(out, CPUTimes{Busy: total - idle, Total: total})
	}
	return out, len(out) > 0
}

// CPUPercent calcula o uso (0..100) de cada núcleo entre duas leituras.
func CPUPercent(prev, cur []CPUTimes) []int {
	n := min(len(prev), len(cur))
	out := make([]int, n)
	for i := 0; i < n; i++ {
		dt := cur[i].Total - prev[i].Total
		if dt == 0 || cur[i].Total < prev[i].Total {
			continue
		}
		out[i] = int((cur[i].Busy - prev[i].Busy) * 100 / dt)
	}
	return out
}

// ReadThermal devolve a maior temperatura (°C) das zonas térmicas legíveis.
func (t *Toolbox) ReadThermal() (float64, bool) {
	zones, _ := filepath.Glob(filepath.Join(t.sysfs(), "class/thermal/thermal_zone*/temp"))
	best, ok := 0.0, false
	for _, z := range zones {
		b, err := os.ReadFile(z)
		if err != nil {
			continue
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(string(b)), 64)
		if err != nil {
			continue
		}
		if v > 1000 {
			v /= 1000
		}
		if v > 0 && v < 130 && v > best {
			best, ok = v, true
		}
	}
	return best, ok
}

// ---------------------------------------------------------------- processos

// Proc é um processo.
type Proc struct {
	PID   int
	Name  string
	RSSMB int64
}

// TopProcs lista os processos que mais usam memória (o Android só deixa
// ver os do próprio app).
func (t *Toolbox) TopProcs(n int) []Proc {
	if t.Fake && t.Proc == "" {
		return []Proc{{101, "node", 310}, {102, "claude", 240}, {103, "bash", 12}, {104, "termux-setup", 24}}
	}
	dirs, _ := filepath.Glob(filepath.Join(t.proc(), "[0-9]*"))
	var ps []Proc
	for _, d := range dirs {
		pid, err := strconv.Atoi(filepath.Base(d))
		if err != nil {
			continue
		}
		b, err := os.ReadFile(filepath.Join(d, "status"))
		if err != nil {
			continue
		}
		p := Proc{PID: pid}
		for _, l := range strings.Split(string(b), "\n") {
			switch {
			case strings.HasPrefix(l, "Name:"):
				p.Name = strings.TrimSpace(strings.TrimPrefix(l, "Name:"))
			case strings.HasPrefix(l, "VmRSS:"):
				fs := strings.Fields(l)
				if len(fs) >= 2 {
					kb, _ := strconv.ParseInt(fs[1], 10, 64)
					p.RSSMB = kb / 1024
				}
			}
		}
		if p.Name != "" {
			ps = append(ps, p)
		}
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].RSSMB > ps[j].RSSMB })
	if len(ps) > n {
		ps = ps[:n]
	}
	return ps
}

// Terminate pede (SIGTERM) que um processo termine. Nunca o próprio app.
func Terminate(pid int) error {
	if pid <= 1 || pid == os.Getpid() {
		return os.ErrInvalid
	}
	return syscall.Kill(pid, syscall.SIGTERM)
}

// ---------------------------------------------------------------- disco

// Disk é o espaço do armazenamento, em MB.
type Disk struct{ TotalMB, FreeMB int64 }

// UsedPct é a porcentagem usada.
func (d Disk) UsedPct() int {
	if d.TotalMB == 0 {
		return 0
	}
	return int(100 - d.FreeMB*100/d.TotalMB)
}

// ReadDisk mede o armazenamento que contém a casa do Termux.
func (t *Toolbox) ReadDisk() Disk {
	var st syscall.Statfs_t
	if syscall.Statfs(t.Env.Home, &st) != nil {
		return Disk{}
	}
	bs := int64(st.Bsize)
	return Disk{TotalMB: int64(st.Blocks) * bs / (1 << 20), FreeMB: int64(st.Bavail) * bs / (1 << 20)}
}

// DirSize soma os arquivos de uma pasta (recursivo). Respeita o cancelamento.
func DirSize(ctx context.Context, dir string) int64 {
	var n int64
	_ = filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			return nil
		}
		if d.Type().IsRegular() {
			if info, err := d.Info(); err == nil {
				n += info.Size()
			}
		}
		return nil
	})
	return n
}
