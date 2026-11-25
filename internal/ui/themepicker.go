package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/Rafael-devR/termux-setup.sh/internal/ui/fx"
)

// ThemePicker é a tela "Tema da interface": enquanto você move o cursor, a
// tela inteira já aparece no tema escolhido; Enter guarda, Esc desfaz.
type ThemePicker struct {
	cur   int
	prev  string
	saved bool
	err   error
}

// NewThemePicker abre a escolha de tema.
func NewThemePicker() *ThemePicker {
	p := &ThemePicker{prev: CurrentTheme()}
	for i, t := range Themes {
		if t.ID == p.prev {
			p.cur = i
		}
	}
	return p
}

func (p *ThemePicker) Init(*Ctx) tea.Cmd { return nil }

func (p *ThemePicker) NoReveal() {}

func (p *ThemePicker) Update(c *Ctx, msg tea.Msg) (Screen, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return p, nil
	}
	n := len(Themes)
	switch k.String() {
	case "up", "k":
		p.cur = (p.cur + n - 1) % n
		SetTheme(Themes[p.cur].ID)
	case "down", "j", "tab":
		p.cur = (p.cur + 1) % n
		SetTheme(Themes[p.cur].ID)
	case "enter":
		p.err = SaveTheme(c.S.Env.Home)
		p.saved = p.err == nil
		if p.saved {
			return p, Pop
		}
	case "esc", "q", "left":
		SetTheme(p.prev) // desfaz a pré-visualização
		return p, Pop
	case "ctrl+c":
		SetTheme(p.prev)
		return p, tea.Quit
	}
	return p, nil
}

// swatch é uma fileira de amostras das cores principais do tema.
func swatch(t Theme) string {
	var b strings.Builder
	for _, hex := range []string{t.Primary, t.Secondary, t.Section, t.Green, t.Yellow, t.Red} {
		b.WriteString(paint("██", fx.Hex(hex)) + " ")
	}
	return b.String()
}

func (p *ThemePicker) View(c *Ctx) string {
	lines := header(c, "Tema da interface")
	lines = append(lines, "", "  "+paint("A tela muda enquanto você navega.", Dim), "")
	for i, t := range Themes {
		row := t.Name
		if i == p.cur {
			lines = append(lines, band(tag(row, "", c.W-2), c.W))
		} else {
			lines = append(lines, "  "+paint(row, Text))
		}
	}
	t := Themes[p.cur]
	lines = append(lines, "")
	for _, l := range card(c.W-3, Cyan, bold(t.Name, White), paint(t.Desc, Dim), swatch(t)) {
		lines = append(lines, " "+l)
	}
	if p.err != nil {
		lines = append(lines, "", "  "+paint("não consegui salvar: "+p.err.Error(), Red))
	}
	lines = append(lines, "", paint("  texto principal ", White)+paint("secundário ", Text)+paint("discreto ", Dim)+paint("apagado", Faint))
	return strings.Join(append(lines, "", footer(c.W, "↑↓ ver", "⏎ usar este", "esc cancelar")), "\n")
}
