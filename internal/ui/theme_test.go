package ui

import (
	"math"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Rafael-devR/termux-setup.sh/internal/ui/fx"
	"github.com/charmbracelet/x/ansi"
)

func lum(c fx.Color) float64 {
	f := func(v uint8) float64 {
		x := float64(v) / 255
		if x <= 0.03928 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	return 0.2126*f(c.R) + 0.7152*f(c.G) + 0.0722*f(c.B)
}

// contrast é a razão de contraste WCAG entre duas cores (1 a 21).
func contrast(a, b fx.Color) float64 {
	la, lb := lum(a), lum(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// Todo tema precisa ser legível: o design system garante contraste mínimo
// entre cada papel de cor e a superfície onde ele aparece.
func TestEveryThemeIsReadable(t *testing.T) {
	defer SetTheme(CurrentTheme())
	for _, th := range Themes {
		if !SetTheme(th.ID) {
			t.Fatalf("%s: não ativou", th.ID)
		}
		checks := []struct {
			name   string
			fg, bg fx.Color
			min    float64
		}{
			{"White/Panel", White, Panel, 12},
			{"Text/Panel", Text, Panel, 7},
			{"Dim/Panel", Dim, Panel, 4.5},
			{"Faint/Panel", Faint, Panel, 2.5}, // só dicas e rótulos de apoio
			{"Ink/Select", Ink, Select, 9},
			{"Primary/Panel", Accent, Panel, 5},
			{"Secondary/Panel", Cyan, Panel, 6},
			{"Section/Panel", Purple, Panel, 6},
			{"Green/Panel", Green, Panel, 6},
			{"Yellow/Panel", Yellow, Panel, 6},
			{"Red/Panel", Red, Panel, 5},
			{"Text/Raised", Text, Raised, 6},
			{"Dim/Raised", Dim, Raised, 4},
			{"Ink/Yellow", Ink, Yellow, 8}, // selo TESTE
		}
		for _, c := range checks {
			if got := contrast(c.fg, c.bg); got < c.min {
				t.Errorf("%s: %s contraste %.1f (mínimo %.1f)", th.ID, c.name, got, c.min)
			}
		}
		// as três superfícies precisam se distinguir (camadas visíveis)
		if !(lum(BG) < lum(Panel) && lum(Panel) < lum(Raised)) {
			t.Errorf("%s: camadas BG < Panel < Raised fora de ordem", th.ID)
		}
		// o degradê tem 5 paradas válidas
		for i := 0; i < 5; i++ {
			if !Grad.At(float64(i) / 4).OK {
				t.Errorf("%s: degradê inválido", th.ID)
			}
		}
	}
}

func TestThemeLookupAndPersistence(t *testing.T) {
	defer SetTheme(CurrentTheme())
	if _, ok := FindTheme("  FLORESTA "); !ok {
		t.Fatal("deveria achar por nome ignorando caixa")
	}
	if SetTheme("nao-existe") {
		t.Fatal("tema inexistente foi aceito")
	}
	home := t.TempDir()
	SetTheme("brasa")
	if err := SaveTheme(home); err != nil {
		t.Fatal(err)
	}
	SetTheme("meia-noite")
	LoadTheme(home)
	if CurrentTheme() != "brasa" {
		t.Fatal(CurrentTheme())
	}
	t.Setenv("TSETUP_TEMA", "carvao")
	LoadTheme(home)
	if CurrentTheme() != "carvao" {
		t.Fatalf("TSETUP_TEMA deveria vencer: %s", CurrentTheme())
	}
}

func TestThemePickerPreviewsSavesAndCancels(t *testing.T) {
	defer SetTheme(CurrentTheme())
	s := services(t, false)
	c := &Ctx{W: 46, H: 40, S: s, RNG: newRNG()}
	SetTheme("meia-noite")

	p := NewThemePicker()
	p.Update(c, key("down")) // pré-visualiza o 2º
	if CurrentTheme() != Themes[1].ID {
		t.Fatal("não pré-visualizou:", CurrentTheme())
	}
	if v := ansi.Strip(p.View(c)); !strings.Contains(v, Themes[1].Name) {
		t.Fatal(v)
	}
	p.Update(c, key("esc")) // cancela: volta ao tema de antes
	if CurrentTheme() != "meia-noite" {
		t.Fatal("esc não desfez:", CurrentTheme())
	}

	p = NewThemePicker()
	p.Update(c, key("down"))
	_, cmd := p.Update(c, tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil || !p.saved {
		t.Fatal("enter deveria salvar e voltar")
	}
	SetTheme("meia-noite")
	LoadTheme(s.Env.Home)
	if CurrentTheme() != Themes[1].ID {
		t.Fatal("não salvou:", CurrentTheme())
	}
}
