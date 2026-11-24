package cli

import (
	"bytes"
	"strings"
	"testing"
)

func run(args ...string) (int, string) {
	var out, errb bytes.Buffer
	code := Execute(args, &out, &errb)
	return code, out.String() + errb.String()
}

func TestHelpInPortuguese(t *testing.T) {
	code, out := run("--help")
	if code != 0 || !strings.Contains(out, "Comandos:") || !strings.Contains(out, "desfazer") {
		t.Fatal(code, out)
	}
}

func TestModulesList(t *testing.T) {
	code, out := run("modulos", "--teste")
	if code != 0 || !strings.Contains(out, "codex") || !strings.Contains(out, "precisa: ubuntu, ubnode") {
		t.Fatal(code, out)
	}
}

func TestUnknownModule(t *testing.T) {
	if code, out := run("instalar", "xyz", "--teste", "--sim"); code != 1 || !strings.Contains(out, "desconhecido") {
		t.Fatal(code, out)
	}
}

func TestNeedsSimWithoutTerminal(t *testing.T) {
	if code, out := run("dev", "--teste"); code != 1 || !strings.Contains(out, "--sim") {
		t.Fatal(code, out)
	}
}

func TestTextModeInstallAndFailureExitCode(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	code, out := run("instalar", "python", "theme", "--teste", "--sim")
	if code != 0 || !strings.Contains(out, "COMPLETO") {
		t.Fatal(code, out)
	}
	t.Setenv("TSETUP_FALHAR", "python")
	if code, out := run("instalar", "python", "--teste", "--sim"); code != 1 || !strings.Contains(out, "FALHAS") {
		t.Fatal(code, out)
	}
}

func TestLegacyFlags(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	if code, out := run("--desfazer", "--teste", "--sim"); code != 0 || !strings.Contains(out, "Desfazendo") {
		t.Fatal(code, out)
	}
}
