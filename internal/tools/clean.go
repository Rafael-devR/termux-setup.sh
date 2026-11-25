package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rafael-devR/termux-setup.sh/internal/sys"
)

// CleanItem é algo que dá para limpar.
type CleanItem struct {
	ID    string
	Label string
	Bytes int64
}

// logAge: logs mais antigos que isso podem ir embora.
const logAge = 14 * 24 * time.Hour

func (t *Toolbox) cachePaths() []string {
	return []string{filepath.Join(t.Env.Home, ".cache"), filepath.Join(t.Env.Home, ".npm", "_cacache")}
}

func (t *Toolbox) oldLogs() []string {
	var out []string
	ents, _ := os.ReadDir(t.Env.LogDir)
	for _, e := range ents {
		n := e.Name()
		if !e.Type().IsRegular() || !(strings.HasPrefix(n, "setup-") || strings.HasPrefix(n, "resumo-")) {
			continue
		}
		if info, err := e.Info(); err == nil && time.Since(info.ModTime()) > logAge {
			out = append(out, filepath.Join(t.Env.LogDir, n))
		}
	}
	return out
}

// CleanScan mede o que dá para liberar. Os backups do espelho/aparência
// (usados pelo "Desfazer") nunca entram.
func (t *Toolbox) CleanScan(ctx context.Context) []CleanItem {
	items := []CleanItem{{ID: "apt", Label: "Cache de pacotes (apt)", Bytes: DirSize(ctx, filepath.Join(t.Env.Prefix, "var/cache/apt"))}}
	var cache int64
	for _, p := range t.cachePaths() {
		cache += DirSize(ctx, p)
	}
	items = append(items, CleanItem{ID: "cache", Label: "Caches de programas (~/.cache, npm)", Bytes: cache})
	var logs int64
	for _, p := range t.oldLogs() {
		if info, err := os.Stat(p); err == nil {
			logs += info.Size()
		}
	}
	items = append(items, CleanItem{ID: "logs", Label: "Logs do termux-setup com mais de 14 dias", Bytes: logs})
	return items
}

// Clean apaga os itens escolhidos e devolve quantos bytes liberou.
func (t *Toolbox) Clean(ctx context.Context, ids []string) (int64, error) {
	var freed int64
	for _, id := range ids {
		switch id {
		case "apt":
			dir := filepath.Join(t.Env.Prefix, "var/cache/apt")
			before := DirSize(ctx, dir)
			if err := t.Run.Run(ctx, sys.C("apt-get", "clean"), nil); err != nil {
				return freed, err
			}
			freed += before - DirSize(ctx, dir)
		case "cache":
			for _, p := range t.cachePaths() {
				ents, _ := os.ReadDir(p)
				for _, e := range ents { // esvazia a pasta, mantém ela
					full := filepath.Join(p, e.Name())
					freed += DirSize(ctx, full)
					_ = os.RemoveAll(full)
				}
			}
		case "logs":
			for _, p := range t.oldLogs() {
				if info, err := os.Stat(p); err == nil {
					freed += info.Size()
				}
				_ = os.Remove(p)
			}
		}
	}
	return freed, nil
}
