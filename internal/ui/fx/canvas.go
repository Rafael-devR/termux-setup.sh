// Package fx tem os efeitos visuais: canvas de células coloridas, logo em
// blocos, chuva de caracteres, texto que se decifra, ondas, dígitos
// grandes e espectro. Tudo é função pura do número do quadro (e de um
// gerador aleatório), então dá para testar sem terminal.
package fx

import (
	"fmt"
	"strings"
)

// Color é uma cor RGB; a zero (OK=false) é "cor padrão do terminal".
type Color struct {
	R, G, B uint8
	OK      bool
}

// RGB cria uma cor.
func RGB(r, g, b uint8) Color { return Color{r, g, b, true} }

// Hex lê "#rrggbb".
func Hex(s string) Color {
	var r, g, b uint8
	if _, err := fmt.Sscanf(strings.TrimPrefix(s, "#"), "%02x%02x%02x", &r, &g, &b); err != nil {
		return Color{}
	}
	return RGB(r, g, b)
}

// Lerp mistura a e b (t de 0 a 1).
func Lerp(a, b Color, t float64) Color {
	t = clamp01(t)
	m := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t + 0.5) }
	return RGB(m(a.R, b.R), m(a.G, b.G), m(a.B, b.B))
}

// Dim escurece a cor (f de 0 a 1).
func Dim(c Color, f float64) Color { return Lerp(RGB(0, 0, 0), c, f) }

// Hexs devolve "#rrggbb" (para o lipgloss).
func (c Color) Hexs() string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

func clamp01(t float64) float64 { return min(1, max(0, t)) }

// Gradient é uma sequência de cores com interpolação.
type Gradient []Color

// At devolve a cor na posição t (0 a 1).
func (g Gradient) At(t float64) Color {
	if len(g) == 0 {
		return Color{}
	}
	if len(g) == 1 {
		return g[0]
	}
	t = clamp01(t) * float64(len(g)-1)
	i := int(t)
	if i >= len(g)-1 {
		return g[len(g)-1]
	}
	return Lerp(g[i], g[i+1], t-float64(i))
}

// Cell é uma célula do canvas.
type Cell struct {
	R    rune
	FG   Color
	BG   Color
	Bold bool
}

// Canvas é uma grade de células.
type Canvas struct {
	W, H  int
	cells []Cell
}

// NewCanvas cria um canvas vazio (espaços).
func NewCanvas(w, h int) *Canvas {
	c := &Canvas{W: max(0, w), H: max(0, h), cells: make([]Cell, max(0, w)*max(0, h))}
	for i := range c.cells {
		c.cells[i].R = ' '
	}
	return c
}

func (c *Canvas) in(x, y int) bool { return x >= 0 && y >= 0 && x < c.W && y < c.H }

// At devolve a célula (zero fora do canvas).
func (c *Canvas) At(x, y int) Cell {
	if !c.in(x, y) {
		return Cell{R: ' '}
	}
	return c.cells[y*c.W+x]
}

// Set escreve um caractere.
func (c *Canvas) Set(x, y int, r rune, fg Color) {
	if c.in(x, y) {
		c.cells[y*c.W+x] = Cell{R: r, FG: fg, BG: c.cells[y*c.W+x].BG}
	}
}

// SetCell escreve uma célula completa.
func (c *Canvas) SetCell(x, y int, cell Cell) {
	if c.in(x, y) {
		c.cells[y*c.W+x] = cell
	}
}

// Text escreve um texto a partir de (x, y).
func (c *Canvas) Text(x, y int, s string, fg Color, bold bool) {
	for _, r := range s {
		if c.in(x, y) {
			c.cells[y*c.W+x] = Cell{R: r, FG: fg, Bold: bold, BG: c.cells[y*c.W+x].BG}
		}
		x++
	}
}

// GradText escreve um texto com degradê da esquerda para a direita.
func (c *Canvas) GradText(x, y int, s string, g Gradient, bold bool) {
	rs := []rune(s)
	for i, r := range rs {
		c.SetCell(x+i, y, Cell{R: r, FG: g.At(float64(i) / float64(max(1, len(rs)-1))), Bold: bold})
	}
}

// Clear apaga uma linha inteira.
func (c *Canvas) Clear(y int) {
	for x := 0; x < c.W; x++ {
		c.SetCell(x, y, Cell{R: ' '})
	}
}

// Render gera o texto ANSI (24 bits) trocando de cor só quando precisa.
func (c *Canvas) Render() string {
	var b strings.Builder
	b.Grow(c.W * c.H * 4)
	for y := 0; y < c.H; y++ {
		var cur Cell
		curSet := false
		for x := 0; x < c.W; x++ {
			cell := c.cells[y*c.W+x]
			if !curSet || cell.FG != cur.FG || cell.BG != cur.BG || cell.Bold != cur.Bold {
				b.WriteString(sgr(cell))
				cur, curSet = cell, true
			}
			b.WriteRune(cell.R)
		}
		b.WriteString("\x1b[0m")
		if y < c.H-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func sgr(c Cell) string {
	s := "\x1b[0"
	if c.Bold {
		s += ";1"
	}
	if c.FG.OK {
		s += fmt.Sprintf(";38;2;%d;%d;%d", c.FG.R, c.FG.G, c.FG.B)
	}
	if c.BG.OK {
		s += fmt.Sprintf(";48;2;%d;%d;%d", c.BG.R, c.BG.G, c.BG.B)
	}
	return s + "m"
}

// Plain devolve só os caracteres (para testes).
func (c *Canvas) Plain() string {
	var b strings.Builder
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			b.WriteRune(c.cells[y*c.W+x].R)
		}
		if y < c.H-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
