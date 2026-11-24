// Package sys concentra tudo o que toca o sistema: caminhos do Termux,
// execução de comandos e informações do aparelho. O resto do app só fala
// com o sistema por aqui, o que permite trocar tudo por versões falsas nos
// testes e no modo --teste.
package sys

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TermuxPrefix é o $PREFIX padrão do Termux.
const TermuxPrefix = "/data/data/com.termux/files/usr"

// Env são os caminhos e o modo de execução.
type Env struct {
	Prefix string // $PREFIX (onde ficam bin/, etc/apt/...)
	Home   string // $HOME
	LogDir string // logs, resumos e backups
	Work   string // pasta temporária desta execução
	DryRun bool   // modo --teste: nada é instalado de verdade
}

// AptSources é o sources.list do Termux.
func (e *Env) AptSources() string { return filepath.Join(e.Prefix, "etc/apt/sources.list") }

// AptSourcesD é a pasta sources.list.d do Termux.
func (e *Env) AptSourcesD() string { return filepath.Join(e.Prefix, "etc/apt/sources.list.d") }

// TermuxDir é ~/.termux (tema, fonte, barra de teclas).
func (e *Env) TermuxDir() string { return filepath.Join(e.Home, ".termux") }

// BinDir é ~/bin (comandos criados pelo app).
func (e *Env) BinDir() string { return filepath.Join(e.Home, "bin") }

// InTermux diz se estamos rodando dentro do Termux de verdade.
func InTermux() bool {
	if os.Getenv("TERMUX_VERSION") != "" {
		return true
	}
	_, err := os.Stat(filepath.Join(TermuxPrefix, "bin/bash"))
	return err == nil
}

// DetectEnv monta o ambiente real. Com dry=true cria um Termux "de mentira"
// em pastas temporárias (com um sources.list antigo, para os espelhos
// terem o que trocar), igual ao antigo modo --teste do script bash.
func DetectEnv(dry bool) (*Env, error) {
	if dry {
		root, err := os.MkdirTemp("", "tsetup-teste-")
		if err != nil {
			return nil, err
		}
		e := &Env{Prefix: filepath.Join(root, "usr"), Home: filepath.Join(root, "home"), DryRun: true}
		if err := os.MkdirAll(e.AptSourcesD(), 0o755); err != nil {
			return nil, err
		}
		if err := os.MkdirAll(e.Home, 0o755); err != nil {
			return nil, err
		}
		seed := "deb https://velho.exemplo/termux-main stable main\n"
		if err := os.WriteFile(e.AptSources(), []byte(seed), 0o644); err != nil {
			return nil, err
		}
		return e.finish()
	}
	e := &Env{Prefix: os.Getenv("PREFIX"), Home: os.Getenv("HOME")}
	if e.Prefix == "" {
		e.Prefix = TermuxPrefix
	}
	if e.Home == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("sem HOME: %w", err)
		}
		e.Home = h
	}
	return e.finish()
}

func (e *Env) finish() (*Env, error) {
	e.LogDir = filepath.Join(e.Home, "termux-setup-logs")
	if err := os.MkdirAll(e.LogDir, 0o755); err != nil {
		return nil, err
	}
	base := filepath.Join(e.Prefix, "tmp")
	if err := os.MkdirAll(base, 0o755); err != nil {
		base = ""
	}
	w, err := os.MkdirTemp(base, "tsetup.")
	if err != nil {
		return nil, err
	}
	e.Work = w
	return e, nil
}

// Cleanup apaga a pasta temporária da execução.
func (e *Env) Cleanup() { _ = os.RemoveAll(e.Work) }

// Stamp é o carimbo de data usado nos nomes de log e backup.
func Stamp(t time.Time) string { return t.Format("20060102-150405") }

// Tilde troca $HOME por ~ para mostrar caminhos curtos.
func (e *Env) Tilde(p string) string {
	if strings.HasPrefix(p, e.Home) {
		return "~" + strings.TrimPrefix(p, e.Home)
	}
	return p
}
