package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/Rafael-devR/termux-setup.sh/internal/app"
	"github.com/Rafael-devR/termux-setup.sh/internal/engine"
	"github.com/Rafael-devR/termux-setup.sh/internal/mirror"
	"github.com/Rafael-devR/termux-setup.sh/internal/plan"
	"github.com/Rafael-devR/termux-setup.sh/internal/sys"
	"github.com/Rafael-devR/termux-setup.sh/internal/ui/fx"
)

func ctxBG() context.Context { return context.Background() }

// ---------------------------------------------------------------- Result

// Result é a tela final de uma execução.
type Result struct {
	r     *engine.Result
	start int
}

// NewResult cria a tela de resultado.
func NewResult(r *engine.Result) *Result { return &Result{r: r, start: -1} }

func (v *Result) Init(c *Ctx) tea.Cmd { v.start = c.Frame; return nil }

const resultAnim = FPS * 2

func (v *Result) Animating(c *Ctx) bool { return c.Frame-v.start < resultAnim }

func (v *Result) Update(c *Ctx, msg tea.Msg) (Screen, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "l":
			return v, Push(NewLogView())
		case "ctrl+c":
			return v, tea.Quit
		default:
			if c.Auto {
				return v, tea.Quit
			}
			return v, Root
		}
	}
	return v, nil
}

func (v *Result) View(c *Ctx) string {
	r := v.r
	t := min(1, float64(c.Frame-v.start)/float64(resultAnim))
	col, icon := Green, "◉"
	switch r.Overall {
	case engine.Incomplete:
		col, icon = Red, "■"
	case engine.WithFailures:
		col, icon = Yellow, "▲"
	}
	lines := append(header(c, "Resultado"), "")
	// caixa: a borda tem um degradê que gira enquanto anima
	status := fx.Decrypt(r.Overall, t*2, c.RNG)
	inner := len([]rune(r.Overall)) + 10
	border := func(s string, row int) string {
		var b strings.Builder
		rs := []rune(s)
		for i, ch := range rs {
			k := float64(i)/float64(len(rs)) + float64(c.Frame)/40 + float64(row)*0.1
			k -= float64(int(k))
			cc := fx.Lerp(Grad.At(k), col, t)
			b.WriteString(paint(string(ch), cc))
		}
		return b.String()
	}
	lines = append(lines,
		" "+border("╭"+strings.Repeat("─", inner)+"╮", 0),
		" "+border("│", 1)+"  "+bold(icon, col)+"  "+bold(pad(status, inner-5), col)+border("│", 1),
		" "+border("╰"+strings.Repeat("─", inner)+"╯", 2), "")
	count := func(n int) string { return itoa(int(float64(n) * min(1, t*1.5))) }
	lines = append(lines, " "+bold("✓ "+count(r.NOK)+" ok", Green)+"   "+bold("✗ "+count(r.NFailed)+" falhas", Red)+"   "+bold("· "+count(r.NPending)+" pendentes", Yellow))
	lines = append(lines, " "+paint("tempo ", Dim)+bold(mmss(int(r.Elapsed().Seconds())), White)+paint(fmt.Sprintf(" · +%d MB · livre %s", r.UsedMB(), humanMB(r.FreeAfter)), Dim))
	if r.MirrorUsed != "" {
		lines = append(lines, " "+paint("espelho: "+mirror.Host(r.MirrorUsed), Faint))
	}
	for i, st := range r.Status {
		switch st {
		case engine.Failed:
			lines = append(lines, " "+bold("✗", Red)+" "+paint(fit(r.Plan.Steps[i].Label, c.W-4), Text))
		case engine.Interrupted:
			lines = append(lines, " "+bold("■", Yellow)+" "+paint(fit(r.Plan.Steps[i].Label+" (interrompido)", c.W-4), Text))
		}
	}
	for _, w := range r.Warnings {
		lines = append(lines, " "+paint("! "+fit(w, c.W-4), Yellow))
	}
	for _, h := range r.Hints {
		lines = append(lines, " "+paint("- "+fit(h, c.W-4), Yellow))
	}
	lines = append(lines, "")
	if r.Overall == engine.Incomplete {
		lines = append(lines, " "+bold("O que fazer", White), " "+paint("Rode de novo: o que já está instalado é pulado.", Dim))
	} else if r.Plan.Kind == plan.KindInstall {
		lines = append(lines, " "+bold("Próximos passos", White),
			"  "+bold("source ~/.bashrc", Cyan)+paint(" (ou reabra o Termux)", Dim))
		for _, ia := range []struct{ id, how string }{{"claude", "login pelo navegador"}, {"codex", "Sign in with ChatGPT"}, {"gemini", "Google ou GEMINI_API_KEY"}, {"opencode", "opencode auth login"}} {
			if r.Plan.Has(ia.id) {
				lines = append(lines, fit("  "+bold(pad(ia.id, 9), Cyan)+paint(ia.how, Dim), c.W))
			}
		}
		if r.Plan.Has("ubuntu") {
			lines = append(lines, "  "+bold("ubuntu ", Cyan)+paint("shell do Ubuntu na pasta atual", Dim))
		}
		if slices.ContainsFunc([]string{"theme", "font", "keys"}, r.Plan.Has) {
			lines = append(lines, "  "+paint("tema/fonte não mudou? reabra o Termux", Dim))
		}
	}
	lines = append(lines, "", " "+paint("log: "+fit(c.S.Env.Tilde(r.SummaryPath), c.W-8), Faint))
	return strings.Join(append(lines, footer(c.W, "⏎ menu", "l ver log")), "\n")
}

// ---------------------------------------------------------------- Verify

// Verify mostra as versões instaladas, chegando uma a uma.
type Verify struct {
	rows []app.VerifyRow
	ch   chan app.VerifyRow
	done bool
	at   []int
}

type verifyRowMsg struct {
	row app.VerifyRow
	ok  bool
}

func (v *Verify) Init(c *Ctx) tea.Cmd {
	v.ch = make(chan app.VerifyRow, 64)
	go func() {
		c.S.Verify(ctxBG(), func(r app.VerifyRow) { v.ch <- r })
		close(v.ch)
	}()
	return v.next()
}

func (v *Verify) next() tea.Cmd {
	return func() tea.Msg { r, ok := <-v.ch; return verifyRowMsg{r, ok} }
}

func (v *Verify) Animating(c *Ctx) bool {
	return !v.done || (len(v.at) > 0 && c.Frame-v.at[len(v.at)-1] < decryptFrames)
}

func (v *Verify) Update(c *Ctx, msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case verifyRowMsg:
		if !msg.ok {
			v.done = true
			return v, nil
		}
		v.rows = append(v.rows, msg.row)
		v.at = append(v.at, c.Frame)
		return v, v.next()
	case tea.KeyPressMsg:
		return v, Pop
	}
	return v, nil
}

func (v *Verify) View(c *Ctx) string {
	lines := header(c, "Verificar instalação")
	where := ""
	installed := 0
	for i, r := range v.rows {
		if r.Version != "" {
			installed++
		}
		if r.Where != where {
			where = r.Where
			title := "Termux"
			if where == "ubuntu" {
				title = "Ubuntu (proot)"
			}
			lines = append(lines, "", " "+grad(title, 0, true))
		}
		ver := r.Version
		if ver == "" {
			lines = append(lines, "   "+paint("·", Ghost)+" "+paint(pad(r.Name, 9)+"não instalado", Faint))
			continue
		}
		ver = fx.Decrypt(fit(ver, c.W-16), float64(c.Frame-v.at[i])/decryptFrames, c.RNG)
		lines = append(lines, "   "+bold("✓", Green)+" "+bold(pad(r.Name, 9), White)+paint(ver, Dim))
	}
	if !v.done {
		lines = append(lines, "", " "+paint(spinner[c.Frame%len(spinner)], Cyan)+paint(" verificando…", Dim))
	} else {
		lines = append(lines, "", " "+paint(fmt.Sprintf("%d de %d instalados", installed, len(v.rows)), Dim))
	}
	// mostra o fim da lista se não couber
	if extra := len(lines) - (c.H - 1); extra > 0 {
		lines = append(lines[:2], lines[2+extra:]...)
	}
	return strings.Join(append(lines, footer(c.W, "qualquer tecla volta")), "\n")
}

// ---------------------------------------------------------------- LogView

// LogView mostra o resumo mais recente, com rolagem.
type LogView struct {
	vp    viewport.Model
	path  string
	ready bool
}

// NewLogView cria o visualizador.
func NewLogView() *LogView { return &LogView{} }

func (l *LogView) Init(c *Ctx) tea.Cmd {
	files, _ := filepath.Glob(filepath.Join(c.S.Env.LogDir, "resumo-*.txt"))
	sort.Strings(files)
	l.vp = viewport.New(viewport.WithWidth(c.W), viewport.WithHeight(max(3, c.H-4)))
	if len(files) == 0 {
		l.vp.SetContent(" Nenhum log ainda.")
		return nil
	}
	l.path = files[len(files)-1]
	data, _ := os.ReadFile(l.path)
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		col := Text
		switch {
		case strings.HasPrefix(line, "[OK]"):
			col = Green
		case strings.HasPrefix(line, "[FALHOU]"):
			col = Red
		case strings.HasPrefix(line, "---") || strings.HasPrefix(line, "==="):
			col = Cyan
		case strings.HasPrefix(line, "[") || strings.HasPrefix(line, "- "):
			col = Yellow
		}
		out = append(out, " "+paint(line, col))
	}
	l.vp.SetContent(strings.Join(out, "\n"))
	return nil
}

func (l *LogView) Update(c *Ctx, msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		l.vp.SetWidth(c.W)
		l.vp.SetHeight(max(3, c.H-4))
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "esc", "enter", "backspace", "left":
			return l, Pop
		}
	}
	var cmd tea.Cmd
	l.vp, cmd = l.vp.Update(msg)
	return l, cmd
}

func (l *LogView) View(c *Ctx) string {
	lines := header(c, "Último log")
	return strings.Join(append(lines, l.vp.View(), footer(c.W, "↑↓ rolar", "q voltar")), "\n")
}

// ---------------------------------------------------------------- Mirrors

// Mirrors mede os espelhos ao vivo e deixa escolher um.
type Mirrors struct {
	ch     chan mirror.Probe
	probes []mirror.Probe
	at     map[string]int
	done   bool
	cur    int
	start  time.Time
}

type probeMsg struct {
	p  mirror.Probe
	ok bool
}

func (m *Mirrors) Init(c *Ctx) tea.Cmd {
	m.ch = make(chan mirror.Probe, len(mirror.Defaults))
	m.at = map[string]int{}
	m.start = time.Now()
	go func() {
		mirror.ProbeAll(ctxBG(), c.S.Fetch, mirror.Defaults, c.S.Env.Work, func(p mirror.Probe) { m.ch <- p })
		close(m.ch)
	}()
	return m.next()
}

func (m *Mirrors) next() tea.Cmd {
	return func() tea.Msg { p, ok := <-m.ch; return probeMsg{p, ok} }
}

func (m *Mirrors) Animating(c *Ctx) bool { return !m.done || c.Frame < 10 }

func (m *Mirrors) Update(c *Ctx, msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case probeMsg:
		if !msg.ok {
			m.done = true
			m.probes = mirror.Rank(m.probes)
			return m, nil
		}
		m.probes = append(m.probes, msg.p)
		m.at[msg.p.URL] = c.Frame
		return m, m.next()
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			m.cur = max(0, m.cur-1)
		case "down", "j":
			m.cur = min(len(m.probes)-1, m.cur+1)
		case "enter":
			if m.done && len(m.probes) > 0 {
				return m, Replace(NewInstall(plan.Mirror(m.probes[m.cur].URL)))
			}
		case "esc", "q", "backspace":
			return m, Pop
		}
	}
	return m, nil
}

func (m *Mirrors) View(c *Ctx) string {
	lines := header(c, "Espelhos")
	if !m.done {
		// radar: varredura girando enquanto os espelhos respondem
		lines = append(lines, " "+paint(spinner[c.Frame%len(spinner)], Cyan)+paint(fmt.Sprintf(" medindo %d espelhos… %d responderam", len(mirror.Defaults), len(m.probes)), Dim))
	} else {
		lines = append(lines, " "+paint("o 1º é o melhor · ", Dim)+bold("●", Green)+paint(" < 48h  ", Dim)+bold("●", Red)+paint(" velho", Dim))
	}
	lines = append(lines, "")
	slow := 0.0
	for _, p := range m.probes {
		slow = max(slow, p.Latency.Seconds())
	}
	for i, p := range m.probes {
		bw := 8
		fill := 0
		if slow > 0 {
			fill = int(p.Latency.Seconds() / slow * float64(bw))
		}
		bar := gradRev(strings.Repeat("■", max(1, fill))) + paint(strings.Repeat("·", bw-max(1, fill)), Ghost)
		tag := bold("●", Green)
		if !p.Fresh() {
			tag = bold("●", Red)
		}
		hw := min(22, max(8, c.W-28))
		host := fx.Decrypt(pad(fit(p.Host(), hw), hw), float64(c.Frame-m.at[p.URL])/decryptFrames, c.RNG)
		row := fmt.Sprintf("%s %s %s %s", tag, host, bar, paint(fmt.Sprintf("%.2fs %3dh", p.Latency.Seconds(), min(999, p.AgeHours)), Dim))
		if m.done && i == m.cur {
			lines = append(lines, band(bold("▌", Accent)+" "+bold(row, White), c.W))
		} else {
			lines = append(lines, "  "+paint(row, Text))
		}
	}
	if m.done && len(m.probes) == 0 {
		lines = append(lines, " "+bold("Nenhum espelho respondeu.", Red)+paint(" Verifique a internet.", Dim))
	}
	lines = append(lines, "", " "+paint("atual: "+c.S.Mirror(), Faint))
	return strings.Join(append(lines, footer(c.W, "↑↓ escolher", "⏎ aplicar", "q voltar")), "\n")
}

// ---------------------------------------------------------------- GitHub

// GitHub faz login no gh com token (o login pelo navegador trava no Termux).
type GitHub struct {
	state  string // "checking" "ask" "running" "done"
	status string
	ok     bool
	input  textinput.Model
}

type ghStatusMsg struct {
	text   string
	authed bool
	hasGh  bool
}

type ghLoginMsg struct {
	ok   bool
	text string
}

func (g *GitHub) Init(c *Ctx) tea.Cmd {
	g.state = "checking"
	g.input = textinput.New()
	g.input.Prompt = "› "
	g.input.Placeholder = "cole o token aqui"
	g.input.EchoMode = textinput.EchoPassword
	g.input.SetWidth(max(10, c.W-6))
	return func() tea.Msg {
		if !c.S.Runner.Has("gh") {
			return ghStatusMsg{hasGh: false}
		}
		out, err := sys.Output(ctxBG(), c.S.Runner, sys.C("gh", "auth", "status"))
		return ghStatusMsg{text: out, authed: err == nil, hasGh: true}
	}
}

func (g *GitHub) Animating(*Ctx) bool { return g.state == "checking" || g.state == "running" }

func (g *GitHub) Update(c *Ctx, msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case ghStatusMsg:
		if !msg.hasGh {
			g.state, g.status = "nogh", "O gh não está instalado. Enter instala."
			return g, nil
		}
		g.state, g.ok, g.status = "ask", msg.authed, strings.TrimSpace(msg.text)
		return g, g.input.Focus()
	case ghLoginMsg:
		g.state, g.ok, g.status = "done", msg.ok, msg.text
		return g, nil
	case tea.KeyPressMsg:
		switch {
		case msg.String() == "esc" || (g.state == "done" && msg.String() != ""):
			return g, Pop
		case g.state == "nogh" && msg.String() == "enter":
			return g, Replace(NewInstall(&plan.Plan{Title: "GitHub CLI", Steps: []plan.Step{
				{Label: "Atualizar listas (apt)", Action: "apt-update", Seconds: 5},
				{Label: "Instalar gh", Action: "termux-install", Args: []string{"gh"}, Seconds: 15}}}))
		case g.state == "ask" && msg.String() == "enter":
			tok := strings.TrimSpace(g.input.Value())
			g.input.SetValue("")
			if tok == "" {
				return g, Pop
			}
			g.state = "running"
			return g, func() tea.Msg {
				cmd := sys.C("gh", "auth", "login", "--hostname", "github.com", "--git-protocol", "https", "--with-token")
				cmd.Stdin = tok + "\n"
				out, err := sys.Output(ctxBG(), c.S.Runner, cmd)
				if err != nil {
					return ghLoginMsg{false, "Token recusado. Confira os escopos (repo, read:org) e a validade.\n" + out}
				}
				_ = c.S.Runner.Run(ctxBG(), sys.C("gh", "auth", "setup-git"), nil)
				st, _ := sys.Output(ctxBG(), c.S.Runner, sys.C("gh", "auth", "status"))
				return ghLoginMsg{true, "Git configurado para usar o gh.\n" + st}
			}
		}
		if g.state == "ask" {
			var cmd tea.Cmd
			g.input, cmd = g.input.Update(msg)
			return g, cmd
		}
	}
	return g, nil
}

func (g *GitHub) View(c *Ctx) string {
	lines := append(header(c, "GitHub"), "")
	switch g.state {
	case "checking", "running":
		lines = append(lines, " "+paint(spinner[c.Frame%len(spinner)], Cyan)+paint(" conversando com o GitHub…", Dim))
	case "nogh":
		lines = append(lines, " "+paint(g.status, Yellow))
	case "ask":
		if g.ok {
			lines = append(lines, " "+bold("✓ Já autenticado", Green))
			for _, l := range strings.Split(g.status, "\n") {
				lines = append(lines, "   "+paint(fit(strings.TrimSpace(l), c.W-4), Dim))
			}
			lines = append(lines, "", " "+paint("Para trocar, cole outro token (Enter vazio volta).", Text))
		}
		lines = append(lines, "",
			" "+paint("O login pelo navegador trava no Termux. Use um token:", Text),
			" "+paint("1. abra ", Dim)+bold("github.com/settings/tokens", Cyan),
			" "+paint("2. crie um token clássico: ", Dim)+bold("repo", White)+paint(" + ", Dim)+bold("read:org", White),
			" "+paint("3. cole abaixo (não aparece na tela)", Dim), "", " "+g.input.View())
	case "done":
		col := Green
		if !g.ok {
			col = Red
		}
		for i, l := range strings.Split(g.status, "\n") {
			c2 := Dim
			if i == 0 {
				c2 = col
			}
			lines = append(lines, " "+paint(fit(strings.TrimSpace(l), c.W-2), c2))
		}
	}
	return strings.Join(append(lines, "", footer(c.W, "esc voltar")), "\n")
}
