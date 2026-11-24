package actions_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Rafael-devR/termux-setup.sh/internal/actions"
	"github.com/Rafael-devR/termux-setup.sh/internal/catalog"
	"github.com/Rafael-devR/termux-setup.sh/internal/engine"
	"github.com/Rafael-devR/termux-setup.sh/internal/plan"
	"github.com/Rafael-devR/termux-setup.sh/internal/sys"
	"github.com/Rafael-devR/termux-setup.sh/internal/testutil"
	"github.com/Rafael-devR/termux-setup.sh/modules"
)

func TestEveryManifestActionExists(t *testing.T) {
	c, _ := catalog.Load(modules.FS)
	if err := c.Validate(actions.Registry().Has); err != nil {
		t.Fatal(err)
	}
	for _, p := range []*plan.Plan{plan.Update(true), plan.Repair(), plan.Undo(), plan.Mirror("x")} {
		for _, s := range p.Steps {
			if !actions.Registry().Has(s.Action) {
				t.Errorf("plano %s usa ação inexistente %s", p.Title, s.Action)
			}
		}
	}
}

// Ponta a ponta: instala TUDO no modo teste e depois desfaz.
func TestDryRunEverythingThenUndo(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	c, _ := catalog.Load(modules.FS)
	env := testutil.Env(t)
	e := &engine.Engine{Reg: actions.Registry(), Env: env, Fetch: sys.DryFetcher{},
		Runner: sys.DryRunner{Delay: time.Millisecond, Unavailable: []string{"nmap"}}, Info: sys.Info{Arch: "aarch64"}}
	p, err := plan.Install(c, append(c.Preset("tudo"), "root"), plan.Options{GitName: "Ana", KeysPreset: 1}, "aarch64", 100)
	if err != nil {
		t.Fatal(err)
	}
	r := e.Run(context.Background(), p)
	if r.Overall != engine.Complete {
		t.Fatalf("%s\n%s", r.Overall, testutil.Read(t, r.SummaryPath))
	}
	if r.MirrorUsed == "" || len(r.Unavailable) != 1 {
		t.Fatalf("espelho %q indisponíveis %v", r.MirrorUsed, r.Unavailable)
	}
	if !strings.Contains(testutil.Read(t, env.Home+"/.bashrc"), "termux-setup") {
		t.Fatal("bashrc")
	}
	u := e.Run(context.Background(), plan.Undo())
	if u.Overall != engine.Complete {
		t.Fatal(testutil.Read(t, u.SummaryPath))
	}
	if got := testutil.Read(t, env.AptSources()); !strings.Contains(got, "velho.exemplo") {
		t.Fatalf("desfazer não voltou o espelho: %q", got)
	}
}
