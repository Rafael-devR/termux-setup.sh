package tools

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Rafael-devR/termux-setup.sh/internal/sys"
)

const (
	speedDown = "https://speed.cloudflare.com/__down?bytes=25000000"
	speedUp   = "https://speed.cloudflare.com/__up"
	pingURL   = "https://1.1.1.1/cdn-cgi/trace"
)

// curlW roda o curl e devolve o que -w imprime (campos separados por espaço).
func (t *Toolbox) curlW(ctx context.Context, format string, maxSec int, args ...string) ([]string, error) {
	a := append([]string{"-s", "-o", "/dev/null", "-w", format, "--max-time", strconv.Itoa(maxSec)}, args...)
	out, err := sys.Output(ctx, t.Run, sys.C("curl", a...))
	return strings.Fields(out), err
}

// PingOnce mede o tempo de conexão (ms) até o servidor da Cloudflare.
func (t *Toolbox) PingOnce(ctx context.Context) (float64, error) {
	if t.Fake {
		return 18 + rand.Float64()*14, nil
	}
	f, err := t.curlW(ctx, "%{time_connect}", 5, pingURL)
	if err != nil || len(f) < 1 {
		return 0, fmt.Errorf("sem resposta")
	}
	s, _ := strconv.ParseFloat(f[0], 64)
	if s <= 0 {
		return 0, fmt.Errorf("sem resposta")
	}
	return s * 1000, nil
}

// PingStats resume uma série de pings (ms); perdas são ignoradas na média.
type PingStats struct {
	Sent, Lost         int
	Min, Avg, Max, Jit float64
}

// Summarize calcula min/média/máx/jitter. samples tem 0 nas perdas.
func Summarize(samples []float64) PingStats {
	st := PingStats{Sent: len(samples), Min: math.MaxFloat64}
	var sum float64
	var ok []float64
	for _, s := range samples {
		if s <= 0 {
			st.Lost++
			continue
		}
		ok = append(ok, s)
		sum += s
		st.Min, st.Max = math.Min(st.Min, s), math.Max(st.Max, s)
	}
	if len(ok) == 0 {
		st.Min = 0
		return st
	}
	st.Avg = sum / float64(len(ok))
	for i := 1; i < len(ok); i++ {
		st.Jit += math.Abs(ok[i] - ok[i-1])
	}
	if len(ok) > 1 {
		st.Jit /= float64(len(ok) - 1)
	}
	return st
}

// SpeedResult é o resultado do teste de velocidade (Mbps / ms).
type SpeedResult struct {
	PingMs, JitterMs float64
	DownMbps         float64
	UpMbps           float64
}

// SpeedTest mede ping, download e upload. say recebe o estágio atual.
func (t *Toolbox) SpeedTest(ctx context.Context, say func(stage string, partial SpeedResult)) (SpeedResult, error) {
	var r SpeedResult
	say("ping", r)
	var samples []float64
	for i := 0; i < 5; i++ {
		ms, err := t.PingOnce(ctx)
		if err != nil {
			ms = 0
		}
		samples = append(samples, ms)
		if ctx.Err() != nil {
			return r, ctx.Err()
		}
	}
	st := Summarize(samples)
	if st.Lost == st.Sent {
		return r, fmt.Errorf("sem internet (nenhum ping respondeu)")
	}
	r.PingMs, r.JitterMs = st.Avg, st.Jit
	say("download", r)
	if t.Fake {
		time.Sleep(900 * time.Millisecond)
		r.DownMbps = 80 + rand.Float64()*40
	} else {
		f, err := t.curlW(ctx, "%{speed_download}", 12, speedDown)
		if ctx.Err() != nil {
			return r, ctx.Err()
		}
		if err != nil && len(f) == 0 {
			return r, fmt.Errorf("download falhou")
		}
		if len(f) > 0 {
			b, _ := strconv.ParseFloat(f[0], 64)
			r.DownMbps = b * 8 / 1e6
		}
	}
	say("upload", r)
	if t.Fake {
		time.Sleep(900 * time.Millisecond)
		r.UpMbps = 20 + rand.Float64()*15
	} else {
		body := filepath.Join(t.Env.Work, "upload.bin")
		if err := os.WriteFile(body, make([]byte, 6<<20), 0o600); err != nil {
			return r, err
		}
		defer os.Remove(body)
		f, _ := t.curlW(ctx, "%{speed_upload}", 12, "--data-binary", "@"+body, speedUp)
		if ctx.Err() != nil {
			return r, ctx.Err()
		}
		if len(f) > 0 {
			b, _ := strconv.ParseFloat(f[0], 64)
			r.UpMbps = b * 8 / 1e6
		}
	}
	say("fim", r)
	return r, nil
}

// Check é o resultado de uma verificação.
type Check struct {
	Name   string
	Status Status
	Detail string
	Fix    string // o que fazer quando não está OK
}

// Status de uma verificação.
type Status int

const (
	OK Status = iota
	Warn
	Fail
)

// NetCheck confere, em ordem, onde a internet quebra.
func (t *Toolbox) NetCheck(ctx context.Context, emit func(Check)) {
	type target struct{ name, url, fix string }
	for _, g := range []target{
		{"Internet (IP direto)", "https://1.1.1.1/cdn-cgi/trace", "Ligue o Wi-Fi/dados móveis. Se houver VPN, desligue e teste."},
		{"DNS (nomes de site)", "https://github.com", "A internet funciona mas os nomes não resolvem: troque o DNS do Wi-Fi (ex.: 1.1.1.1) ou desligue o DNS privado do Android."},
		{"GitHub (releases)", "https://github.com/Rafael-devR/termux-setup.sh/releases/latest", "O GitHub está bloqueado ou instável nesta rede."},
		{"Repositório do Termux", "https://packages.termux.dev/apt/termux-main/dists/stable/Release", "Use Espelhos para escolher outro servidor."},
		{"Repositório do Ubuntu", "http://archive.ubuntu.com/ubuntu/dists/resolute/Release", "Tente outra rede; o espelho do Ubuntu não respondeu."},
	} {
		if ctx.Err() != nil {
			return
		}
		if t.Fake {
			time.Sleep(350 * time.Millisecond)
			emit(Check{Name: g.name, Status: OK, Detail: fmt.Sprintf("%d ms", 40+rand.IntN(120))})
			continue
		}
		f, _ := t.curlW(ctx, "%{http_code} %{time_total}", 8, "-L", g.url)
		c := Check{Name: g.name}
		if len(f) == 2 && f[0] != "000" && (strings.HasPrefix(f[0], "2") || strings.HasPrefix(f[0], "3") || f[0] == "404") {
			s, _ := strconv.ParseFloat(f[1], 64)
			c.Detail = fmt.Sprintf("%d ms", int(s*1000))
		} else {
			c.Status, c.Detail, c.Fix = Fail, "sem resposta", g.fix
		}
		emit(c)
	}
}
