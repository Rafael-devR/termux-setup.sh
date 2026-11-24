package ui

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/Rafael-devR/termux-setup.sh/internal/plan"
	"github.com/Rafael-devR/termux-setup.sh/internal/ui/fx"
)

// ---------------------------------------------------------------- Loading

// Loading mostra um spinner enquanto work roda em segundo plano; work
// devolve o próximo comando (normalmente Replace(outra tela)).
type Loading struct {
	title, text string
	work        func() tea.Cmd
	done        bool
}

type loadDoneMsg struct{ next tea.Cmd }

// NewLoading cria a tela de espera.
func NewLoading(title, text string, work func() tea.Cmd) *Loading {
	return &Loading{title: title, text: text, work: work}
}

func (l *Loading) NoReveal() {}
func (l *Loading) Init(*Ctx) tea.Cmd {
	return func() tea.Msg { return loadDoneMsg{l.work()} }
}
func (l *Loading) Animating(*Ctx) bool { return !l.done }
func (l *Loading) Update(_ *Ctx, msg tea.Msg) (Screen, tea.Cmd) {
	if d, ok := msg.(loadDoneMsg); ok {
		l.done = true
		return l, d.next
	}
	return l, nil
}
func (l *Loading) View(c *Ctx) string {
	lines := header(c, l.title)
	sp := spinner[c.Frame%len(spinner)]
	return strings.Join(append(lines, "", " "+paint(sp, Grad.At(float64(c.Frame%30)/30))+" "+paint(l.text+"…", Dim)), "\n")
}

// spinner gira um ponto em volta de um quadrado (quadrantes existem na
// JetBrains Mono; os braille ⠋⠙ não existem).
var spinner = []string{"▘", "▘", "▝", "▝", "▗", "▗", "▖", "▖"}

// ---------------------------------------------------------------- Message

// Message é uma tela simples de aviso (qualquer tecla volta).
type Message struct {
	title string
	color fx.Color
	lines []string
}

// NewMessage cria um aviso.
func NewMessage(title string, color fx.Color, lines ...string) *Message {
	return &Message{title: title, color: color, lines: lines}
}

func (m *Message) Init(*Ctx) tea.Cmd { return nil }
func (m *Message) Update(_ *Ctx, msg tea.Msg) (Screen, tea.Cmd) {
	if _, ok := msg.(tea.KeyPressMsg); ok {
		return m, Pop
	}
	return m, nil
}
func (m *Message) View(c *Ctx) string {
	lines := append(header(c, m.title), "")
	for i, l := range m.lines {
		col := Text
		if i == 0 {
			col = m.color
		}
		lines = append(lines, " "+paint(fit(l, c.W-2), col))
	}
	return strings.Join(append(lines, "", footer(c.W, "qualquer tecla volta")), "\n")
}

// ---------------------------------------------------------------- Checklist

// Checklist marca módulos (agrupados por seção) e segue para a instalação.
type Checklist struct {
	title  string
	groups []string
	sel    map[string]bool
	rows   []string // ids, com "#grupo" como cabeçalho
	cur    int
}

// NewChecklist cria a lista com as seções e a pré-seleção.
func NewChecklist(title string, groups, preselect []string) *Checklist {
	k := &Checklist{title: title, groups: groups, sel: map[string]bool{}}
	for _, id := range preselect {
		k.sel[id] = true
	}
	return k
}

func (k *Checklist) Init(c *Ctx) tea.Cmd {
	k.rows = nil
	for _, g := range k.groups {
		ids := c.S.Cat.InGroup(g)
		if len(ids) == 0 {
			continue
		}
		if len(k.groups) > 1 {
			k.rows = append(k.rows, "#"+g)
		}
		k.rows = append(k.rows, ids...)
	}
	k.skip(c, 1)
	return nil
}

func (k *Checklist) skip(c *Ctx, d int) {
	for i := 0; i < len(k.rows) && strings.HasPrefix(k.rows[k.cur], "#"); i++ {
		k.cur = (k.cur + d + len(k.rows)) % len(k.rows)
	}
}

func (k *Checklist) selected(c *Ctx) []string {
	var ids []string
	for _, r := range k.rows {
		if k.sel[r] && k.ok(c, r) {
			ids = append(ids, r)
		}
	}
	return ids
}

func (k *Checklist) ok(c *Ctx, id string) bool {
	m, found := c.S.Cat.Get(id)
	return found && m.SupportsArch(c.S.Info.Arch)
}

func (k *Checklist) Update(c *Ctx, msg tea.Msg) (Screen, tea.Cmd) {
	key, isKey := msg.(tea.KeyPressMsg)
	if !isKey {
		return k, nil
	}
	switch key.String() {
	case "up", "k":
		k.cur = (k.cur - 1 + len(k.rows)) % len(k.rows)
		k.skip(c, -1)
	case "down", "j", "tab":
		k.cur = (k.cur + 1) % len(k.rows)
		k.skip(c, 1)
	case "space", "x":
		if id := k.rows[k.cur]; k.ok(c, id) {
			k.sel[id] = !k.sel[id]
		}
	case "a":
		for _, r := range k.rows {
			k.sel[r] = !strings.HasPrefix(r, "#")
		}
	case "n":
		k.sel = map[string]bool{}
	case "enter":
		if ids := k.selected(c); len(ids) > 0 {
			return k, StartInstall(c, ids)
		}
	case "esc", "q", "backspace":
		return k, Pop
	}
	return k, nil
}

func (k *Checklist) View(c *Ctx) string {
	ids := k.selected(c)
	res, _ := c.S.Cat.Resolve(ids, c.S.Info.Arch)
	dl := 0
	for _, m := range res.Modules {
		dl += m.Estimate.DownloadMB * c.S.Info.SizeFactor() / 100
	}
	lines := header(c, k.title)
	lines = append(lines, " "+paint("◉ ", Cyan)+bold(itoa(len(ids)), White)+paint(" marcados · ~"+itoa(dl)+" MB", Dim))
	room := max(4, c.H-len(lines)-4)
	first := max(0, min(k.cur-room/2, len(k.rows)-room))
	for i := first; i < min(len(k.rows), first+room); i++ {
		r := k.rows[i]
		if strings.HasPrefix(r, "#") {
			lines = append(lines, " "+section(c.S.Cat.GroupTitle(r[1:])))
			continue
		}
		m, _ := c.S.Cat.Get(r)
		mark, col := paint("○", Faint), Text
		switch {
		case !k.ok(c, r):
			mark, col = paint("⊘", Faint), Faint
		case k.sel[r]:
			mark = bold("◉", Green)
		case slices.Contains(res.Added, r):
			mark = paint("◈", Cyan) // entra por dependência
		}
		size := paint(fmt.Sprintf(" ~%dMB", m.Estimate.DownloadMB*c.S.Info.SizeFactor()/100), Faint)
		label := fit(m.Name, c.W-14)
		if !k.ok(c, r) {
			size = paint(" (sem suporte a "+c.S.Info.Arch+")", Faint)
		}
		if i == k.cur {
			lines = append(lines, band(bold("▌", Accent)+" "+mark+" "+bold(label, White)+size, c.W))
		} else {
			lines = append(lines, "   "+mark+" "+paint(label, col)+size)
		}
	}
	if m, ok := c.S.Cat.Get(k.rows[k.cur]); ok {
		lines = append(lines, "")
		for _, l := range card(c.W-3, Cyan, paint(fit(m.Description, c.W-9), Dim)) {
			lines = append(lines, " "+l)
		}
	}
	if len(res.Added) > 0 {
		lines = append(lines, " "+paint("◈ incluído por dependência", Cyan))
	}
	return strings.Join(append(lines, footer(c.W, "espaço marca", "a todos", "n nenhum", "⏎ seguir")), "\n")
}

// ---------------------------------------------------------------- Options

// Options faz as perguntas antes de instalar (só as necessárias).
type Options struct {
	ids   []string
	opt   plan.Options
	steps []string // "name" "email" "storage" "keys"
	step  int
	input textinput.Model
	pick  int
}

// NewOptions cria as perguntas para os módulos escolhidos.
func NewOptions(ids []string) *Options { return &Options{ids: ids, opt: plan.Options{KeysPreset: 1}} }

func (o *Options) Init(c *Ctx) tea.Cmd {
	o.steps = nil
	if !c.S.GitConfigured(ctxBG()) {
		o.steps = append(o.steps, "name", "email")
	}
	if !c.S.StorageReady() {
		o.steps = append(o.steps, "storage")
	}
	if res, err := c.S.Cat.Resolve(o.ids, c.S.Info.Arch); err == nil {
		for _, m := range res.Modules {
			if m.ID == "keys" {
				o.steps = append(o.steps, "keys")
			}
		}
	}
	o.input = textinput.New()
	o.input.Prompt = "› "
	o.input.SetWidth(max(10, c.W-6))
	if len(o.steps) == 0 {
		return o.finish(c)
	}
	return o.enter()
}

func (o *Options) enter() tea.Cmd {
	o.input.SetValue("")
	o.pick = 0
	switch o.cur() {
	case "name":
		o.input.Placeholder = "seu nome (Enter pula)"
		return o.input.Focus()
	case "email":
		o.input.Placeholder = "seu e-mail"
		return o.input.Focus()
	}
	o.input.Blur()
	return nil
}

func (o *Options) cur() string {
	if o.step < len(o.steps) {
		return o.steps[o.step]
	}
	return ""
}

func (o *Options) next(c *Ctx) tea.Cmd {
	o.step++
	if o.cur() == "email" && o.opt.GitName == "" {
		o.step++
	}
	if o.step >= len(o.steps) {
		return o.finish(c)
	}
	return o.enter()
}

func (o *Options) finish(c *Ctx) tea.Cmd {
	p, err := plan.Install(c.S.Cat, o.ids, o.opt, c.S.Info.Arch, c.S.Info.SizeFactor())
	if err != nil {
		return Replace(NewMessage("Erro", Red, err.Error()))
	}
	return Replace(NewConfirm(p, nil))
}

func (o *Options) Update(c *Ctx, msg tea.Msg) (Screen, tea.Cmd) {
	if _, ok := msg.(tea.WindowSizeMsg); ok {
		o.input.SetWidth(max(10, c.W-6))
		return o, nil
	}
	key, isKey := msg.(tea.KeyPressMsg)
	switch o.cur() {
	case "name", "email":
		if isKey && key.String() == "enter" {
			if o.cur() == "name" {
				o.opt.GitName = strings.TrimSpace(o.input.Value())
			} else {
				o.opt.GitEmail = strings.TrimSpace(o.input.Value())
			}
			return o, o.next(c)
		}
		if isKey && key.String() == "esc" {
			return o, Pop
		}
		var cmd tea.Cmd
		o.input, cmd = o.input.Update(msg)
		return o, cmd
	}
	if !isKey {
		return o, nil
	}
	n := 2
	if o.cur() == "keys" {
		n = 3
	}
	switch key.String() {
	case "up", "k", "left":
		o.pick = (o.pick - 1 + n) % n
	case "down", "j", "right", "tab":
		o.pick = (o.pick + 1) % n
	case "s", "y":
		o.pick = 0
	case "esc", "q":
		return o, Pop
	case "enter":
		if o.cur() == "storage" {
			o.opt.Storage = o.pick == 0
		} else {
			o.opt.KeysPreset = o.pick + 1
		}
		return o, o.next(c)
	}
	return o, nil
}

func (o *Options) View(c *Ctx) string {
	lines := append(header(c, "Opções"), "")
	progress := paint(fmt.Sprintf("pergunta %d de %d", o.step+1, max(1, len(o.steps))), Faint)
	lines = append(lines, " "+progress, "")
	choice := func(opts ...string) {
		for i, s := range opts {
			if i == o.pick {
				lines = append(lines, band(bold("▌", Accent)+" "+bold(s, White), c.W))
			} else {
				lines = append(lines, "   "+paint(s, Text))
			}
		}
	}
	switch o.cur() {
	case "name":
		lines = append(lines, " "+bold("Nome para o Git", White), " "+paint("aparece nos seus commits", Dim), "", " "+o.input.View())
	case "email":
		lines = append(lines, " "+bold("E-mail para o Git", White), "", " "+o.input.View())
	case "storage":
		lines = append(lines, " "+bold("Liberar acesso ao armazenamento?", White),
			" "+paint("cria ~/storage (Downloads, fotos...) e guarda cópia dos logs", Dim), "")
		choice("Sim", "Não")
	case "keys":
		lines = append(lines, " "+bold("Barra de teclas", White), "")
		choice("Compacta (2 linhas, recomendada)", "Grande (4 linhas)", "Padrão do Termux")
		lines = append(lines, "")
		switch o.pick {
		case 0:
			lines = append(lines, " "+paint("ESC TAB / PASTE ENTER ↑ DRAWER", Dim), " "+paint("CTRL ALT ^C ^D ← ↓ →", Dim),
				" "+paint("deslize p/ cima: ESC=claude TAB=codex /=gemini", Faint))
		case 1:
			lines = append(lines, " "+paint("4 linhas: símbolos, ^C ^D ^L ^Z ^R,", Dim), " "+paint("Home/End/PgUp/PgDn e atalhos de comandos", Dim))
		}
	}
	return strings.Join(append(lines, "", footer(c.W, "⏎ ok", "esc voltar")), "\n")
}

// ---------------------------------------------------------------- Confirm

// Confirm mostra o resumo do plano e pede para começar.
type Confirm struct {
	p     *plan.Plan
	notes []string
	menu  *Menu
}

// NewConfirm cria a confirmação (notes = linhas extras de explicação).
func NewConfirm(p *plan.Plan, notes []string) *Confirm { return &Confirm{p: p, notes: notes} }

func (k *Confirm) Init(c *Ctx) tea.Cmd {
	if c.Auto {
		return Replace(NewInstall(k.p))
	}
	k.menu = &Menu{Title: "Confirmar", Items: []MenuItem{
		{Key: "1", Label: "Iniciar", Run: func(*Ctx) tea.Cmd { return Replace(NewInstall(k.p)) }},
		{Key: "0", Label: "Voltar", Run: func(*Ctx) tea.Cmd { return Pop }},
	}}
	k.menu.Pre = func(c *Ctx) []string { return k.summary(c) }
	return k.menu.Init(c)
}

func (k *Confirm) summary(c *Ctx) []string {
	p := k.p
	free := c.S.FreeMB()
	l := []string{
		bold("Passos ", Dim) + bold(itoa(len(p.Steps)), White) + "   " + bold("Tempo ", Dim) + bold("~"+mmss(p.Seconds()), White) + paint(" (estimado)", Faint),
	}
	if p.Kind == plan.KindInstall {
		l = append(l, bold("Download ", Dim)+bold("~"+humanMB(int64(p.Est.DownloadMB)), White)+"   "+bold("Disco ", Dim)+bold("~"+humanMB(int64(p.Est.DiskMB)), White))
	}
	l = append(l, bold("Livre ", Dim)+bold(humanMB(free), White)+paint(" (valores aproximados)", Faint))
	if p.Kind == plan.KindInstall && free < int64(p.Est.DiskMB)*12/10+300 {
		l = append(l, bold("Pouco espaço livre para esta seleção!", Red))
	}
	if len(p.Res.Added) > 0 {
		l = append(l, paint("◈ dependências: "+strings.Join(p.Res.Added, ", "), Cyan))
	}
	if len(p.Res.Dropped) > 0 {
		l = append(l, paint("⊘ sem suporte a "+c.S.Info.Arch+": "+strings.Join(p.Res.Dropped, ", "), Yellow))
	}
	if p.Has("claude") {
		l = append(l, paint("Claude Code: precisa de 4 GB de RAM.", Faint))
	}
	if p.Kind == plan.KindInstall {
		l = append(l, paint("Recomendado: Wi-Fi (downloads grandes).", Faint))
	}
	for _, n := range k.notes {
		l = append(l, paint(n, Text))
	}
	return l
}

func (k *Confirm) Animating(c *Ctx) bool { return k.menu != nil && k.menu.Animating(c) }
func (k *Confirm) Update(c *Ctx, msg tea.Msg) (Screen, tea.Cmd) {
	if k.menu == nil {
		return k, nil
	}
	_, cmd := k.menu.Update(c, msg)
	return k, cmd
}
func (k *Confirm) View(c *Ctx) string {
	if k.menu == nil {
		return ""
	}
	return k.menu.View(c)
}
