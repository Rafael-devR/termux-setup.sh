package tools

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rafael-devR/termux-setup.sh/internal/sys"
)

func box(t *testing.T) *Toolbox {
	t.Helper()
	d := t.TempDir()
	env := &sys.Env{Home: filepath.Join(d, "home"), Prefix: filepath.Join(d, "usr"), LogDir: filepath.Join(d, "home", "termux-setup-logs"), Work: d}
	os.MkdirAll(env.Home, 0o755)
	return &Toolbox{Run: &sys.FakeRunner{}, Env: env, Info: sys.Info{Arch: "aarch64"}, Proc: filepath.Join(d, "proc"), Sysfs: filepath.Join(d, "sys")}
}

func write(t *testing.T, p, body string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBatteryFromTermuxAPI(t *testing.T) {
	b := box(t)
	b.Run = &sys.FakeRunner{Rules: []sys.Rule{{Match: "termux-battery-status",
		Out: `{"health":"GOOD","percentage":57,"plugged":"UNPLUGGED","status":"DISCHARGING","temperature":33.4}`}}}
	got, err := b.ReadBattery(context.Background())
	if err != nil || got.Percent != 57 || got.Temp != 33.4 || got.Charging() {
		t.Fatal(got, err)
	}
	// sem o pacote: erro claro
	b.Run = &sys.FakeRunner{Missing: []string{"termux-battery-status"}}
	if _, err := b.ReadBattery(context.Background()); err != ErrNoAPI {
		t.Fatal(err)
	}
	// resposta que não é JSON (app Termux:API ausente) também vira ErrNoAPI
	b.Run = &sys.FakeRunner{Rules: []sys.Rule{{Match: "termux-battery-status", Out: "oops"}}}
	if _, err := b.ReadBattery(context.Background()); err != ErrNoAPI {
		t.Fatal(err)
	}
}

func TestMemoryCPUThermalProcs(t *testing.T) {
	b := box(t)
	write(t, filepath.Join(b.Proc, "meminfo"), "MemTotal: 6144000 kB\nMemAvailable: 2048000 kB\nSwapTotal: 2097152 kB\nSwapFree: 1048576 kB\n")
	m, ok := b.ReadMemory()
	if !ok || m.TotalMB != 6000 || m.AvailMB != 2000 || m.UsedPct() != 67 || m.SwapTotalMB != 2048 {
		t.Fatal(m, ok)
	}

	write(t, filepath.Join(b.Proc, "stat"), "cpu 100 0 100 700 100 0 0 0\ncpu0 50 0 50 350 50 0 0 0\n")
	a, ok := b.ReadCPU()
	if !ok || len(a) != 2 {
		t.Fatal(a, ok)
	}
	write(t, filepath.Join(b.Proc, "stat"), "cpu 300 0 200 800 100 0 0 0\ncpu0 100 0 50 450 50 0 0 0\n")
	c, _ := b.ReadCPU()
	if p := CPUPercent(a, c); p[0] != 75 || p[1] != 33 {
		t.Fatal(p)
	}
	// bloqueado pelo Android: ok=false, sem pânico
	if _, ok := (&Toolbox{Proc: filepath.Join(t.TempDir(), "nada")}).ReadCPU(); ok {
		t.Fatal("deveria falhar")
	}

	write(t, filepath.Join(b.Sysfs, "class/thermal/thermal_zone0/temp"), "38500\n")
	write(t, filepath.Join(b.Sysfs, "class/thermal/thermal_zone1/temp"), "41.0\n")
	write(t, filepath.Join(b.Sysfs, "class/thermal/thermal_zone2/temp"), "-274000\n")
	if v, ok := b.ReadThermal(); !ok || v != 41.0 {
		t.Fatal(v, ok)
	}

	write(t, filepath.Join(b.Proc, "10/status"), "Name:\tnode\nVmRSS:\t   307200 kB\n")
	write(t, filepath.Join(b.Proc, "11/status"), "Name:\tbash\nVmRSS:\t 4096 kB\n")
	write(t, filepath.Join(b.Proc, "self/status"), "Name:\tx\n")
	ps := b.TopProcs(5)
	if len(ps) != 2 || ps[0].Name != "node" || ps[0].RSSMB != 300 {
		t.Fatal(ps)
	}
	if Terminate(os.Getpid()) == nil || Terminate(1) == nil {
		t.Fatal("não pode encerrar o próprio app nem o init")
	}
}

func TestSummarize(t *testing.T) {
	st := Summarize([]float64{10, 20, 0, 30})
	if st.Sent != 4 || st.Lost != 1 || st.Min != 10 || st.Max != 30 || st.Avg != 20 || st.Jit != 10 {
		t.Fatal(st)
	}
	if z := Summarize([]float64{0, 0}); z.Lost != 2 || z.Avg != 0 {
		t.Fatal(z)
	}
}

func TestSpeedTestParsesCurl(t *testing.T) {
	b := box(t)
	b.Run = &sys.FakeRunner{Rules: []sys.Rule{
		{Match: "time_connect", Out: "0.025"},
		{Match: "speed_download", Out: "12500000"}, // 100 Mbps
		{Match: "speed_upload", Out: "2500000"},    // 20 Mbps
	}}
	var stages []string
	r, err := b.SpeedTest(context.Background(), func(s string, _ SpeedResult) { stages = append(stages, s) })
	if err != nil || r.DownMbps != 100 || r.UpMbps != 20 || r.PingMs < 24 || r.PingMs > 26 {
		t.Fatal(r, err)
	}
	if strings.Join(stages, ",") != "ping,download,upload,fim" {
		t.Fatal(stages)
	}
	// sem internet: erro claro
	b.Run = &sys.FakeRunner{Rules: []sys.Rule{{Match: "time_connect", Out: "0.000"}}}
	if _, err := b.SpeedTest(context.Background(), func(string, SpeedResult) {}); err == nil || !strings.Contains(err.Error(), "sem internet") {
		t.Fatal(err)
	}
}

func TestNetCheckReportsWhereItBreaks(t *testing.T) {
	b := box(t)
	b.Run = &sys.FakeRunner{Rules: []sys.Rule{{Match: "github.com", Out: "000 0.0"}, {Match: "", Out: "200 0.050"}}}
	var got []Check
	b.NetCheck(context.Background(), func(c Check) { got = append(got, c) })
	if len(got) != 5 {
		t.Fatal(len(got))
	}
	bad := 0
	for _, c := range got {
		if c.Status == Fail {
			bad++
			if c.Fix == "" {
				t.Fatal("falha sem correção sugerida:", c.Name)
			}
		}
	}
	if bad != 2 { // DNS e GitHub usam github.com
		t.Fatalf("falhas: %d (%v)", bad, got)
	}
}

func TestDoctorFindsProblems(t *testing.T) {
	b := box(t)
	t.Setenv("TERMUX_APK_RELEASE", "GOOGLE_PLAY")
	b.Run = &sys.FakeRunner{Rules: []sys.Rule{{Match: "curl", Out: "200 0.01"}}, Missing: []string{"termux-battery-status"}}
	var got []Check
	b.Doctor(context.Background(), func(c Check) { got = append(got, c) })
	find := func(name string) *Check {
		for i := range got {
			if got[i].Name == name {
				return &got[i]
			}
		}
		return nil
	}
	if c := find("Origem do Termux"); c == nil || c.Status != Fail || !strings.Contains(c.Fix, "F-Droid") {
		t.Fatal(c)
	}
	if c := find("Termux:API"); c == nil || c.Status != Warn {
		t.Fatal(c)
	}
	if c := find("Internet, DNS e repositórios"); c == nil || c.Status != OK {
		t.Fatal(c)
	}
}

func TestCleanKeepsRestorePoints(t *testing.T) {
	b := box(t)
	cache := filepath.Join(b.Env.Home, ".cache", "pip", "x.whl")
	write(t, cache, strings.Repeat("x", 1000))
	old := filepath.Join(b.Env.LogDir, "setup-antigo.log")
	write(t, old, "velho")
	os.Chtimes(old, time.Now().Add(-30*24*time.Hour), time.Now().Add(-30*24*time.Hour))
	novo := filepath.Join(b.Env.LogDir, "setup-novo.log")
	write(t, novo, "novo")
	bk := filepath.Join(b.Env.LogDir, "backup-apt-1", "sources.list")
	write(t, bk, "restauração")
	os.Chtimes(filepath.Dir(bk), time.Now().Add(-90*24*time.Hour), time.Now().Add(-90*24*time.Hour))

	items := b.CleanScan(context.Background())
	byID := map[string]int64{}
	for _, it := range items {
		byID[it.ID] = it.Bytes
	}
	if byID["cache"] != 1000 || byID["logs"] != 5 {
		t.Fatal(byID)
	}
	freed, err := b.Clean(context.Background(), []string{"cache", "logs"})
	if err != nil || freed != 1005 {
		t.Fatal(freed, err)
	}
	if _, err := os.Stat(cache); err == nil {
		t.Fatal("cache ficou")
	}
	if _, err := os.Stat(old); err == nil {
		t.Fatal("log antigo ficou")
	}
	for _, keep := range []string{novo, bk, filepath.Join(b.Env.Home, ".cache")} {
		if _, err := os.Stat(keep); err != nil {
			t.Fatal("apagou o que não devia:", keep)
		}
	}
}

func TestBackupRoundTripAndTraversal(t *testing.T) {
	b := box(t)
	write(t, filepath.Join(b.Env.Home, ".bashrc"), "alias x=y\n")
	write(t, filepath.Join(b.Env.Home, ".termux", "colors.properties"), "background=#1e1e1e\n")
	write(t, filepath.Join(b.Env.Home, "bin", "atualizar"), "#!/bin/sh\n")
	file, err := b.Backup()
	if err != nil {
		t.Fatal(err)
	}
	os.Remove(filepath.Join(b.Env.Home, ".bashrc"))
	os.WriteFile(filepath.Join(b.Env.Home, ".termux", "colors.properties"), []byte("estragado"), 0o644)
	n, err := b.Restore(file)
	if err != nil || n != 3 {
		t.Fatal(n, err)
	}
	if got, _ := os.ReadFile(filepath.Join(b.Env.Home, ".termux", "colors.properties")); string(got) != "background=#1e1e1e\n" {
		t.Fatalf("%q", got)
	}
	if len(b.Backups()) != 1 {
		t.Fatal(b.Backups())
	}

	// backup malicioso: caminho com ../
	evil := filepath.Join(t.TempDir(), "evil.tar.gz")
	f, _ := os.Create(evil)
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: "../fora.txt", Mode: 0o644, Size: 1, Typeflag: tar.TypeReg})
	tw.Write([]byte("x"))
	tw.Close()
	gz.Close()
	f.Close()
	if _, err := b.Restore(evil); err == nil {
		t.Fatal("aceitou caminho com ../")
	}
	if _, err := os.Stat(filepath.Join(b.Env.Home, "..", "fora.txt")); err == nil {
		t.Fatal("escreveu fora da casa")
	}
	// nada para guardar
	empty := box(t)
	if _, err := empty.Backup(); err == nil {
		t.Fatal("backup vazio deveria falhar")
	}
}

func TestWiFiSignal(t *testing.T) {
	for rssi, want := range map[int]string{-50: "ótimo", -60: "bom", -70: "fraco", -85: "muito fraco"} {
		if got := (WiFi{RSSI: rssi}).Signal(); got != want {
			t.Errorf("%d: %s", rssi, got)
		}
	}
}
