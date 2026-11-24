// Package mirror testa os espelhos do Termux, escolhe o melhor e troca o
// sources.list (formato antigo e deb822), sempre com backup.
package mirror

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Rafael-devR/termux-setup.sh/internal/sys"
)

// Defaults são os espelhos da wiki oficial do Termux.
var Defaults = []string{
	"https://packages-cf.termux.dev/apt/termux-main",
	"https://packages.termux.dev/apt/termux-main",
	"https://mirror.accum.se/mirror/termux.dev/termux-main",
	"https://mirrors.sahilister.in/termux/termux-main",
	"https://mirror.mwt.me/termux/main",
	"https://mirror.vern.cc/termux/termux-main",
	"https://mirror.fcix.net/termux/termux-main",
	"https://mirror.csclub.uwaterloo.ca/termux/termux-main",
	"https://mirror.freedif.org/termux/termux-main",
	"https://mirror.leitecastro.com/termux/termux-main",
	"https://termux.librehat.com/apt/termux-main",
	"https://mirrors.utermux.dev/termux/termux-main",
}

// FreshHours: espelho atualizado há menos que isso é "fresco".
const FreshHours = 48

// Probe é o resultado do teste de um espelho.
type Probe struct {
	URL      string
	AgeHours int           // idade da última atualização do espelho
	Latency  time.Duration // tempo para baixar o Release
}

// Fresh diz se o espelho está atualizado.
func (p Probe) Fresh() bool { return p.AgeHours <= FreshHours }

// Host é o nome do servidor (para mostrar).
func (p Probe) Host() string { return Host(p.URL) }

// Host extrai o servidor de uma URL.
func Host(url string) string {
	h := strings.TrimPrefix(strings.TrimPrefix(url, "https://"), "http://")
	if i := strings.IndexByte(h, '/'); i >= 0 {
		h = h[:i]
	}
	return h
}

var dateRe = regexp.MustCompile(`(?m)^Date:\s*(.+)$`)

// ReleaseAge lê o campo Date de um arquivo Release.
func ReleaseAge(data []byte, now time.Time) (int, error) {
	m := dateRe.FindSubmatch(data)
	if m == nil {
		return 0, errors.New("Release sem Date")
	}
	s := strings.TrimSpace(string(m[1]))
	for _, layout := range []string{time.RFC1123, time.RFC1123Z, "Mon, 2 Jan 2006 15:04:05 MST", "Mon, 2 Jan 2006 15:04:05 -0700"} {
		if t, err := time.Parse(layout, s); err == nil {
			return max(0, int(now.Sub(t).Hours())), nil
		}
	}
	return 0, fmt.Errorf("data inválida: %q", s)
}

// ProbeOne baixa o Release de um espelho e mede idade e tempo.
func ProbeOne(ctx context.Context, f sys.Fetcher, url, dir string) (Probe, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	tmp, err := os.CreateTemp(dir, "rel.")
	if err != nil {
		return Probe{}, err
	}
	tmp.Close()
	defer os.Remove(tmp.Name())
	t0 := time.Now()
	if err := f.Fetch(ctx, strings.TrimSuffix(url, "/")+"/dists/stable/Release", tmp.Name()); err != nil {
		return Probe{}, err
	}
	lat := time.Since(t0)
	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		return Probe{}, err
	}
	age, err := ReleaseAge(data, time.Now())
	if err != nil {
		age = 9999
	}
	return Probe{URL: url, AgeHours: age, Latency: lat}, nil
}

// ProbeAll testa todos em paralelo; onResult (opcional) é chamado a cada
// resposta, para a tela mostrar os resultados chegando.
func ProbeAll(ctx context.Context, f sys.Fetcher, urls []string, dir string, onResult func(Probe)) []Probe {
	var (
		mu  sync.Mutex
		out []Probe
		wg  sync.WaitGroup
	)
	for _, u := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			p, err := ProbeOne(ctx, f, u, dir)
			if err != nil {
				return
			}
			mu.Lock()
			out = append(out, p)
			mu.Unlock()
			if onResult != nil {
				onResult(p)
			}
		}(u)
	}
	wg.Wait()
	return Rank(out)
}

// Rank ordena: primeiro os frescos (mais rápido antes), depois os velhos
// (menos velho antes).
func Rank(ps []Probe) []Probe {
	out := append([]Probe(nil), ps...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Fresh() != b.Fresh() {
			return a.Fresh()
		}
		if a.Fresh() {
			return a.Latency < b.Latency
		}
		return a.AgeHours < b.AgeHours
	})
	return out
}

// URLs extrai as URLs na ordem.
func URLs(ps []Probe) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.URL
	}
	return out
}

// BackupOnce copia sources.list e sources.list.d para os logs na 1ª troca
// da execução (as próximas não sobrescrevem o original).
func BackupOnce(env *sys.Env, stamp string) (string, error) {
	bk := filepath.Join(env.LogDir, "backup-apt-"+stamp)
	if _, err := os.Stat(bk); err == nil {
		return bk, nil
	}
	if err := os.MkdirAll(filepath.Join(bk, "sources.list.d"), 0o755); err != nil {
		return "", err
	}
	if data, err := os.ReadFile(env.AptSources()); err == nil {
		if err := os.WriteFile(filepath.Join(bk, "sources.list"), data, 0o644); err != nil {
			return "", err
		}
	}
	entries, _ := os.ReadDir(env.AptSourcesD())
	for _, e := range entries {
		if data, err := os.ReadFile(filepath.Join(env.AptSourcesD(), e.Name())); err == nil {
			_ = os.WriteFile(filepath.Join(bk, "sources.list.d", e.Name()), data, 0o644)
		}
	}
	return bk, nil
}

var (
	suitesStable = regexp.MustCompile(`(?m)^Suites:\s*stable`)
	urisLine     = regexp.MustCompile(`(?m)^URIs:.*$`)
	debLine      = regexp.MustCompile(`(?m)^deb `)
)

// Apply troca o espelho. No formato novo (deb822) muda só a linha URIs do
// repositório "stable" e comenta o sources.list antigo; senão reescreve o
// sources.list.
func Apply(env *sys.Env, url, stamp string) (backup string, err error) {
	url = strings.TrimSuffix(url, "/")
	if url == "" {
		return "", errors.New("URL vazia")
	}
	if backup, err = BackupOnce(env, stamp); err != nil {
		return "", err
	}
	changed := false
	files, _ := filepath.Glob(filepath.Join(env.AptSourcesD(), "*.sources"))
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if suitesStable.Match(data) && urisLine.Match(data) {
			data = urisLine.ReplaceAll(data, []byte("URIs: "+url))
			if err := os.WriteFile(f, data, 0o644); err != nil {
				return backup, err
			}
			changed = true
		}
	}
	if changed {
		if data, err := os.ReadFile(env.AptSources()); err == nil {
			return backup, os.WriteFile(env.AptSources(), debLine.ReplaceAll(data, []byte("#deb ")), 0o644)
		}
		return backup, nil
	}
	return backup, os.WriteFile(env.AptSources(), []byte("deb "+url+" stable main\n"), 0o644)
}

// Restore volta os arquivos de um backup.
func Restore(env *sys.Env, backup string) error {
	if data, err := os.ReadFile(filepath.Join(backup, "sources.list")); err == nil {
		if err := os.WriteFile(env.AptSources(), data, 0o644); err != nil {
			return err
		}
	}
	entries, _ := os.ReadDir(filepath.Join(backup, "sources.list.d"))
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(backup, "sources.list.d", e.Name()))
		if err != nil {
			continue
		}
		if err := os.WriteFile(filepath.Join(env.AptSourcesD(), e.Name()), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

var mirrorLine = regexp.MustCompile(`(?m)^(?:deb |URIs:)\s*(\S+)`)

// Current devolve o servidor do espelho em uso.
func Current(env *sys.Env) string {
	files, _ := filepath.Glob(filepath.Join(env.AptSourcesD(), "*.sources"))
	for _, f := range append([]string{env.AptSources()}, files...) {
		if data, err := os.ReadFile(f); err == nil {
			if m := mirrorLine.FindSubmatch(data); m != nil {
				return Host(string(m[1]))
			}
		}
	}
	return "?"
}
