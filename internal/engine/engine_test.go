package engine_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Rafael-devR/termux-setup.sh/internal/engine"
	"github.com/Rafael-devR/termux-setup.sh/internal/plan"
	"github.com/Rafael-devR/termux-setup.sh/internal/sys"
	"github.com/Rafael-devR/termux-setup.sh/internal/testutil"
)

func newEngine(t *testing.T, reg engine.Registry) *engine.Engine {
	return &engine.Engine{Reg: reg, Env: testutil.Env(t), Runner: &sys.FakeRunner{}, Fetch: sys.DryFetcher{}}
}

func steps(actions ...string) *plan.Plan {
	p := &plan.Plan{Title: "teste"}
	for _, a := range actions {
		p.Steps = append(p.Steps, plan.Step{Label: "passo " + a, Action: a, Seconds: 1})
	}
	return p
}

var reg = engine.Registry{
	"ok":    func(_ context.Context, s *engine.StepCtx) error { s.Logf("tudo certo\rprogresso"); return nil },
	"fail":  func(_ context.Context, s *engine.StepCtx) error { s.Logf("E: Unable to locate package x"); return errors.New("deu ruim") },
	"warn":  func(_ context.Context, s *engine.StepCtx) error { s.Warn("cuidado"); s.Unavailable("pkg"); return nil },
	"panic": func(context.Context, *engine.StepCtx) error { panic("bug") },
	"slow": func(ctx context.Context, _ *engine.StepCtx) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			return nil
		}
	},
}

func TestRunCompleteWritesSummary(t *testing.T) {
	e := newEngine(t, reg)
	prog := &engine.Progress{}
	r := e.Run(context.Background(), steps("ok", "warn"), prog)
	if r.Overall != engine.Complete || r.NOK != 2 {
		t.Fatalf("resultado: %+v", r)
	}
	sum := testutil.Read(t, r.SummaryPath)
	for _, want := range []string{"Status : COMPLETO", "--- Avisos ---", "cuidado", "pkg"} {
		if !strings.Contains(sum, want) {
			t.Errorf("resumo sem %q:\n%s", want, sum)
		}
	}
	snap := prog.Snapshot()
	if snap.Result == nil || snap.NLines < 2 {
		t.Fatalf("progresso: %+v", snap)
	}
	if got := snap.Lines; got[0] != "tudo certo" || got[1] != "progresso" {
		t.Fatalf("linhas (\\r deve quebrar): %q", got)
	}
}

func TestFailureContinuesAndDiagnoses(t *testing.T) {
	r := newEngine(t, reg).Run(context.Background(), steps("fail", "ok", "panic"))
	if r.Overall != engine.WithFailures || r.NFailed != 2 || r.Status[1] != engine.OK {
		t.Fatalf("status: %v %s", r.Status, r.Overall)
	}
	if len(r.Hints) == 0 || !strings.Contains(r.Hints[0], "não existe") {
		t.Fatalf("diagnóstico: %v", r.Hints)
	}
}

func TestUnknownActionFails(t *testing.T) {
	r := newEngine(t, reg).Run(context.Background(), steps("nao-existe"))
	if r.Status[0] != engine.Failed {
		t.Fatal(r.Status)
	}
}

func TestCancelMarksInterrupted(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	r := newEngine(t, reg).Run(ctx, steps("ok", "slow", "ok"))
	if !r.Interrupted || r.Overall != engine.Incomplete {
		t.Fatalf("%+v", r)
	}
	if r.Status[1] != engine.Interrupted || r.Status[2] != engine.NotRun {
		t.Fatal(r.Status)
	}
}
