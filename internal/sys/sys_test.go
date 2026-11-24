package sys

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExecRunnerOutputAndExitCode(t *testing.T) {
	r := ExecRunner{}
	out, err := Output(context.Background(), r, C("sh", "-c", "echo ola; echo erro >&2; exit 3"))
	var ee *ExitError
	if !errors.As(err, &ee) || ee.Code != 3 {
		t.Fatalf("erro: %v", err)
	}
	if !strings.Contains(out, "ola") || !strings.Contains(out, "erro") {
		t.Fatalf("stdout e stderr juntos: %q", out)
	}
	cmd := C("cat")
	cmd.Stdin = "segredo"
	if out, _ := Output(context.Background(), r, cmd); out != "segredo" {
		t.Fatal(out)
	}
}

func TestExecRunnerCancelKillsGroup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	t0 := time.Now()
	err := ExecRunner{}.Run(ctx, C("sh", "-c", "sleep 30 & sleep 30"), nil)
	if err == nil || time.Since(t0) > 5*time.Second {
		t.Fatalf("cancelamento demorou %v (%v)", time.Since(t0), err)
	}
}

func TestDryRunner(t *testing.T) {
	d := DryRunner{Fail: "quebra", Unavailable: []string{"nmap"}, Delay: time.Millisecond}
	var b strings.Builder
	if err := d.Run(context.Background(), C("apt-get", "install", "git"), &b); err != nil || !strings.HasPrefix(b.String(), "[teste] apt-get") {
		t.Fatal(err, b.String())
	}
	if d.Run(context.Background(), C("apt-cache", "show", "nmap"), nil) == nil {
		t.Fatal("nmap deveria estar indisponível")
	}
	if d.Run(context.Background(), C("echo", "quebra"), nil) == nil {
		t.Fatal("deveria simular falha")
	}
}

func TestDryEnvIsIsolatedAndCleaned(t *testing.T) {
	e, err := DetectEnv(true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(e.Home, os.TempDir()) || !e.DryRun {
		t.Fatalf("HOME de teste fora do temporário: %s", e.Home)
	}
	if data, _ := os.ReadFile(e.AptSources()); !strings.Contains(string(data), "velho.exemplo") {
		t.Fatal("sources.list semente")
	}
	root := filepath.Dir(e.Home)
	e.Cleanup()
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("o Termux de mentira deveria ser apagado")
	}
}

func TestDryFetcher(t *testing.T) {
	dir := t.TempDir()
	f := DryFetcher{}
	rel := filepath.Join(dir, "Release")
	if err := f.Fetch(context.Background(), "https://x/dists/stable/Release", rel); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(rel); !strings.Contains(string(data), "Date: ") {
		t.Fatal(string(data))
	}
}

func TestIs64(t *testing.T) {
	if !Is64("aarch64") || Is64("armv7l") || (Info{Arch: "armv7l"}).SizeFactor() != 80 {
		t.Fatal()
	}
}
