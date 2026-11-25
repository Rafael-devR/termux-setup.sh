// Package ui é a interface (Bubble Tea v2). O modelo raiz guarda uma pilha
// de telas; cada tela é um Screen. A raiz cuida do relógio de animação (só
// liga quando alguma coisa está animando), da entrada linha a linha de
// toda tela nova e das teclas globais.
package ui

import (
	"context"
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
	// Update é a versão nova achada em segundo plano ("" = em dia/ainda não sabe).
	Update string
}

// Screen é uma tela.
type Screen interface {
	Init(c *Ctx) tea.Cmd
	Update(c *Ctx, msg tea.Msg) (Screen, tea.Cmd)
	View(c *Ctx) string
}

// Animator é uma tela que precisa do relógio enquanto Animating for true.
type Animator interface{ Animating(c *Ctx) bool }

// FullScreen é uma tela que usa o terminal inteiro (sem painel centralizado).
type FullScreen interface{ FullScreen() }

// MaxW é a largura máxima do painel; em telas largas ele fica centralizado.
const MaxW = 78

// NoReveal desliga a entrada linha a linha (telas que já animam sozinhas).
type NoReveal interface{ NoReveal() }

type (
	tickMsg    time.Time
	pushMsg    struct{ s Screen }
	popMsg     struct{}
	replaceMsg struct{ s Screen }
	rootMsg    struct{}
	updateMsg  struct{ version string }
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
	termW   int // tamanho real do terminal
	termH   int
}

// New cria a interface começando em first.
func New(s *app.Services, first Screen, auto bool) *Model {
	if s != nil && s.Env != nil {
		LoadTheme(s.Env.Home) // tema salvo (ou TSETUP_TEMA)
	}
	return &Model{ctx: &Ctx{W: 80, H: 24, S: s, RNG: newRNG(), Auto: auto},
		stack: []Screen{first}, reveal: 1 << 20, termW: 80, termH: 24}
}

// layout calcula o tamanho que a tela de cima enxerga: painel de até MaxW
// colunas e uma linha a menos (a barra de status fica embaixo).
func (m *Model) layout() {
	if _, full := m.top().(FullScreen); full {
		m.ctx.W, m.ctx.H = m.termW, m.termH
		return
	}
	m.ctx.W, m.ctx.H = min(m.termW, MaxW), max(1, m.termH-1)
}

func (m *Model) top() Screen { return m.stack[len(m.stack)-1] }

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.top().Init(m.ctx), m.ensureTick(), m.checkUpdate())
}

// checkUpdate procura versão nova em segundo plano (no máx. a cada 12 h);
// qualquer erro (sem internet) é ignorado em silêncio.
func (m *Model) checkUpdate() tea.Cmd {
	if m.ctx.S == nil || m.ctx.S.Updater == nil || m.ctx.Auto {
		return nil
	}
	u := m.ctx.S.Updater
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		v, _ := u.Check(ctx, false)
		return updateMsg{v}
	}
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
	m.layout()
	m.reveal = 0
	if _, ok := s.(NoReveal); ok || !m.ctx.S.Anim {
		m.reveal = 1 << 20
	}
	return s.Init(m.ctx)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m.layout()
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Teclado abrindo, tela girando: o terminal (o Termux, por exemplo)
		// reorganiza o texto que já estava na tela, e o renderizador não
		// sabe disso. Sem limpar, ele desenharia só as diferenças por cima
		// de uma tela que já mudou (é daí que vem a "sujeira"). Então:
		// limpa tudo e redesenha do zero no tamanho novo.
		resized := msg.Width != m.termW || msg.Height != m.termH
		m.termW, m.termH = msg.Width, msg.Height
		m.layout()
		m.reveal = 1 << 20
		next, cmd := m.top().Update(m.ctx, msg)
		m.stack[len(m.stack)-1] = next
		if resized {
			return m, tea.Batch(tea.ClearScreen, cmd, m.ensureTick())
		}
		return m, tea.Batch(cmd, m.ensureTick())
	case updateMsg:
		m.ctx.Update = msg.version
		return m, nil
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
		// Ctrl+C sai de qualquer tela (na instalação, interrompe primeiro)
		if _, installing := m.top().(*Install); msg.String() == "ctrl+c" && !installing {
			return m, tea.Quit
		}
	}
	next, cmd := m.top().Update(m.ctx, msg)
	m.stack[len(m.stack)-1] = next
	cmds = append(cmds, cmd, m.ensureTick())
	return m, tea.Batch(cmds...)
}

func (m *Model) View() tea.View {
	m.layout()
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
	_, full := m.top().(FullScreen)
	out := compose(lines, m.ctx.W, m.termW, m.termH, full, m.statusBar())
	v := tea.NewView(out)
	v.AltScreen = true
	v.WindowTitle = "termux-setup"
	return v
}

// statusBar é a linha de baixo: onde estamos (esq.) e a versão (dir.).
func (m *Model) statusBar() string {
	i := m.ctx.S.Info
	left := paint("Android "+i.Android+" · "+i.Arch, Faint)
	return tag(" "+left, paint(app.Version, Faint), m.ctx.W)
}

// compose monta a tela final: pinta o fundo, o painel (do marcador de
// cabeçalho ao de rodapé), centraliza o bloco e coloca a barra de status.
func compose(lines []string, w, termW, termH int, full bool, status string) string {
	if full {
		out := make([]string, 0, termH)
		for _, l := range lines {
			out = append(out, fill(l, termW, BG))
		}
		for len(out) < termH {
			out = append(out, fill("", termW, BG))
		}
		return strings.Join(out[:termH], "\n")
	}
	margin := max(0, (termW-w)/2)
	inPanel := false
	body := make([]string, 0, termH)
	for _, l := range lines {
		bg := BG
		switch {
		case strings.HasPrefix(l, markHead):
			inPanel, bg = true, Panel
			l = " " + strings.TrimPrefix(l, markHead)
		case strings.HasPrefix(l, markFoot):
			inPanel = false
			l = strings.TrimPrefix(l, markFoot)
			body = append(body, fill("", w, Panel)) // respiro final do painel
		case inPanel:
			bg = Panel
		}
		body = append(body, fill(l, w, bg))
	}
	// preenche até a barra de status
	for len(body) < termH-1 {
		body = append(body, fill("", w, BG))
	}
	body = body[:max(0, termH-1)]
	body = append(body, fill(status, w, BG))
	pre := fill("", margin, BG)
	post := ""
	out := make([]string, len(body))
	for i, l := range body {
		if rest := termW - margin - w; rest > 0 {
			post = fill("", rest, BG)
		}
		out[i] = pre + l + post
		if margin == 0 {
			out[i] = l + post
		}
	}
	return strings.Join(out, "\n")
}

func newRNG() *rand.Rand { return rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 7)) }
