package sys

import (
	"bufio"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

// Info são dados do aparelho mostrados na interface.
type Info struct {
	Arch    string // saída de "uname -m" (aarch64, armv7l, x86_64...)
	Android string // versão do Android
	Model   string // modelo do aparelho
}

// Is64 diz se a arquitetura é de 64 bits (Claude Code e Codex só rodam nelas).
func (i Info) Is64() bool { return Is64(i.Arch) }

// Is64 diz se a arquitetura (uname -m) é de 64 bits.
func Is64(arch string) bool {
	switch arch {
	case "aarch64", "arm64", "x86_64", "amd64":
		return true
	}
	return false
}

// SizeFactor: os tamanhos do catálogo são para arm64; em 32 bits ~80%.
func (i Info) SizeFactor() int {
	if i.Is64() {
		return 100
	}
	return 80
}

// DetectInfo lê arquitetura, versão do Android e modelo.
func DetectInfo(dry bool) Info {
	var u unix.Utsname
	arch := "?"
	if unix.Uname(&u) == nil {
		arch = unix.ByteSliceToString(u.Machine[:])
	}
	if dry {
		return Info{Arch: arch, Android: "teste", Model: "teste"}
	}
	return Info{Arch: arch, Android: getprop("ro.build.version.release"), Model: getprop("ro.product.model")}
}

func getprop(key string) string {
	out, err := exec.Command("getprop", key).Output()
	if err != nil {
		return "?"
	}
	if s := strings.TrimSpace(string(out)); s != "" {
		return s
	}
	return "?"
}

// FreeMB é o espaço livre (MB) no disco que contém path.
func FreeMB(path string) int64 {
	var st syscall.Statfs_t
	if syscall.Statfs(path, &st) != nil {
		return 0
	}
	return int64(st.Bavail) * int64(st.Bsize) / (1024 * 1024)
}

// Mem é o uso de memória lido de /proc/meminfo (o Android costuma deixar ler).
type Mem struct{ TotalMB, AvailMB int64 }

// UsedPct é a porcentagem de memória em uso.
func (m Mem) UsedPct() int {
	if m.TotalMB == 0 {
		return 0
	}
	return int(100 - m.AvailMB*100/m.TotalMB)
}

// ReadMem lê /proc/meminfo; ok=false se o sistema não deixar.
func ReadMem() (Mem, bool) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return Mem{}, false
	}
	defer f.Close()
	var m Mem
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		kb, _ := strconv.ParseInt(fields[1], 10, 64)
		switch fields[0] {
		case "MemTotal:":
			m.TotalMB = kb / 1024
		case "MemAvailable:":
			m.AvailMB = kb / 1024
		}
	}
	return m, m.TotalMB > 0
}

// DirBytes soma o tamanho dos arquivos de uma pasta (sem subpastas). Usado
// para medir a velocidade real de download: o apt baixa para o cache.
func DirBytes(dir string) int64 {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	var n int64
	for _, e := range entries {
		if info, err := e.Info(); err == nil && info.Mode().IsRegular() {
			n += info.Size()
		}
	}
	return n
}
