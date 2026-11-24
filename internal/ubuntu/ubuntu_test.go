package ubuntu_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/Rafael-devR/termux-setup.sh/internal/ubuntu"
)

func TestScriptQuotesVariables(t *testing.T) {
	s := ubuntu.Script("npm", map[string]string{"PKG": "x'; rm -rf / #"})
	if !strings.HasPrefix(s, `PKG='x'\''; rm -rf / #'`) || !strings.Contains(s, "fetch_check()") {
		t.Fatal(s[:120])
	}
}

func TestScriptsAreValidBash(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("sem bash")
	}
	for _, name := range []string{"base", "apt", "node", "claude", "npm", "pyvenv", "upgrade", "ia-update"} {
		cmd := exec.Command("bash", "-n")
		cmd.Stdin = strings.NewReader(ubuntu.Script(name, map[string]string{"U": "dev"}))
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("%s: %v %s", name, err, out)
		}
	}
	cmd := exec.Command("bash", "-n")
	cmd.Stdin = strings.NewReader(ubuntu.VerifyScript([]string{"node --version", "java -version"}))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("verify: %v %s", err, out)
	}
}

// O npm do proot pode recusar a plataforma "android": npm_global precisa
// pedir --os=linux e, se ainda recusar, repetir com --force.
func TestNpmGlobalRetriesWithForce(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("sem bash")
	}
	dir := t.TempDir()
	log := dir + "/chamadas.txt"
	fake := "#!/bin/sh\necho \"$*\" >> " + log + "\ncase \"$*\" in *--force*) exit 0;; esac\necho 'npm error code EBADPLATFORM' >&2\nexit 1\n"
	if err := os.WriteFile(dir+"/npm", []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	script := ubuntu.Script("npm", nil)
	// só a função e a chamada: tira as linhas que configuram o prefix real
	script = script[:strings.Index(script, "set -e")] + "npm_global opencode-ai\n"
	cmd := exec.Command("bash", "-c", script)
	cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	data, _ := os.ReadFile(log)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], "--os=linux") || strings.Contains(lines[0], "--force") ||
		!strings.Contains(lines[1], "--force") || !strings.Contains(lines[1], "--os=linux") {
		t.Fatalf("chamadas: %q", lines)
	}
}

func TestNpmScriptsUseNpmGlobal(t *testing.T) {
	for _, name := range []string{"npm", "ia-update"} {
		s := ubuntu.Script(name, map[string]string{"PKG": "x"})
		body := s[strings.Index(s, "npm_global()")+10:]
		if strings.Contains(body, "npm install -g @") || strings.Contains(body, `npm install -g "$PKG"`) {
			t.Errorf("%s ainda chama npm install -g direto", name)
		}
	}
}
