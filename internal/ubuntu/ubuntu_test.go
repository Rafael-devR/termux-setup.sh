package ubuntu_test

import (
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
