package fx

import (
	"math/rand/v2"
	"strings"
	"testing"
	"unicode/utf8"
)

func rng() *rand.Rand { return rand.New(rand.NewPCG(1, 2)) }

func TestDecryptConverges(t *testing.T) {
	r := rng()
	text := "Instalação completa"
	if got := Decrypt(text, 1, r); got != text {
		t.Fatal(got)
	}
	half := Decrypt(text, 0.5, r)
	if utf8.RuneCountInString(half) != utf8.RuneCountInString(text) || !strings.HasPrefix(half, "Instalaç") {
		t.Fatal(half)
	}
	if strings.Count(Decrypt(text, 0, r), " ") != 1 {
		t.Fatal("espaços devem ficar no lugar")
	}
}

func TestLogoShapes(t *testing.T) {
	big := FullBlocks(Bitmap("TERMUX"))
	if len(big) != 5 || utf8.RuneCountInString(big[0]) != 35 {
		t.Fatalf("logo grande: %d linhas, %d colunas", len(big), utf8.RuneCountInString(big[0]))
	}
	small := HalfBlocks(Bitmap("TERMUX"))
	if len(small) != 3 || !strings.HasPrefix(small[0], "▀▀█▀▀") {
		t.Fatalf("%q", small)
	}
	digits := HalfBlocks(Bitmap("42%"))
	if len(digits) != 3 || utf8.RuneCountInString(digits[0]) != 11 {
		t.Fatalf("%q", digits)
	}
}

func TestCanvasRenderAndShadow(t *testing.T) {
	c := NewCanvas(6, 3)
	DrawBlocks(c, 0, 0, []string{"██"}, Gradient{Hex("#00ffff"), Hex("#ff00ff")}, Hex("#333333"))
	if got := c.Plain(); got != "██    \n ░░   \n      " {
		t.Fatalf("%q", got)
	}
	out := c.Render()
	if !strings.Contains(out, "38;2;0;255;255") || strings.Count(out, "\n") != 2 {
		t.Fatal(out)
	}
}

func TestRainStaysInside(t *testing.T) {
	c := NewCanvas(20, 10)
	r := NewRain(20, 10, rng())
	for range 50 {
		r.Step()
		r.Draw(c, Hex("#ffffff"), Hex("#00ff00"), func(x, y int) bool { return x < 4 })
	}
	for y := range 10 {
		if c.At(0, y).R != ' ' {
			t.Fatal("desenhou na área protegida")
		}
	}
}

func TestWaveAndSpectrum(t *testing.T) {
	w := Wave(30, 1.5)
	for _, v := range w {
		if v < 0 || v > 7 {
			t.Fatal(w)
		}
	}
	sp := Spectrum(12, 3, 2, 0.8, rng())
	if len(sp) != 3 || utf8.RuneCountInString(sp[0]) != 12 {
		t.Fatal(sp)
	}
	if Spark([]float64{0, 1, 2}) != "▁▄█" {
		t.Fatal(Spark([]float64{0, 1, 2}))
	}
}

func TestGradient(t *testing.T) {
	g := Gradient{RGB(0, 0, 0), RGB(200, 100, 0)}
	if g.At(0.5) != RGB(100, 50, 0) || g.At(2) != RGB(200, 100, 0) {
		t.Fatal(g.At(0.5))
	}
}
