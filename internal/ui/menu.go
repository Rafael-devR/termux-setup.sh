package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/Rafael-devR/termux-setup.sh/internal/ui/fx"
)

// MenuItem é uma opção do menu.
type MenuItem struct {
	Key   string // atalho ("1", "v"...)
	Label string
	Hint  string // descrição mostrada embaixo quando selecionado
	Run   func(c *Ctx) tea.Cmd
}

// Menu é uma lista navegável com setas. O item selecionado "se decifra"
// quando o cursor chega nele.
type Menu struct {
	Title     string
	Pre       func(c *Ctx) []string // linhas acima da lista
	Items     []MenuItem
	OnBack    func(c *Ctx) tea.Cmd // Esc/q (nil = voltar)
	cur       int
	changedAt int
}

const decryptFrames = 7

func (m *Menu) Init(c *Ctx) tea.Cmd { m.changedAt = c.Frame; return nil }

func (m *Menu) Animating(c *Ctx) bool { return c.Frame-m.changedAt < decryptFrames }

func (m *Menu) move(c *Ctx, d int) {
	n := len(m.Items)
	m.cur = (m.cur + d + n) % n
	m.changedAt = c.Frame
}

func (m *Menu) Update(c *Ctx, msg tea.Msg) (Screen, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch k.String() {
	case "up", "k":
		m.move(c, -1)
	case "down", "j", "tab":
		m.move(c, 1)
	case "enter", "right":
		return m, m.Items[m.cur].Run(c)
	case "esc", "q", "left", "backspace":
		if m.OnBack != nil {
			return m, m.OnBack(c)
		}
		return m, Pop
	case "ctrl+c":
		return m, tea.Quit
	default:
		for i, it := range m.Items {
			if it.Key != "" && it.Key == k.String() {
				m.cur, m.changedAt = i, c.Frame
				return m, it.Run(c)
			}
		}
	}
	return m, nil
}

func (m *Menu) View(c *Ctx) string {
	lines := header(c, m.Title)
	if m.Pre != nil {
		for _, l := range m.Pre(c) {
			lines = append(lines, fit(" "+l, c.W))
		}
	}
	lines = append(lines, "")
	// janela de itens que cabe na tela
	room := max(3, c.H-len(lines)-4)
	first := max(0, min(m.cur-room/2, len(m.Items)-room))
	last := min(len(m.Items), first+room)
	lw := c.W - 7
	for i := first; i < last; i++ {
		it := m.Items[i]
		key := it.Key
		if key == "" {
			key = "·"
		}
		if i == m.cur {
			p := float64(c.Frame-m.changedAt) / decryptFrames
			label := fx.Decrypt(fit(it.Label, lw), p, c.RNG)
			lines = append(lines, band(bold("▌", Accent)+" "+bold(key, Accent)+"  "+bold(label, White), c.W))
		} else {
			lines = append(lines, "  "+paint(key, Faint)+"  "+paint(fit(it.Label, lw), Text))
		}
	}
	lines = append(lines, "")
	if h := m.Items[m.cur].Hint; h != "" {
		lines = append(lines, " "+paint("› "+fit(h, c.W-4), Dim))
	}
	pos := paint(strings.Repeat("0", 2-len(itoa(m.cur+1)))+itoa(m.cur+1), Cyan) + paint("/"+itoa(len(m.Items)), Faint)
	lines = append(lines, footer(c.W-8, "↑↓ navegar", "⏎ ok", "q voltar")+"   "+pos)
	return strings.Join(lines, "\n")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}
