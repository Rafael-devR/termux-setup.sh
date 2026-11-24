// Package ui é a interface (Bubble Tea v2). O modelo raiz guarda uma pilha
// de telas; cada tela é um Screen. A raiz cuida do relógio de animação (só
// liga quando alguma coisa está animando), da entrada linha a linha de
// toda tela nova e das teclas globais.
package ui

import (
	"math/rand/v2"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/Rafael-devR/termux-setup.sh/internal/app"
)

// FPS das animações.
const FPS = 30

// Ctx é o que toda tela recebe.
type Ctx struct {
	W, H  int
	Frame int // quadros desde o início (avança só enquanto anima)
	S     *app.Services
	RNG   *rand.Rand
	Auto  bool // --sim: sem perguntas
}

// Screen é uma tela.
type Screen interface {
	Init(c *Ctx) tea.Cmd
	Update(c *Ctx, msg tea.Msg) (Screen, tea.Cmd)
	View(c *Ctx) string
}

// Animator é uma tela que precisa do relógio enquanto Animating for true.
type Animator interface{ Animating(c *Ctx) bool }

// NoReveal desliga a entrada linha a linha (telas que já animam sozinhas).
type NoReveal interface{ NoReveal() }

type (
	tickMsg    time.Time
	pushMsg    struct{ s Screen }
	popMsg     struct{}
	replaceMsg struct{ s Screen }
	rootMsg    struct{}
)

// Push abre uma tela por cima.
func Push(s Screen) tea.Cmd { return func() tea.Msg { return pushMsg{s} } }

// Pop volta para a tela anterior.
func Pop() tea.Msg { return popMsg{} }

// Replace troca a tela atual.
func Replace(s Screen) tea.Cmd { return func() tea.Msg { return replaceMsg{s} } }

// Root volta ao menu principal.
func Root() tea.Msg { return rootMsg{} }

// Model é o modelo raiz.
type Model struct {
	ctx     *Ctx
	stack   []Screen
	ticking bool
	reveal  int // linhas já reveladas da tela atual
}

// New cria a interface começando em first.
func New(s *app.Services, first Screen, auto bool) *Model {
	return &Model{ctx: &Ctx{W: 80, H: 24, S: s, RNG: newRNG(), Auto: auto},
		stack: []Screen{first}, reveal: 1 << 20}
}

func (m *Model) top() Screen { return m.stack[len(m.stack)-1] }

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.top().Init(m.ctx), m.ensureTick())
}

func tick() tea.Cmd {
	return tea.Tick(time.Second/FPS, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m *Model) revealing() bool { return m.ctx.S.Anim && m.reveal < m.ctx.H }

func (m *Model) wantsTick() bool {
	if m.revealing() {
		return true
	}
	if a, ok := m.top().(Animator); ok {
		return a.Animating(m.ctx)
	}
	return false
}

func (m *Model) ensureTick() tea.Cmd {
	if m.ticking || !m.wantsTick() {
		return nil
	}
	m.ticking = true
	return tick()
}

func (m *Model) enter(s Screen) tea.Cmd {
	m.reveal = 0
	if _, ok := s.(NoReveal); ok || !m.ctx.S.Anim {
		m.reveal = 1 << 20
	}
	return s.Init(m.ctx)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ctx.W, m.ctx.H = msg.Width, msg.Height
	case tickMsg:
		m.ticking = false
		m.ctx.Frame++
		m.reveal += 3
	case pushMsg:
		m.stack = append(m.stack, msg.s)
		return m, tea.Batch(m.enter(msg.s), m.ensureTick())
	case popMsg:
		if len(m.stack) > 1 {
			m.stack = m.stack[:len(m.stack)-1]
			m.reveal = 1 << 20
		}
		return m, m.ensureTick()
	case replaceMsg:
		m.stack[len(m.stack)-1] = msg.s
		return m, tea.Batch(m.enter(msg.s), m.ensureTick())
	case rootMsg:
		m.stack = m.stack[:1]
		m.reveal = 1 << 20
		return m, tea.Batch(m.top().Init(m.ctx), m.ensureTick())
	case tea.KeyPressMsg:
		if m.revealing() { // tecla durante a entrada: mostra tudo de uma vez
			m.reveal = 1 << 20
		}
	}
	next, cmd := m.top().Update(m.ctx, msg)
	m.stack[len(m.stack)-1] = next
	cmds = append(cmds, cmd, m.ensureTick())
	return m, tea.Batch(cmds...)
}

func (m *Model) View() tea.View {
	content := m.top().View(m.ctx)
	lines := strings.Split(content, "\n")
	if len(lines) > m.ctx.H {
		lines = lines[:m.ctx.H]
	}
	for i := range lines {
		lines[i] = fit(lines[i], m.ctx.W)
	}
	if m.revealing() && m.reveal < len(lines) {
		// entrada: linhas aparecem de cima para baixo com uma linha de luz
		cut := max(0, m.reveal)
		scan := grad(strings.Repeat("─", max(1, m.ctx.W-2)), float64(m.ctx.Frame)/20, false)
		lines = append(lines[:cut], " "+scan)
	}
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	v.WindowTitle = "termux-setup"
	return v
}

func newRNG() *rand.Rand { return rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 7)) }
