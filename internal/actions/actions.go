// Package actions liga os nomes de ação usados em modules/*.toml (e nos
// planos do sistema) às funções que fazem o trabalho. Para criar uma ação
// nova: escreva a função aqui e registre em Registry().
package actions

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rafael-devR/termux-setup.sh/internal/apt"
	"github.com/Rafael-devR/termux-setup.sh/internal/engine"
	"github.com/Rafael-devR/termux-setup.sh/internal/mirror"
	"github.com/Rafael-devR/termux-setup.sh/internal/termux"
	"github.com/Rafael-devR/termux-setup.sh/internal/ubuntu"
)

// Registry devolve todas as ações conhecidas.
func Registry() engine.Registry {
	return engine.Registry{
		// apt / espelhos
		"mirror-probe":   mirrorProbe,
		"mirror-best":    mirrorBest,
		"mirror-apply":   mirrorApply,
		"apt-update":     aptUpdate,
		"apt-upgrade":    aptUpgrade,
		"apt-repair":     apt.Repair,
		"apt-cleanup":    aptCleanup,
		"termux-install": termuxInstall,
		// Termux
		"termux-pyvenv":   termuxPyvenv,
		"npm-global":      npmGlobal,
		"git-config":      gitConfig,
		"storage":         storage,
		"termux-commands": termuxCommands,
		"root-tsu":        rootTsu,
		// aparência
		"hush":            func(_ context.Context, s *engine.StepCtx) error { return termux.Hush(s.Env) },
		"theme":           theme,
		"font":            font,
		"keys":            keys,
		"reload-settings": reloadSettings,
		// Ubuntu
		"ubuntu-install": ubuntu.Install,
		"ubuntu-base":    ubBase,
		"ubuntu-apt":     ubApt,
		"ubuntu-node":    ubNode,
		"ubuntu-pyvenv":  ubPyvenv,
		"ubuntu-upgrade": ubUpgrade,
		"claude-install": claudeInstall,
		"ub-npm":         ubNpm,
		"ia-update":      iaUpdate,
		"ub-commands":    ubCommands,
		// desfazer
		"undo-apt":    undoApt,
		"undo-look":   func(_ context.Context, s *engine.StepCtx) error { return termux.UndoLook(s.Env, s.Out) },
		"undo-bashrc": undoBashrc,
	}
}

func arg(s *engine.StepCtx, i int) string {
	if i < len(s.Args) {
		return s.Args[i]
	}
	return ""
}

// ---------------------------------------------------------------- espelhos

func mirrorProbe(ctx context.Context, s *engine.StepCtx) error {
	ps := mirror.ProbeAll(ctx, s.Fetch, mirror.Defaults, s.Env.Work, nil)
	s.Logf("--- ranking (idade_h tempo url) ---")
	for _, p := range ps {
		s.Logf("%dh %.2fs %s", p.AgeHours, p.Latency.Seconds(), p.URL)
	}
	if len(ps) == 0 {
		return errors.New("nenhum espelho respondeu (sem internet?)")
	}
	s.Shared.SetCandidates(mirror.URLs(ps))
	return nil
}

func applyMirror(s *engine.StepCtx, url string) error {
	bk, err := mirror.Apply(s.Env, url, s.Stamp)
	if err != nil {
		return err
	}
	s.Shared.MarkMirror(url)
	s.Logf("Espelho aplicado: %s (backup em %s)", url, s.Env.Tilde(bk))
	return nil
}

func mirrorBest(_ context.Context, s *engine.StepCtx) error {
	url := s.Shared.NextMirror()
	if url == "" {
		s.Logf("Sem candidato; mantendo espelho atual.")
		return errors.New("nenhum espelho testado")
	}
	return applyMirror(s, url)
}

func mirrorApply(_ context.Context, s *engine.StepCtx) error { return applyMirror(s, arg(s, 0)) }

// aptUpdate tenta até 5 vezes, trocando de espelho a cada falha.
func aptUpdate(ctx context.Context, s *engine.StepCtx) error {
	for try := 0; ; try++ {
		err := apt.Get(ctx, s, "update")
		if err == nil || ctx.Err() != nil || try >= 4 {
			return err
		}
		url := s.Shared.NextMirror()
		if url == "" {
			return err
		}
		s.Logf("[aviso] apt update falhou; tentando outro espelho: %s", url)
		if err := applyMirror(s, url); err != nil {
			return err
		}
	}
}

func aptUpgrade(ctx context.Context, s *engine.StepCtx) error {
	if apt.Get(ctx, s, "full-upgrade") == nil || ctx.Err() != nil {
		return ctx.Err()
	}
	s.Logf("[aviso] upgrade falhou; reparando e tentando de novo")
	return apt.Repair(ctx, s)
}

func aptCleanup(ctx context.Context, s *engine.StepCtx) error {
	_ = s.Exec(ctx, "apt-get", "autoremove", "-y")
	_ = s.Exec(ctx, "apt-get", "clean")
	return nil
}

func termuxInstall(ctx context.Context, s *engine.StepCtx) error { return apt.Install(ctx, s, s.Args) }

// ---------------------------------------------------------------- Termux

func need(s *engine.StepCtx, prog string) error {
	if !s.Run.Has(prog) {
		return fmt.Errorf("%s não encontrado", prog)
	}
	return nil
}

func termuxPyvenv(ctx context.Context, s *engine.StepCtx) error {
	if err := need(s, "python"); err != nil {
		return err
	}
	venv := filepath.Join(s.Env.Home, "venv")
	if _, err := os.Stat(venv); err != nil {
		if err := s.Exec(ctx, "python", "-m", "venv", venv); err != nil {
			return err
		}
	}
	_ = s.Exec(ctx, filepath.Join(venv, "bin/python"), "-m", "pip", "install", "--upgrade", "pip")
	return s.Exec(ctx, filepath.Join(venv, "bin/pip"), "install", "requests", "rich")
}

func npmGlobal(ctx context.Context, s *engine.StepCtx) error {
	if err := need(s, "npm"); err != nil {
		return err
	}
	return s.Exec(ctx, "npm", append([]string{"install", "-g"}, s.Args...)...)
}

func gitConfig(ctx context.Context, s *engine.StepCtx) error {
	if err := need(s, "git"); err != nil {
		return err
	}
	if n := arg(s, 0); n != "" {
		if err := s.Exec(ctx, "git", "config", "--global", "user.name", n); err != nil {
			return err
		}
	}
	if m := arg(s, 1); m != "" {
		if err := s.Exec(ctx, "git", "config", "--global", "user.email", m); err != nil {
			return err
		}
	}
	return s.Exec(ctx, "git", "config", "--global", "init.defaultBranch", "main")
}

func storage(ctx context.Context, s *engine.StepCtx) error {
	_ = s.Exec(ctx, "termux-setup-storage")
	return nil
}

func termuxCommands(_ context.Context, s *engine.StepCtx) error {
	if err := termux.WriteCommands(s.Env); err != nil {
		return err
	}
	if err := termux.EnsureBashrc(s.Env.Home); err != nil {
		return err
	}
	s.Logf("Comandos criados: atualizar limpar info novo (projetos em ~/projetos)")
	return nil
}

func rootTsu(ctx context.Context, s *engine.StepCtx) error {
	if err := apt.Install(ctx, s, []string{"root-repo"}); err != nil {
		return err
	}
	_ = apt.Get(ctx, s, "update")
	return apt.Install(ctx, s, []string{"tsu"})
}

// ---------------------------------------------------------------- aparência

func theme(_ context.Context, s *engine.StepCtx) error {
	if err := termux.WriteTheme(s.Env, s.Stamp); err != nil {
		return err
	}
	s.Logf("Tema aplicado: cores do VS Code (fundo #1e1e1e)")
	return nil
}

func font(ctx context.Context, s *engine.StepCtx) error {
	return termux.InstallFont(ctx, s.Env, s.Fetch, s.Stamp, s.Out)
}

func keys(_ context.Context, s *engine.StepCtx) error {
	p := arg(s, 0)
	if p == "" {
		p = "1"
	}
	if err := termux.SetKeys(s.Env, s.Stamp, p); err != nil {
		return err
	}
	s.Logf("Barra de teclas: preset %s", p)
	return nil
}

func reloadSettings(ctx context.Context, s *engine.StepCtx) error {
	if s.Run.Has("termux-reload-settings") {
		_ = s.Exec(ctx, "termux-reload-settings")
	}
	s.Logf("Configurações recarregadas (se a fonte não mudar, feche e abra o Termux)")
	return nil
}

// ---------------------------------------------------------------- Ubuntu

func withUbuntu(fn func(ctx context.Context, s *engine.StepCtx) error) engine.ActionFunc {
	return func(ctx context.Context, s *engine.StepCtx) error {
		if err := ubuntu.Require(ctx, s); err != nil {
			return err
		}
		return fn(ctx, s)
	}
}

var (
	ubBase = withUbuntu(func(ctx context.Context, s *engine.StepCtx) error {
		return ubuntu.AsRoot(ctx, s, "base", map[string]string{"U": ubuntu.User})
	})
	ubApt = withUbuntu(func(ctx context.Context, s *engine.StepCtx) error {
		return ubuntu.AsRoot(ctx, s, "apt", map[string]string{"P": strings.Join(s.Args, " ")})
	})
	ubNode = withUbuntu(func(ctx context.Context, s *engine.StepCtx) error {
		return ubuntu.AsRoot(ctx, s, "node", map[string]string{"URL": ubuntu.NodeSourceURL})
	})
	ubPyvenv = withUbuntu(func(ctx context.Context, s *engine.StepCtx) error {
		return ubuntu.AsUser(ctx, s, "pyvenv", nil)
	})
	ubUpgrade = withUbuntu(func(ctx context.Context, s *engine.StepCtx) error {
		return ubuntu.AsRoot(ctx, s, "upgrade", nil)
	})
	claudeInstall = withUbuntu(func(ctx context.Context, s *engine.StepCtx) error {
		return ubuntu.AsUser(ctx, s, "claude", map[string]string{"URL": ubuntu.ClaudeURL})
	})
	ubNpm = withUbuntu(func(ctx context.Context, s *engine.StepCtx) error {
		return ubuntu.AsUser(ctx, s, "npm", map[string]string{"PKG": arg(s, 0)})
	})
	iaUpdate = withUbuntu(func(ctx context.Context, s *engine.StepCtx) error {
		return ubuntu.AsUser(ctx, s, "ia-update", nil)
	})
)

func ubCommands(_ context.Context, s *engine.StepCtx) error {
	if err := termux.WriteUbuntuCommands(s.Env, termux.UbuntuNames{Distro: ubuntu.Distro, User: ubuntu.User}); err != nil {
		return err
	}
	s.Logf("Comandos: claude codex gemini opencode ubuntu ubuntu-root ia-run ia-update")
	return nil
}

// ---------------------------------------------------------------- desfazer

func undoApt(ctx context.Context, s *engine.StepCtx) error {
	bk := termux.Oldest(s.Env, "backup-apt")
	if bk == "" {
		s.Logf("Nenhum backup do espelho (nunca foi trocado).")
		return nil
	}
	s.Logf("Restaurando espelho de: %s", s.Env.Tilde(bk))
	if err := mirror.Restore(s.Env, bk); err != nil {
		return err
	}
	_ = apt.Get(ctx, s, "update")
	return nil
}

func undoBashrc(_ context.Context, s *engine.StepCtx) error {
	if err := termux.UndoBashrc(s.Env.Home); err != nil {
		return err
	}
	s.Logf("Bloco do termux-setup removido do ~/.bashrc (comandos em ~/bin continuam lá)")
	return nil
}
