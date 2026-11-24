package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/Rafael-devR/termux-setup.sh/internal/app"
	"github.com/Rafael-devR/termux-setup.sh/internal/ui/fx"
)

// Intro é a abertura: chuva de caracteres com fórmulas ao fundo → logo se
// decifra por cima da chuva → varredura de luz limpa a tela → glitch →
// onda com a própria fórmula → boot log → "ACESSO LIBERADO".
// Qualquer tecla pula.
type Intro struct {
	rain     *fx.Rain
	start    int
	formulas []formula
	next     Screen
}

type formula struct {
	x, y int
	text string
}

// NewIntro cria a abertura; next é a tela que vem depois.
func NewIntro(next Screen) *Intro { return &Intro{next: next, start: -1} }

// Linha do tempo (em quadros, a 30 fps).
const (
	tLogo    = 12  // a chuva deixa de cair na área do logo
	tLocked  = 42  // logo totalmente decifrado
	tSweep   = 44  // começa a varredura
	tGlitch  = 64  // glitch
	tSub     = 72  // subtítulo + onda
	tBoot    = 84  // boot log
	tGranted = 122 // ACESSO LIBERADO
	tEnd     = 150
)

func (in *Intro) NoReveal() {}

func (in *Intro) Init(c *Ctx) tea.Cmd {
	if !c.S.Anim {
		return Replace(in.next)
	}
	in.start = c.Frame
	in.rain = fx.NewRain(c.W, c.H, c.RNG)
	texts := []string{"∇·E = ρ/ε", "e^iπ + 1 = 0", "∑ 1/n² = π²/6", "∫ f(x) dx", "λ = h/p", "0xDEADBEEF",
		"ψ(x,t)", "Δx·Δp ≥ ħ/2", "√2 ≈ 1.4142", "SHA-256", "0x7F454C46", "∞", "Ω", "lim x→0", "E = mc²", "∂ψ/∂t"}
	for _, t := range texts {
		in.formulas = append(in.formulas, formula{x: c.RNG.IntN(max(1, c.W-12)), y: c.RNG.IntN(max(1, c.H)), text: t})
	}
	return nil
}

func (in *Intro) Animating(*Ctx) bool { return true }

func (in *Intro) Update(c *Ctx, msg tea.Msg) (Screen, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyPressMsg:
		return in, Replace(in.next)
	case tea.WindowSizeMsg:
		in.rain = fx.NewRain(c.W, c.H, c.RNG)
	case tickMsg:
		f := c.Frame - in.start
		if f < tSweep {
			in.rain.Step()
		}
		if f >= tEnd {
			return in, Replace(in.next)
		}
	}
	return in, nil
}

func (in *Intro) View(c *Ctx) string {
	if in.rain == nil { // sem animação: a troca para o menu ainda não chegou
		return ""
	}
	f := c.Frame - in.start
	cv := fx.NewCanvas(c.W, c.H)
	big := c.W >= 40 && c.H >= 20
	var logo []string
	if big {
		logo = fx.FullBlocks(fx.Bitmap("TERMUX"))
	} else {
		logo = fx.HalfBlocks(fx.Bitmap("TERMUX"))
	}
	lw, lh := len([]rune(logo[0])), len(logo)
	lx := max(1, (c.W-lw)/2)
	ly := max(1, c.H/2-lh-3)
	inLogo := func(x, y int) bool { return f >= tLogo && x >= lx-2 && x <= lx+lw+1 && y >= ly-1 && y <= ly+lh }
	sweepY := (f - tSweep) * c.H / 18 // linha da varredura

	// camada 1: fórmulas apagadas; camada 2: chuva (apagadas acima da varredura)
	if f < tSweep+18 {
		for _, fm := range in.formulas {
			if fm.y > sweepY || f < tSweep {
				cv.Text(fm.x, fm.y, fm.text, Ghost, false)
			}
		}
		in.rain.Draw(cv, White, Matrix, func(x, y int) bool { return inLogo(x, y) || (f >= tSweep && y <= sweepY) })
	}
	if f >= tSweep && f < tSweep+18 && sweepY < c.H {
		for x := 0; x < c.W; x++ {
			cv.Set(x, sweepY, '━', Grad.At(float64(sweepY)/float64(c.H)))
		}
	}

	// camada 3: logo (decifrando coluna por coluna, depois com sombra)
	if f >= tLogo {
		for y := ly - 1; y <= ly+lh; y++ {
			for x := lx - 2; x <= lx+lw+1; x++ {
				cv.Set(x, y, ' ', fx.Color{})
			}
		}
		if f >= tLocked {
			shadow := Faint
			gx := 0
			if f >= tGlitch && f < tGlitch+6 && f%2 == 0 {
				gx = 2 - (f % 4) // a linha "escorrega"
			}
			fx.DrawBlocks(cv, lx, ly, logo, Grad, shadow)
			if gx != 0 {
				row := (f / 2) % lh
				cv.Clear(ly + row)
				cv.Text(lx+gx, ly+row, logo[row], Accent, false)
			}
		} else {
			for y, line := range logo {
				for x, r := range []rune(line) {
					if r == ' ' {
						continue
					}
					lock := tLogo + 4 + x*24/lw + (x*7+y*3)%4
					if f >= lock {
						cv.Set(lx+x, ly+y, r, Grad.At(float64(x)/float64(lw-1)))
					} else {
						cv.Set(lx+x, ly+y, fx.Glyph(c.RNG), Matrix)
					}
				}
			}
		}
	}

	// subtítulo, onda e fórmula
	sub := ly + lh + 2
	if f >= tSub {
		s := "S E T U P   //   " + app.Version
		cv.Text((c.W-len([]rune(s)))/2, sub, fx.Decrypt(s, float64(f-tSub)/10, c.RNG), White, false)
		wave := fx.Wave(lw, float64(f)/5)
		for i, v := range wave {
			cv.Set(lx+i, sub+2, fx.SparkRunes[v], Grad.At(float64(i)/float64(lw-1)))
		}
		form := "ƒ(x,t) = ½[sin(x+t) + sin(3x+2t)]"
		cv.Text((c.W-len([]rune(form)))/2, sub+3, form, Faint, false)
	}

	// boot log digitado
	if f >= tBoot {
		i := c.S.Info
		boot := []string{
			"kernel ......... android " + i.Android,
			"arquitetura .... " + i.Arch,
			"armazenamento .. " + humanMB(c.S.FreeMB()) + " livres",
			"espelho ........ " + c.S.Mirror(),
			"módulos ........ " + itoa(len(c.S.Cat.All())) + " no catálogo",
			"status ......... ONLINE",
		}
		bx := max(1, lx)
		for k, l := range boot {
			y := sub + 5 + k
			if y >= c.H-2 {
				break
			}
			typed := (f - tBoot - k*5) * 4
			if typed <= 0 {
				break
			}
			rs := []rune(l)
			cv.Text(bx, y, fmt.Sprintf("0x%02X", k*31+16), Faint, false)
			cv.Text(bx+5, y, "›", Grad.At(float64(k)/5), false)
			cv.Text(bx+7, y, string(rs[:min(len(rs), typed)]), Dim, false)
			if typed >= len(rs) && k == len(boot)-1 {
				cv.Text(bx+8+len(rs), y, "✓", Green, true)
			}
		}
	}
	if f >= tGranted {
		s := "▸ ACESSO LIBERADO"
		cv.Text((c.W-len([]rune(s)))/2, min(c.H-1, sub+12), fx.Decrypt(s, float64(f-tGranted)/12, c.RNG), Green, true)
	}
	return cv.Render()
}

// ---------------------------------------------------------------- Outro

// Outro desliga a tela como uma TV de tubo: a imagem se fecha numa linha,
// a linha num ponto, e some.
type Outro struct {
	start int
	base  []string
}

func (o *Outro) NoReveal() {}
func (o *Outro) Init(c *Ctx) tea.Cmd {
	if !c.S.Anim {
		return tea.Quit
	}
	o.start = c.Frame
	o.base = strings.Split(MainMenu().View(c), "\n")
	return nil
}
func (o *Outro) Animating(*Ctx) bool { return true }
func (o *Outro) Update(c *Ctx, msg tea.Msg) (Screen, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyPressMsg:
		return o, tea.Quit
	case tickMsg:
		if c.Frame-o.start > 26 {
			return o, tea.Quit
		}
	}
	return o, nil
}
func (o *Outro) View(c *Ctx) string {
	f := c.Frame - o.start
	lines := make([]string, c.H)
	mid := c.H / 2
	switch {
	case f < 10: // fecha na vertical
		half := c.H / 2 * (10 - f) / 10
		for y := mid - half; y <= mid+half && y < c.H; y++ {
			if y >= 0 && y < len(o.base) {
				lines[y] = o.base[y]
			}
		}
		if t := mid - half - 1; t >= 0 {
			lines[t] = paint(strings.Repeat("▔", c.W), Cyan)
		}
		if b := mid + half + 1; b < c.H {
			lines[b] = paint(strings.Repeat("▁", c.W), Accent)
		}
	case f < 20: // linha encolhendo
		w := c.W * (20 - f) / 10
		lines[mid] = strings.Repeat(" ", (c.W-w)/2) + bold(strings.Repeat("━", w), White)
	case f < 24:
		lines[mid] = strings.Repeat(" ", c.W/2) + bold("●", White)
	default:
		lines[mid] = strings.Repeat(" ", c.W/2) + paint("·", Faint)
	}
	return strings.Join(lines, "\n")
}
