package fx

import (
	"math"
	"math/rand/v2"
	"strings"
)

// Glyphs são os caracteres da "chuva" e do texto sendo decifrado. Todos
// existem na JetBrains Mono e ocupam uma célula.
var Glyphs = []rune("0123456789abcdefABCDEF#$%&*+=<>/\\|{}[]?!~^λΣπΔ∇∫√∞≈Ωµφθψ")

// Glyph sorteia um caractere.
func Glyph(rng *rand.Rand) rune { return Glyphs[rng.IntN(len(Glyphs))] }

// Decrypt mostra o texto "se decifrando": até progress (0..1) as letras
// já estão certas, o resto é sorteado. Espaços ficam sempre.
func Decrypt(text string, progress float64, rng *rand.Rand) string {
	if progress >= 1 {
		return text
	}
	rs := []rune(text)
	locked := int(float64(len(rs)) * clamp01(progress))
	for i := locked; i < len(rs); i++ {
		if rs[i] != ' ' {
			rs[i] = Glyph(rng)
		}
	}
	return string(rs)
}

// ---------------------------------------------------------------- logo

// font5 é uma fonte 5x5 ('#' = pixel). Tem as letras do logo, dígitos e %.
var font5 = map[rune][5]string{
	'T': {"#####", "  #  ", "  #  ", "  #  ", "  #  "},
	'E': {"#####", "#    ", "#### ", "#    ", "#####"},
	'R': {"#### ", "#   #", "#### ", "#  # ", "#   #"},
	'M': {"#   #", "## ##", "# # #", "#   #", "#   #"},
	'U': {"#   #", "#   #", "#   #", "#   #", " ### "},
	'X': {"#   #", " # # ", "  #  ", " # # ", "#   #"},
	'0': {"###", "# #", "# #", "# #", "###"},
	'1': {" # ", "## ", " # ", " # ", "###"},
	'2': {"###", "  #", "###", "#  ", "###"},
	'3': {"###", "  #", " ##", "  #", "###"},
	'4': {"# #", "# #", "###", "  #", "  #"},
	'5': {"###", "#  ", "###", "  #", "###"},
	'6': {"###", "#  ", "###", "# #", "###"},
	'7': {"###", "  #", " # ", " # ", " # "},
	'8': {"###", "# #", "###", "# #", "###"},
	'9': {"###", "# #", "###", "  #", "###"},
	'%': {"# #", "  #", " # ", "#  ", "# #"},
	' ': {"  ", "  ", "  ", "  ", "  "},
}

// Bitmap monta as 5 linhas de pixels de um texto (1 coluna entre letras).
func Bitmap(text string) [5]string {
	var rows [5]string
	rs := []rune(text)
	for i, r := range rs {
		g, ok := font5[r]
		if !ok {
			g = font5[' ']
		}
		for y := range 5 {
			rows[y] += g[y]
			if i < len(rs)-1 {
				rows[y] += " "
			}
		}
	}
	return rows
}

// HalfBlocks junta pares de linhas de pixels em meio-blocos (▀ ▄ █): 5
// linhas de pixels viram 3 linhas de texto.
func HalfBlocks(rows [5]string) []string {
	var out []string
	for y := 0; y < 5; y += 2 {
		top := []rune(rows[y])
		var bot []rune
		if y+1 < 5 {
			bot = []rune(rows[y+1])
		}
		var b strings.Builder
		for x := range top {
			t := top[x] == '#'
			d := x < len(bot) && bot[x] == '#'
			switch {
			case t && d:
				b.WriteRune('█')
			case t:
				b.WriteRune('▀')
			case d:
				b.WriteRune('▄')
			default:
				b.WriteRune(' ')
			}
		}
		out = append(out, b.String())
	}
	return out
}

// FullBlocks troca '#' por █ (logo grande, 5 linhas).
func FullBlocks(rows [5]string) []string {
	out := make([]string, 5)
	for i, r := range rows {
		out[i] = strings.ReplaceAll(r, "#", "█")
	}
	return out
}

// DrawBlocks desenha linhas de blocos com degradê horizontal. Se shadow,
// desenha antes uma sombra ░ deslocada 1↘ (fica por baixo do logo).
func DrawBlocks(c *Canvas, x, y int, lines []string, g Gradient, shadow Color) {
	w := 0
	for _, l := range lines {
		w = max(w, len([]rune(l)))
	}
	if shadow.OK {
		for dy, l := range lines {
			for dx, r := range []rune(l) {
				if r != ' ' && c.At(x+dx+1, y+dy+1).R == ' ' {
					c.Set(x+dx+1, y+dy+1, '░', shadow)
				}
			}
		}
	}
	for dy, l := range lines {
		for dx, r := range []rune(l) {
			if r != ' ' {
				c.SetCell(x+dx, y+dy, Cell{R: r, FG: g.At(float64(dx) / float64(max(1, w-1)))})
			}
		}
	}
}

// ---------------------------------------------------------------- chuva

type drop struct {
	x       int
	y       float64
	speed   float64
	length  int
	glyphs  []rune
	flicker int
}

// Rain é a chuva de caracteres (estilo Matrix).
type Rain struct {
	W, H  int
	drops []drop
	rng   *rand.Rand
}

// NewRain cria a chuva (uma coluna sim, outra não).
func NewRain(w, h int, rng *rand.Rand) *Rain {
	r := &Rain{W: w, H: h, rng: rng}
	for x := 0; x < w; x += 2 {
		r.drops = append(r.drops, r.newDrop(x, -float64(rng.IntN(max(1, h)))))
	}
	return r
}

func (r *Rain) newDrop(x int, y float64) drop {
	d := drop{x: x, y: y, speed: 0.5 + r.rng.Float64()*0.9, length: 6 + r.rng.IntN(10)}
	d.glyphs = make([]rune, r.H+d.length+2)
	for i := range d.glyphs {
		d.glyphs[i] = Glyph(r.rng)
	}
	return d
}

// Step avança um quadro.
func (r *Rain) Step() {
	for i := range r.drops {
		d := &r.drops[i]
		d.y += d.speed
		if r.rng.IntN(4) == 0 { // caracteres trocam sozinhos (cintilam)
			d.glyphs[r.rng.IntN(len(d.glyphs))] = Glyph(r.rng)
		}
		if int(d.y)-d.length > r.H {
			r.drops[i] = r.newDrop(d.x, -float64(r.rng.IntN(6)))
		}
	}
}

// Draw desenha a chuva; skip diz onde não desenhar (ex.: área do logo).
func (r *Rain) Draw(c *Canvas, head, body Color, skip func(x, y int) bool) {
	for _, d := range r.drops {
		hy := int(d.y)
		for k := 0; k < d.length; k++ {
			y := hy - k
			if y < 0 || y >= r.H || (skip != nil && skip(d.x, y)) {
				continue
			}
			col := Lerp(body, Dim(body, 0.15), float64(k)/float64(d.length))
			if k == 0 {
				col = head
			}
			gi := (y + len(d.glyphs)) % len(d.glyphs)
			c.Set(d.x, y, d.glyphs[gi], col)
		}
	}
}

// ---------------------------------------------------------------- ondas

// SparkRunes são os 8 níveis de altura.
var SparkRunes = []rune("▁▂▃▄▅▆▇█")

// Wave devolve níveis 0..7 de ½[sin(x+t) + sin(3x+2t)] (a fórmula aparece
// embaixo da onda na intro).
func Wave(width int, t float64) []int {
	out := make([]int, width)
	for i := range out {
		x := float64(i) * 0.35
		v := (math.Sin(x+t) + math.Sin(3*x+2*t)) / 2 // -1..1
		out[i] = int(math.Round((v + 1) / 2 * 7))
	}
	return out
}

// Spark transforma valores em ▁▂▃▅▇ (escala pelo maior valor).
func Spark(vals []float64) string {
	top := 0.0
	for _, v := range vals {
		top = max(top, v)
	}
	var b strings.Builder
	for _, v := range vals {
		i := 0
		if top > 0 {
			i = int(v / top * 7)
		}
		b.WriteRune(SparkRunes[min(7, max(0, i))])
	}
	return b.String()
}

// Spectrum são barras verticais (como um equalizador) com `rows` linhas
// de altura; level 0..1 controla a energia. Devolve as linhas de cima
// para baixo.
func Spectrum(width, rows int, t, level float64, rng *rand.Rand) []string {
	heights := make([]float64, width)
	for i := range heights {
		base := (math.Sin(float64(i)*0.7+t*1.3) + math.Sin(float64(i)*0.31-t*2.1) + 2) / 4 // 0..1
		heights[i] = clamp01(base*(0.25+0.75*level) + rng.Float64()*0.25*level)
	}
	out := make([]string, rows)
	for r := range rows {
		var b strings.Builder
		for _, h := range heights {
			fill := h*float64(rows) - float64(rows-1-r) // quanto desta linha está cheio
			switch {
			case fill >= 1:
				b.WriteRune('█')
			case fill <= 0:
				b.WriteRune(' ')
			default:
				b.WriteRune(SparkRunes[int(fill*7)])
			}
		}
		out[r] = b.String()
	}
	return out
}
