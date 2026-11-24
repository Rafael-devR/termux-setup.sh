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

// O npm do proot pode recusar a plataforma "android": npm_global liga o Node
// "linux", pede --os=linux e, se ainda recusar, repete com --force; no fim
// roda "npm rebuild -g" (conserta instalações antigas sem o postinstall).
func TestNpmGlobalRetriesWithForceAndRebuilds(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("sem bash")
	}
	dir := t.TempDir()
	log := dir + "/chamadas.txt"
	fake := "#!/bin/sh\necho \"$*\" >> " + log + "\ncase \"$1\" in rebuild) exit 0;; esac\ncase \"$*\" in *--force*) exit 0;; esac\necho 'npm error code EBADPLATFORM' >&2\nexit 1\n"
	if err := os.WriteFile(dir+"/npm", []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	script := ubuntu.Script("npm", nil)
	// só as funções e a chamada: tira as linhas que configuram o prefix real
	script = script[:strings.Index(script, "set -e")] + "npm_global opencode-ai@latest @openai/codex\n"
	cmd := exec.Command("bash", "-c", script)
	cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"), "HOME="+dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	data, _ := os.ReadFile(log)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 4 {
		t.Fatalf("esperava 4 chamadas, vieram %q", lines)
	}
	if strings.Contains(lines[0], "--force") || !strings.Contains(lines[0], "--os=linux") {
		t.Errorf("1ª tentativa: %q", lines[0])
	}
	if !strings.Contains(lines[1], "--force") {
		t.Errorf("2ª tentativa deveria ter --force: %q", lines[1])
	}
	// o rebuild usa o NOME do pacote, sem a versão (@latest), inclusive nos escopados
	if lines[2] != "rebuild -g opencode-ai" || lines[3] != "rebuild -g @openai/codex" {
		t.Errorf("rebuild: %q", lines[2:])
	}
}

// O arquivo node-linux.js precisa fazer um Node que se acha "android"
// (como o do proot no celular) dizer "linux".
func TestNodeShimMakesAndroidLookLinux(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("sem node")
	}
	dir := t.TempDir()
	fakeAndroid := dir + "/fake-android.js"
	if err := os.WriteFile(fakeAndroid, []byte("Object.defineProperty(process,'platform',{value:'android'});require('os').platform=()=>'android';"), 0o644); err != nil {
		t.Fatal(err)
	}
	script := ubuntu.Script("npm", nil)
	script = script[:strings.Index(script, "set -e")] + `write_node_shim "$HOME/.s"` + "\n" +
		`node -e 'console.log(process.platform, require("os").platform())'` + "\n"
	run := func(nodeOptions string) string {
		cmd := exec.Command("bash", "-c", script)
		cmd.Env = append(os.Environ(), "HOME="+dir, "NODE_OPTIONS="+nodeOptions)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v\n%s", err, out)
		}
		return strings.TrimSpace(string(out))
	}
	if got := run("--require=" + fakeAndroid); got != "android android" {
		t.Fatalf("o teste precisa começar com o Node achando que é android, veio %q", got)
	}
	if got := run("--require=" + fakeAndroid + " --require=" + dir + "/.s/node-linux.js"); got != "linux linux" {
		t.Fatalf("com o shim deveria dizer linux, veio %q", got)
	}
}

func TestNpmScriptsUseNpmGlobal(t *testing.T) {
	for _, name := range []string{"npm", "ia-update"} {
		s := ubuntu.Script(name, map[string]string{"PKG": "x"})
		body := s[strings.Index(s, "npm_global()")+10:]
		if strings.Contains(body, "npm install -g @") || strings.Contains(body, `npm install -g "$PKG"`) || strings.Contains(body, "npm install -g opencode") {
			t.Errorf("%s ainda chama npm install -g direto", name)
		}
	}
}
