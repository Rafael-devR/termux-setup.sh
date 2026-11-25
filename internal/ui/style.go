package ui

import (
	"fmt"
	"strings"

	"github.com/Rafael-devR/termux-setup.sh/internal/app"
	"github.com/Rafael-devR/termux-setup.sh/internal/ui/fx"
	"github.com/charmbracelet/x/ansi"
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

// fill pinta o fundo bg numa linha inteira de largura w, mantendo o fundo
// depois de cada código de cor que a linha já traga.
func fill(s string, w int, bg fx.Color) string {
	// todo código de cor começa com "\x1b[0" (reset): enxerta o fundo em
	// cada um, senão cada trecho colorido apagaria o fundo
	bgCode := fmt.Sprintf("\x1b[0;48;2;%d;%d;%d", bg.R, bg.G, bg.B)
	s = strings.ReplaceAll(pad(fit(s, w), w), "\x1b[m", "\x1b[0m") // reset curto → longo
	line := sgr(fx.Color{}, bg, false) + s
	return strings.ReplaceAll(line, "\x1b[0", bgCode) + reset
}

// band é a linha selecionada: preenchida de ponta a ponta com a cor de
// destaque e texto escuro (como a paleta de comandos do OpenCode).
func band(s string, w int) string {
	plain := strings.TrimLeft(ansi.Strip(s), "▌ ")
	return sgr(Ink, Select, true) + pad(fit("  "+plain, w), w) + reset
}

// section é o título de um grupo de itens.
func section(s string) string { return bold(s, Purple) }

// tag alinha um rótulo discreto à direita de uma linha de largura w.
func tag(left, right string, w int) string {
	gap := w - width(left) - width(right) - 1
	if gap < 1 {
		return fit(left, w)
	}
	return left + strings.Repeat(" ", gap) + right
}

// card é um bloco com barra de destaque à esquerda e fundo elevado, com a
// borda de baixo em meio-bloco (o "┃ … ╹▀▀▀" do OpenCode).
func card(w int, accent fx.Color, lines ...string) []string {
	out := []string{}
	for _, l := range lines {
		out = append(out, paint("┃", accent)+fill("  "+l, w-1, Raised))
	}
	out = append(out, paint("╹", accent)+paint(strings.Repeat("▀", max(0, w-1)), Raised))
	return out
}

// Marcadores invisíveis que o compositor (app.View) usa para achar o
// painel: header marca o início, footer marca o fim.
const (
	markHead = "\x00H"
	markFoot = "\x00F"
)

// header são as 2 linhas do topo de toda tela: a marca e o título do painel.
func header(c *Ctx, title string) []string {
	brand := paint("termux", Dim) + bold("setup", White) + " " + paint(app.Version, Faint)
	if c.S != nil && c.S.Env.DryRun {
		brand += "  " + sgr(Ink, Yellow, true) + " TESTE " + reset
	}
	return []string{" " + fit(brand, c.W-1), markHead + bold(title, White)}
}

// footer são as dicas de teclas ("tecla rótulo"), alinhadas à direita; as
// que não cabem em w saem pelo fim.
func footer(w int, hints ...string) string {
	var parts []string
	total := 0
	for _, h := range hints {
		k, label, _ := strings.Cut(h, " ")
		part := bold(k, White) + " " + paint(label, Dim)
		pw := width(part) + 2
		if total+pw > w-1 {
			break
		}
		parts = append(parts, part)
		total += pw
	}
	line := strings.Join(parts, "  ")
	return markFoot + strings.Repeat(" ", max(0, w-width(line)-1)) + line
}

func mmss(sec int) string { return fmt.Sprintf("%02d:%02d", sec/60, sec%60) }

func humanMB(mb int64) string {
	if mb >= 1024 {
		return fmt.Sprintf("%.1f GB", float64(mb)/1024)
	}
	return fmt.Sprintf("%d MB", mb)
}
