package diagnose

import (
	"strings"
	"testing"
)

func TestHints(t *testing.T) {
	log := "W: GPG error: NO_PUBKEY 123\nE: Unable to locate package foo\nKilled\n"
	h := strings.Join(Hints(log), "\n")
	for _, want := range []string{"Play Store", "não existe", "matou"} {
		if !strings.Contains(h, want) {
			t.Errorf("faltou %q em %q", want, h)
		}
	}
	if h := Hints("npm error code EBADPLATFORM"); len(h) != 1 || !strings.Contains(h[0], "npm") {
		t.Fatal(h)
	}
	if len(Hints("tudo certo")) != 0 {
		t.Fatal("log limpo não deveria ter dicas")
	}
}

func TestErrorLines(t *testing.T) {
	log := strings.Repeat("ok\nE: falhou\n", 30)
	if got := ErrorLines(log, 15); len(got) != 15 || got[0] != "E: falhou" {
		t.Fatal(got)
	}
}
