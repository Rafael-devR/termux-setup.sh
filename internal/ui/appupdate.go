package ui

import (
	"context"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/Rafael-devR/termux-setup.sh/internal/app"
)

type (
	checkedMsg struct {
		version string
		err     error
	}
	appliedMsg struct{ err error }
	sayMsg     string
)

// AppUpdate é a tela "Atualizar o termux-setup": procura, mostra a versão
// nova e instala por cima do próprio programa.
type AppUpdate struct {
	phase   string // "check" | "ask" | "apply" | "done" | "none" | "error"
	version string
	say     string
	err     error
	ch      chan tea.Msg
}

func NewAppUpdate() *AppUpdate { return &AppUpdate{phase: "check"} }

func (a *AppUpdate) NoReveal() {}

func (a *AppUpdate) Animating(*Ctx) bool { return a.phase == "check" || a.phase == "apply" }

func (a *AppUpdate) Init(c *Ctx) tea.Cmd {
	u := c.S.Updater
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		v, err := u.Check(ctx, true)
		return checkedMsg{v, err}
	}
}

func (a *AppUpdate) Update(c *Ctx, msg tea.Msg) (Screen, tea.Cmd) {
	switch m := msg.(type) {
	case checkedMsg:
		switch {
		case m.err != nil:
			a.phase, a.err = "error", m.err
		case m.version == "":
			a.phase = "none"
		default:
			a.phase, a.version = "ask", m.version
			c.Update = m.version
		}
	case sayMsg:
		a.say = string(m)
		return a, a.wait()
	case appliedMsg:
		if m.err != nil {
			a.phase, a.err = "error", m.err
		} else {
			a.phase, c.Update = "done", ""
		}
	case tea.KeyPressMsg:
		k := m.String()
		switch {
		case a.phase == "ask" && (k == "enter" || k == "y" || k == "s"):
			return a, a.apply(c)
		case a.phase == "apply":
			// não interrompe no meio da troca do programa
		case k == "ctrl+c":
			return a, tea.Quit
		case a.phase == "done" && (k == "enter" || k == "q"):
			return a, tea.Quit
		case k == "esc" || k == "q" || k == "enter" || k == "n":
			return a, Pop
		}
	}
	return a, nil
}

// apply instala em segundo plano; os avisos de progresso chegam por canal.
func (a *AppUpdate) apply(c *Ctx) tea.Cmd {
	a.phase = "apply"
	a.ch = make(chan tea.Msg, 8)
	u, v := c.S.Updater, a.version
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		err := u.Apply(ctx, v, func(s string) { a.ch <- sayMsg(s) })
		a.ch <- appliedMsg{err}
	}()
	return a.wait()
}

func (a *AppUpdate) wait() tea.Cmd {
	ch := a.ch
	if ch == nil {
		return nil
	}
	return func() tea.Msg { return <-ch }
}

func (a *AppUpdate) View(c *Ctx) string {
	lines := header(c, "Atualizar o termux-setup")
	lines = append(lines, "")
	cur := app.Version
	if c.S.Env.DryRun {
		cur = c.S.Updater.Current
	}
	sp := spinner[c.Frame%len(spinner)]
	card1 := func(text string) {
		for _, l := range card(c.W-3, Cyan, text) {
			lines = append(lines, " "+l)
		}
	}
	switch a.phase {
	case "check":
		lines = append(lines, " "+paint(sp, Cyan)+" "+paint("procurando versão nova…", Dim))
	case "none":
		card1(bold("✓ ", Green) + bold("Você já está na versão mais nova ", White) + paint(cur, Dim))
		lines = append(lines, "", footer(c.W, "⏎ voltar"))
	case "ask":
		card1(bold("↑ ", Yellow) + bold(a.version, White) + paint(" disponível · você usa "+cur, Dim))
		lines = append(lines, "", " "+paint("Baixa do GitHub, confere o SHA256 e troca o programa.", Dim),
			" "+paint("O que você já instalou no celular não muda.", Faint))
		lines = append(lines, "", footer(c.W, "⏎ atualizar", "esc agora não"))
	case "apply":
		lines = append(lines, " "+paint(sp, Cyan)+" "+paint(strings.TrimSpace(a.say), Text))
	case "done":
		card1(bold("✓ ", Green) + bold("Atualizado para "+a.version, White))
		lines = append(lines, "", " "+paint("Abra o termux-setup de novo para usar a versão nova.", Dim))
		lines = append(lines, "", footer(c.W, "⏎ sair"))
	case "error":
		msg := "erro"
		if a.err != nil {
			msg = a.err.Error()
		}
		card1(bold("✗ ", Red) + paint(fit(msg, c.W-12), Text))
		lines = append(lines, "", " "+paint("Sem internet? Tente de novo mais tarde, ou rode:", Dim),
			" "+bold("curl -fsSL https://raw.githubusercontent.com/Rafael-devR/termux-setup.sh/main/install.sh | bash", Cyan))
		lines = append(lines, "", footer(c.W, "⏎ voltar"))
	}
	return strings.Join(lines, "\n")
}
