// Package engine executa um plano passo a passo. Não sabe nada de
// interface: quem quer acompanhar (a tela animada, o modo texto, a
// notificação) se registra como Observer.
package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Rafael-devR/termux-setup.sh/internal/diagnose"
	"github.com/Rafael-devR/termux-setup.sh/internal/plan"
	"github.com/Rafael-devR/termux-setup.sh/internal/sys"
)

// Status de um passo.
type Status string

const (
	Pending     Status = ""
	Running     Status = "RODANDO"
	OK          Status = "OK"
	Failed      Status = "FALHOU"
	Interrupted Status = "INTERROMPIDO"
	NotRun      Status = "NAO EXECUTADO"
)

// Situação final da execução.
const (
	Complete     = "COMPLETO"
	WithFailures = "CONCLUIDO COM FALHAS"
	Incomplete   = "INCOMPLETO"
)

// ActionFunc implementa uma ação ("termux-install", "font"...).
type ActionFunc func(ctx context.Context, s *StepCtx) error

// Registry liga nomes de ação (usados nos manifestos) às funções.
type Registry map[string]ActionFunc

// Has diz se a ação existe (usado na validação do catálogo).
func (r Registry) Has(name string) bool { _, ok := r[name]; return ok }

// Shared é o estado que os passos de uma mesma execução compartilham.
type Shared struct {
	mu          sync.Mutex
	Candidates  []string // espelhos do melhor para o pior
	Tried       []string // espelhos já aplicados
	MirrorUsed  string
	Warnings    []string
	Unavailable []string
	FailedPkgs  []string
}

func (s *Shared) add(list *[]string, v string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !slices.Contains(*list, v) {
		*list = append(*list, v)
	}
}

// SetCandidates guarda a lista de espelhos ordenada.
func (s *Shared) SetCandidates(c []string) { s.mu.Lock(); s.Candidates = c; s.mu.Unlock() }

// NextMirror devolve o melhor espelho ainda não tentado.
func (s *Shared) NextMirror() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.Candidates {
		if !slices.Contains(s.Tried, c) {
			return c
		}
	}
	return ""
}

// MarkMirror registra um espelho aplicado.
func (s *Shared) MarkMirror(url string) {
	s.mu.Lock()
	s.MirrorUsed = url
	s.Tried = append(s.Tried, url)
	s.mu.Unlock()
}

// StepCtx é o que uma ação recebe.
type StepCtx struct {
	Env    *sys.Env
	Run    sys.Runner
	Fetch  sys.Fetcher
	Info   sys.Info
	Out    io.Writer // vai para o log e para a linha "ao vivo"
	Args   []string
	Opts   plan.Options
	Stamp  string // carimbo da execução (nomes de backup)
	Shared *Shared
}

// Exec roda um comando mandando a saída para o log.
func (s *StepCtx) Exec(ctx context.Context, name string, args ...string) error {
	return s.Run.Run(ctx, sys.C(name, args...), s.Out)
}

// ExecCmd roda um Cmd completo mandando a saída para o log.
func (s *StepCtx) ExecCmd(ctx context.Context, c sys.Cmd) error { return s.Run.Run(ctx, c, s.Out) }

// Logf escreve uma linha no log.
func (s *StepCtx) Logf(format string, a ...any) { fmt.Fprintf(s.Out, format+"\n", a...) }

// Warn registra um aviso que aparece no resumo final.
func (s *StepCtx) Warn(msg string) { s.Logf("[aviso] %s", msg); s.Shared.add(&s.Shared.Warnings, msg) }

// Unavailable registra um pacote que não existe no espelho.
func (s *StepCtx) Unavailable(pkg string) { s.Shared.add(&s.Shared.Unavailable, pkg) }

// FailedPkg registra um pacote que falhou ao instalar.
func (s *StepCtx) FailedPkg(pkg string) { s.Shared.add(&s.Shared.FailedPkgs, pkg) }

// Observer acompanha a execução.
type Observer interface {
	Start(p *plan.Plan)
	StepStart(i int)
	Line(i int, line string)
	StepEnd(i int, st Status, d time.Duration)
	Finish(r *Result)
}

// Engine executa planos.
type Engine struct {
	Reg    Registry
	Env    *sys.Env
	Runner sys.Runner
	Fetch  sys.Fetcher
	Info   sys.Info
	Now    func() time.Time // trocável nos testes
}

// Result é o resultado de uma execução.
type Result struct {
	Plan        *plan.Plan
	Status      []Status
	Durations   []time.Duration
	Overall     string
	NOK         int
	NFailed     int
	NPending    int
	Interrupted bool
	Start, End  time.Time
	FreeBefore  int64
	FreeAfter   int64
	LogPath     string
	SummaryPath string
	MirrorUsed  string
	Warnings    []string
	Unavailable []string
	FailedPkgs  []string
	Hints       []string
	ErrLines    []string
}

// Elapsed é a duração total.
func (r *Result) Elapsed() time.Duration { return r.End.Sub(r.Start) }

// UsedMB é o espaço gasto (estimado pela diferença de espaço livre).
func (r *Result) UsedMB() int64 { return max(0, r.FreeBefore-r.FreeAfter) }

// Run executa o plano. Cancelar ctx (Ctrl+C) interrompe o passo atual;
// o resumo é escrito mesmo assim.
func (e *Engine) Run(ctx context.Context, p *plan.Plan, obs ...Observer) *Result {
	now := e.Now
	if now == nil {
		now = time.Now
	}
	start := now()
	stamp := sys.Stamp(start)
	n := len(p.Steps)
	r := &Result{Plan: p, Status: make([]Status, n), Durations: make([]time.Duration, n), Start: start,
		FreeBefore: sys.FreeMB(e.Env.Home)}
	r.LogPath = filepath.Join(e.Env.LogDir, "setup-"+stamp+".log")
	r.SummaryPath = filepath.Join(e.Env.LogDir, "resumo-"+stamp+".txt")
	logf, err := os.Create(r.LogPath)
	if err != nil {
		logf = nil
	}
	shared := &Shared{}
	for _, o := range obs {
		o.Start(p)
	}
	for i, st := range p.Steps {
		if ctx.Err() != nil {
			break
		}
		for _, o := range obs {
			o.StepStart(i)
		}
		r.Status[i] = Running
		lw := newLineWriter(func(line string) {
			for _, o := range obs {
				o.Line(i, line)
			}
		})
		var out io.Writer = lw
		if logf != nil {
			fmt.Fprintf(logf, "\n===== [%d/%d] %s (%s) =====\n", i+1, n, st.Label, now().Format("15:04:05"))
			out = io.MultiWriter(logf, lw)
		}
		sc := &StepCtx{Env: e.Env, Run: e.Runner, Fetch: e.Fetch, Info: e.Info, Out: &lockedWriter{w: out},
			Args: st.Args, Opts: p.Options, Stamp: stamp, Shared: shared}
		t0 := now()
		err := e.call(ctx, st.Action, sc)
		lw.Flush()
		r.Durations[i] = now().Sub(t0)
		switch {
		case ctx.Err() != nil:
			r.Status[i] = Interrupted
		case err != nil:
			r.Status[i] = Failed
			fmt.Fprintf(sc.Out, "[erro] %v\n", err)
		default:
			r.Status[i] = OK
		}
		for _, o := range obs {
			o.StepEnd(i, r.Status[i], r.Durations[i])
		}
	}
	for i := range r.Status {
		if r.Status[i] == Pending {
			r.Status[i] = NotRun
		}
	}
	r.Interrupted = ctx.Err() != nil
	r.End = now()
	r.FreeAfter = sys.FreeMB(e.Env.Home)
	r.MirrorUsed, r.Warnings, r.Unavailable, r.FailedPkgs = shared.MirrorUsed, shared.Warnings, shared.Unavailable, shared.FailedPkgs
	if logf != nil {
		logf.Close()
	}
	r.tally()
	if data, err := os.ReadFile(r.LogPath); err == nil {
		r.Hints = diagnose.Hints(string(data))
		r.ErrLines = diagnose.ErrorLines(string(data), 15)
	}
	e.writeSummary(r)
	for _, o := range obs {
		o.Finish(r)
	}
	return r
}

func (e *Engine) call(ctx context.Context, action string, sc *StepCtx) (err error) {
	fn, ok := e.Reg[action]
	if !ok {
		return fmt.Errorf("ação desconhecida: %s", action)
	}
	defer func() { // uma ação com bug não derruba o app inteiro
		if p := recover(); p != nil {
			err = fmt.Errorf("erro interno na ação %s: %v", action, p)
		}
	}()
	return fn(ctx, sc)
}

func (r *Result) tally() {
	for _, s := range r.Status {
		switch s {
		case OK:
			r.NOK++
		case Failed:
			r.NFailed++
		default:
			r.NPending++
		}
	}
	switch {
	case r.Interrupted || r.NPending > 0:
		r.Overall = Incomplete
	case r.NFailed > 0:
		r.Overall = WithFailures
	default:
		r.Overall = Complete
	}
}

func (e *Engine) writeSummary(r *Result) {
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f+"\n", a...) }
	mirror := r.MirrorUsed
	if mirror == "" {
		mirror = "(não alterado)"
	}
	w("=== RESUMO TERMUX SETUP ===")
	w("Status : %s", r.Overall)
	w("Início : %s", r.Start.Format("2006-01-02 15:04:05"))
	w("Fim    : %s (%ds)", r.End.Format("2006-01-02 15:04:05"), int(r.Elapsed().Seconds()))
	w("Disp.  : %s | Android %s | %s", e.Info.Arch, e.Info.Android, e.Info.Model)
	w("Espelho: %s", mirror)
	w("Espaço : ~%d MB usados | livre: %d MB", r.UsedMB(), r.FreeAfter)
	w("Passos : %d ok, %d falha(s), %d pendente(s) de %d", r.NOK, r.NFailed, r.NPending, len(r.Status))
	w("\n--- Passos ---")
	for i, st := range r.Plan.Steps {
		w("[%s] %s (%ds)", r.Status[i], st.Label, int(r.Durations[i].Seconds()))
	}
	section := func(title string, items []string, sep string) {
		if len(items) > 0 {
			w("\n--- %s ---", title)
			w("%s", strings.Join(items, sep))
		}
	}
	section("Avisos", r.Warnings, "\n")
	section("Pacotes indisponíveis (ignorados)", r.Unavailable, " ")
	section("Pacotes que falharam", r.FailedPkgs, " ")
	section("Diagnóstico automático", prefixAll("- ", r.Hints), "\n")
	section("Últimas linhas de erro do log", r.ErrLines, "\n")
	w("\nLog completo: %s", r.LogPath)
	w("Resumo      : %s", r.SummaryPath)
	if err := os.WriteFile(r.SummaryPath, []byte(b.String()), 0o644); err != nil {
		return
	}
	// cópia em Downloads, se o acesso ao armazenamento estiver liberado
	if dl := filepath.Join(e.Env.Home, "storage/downloads"); isDir(dl) {
		for _, f := range []string{r.SummaryPath, r.LogPath} {
			if data, err := os.ReadFile(f); err == nil {
				_ = os.WriteFile(filepath.Join(dl, filepath.Base(f)), data, 0o644)
			}
		}
	}
}

func prefixAll(p string, items []string) []string {
	out := make([]string, len(items))
	for i, s := range items {
		out[i] = p + s
	}
	return out
}

func isDir(p string) bool { st, err := os.Stat(p); return err == nil && st.IsDir() }

// ---------------------------------------------------------------- escrita

// lineWriter quebra a saída em linhas (o apt usa \r para progresso).
type lineWriter struct {
	buf    []byte
	onLine func(string)
}

func newLineWriter(f func(string)) *lineWriter { return &lineWriter{onLine: f} }

func (l *lineWriter) Write(p []byte) (int, error) {
	for _, c := range p {
		if c == '\n' || c == '\r' {
			l.emit()
			continue
		}
		l.buf = append(l.buf, c)
	}
	return len(p), nil
}

func (l *lineWriter) emit() {
	if s := strings.TrimSpace(string(l.buf)); s != "" {
		l.onLine(s)
	}
	l.buf = l.buf[:0]
}

// Flush manda a última linha sem \n.
func (l *lineWriter) Flush() { l.emit() }

type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

// ErrSkip pode ser devolvido por uma ação que decidiu não fazer nada.
var ErrSkip = errors.New("nada a fazer")
