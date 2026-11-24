package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/Rafael-devR/termux-setup.sh/internal/ui/fx"
	"github.com/charmbracelet/x/ansi"
)

// MenuItem é uma opção do menu.
type MenuItem struct {
	Key   string // atalho ("1", "v"...)
	Group string // título da seção (aparece antes do 1º item do grupo)
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
	lines = append(lines, "")
	if m.Pre != nil {
		for _, l := range m.Pre(c) {
			lines = append(lines, fit("  "+l, c.W))
		}
		lines = append(lines, "")
	}
	// linhas do menu: títulos de seção + itens (idx = índice do item, -1 = título)
	type row struct {
		idx  int
		text string
	}
	var rows []row
	last, curRow := "", 0
	for i, it := range m.Items {
		if it.Group != "" && it.Group != last {
			if len(rows) > 0 {
				rows = append(rows, row{-1, ""})
			}
			rows = append(rows, row{-1, "  " + section(it.Group)})
			last = it.Group
		}
		if i == m.cur {
			curRow = len(rows)
		}
		rows = append(rows, row{i, ""})
	}
	// janela de linhas que cabe na tela (reserva: dica, rodapé, respiro)
	room := max(4, c.H-len(lines)-6)
	first := max(0, min(curRow-room/2, len(rows)-room))
	end := min(len(rows), first+room)
	lw := c.W - 4
	for _, r := range rows[first:end] {
		if r.idx < 0 {
			lines = append(lines, r.text)
			continue
		}
		it := m.Items[r.idx]
		key := it.Key
		if i := r.idx; i == m.cur {
			p := float64(c.Frame-m.changedAt) / decryptFrames
			label := fx.Decrypt(fit(it.Label, lw-4), p, c.RNG)
			lines = append(lines, band(tag(ansi.Strip(label), key+" ", c.W-2), c.W))
		} else {
			lines = append(lines, tag("  "+paint(fit(it.Label, lw-4), Text), paint(key, Faint)+" ", c.W))
		}
	}
	if h := m.Items[m.cur].Hint; h != "" {
		lines = append(lines, "")
		for _, l := range card(c.W-3, Cyan, paint(fit(h, c.W-9), Dim)) {
			lines = append(lines, " "+l)
		}
	}
	lines = append(lines, footer(c.W, "↑↓ navegar", "⏎ abrir", "q voltar"))
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
