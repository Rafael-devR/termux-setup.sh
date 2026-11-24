// Package testutil tem ajudas para os testes.
package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rafael-devR/termux-setup.sh/internal/sys"
)

// Env cria um Termux de mentira em pastas temporárias.
func Env(t *testing.T) *sys.Env {
	t.Helper()
	root := t.TempDir()
	e := &sys.Env{Prefix: filepath.Join(root, "usr"), Home: filepath.Join(root, "home"), DryRun: true}
	e.LogDir = filepath.Join(e.Home, "termux-setup-logs")
	e.Work = filepath.Join(root, "work")
	for _, d := range []string{e.AptSourcesD(), e.LogDir, e.Work} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	Write(t, e.AptSources(), "deb https://velho.exemplo/termux-main stable main\n")
	return e
}

// Write grava um arquivo (criando as pastas).
func Write(t *testing.T, p, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Read lê um arquivo ("" se não existir).
func Read(t *testing.T, p string) string {
	t.Helper()
	b, _ := os.ReadFile(p)
	return string(b)
}
