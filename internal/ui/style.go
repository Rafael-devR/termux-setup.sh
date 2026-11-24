package ui

import (
	"fmt"
	"strings"

	"github.com/Rafael-devR/termux-setup.sh/internal/app"
	"github.com/Rafael-devR/termux-setup.sh/internal/ui/fx"
	"github.com/charmbracelet/x/ansi"
)

// Paleta. Grad é o degradê da marca (ciano → azul → violeta → magenta).
var (
	Grad   = fx.Gradient{fx.Hex("#00F0FF"), fx.Hex("#2E9BFF"), fx.Hex("#7B61FF"), fx.Hex("#C04DFF"), fx.Hex("#FF3EC8")}
	White  = fx.Hex("#F2F4FF")
	Text   = fx.Hex("#C9CEDC")
	Dim    = fx.Hex("#8A90A4")
	Faint  = fx.Hex("#555B6E")
	Ghost  = fx.Hex("#2C3140")
	Green  = fx.Hex("#4DFFA6")
	Red    = fx.Hex("#FF5C7A")
	Yellow = fx.Hex("#FFD166")
	Accent = fx.Hex("#FF3EC8")
	Cyan   = fx.Hex("#00F0FF")
	SelBG  = fx.Hex("#1B1F2E")
	Matrix = fx.Hex("#1FE07A")
)

func sgr(fg, bg fx.Color, bold bool) string {
	s := "\x1b[0"
	if bold {
		s += ";1"
	}
	if fg.OK {
		s += fmt.Sprintf(";38;2;%d;%d;%d", fg.R, fg.G, fg.B)
	}
	if bg.OK {
		s += fmt.Sprintf(";48;2;%d;%d;%d", bg.R, bg.G, bg.B)
	}
	return s + "m"
}

const reset = "\x1b[0m"

// paint pinta um texto.
func paint(s string, c fx.Color) string { return sgr(c, fx.Color{}, false) + s + reset }

// bold pinta em negrito.
func bold(s string, c fx.Color) string { return sgr(c, fx.Color{}, true) + s + reset }

// grad pinta com o degradê da marca (offset desloca as cores: animação).
func grad(s string, offset float64, isBold bool) string {
	rs := []rune(s)
	var b strings.Builder
	for i, r := range rs {
		t := float64(i)/float64(max(1, len(rs)-1)) + offset
		t -= float64(int(t)) // volta ao início (cíclico)
		b.WriteString(sgr(Grad.At(t), fx.Color{}, isBold))
		b.WriteRune(r)
	}
	b.WriteString(reset)
	return b.String()
}

// gradRev é o degradê invertido (magenta → ciano).
func gradRev(s string) string {
	rs := []rune(s)
	var b strings.Builder
	for i, r := range rs {
		b.WriteString(sgr(Grad.At(1-float64(i)/float64(max(1, len(rs)-1))), fx.Color{}, false))
		b.WriteRune(r)
	}
	return b.String() + reset
}

// width é a largura visível (sem códigos de cor).
func width(s string) int { return ansi.StringWidth(s) }

// fit corta um texto colorido na largura.
func fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if width(s) <= w {
		return s
	}
	return ansi.Truncate(s, w, "…")
}

// pad completa com espaços até a largura.
func pad(s string, w int) string {
	if d := w - width(s); d > 0 {
		return s + strings.Repeat(" ", d)
	}
	return s
}

// band é uma linha com fundo destacado de largura w (item selecionado).
func band(s string, w int) string {
	// todo código de cor começa com "\x1b[0" (reset): enxerta o fundo em
	// cada um, senão cada trecho colorido apagaria o destaque
	bgCode := fmt.Sprintf("\x1b[0;48;2;%d;%d;%d", SelBG.R, SelBG.G, SelBG.B)
	line := sgr(fx.Color{}, SelBG, false) + pad(fit(s, w), w)
	return strings.ReplaceAll(line, "\x1b[0", bgCode) + reset
}

// header são as 2 linhas do topo de toda tela.
func header(c *Ctx, title string) []string {
	brand := paint("◈ ", Cyan) + grad("TERMUX", 0, true) + paint("//", Faint) + bold("SETUP", White) + " " + paint(app.Version, Faint)
	if c.S != nil && c.S.Env.DryRun {
		brand += "  " + bold("TESTE", Yellow)
	}
	t := strings.ToUpper(title)
	n := max(2, c.W-width(t)-6)
	rule := gradRev(strings.Repeat("━", n))
	return []string{" " + fit(brand, c.W-1), " " + paint("━━", Cyan) + " " + bold(t, White) + " " + rule}
}

// footer é a linha de dicas de teclas; as dicas que não cabem em w saem.
func footer(w int, hints ...string) string {
	out := ""
	for i, h := range hints {
		part := paint(h, Faint)
		if i > 0 {
			part = paint(" · ", Ghost) + part
		}
		if width(out)+width(part)+1 > w {
			break
		}
		out += part
	}
	return " " + out
}

func mmss(sec int) string { return fmt.Sprintf("%02d:%02d", sec/60, sec%60) }

func humanMB(mb int64) string {
	if mb >= 1024 {
		return fmt.Sprintf("%.1f GB", float64(mb)/1024)
	}
	return fmt.Sprintf("%d MB", mb)
}
