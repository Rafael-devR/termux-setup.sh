package apt_test

import (
	"context"
	"io"
	"slices"
	"testing"

	"github.com/Rafael-devR/termux-setup.sh/internal/apt"
	"github.com/Rafael-devR/termux-setup.sh/internal/engine"
	"github.com/Rafael-devR/termux-setup.sh/internal/sys"
)

func ctxFor(r *sys.FakeRunner) (*engine.StepCtx, *engine.Shared) {
	sh := &engine.Shared{}
	return &engine.StepCtx{Run: r, Out: io.Discard, Shared: sh}, sh
}

func TestInstallPicksAlternativesAndSkipsMissing(t *testing.T) {
	r := &sys.FakeRunner{Rules: []sys.Rule{
		{Match: "apt-cache show nodejs-lts", Code: 100},
		{Match: "apt-cache show fantasma", Code: 100},
	}}
	s, sh := ctxFor(r)
	if err := apt.Install(context.Background(), s, []string{"git", "nodejs-lts|nodejs", "fantasma"}); err != nil {
		t.Fatal(err)
	}
	if r.Ran("install -y") != 1 || r.Ran(" git nodejs") != 1 {
		t.Fatalf("chamadas: %v", r.Calls)
	}
	if !slices.Equal(sh.Unavailable, []string{"fantasma"}) {
		t.Fatal(sh.Unavailable)
	}
}

func TestInstallFallsBackOneByOne(t *testing.T) {
	r := &sys.FakeRunner{Rules: []sys.Rule{
		{Match: "Timeout=25 git vim", Code: 100}, // bloco falha
		{Match: "Timeout=25 vim", Code: 100},     // vim sozinho falha
	}}
	s, sh := ctxFor(r)
	if err := apt.Install(context.Background(), s, []string{"git", "vim"}); err == nil {
		t.Fatal("esperava erro")
	}
	if !slices.Equal(sh.FailedPkgs, []string{"vim"}) {
		t.Fatal(sh.FailedPkgs)
	}
}
