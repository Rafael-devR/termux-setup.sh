package engine

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/Rafael-devR/termux-setup.sh/internal/plan"
	"github.com/Rafael-devR/termux-setup.sh/internal/sys"
)

// ---------------------------------------------------------------- Progress

// Progress guarda o estado ao vivo para a interface ler a cada quadro
// (a tela não precisa processar cada linha do apt, só o retrato atual).
type Progress struct {
	mu        sync.Mutex
	plan      *plan.Plan
	cur       int
	status    []Status
	dur       []time.Duration
	start     time.Time
	stepStart time.Time
	lines     []string
	nLines    int
	result    *Result
}

// Snapshot é uma cópia segura do estado.
type Snapshot struct {
	Plan      *plan.Plan
	Current   int
	Status    []Status
	Durations []time.Duration
	Start     time.Time
	StepStart time.Time
	Lines     []string // últimas linhas (a mais nova no fim)
	NLines    int      // total de linhas desde o início
	Result    *Result  // != nil quando terminou
}

const keepLines = 64

func (p *Progress) Start(pl *plan.Plan) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.plan, p.status, p.dur = pl, make([]Status, len(pl.Steps)), make([]time.Duration, len(pl.Steps))
	p.start, p.stepStart = time.Now(), time.Now()
}

func (p *Progress) StepStart(i int) {
	p.mu.Lock()
	p.cur, p.stepStart, p.status[i] = i, time.Now(), Running
	p.mu.Unlock()
}

func (p *Progress) Line(_ int, line string) {
	p.mu.Lock()
	p.lines = append(p.lines, line)
	if len(p.lines) > keepLines {
		p.lines = p.lines[len(p.lines)-keepLines:]
	}
	p.nLines++
	p.mu.Unlock()
}

func (p *Progress) StepEnd(i int, st Status, d time.Duration) {
	p.mu.Lock()
	p.status[i], p.dur[i] = st, d
	p.mu.Unlock()
}

func (p *Progress) Finish(r *Result) {
	p.mu.Lock()
	p.result = r
	copy(p.status, r.Status)
	p.mu.Unlock()
}

// Snapshot devolve o retrato atual.
func (p *Progress) Snapshot() Snapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	return Snapshot{Plan: p.plan, Current: p.cur, Status: append([]Status(nil), p.status...),
		Durations: append([]time.Duration(nil), p.dur...), Start: p.start, StepStart: p.stepStart,
		Lines: append([]string(nil), p.lines...), NLines: p.nLines, Result: p.result}
}

// ---------------------------------------------------------------- texto

// TextReporter mostra o progresso em texto simples (sem terminal
// interativo, ex.: --sim com saída redirecionada).
type TextReporter struct {
	W    io.Writer
	plan *plan.Plan
}

func (t *TextReporter) Start(p *plan.Plan) {
	t.plan = p
	fmt.Fprintf(t.W, "== %s: %d passos ==\n", p.Title, len(p.Steps))
}
func (t *TextReporter) StepStart(i int) {
	fmt.Fprintf(t.W, "[%d/%d] %s\n", i+1, len(t.plan.Steps), t.plan.Steps[i].Label)
}
func (t *TextReporter) Line(int, string) {}
func (t *TextReporter) StepEnd(_ int, st Status, d time.Duration) {
	mark := "✓"
	if st != OK {
		mark = "✗ " + string(st)
	}
	fmt.Fprintf(t.W, "      %s %ds\n", mark, int(d.Seconds()))
}
func (t *TextReporter) Finish(r *Result) {
	fmt.Fprintf(t.W, "\n%s · %d ok · %d falhas · %d pendentes\n", r.Overall, r.NOK, r.NFailed, r.NPending)
	for _, w := range r.Warnings {
		fmt.Fprintf(t.W, "! %s\n", w)
	}
	for _, h := range r.Hints {
		fmt.Fprintf(t.W, "- %s\n", h)
	}
	fmt.Fprintf(t.W, "resumo: %s\n", r.SummaryPath)
}

// ---------------------------------------------------------------- notificação

// Notifier manda uma notificação do Android no fim (precisa do Termux:API).
type Notifier struct{ Run sys.Runner }

func (Notifier) Start(*plan.Plan)                   {}
func (Notifier) StepStart(int)                      {}
func (Notifier) Line(int, string)                   {}
func (Notifier) StepEnd(int, Status, time.Duration) {}
func (n Notifier) Finish(r *Result) {
	if r.Interrupted || !n.Run.Has("termux-notification") {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()
		msg := fmt.Sprintf("%s: %d ok, %d falhas", r.Overall, r.NOK, r.NFailed)
		_ = n.Run.Run(ctx, sys.C("termux-notification", "--title", "Termux Setup", "--content", msg), nil)
		_ = n.Run.Run(ctx, sys.C("termux-vibrate", "-d", "250"), nil)
	}()
}
