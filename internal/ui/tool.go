package ui

import (
	"context"
	"math"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/Rafael-devR/termux-setup.sh/internal/ui/fx"
)

// Tool é o molde das telas do menu Ferramentas: um trabalho em segundo
// plano (Run) manda dados (send), a tela guarda o que chegou (OnData) e
// desenha (Body). Esc cancela o trabalho e volta.
type Tool struct {
	Title string
	Hints []string
	// Run roda em segundo plano até acabar (ou até ctx ser cancelado).
	Run func(ctx context.Context, send func(any))
	// OnData roda na thread da interface para cada send.
	OnData func(v any)
	// Body desenha o conteúdo (sem cabeçalho nem rodapé).
	Body func(c *Ctx, done bool) []string
	// OnKey trata teclas extras; devolve (comando, tratou?).
	OnKey func(c *Ctx, key string, t *Tool) (tea.Cmd, bool)

	ch     chan any
	cancel context.CancelFunc
	done   bool
}

type (
	toolDataMsg struct {
		t *Tool
		v any
	}
	toolDoneMsg struct{ t *Tool }
)

func (t *Tool) NoReveal() {}

func (t *Tool) Animating(*Ctx) bool { return !t.done }

// Restart roda o trabalho de novo (tecla r nos testes).
func (t *Tool) Restart() tea.Cmd {
	if t.cancel != nil {
		t.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.cancel, t.done = cancel, false
	ch := make(chan any, 32)
	t.ch = ch
	go func() {
		t.Run(ctx, func(v any) {
			select {
			case ch <- v:
			case <-ctx.Done():
			}
		})
		select {
		case ch <- toolDoneMsg{t}:
		case <-ctx.Done():
		}
	}()
	return t.wait()
}

func (t *Tool) wait() tea.Cmd {
	ch := t.ch
	return func() tea.Msg {
		v := <-ch
		if d, ok := v.(toolDoneMsg); ok {
			return d
		}
		return toolDataMsg{t, v}
	}
}

func (t *Tool) Init(*Ctx) tea.Cmd { return t.Restart() }

func (t *Tool) stop() {
	if t.cancel != nil {
		t.cancel()
	}
}

func (t *Tool) Update(c *Ctx, msg tea.Msg) (Screen, tea.Cmd) {
	switch m := msg.(type) {
	case toolDataMsg:
		if m.t != t {
			return t, nil
		}
		if t.OnData != nil {
			t.OnData(m.v)
		}
		return t, t.wait()
	case toolDoneMsg:
		if m.t == t {
			t.done = true
		}
	case tea.KeyPressMsg:
		k := m.String()
		if t.OnKey != nil {
			if cmd, ok := t.OnKey(c, k, t); ok {
				return t, cmd
			}
		}
		switch k {
		case "esc", "q", "left", "backspace":
			t.stop()
			return t, Pop
		case "ctrl+c":
			t.stop()
			return t, tea.Quit
		}
	}
	return t, nil
}

func (t *Tool) View(c *Ctx) string {
	lines := header(c, t.Title)
	lines = append(lines, "")
	for _, l := range t.Body(c, t.done) {
		lines = append(lines, fit(" "+l, c.W))
	}
	hints := append([]string{}, t.Hints...)
	hints = append(hints, "esc voltar")
	return strings.Join(append(lines, "", footer(c.W, hints...)), "\n")
}

// ---------------------------------------------------------------- desenho

// meter é uma barra de porcentagem.
func meter(pct, w int, col fx.Color) string {
	pct = max(0, min(100, pct))
	n := pct * w / 100
	return paint(strings.Repeat("█", n), col) + paint(strings.Repeat("░", w-n), Ghost)
}

// spark é um mini gráfico de linha com os últimos w valores.
func spark(vals []float64, w int, lo, hi float64, col fx.Color) string {
	if len(vals) > w {
		vals = vals[len(vals)-w:]
	}
	const bars = "▁▂▃▄▅▆▇█"
	rs := []rune(bars)
	var b strings.Builder
	for _, v := range vals {
		if v <= 0 {
			b.WriteString(paint("·", Faint))
			continue
		}
		t := 0.0
		if hi > lo {
			t = (v - lo) / (hi - lo)
		}
		i := int(math.Round(math.Max(0, math.Min(1, t)) * float64(len(rs)-1)))
		b.WriteString(paint(string(rs[i]), col))
	}
	return b.String()
}

// push guarda um valor numa série de tamanho máximo n.
func push(s []float64, v float64, n int) []float64 {
	s = append(s, v)
	if len(s) > n {
		s = s[len(s)-n:]
	}
	return s
}

func rng(vals []float64) (lo, hi float64) {
	lo, hi = math.MaxFloat64, 0
	for _, v := range vals {
		if v > 0 {
			lo, hi = math.Min(lo, v), math.Max(hi, v)
		}
	}
	if lo == math.MaxFloat64 {
		return 0, 1
	}
	if hi-lo < 2 {
		hi = lo + 2 // série quase reta não vira escada
	}
	return lo, hi
}

// tempColor: verde até 38 °C, amarelo até 42, vermelho acima.
func tempColor(c float64) fx.Color {
	switch {
	case c >= 42:
		return Red
	case c >= 38:
		return Yellow
	}
	return Green
}

// usageColor: verde, amarelo e vermelho conforme a porcentagem de uso.
func usageColor(p int) fx.Color {
	switch {
	case p >= 90:
		return Red
	case p >= 75:
		return Yellow
	}
	return Cyan
}

// panel desenha um bloco com título e linhas dentro de um cartão.
func panel(c *Ctx, accent fx.Color, title string, rows ...string) []string {
	out := []string{}
	body := append([]string{bold(title, White)}, rows...)
	for _, l := range card(c.W-4, accent, body...) {
		out = append(out, l)
	}
	return out
}
