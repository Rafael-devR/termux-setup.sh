// Package app monta as peças (ambiente, executor, catálogo, motor) que a
// CLI e a interface usam. É o único lugar que decide "real ou --teste".
package app

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/Rafael-devR/termux-setup.sh/internal/actions"
	"github.com/Rafael-devR/termux-setup.sh/internal/catalog"
	"github.com/Rafael-devR/termux-setup.sh/internal/engine"
	"github.com/Rafael-devR/termux-setup.sh/internal/mirror"
	"github.com/Rafael-devR/termux-setup.sh/internal/sys"
	"github.com/Rafael-devR/termux-setup.sh/internal/ubuntu"
	"github.com/Rafael-devR/termux-setup.sh/modules"
)

// Version é preenchida no build (-ldflags "-X ...app.Version=v6.0.0").
var Version = "dev"

// Services são as dependências compartilhadas.
type Services struct {
	Env    *sys.Env
	Runner sys.Runner
	Fetch  sys.Fetcher
	Info   sys.Info
	Cat    *catalog.Catalog
	Reg    engine.Registry
	Anim   bool // animações ligadas
}

// Config escolhe o modo.
type Config struct {
	DryRun bool // --teste
	NoAnim bool // --sem-animacao
}

// New monta os serviços. No modo teste tudo é simulado (Termux de mentira
// em pastas temporárias, comandos fingidos).
func New(cfg Config) (*Services, error) {
	env, err := sys.DetectEnv(cfg.DryRun)
	if err != nil {
		return nil, err
	}
	cat, err := catalog.Load(modules.FS)
	if err != nil {
		return nil, err
	}
	reg := actions.Registry()
	if err := cat.Validate(reg.Has); err != nil {
		return nil, err
	}
	s := &Services{Env: env, Cat: cat, Reg: reg, Info: sys.DetectInfo(cfg.DryRun),
		Anim: !cfg.NoAnim && os.Getenv("TSETUP_ANIM") != "0"}
	if cfg.DryRun {
		s.Runner = sys.DryRunner{Fail: os.Getenv("TSETUP_FALHAR"), Unavailable: []string{"nmap"}}
		s.Fetch = sys.DryFetcher{}
	} else {
		s.Runner = sys.ExecRunner{}
		s.Fetch = sys.CurlFetcher{Run: s.Runner}
	}
	return s, nil
}

// Close limpa a pasta temporária.
func (s *Services) Close() { s.Env.Cleanup() }

// Engine cria um motor.
func (s *Services) Engine() *engine.Engine {
	return &engine.Engine{Reg: s.Reg, Env: s.Env, Runner: s.Runner, Fetch: s.Fetch, Info: s.Info}
}

// HasUbuntu diz se o Ubuntu está instalado (demora ~1s: rode em segundo plano).
func (s *Services) HasUbuntu(ctx context.Context) bool { return ubuntu.Has(ctx, s.Runner) }

// Mirror é o servidor do espelho em uso.
func (s *Services) Mirror() string { return mirror.Current(s.Env) }

// FreeMB é o espaço livre.
func (s *Services) FreeMB() int64 { return sys.FreeMB(s.Env.Home) }

// GitConfigured diz se o git já tem nome configurado.
func (s *Services) GitConfigured(ctx context.Context) bool {
	out, err := sys.Output(ctx, s.Runner, sys.C("git", "config", "--global", "user.name"))
	return err == nil && strings.TrimSpace(out) != "" && !s.Env.DryRun
}

// StorageReady diz se o acesso ao armazenamento já foi liberado.
func (s *Services) StorageReady() bool {
	_, err := os.Stat(s.Env.Home + "/storage")
	return err == nil
}

// VerifyRow é uma linha da tela "Verificar".
type VerifyRow struct {
	Where   string // "termux" ou "ubuntu"
	Name    string
	Version string // "" = não instalado
}

// Verify confere as versões de tudo que o catálogo sabe verificar.
// onRow é chamado a cada resultado (a tela mostra chegando).
func (s *Services) Verify(ctx context.Context, onRow func(VerifyRow)) {
	var ub []string
	for _, m := range s.Cat.All() {
		for _, c := range m.Verify {
			if m.VerifyIn == catalog.TargetUbuntu {
				ub = append(ub, c)
				continue
			}
			f := strings.Fields(c)
			row := VerifyRow{Where: catalog.TargetTermux, Name: f[0]}
			if s.Runner.Has(f[0]) {
				cctx, cancel := context.WithTimeout(ctx, 8*time.Second)
				out, err := sys.Output(cctx, s.Runner, sys.C(f[0], f[1:]...))
				cancel()
				if err == nil || out != "" {
					row.Version = firstLine(out)
				}
			}
			onRow(row)
		}
	}
	if len(ub) == 0 || !s.HasUbuntu(ctx) {
		onRow(VerifyRow{Where: catalog.TargetUbuntu, Name: "ubuntu"})
		return
	}
	onRow(VerifyRow{Where: catalog.TargetUbuntu, Name: "ubuntu", Version: ubuntu.Version(ctx, s.Runner)})
	out, _ := ubuntu.RunAsUser(ctx, s.Runner, ubuntu.VerifyScript(ub))
	seen := map[string]bool{}
	for _, l := range strings.Split(out, "\n") {
		f := strings.Fields(l)
		if len(f) < 2 || seen[f[1]] {
			continue
		}
		seen[f[1]] = true
		row := VerifyRow{Where: catalog.TargetUbuntu, Name: f[1]}
		if f[0] == "OK" {
			row.Version = strings.Join(f[2:], " ")
		}
		onRow(row)
	}
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if s == "" {
		s = "instalado"
	}
	return s
}
