// Package ubuntu cuida do Ubuntu que roda dentro do Termux (proot-distro):
// instalar, preparar o usuário comum, instalar linguagens e IAs, atualizar.
// Os comandos que rodam lá dentro são os scripts de scripts/*.sh.
package ubuntu

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strings"

	"github.com/Rafael-devR/termux-setup.sh/internal/engine"
	"github.com/Rafael-devR/termux-setup.sh/internal/sys"
)

// Configuração (mude aqui sem mexer no resto).
var (
	Distro   = "ubuntu"
	User     = "dev" // usuário comum: as IAs não devem rodar como root
	Versions = []string{"26.04", "24.04"}

	NodeSourceURL = "https://deb.nodesource.com/setup_lts.x"
	ClaudeURL     = "https://claude.ai/install.sh"
)

//go:embed scripts/*.sh
var scripts embed.FS

// Script monta um script: variáveis (com aspas seguras) + common.sh + nome.sh.
func Script(name string, vars map[string]string) string {
	common, _ := scripts.ReadFile("scripts/common.sh")
	body, err := scripts.ReadFile("scripts/" + name + ".sh")
	if err != nil {
		panic("script ausente: " + name)
	}
	var b strings.Builder
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%s\n", k, Quote(vars[k]))
	}
	b.Write(common)
	b.WriteString("\n")
	b.Write(body)
	return b.String()
}

// Quote põe aspas simples seguras para o shell.
func Quote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

func login(user, script string) sys.Cmd {
	args := []string{"login", Distro}
	if user != "" {
		args = append(args, "--user", user)
	}
	return sys.C("proot-distro", append(args, "--", "bash", "-lc", script)...)
}

// AsRoot roda um script como root no Ubuntu.
func AsRoot(ctx context.Context, s *engine.StepCtx, name string, vars map[string]string) error {
	return s.ExecCmd(ctx, login("", Script(name, vars)))
}

// AsUser roda um script como o usuário comum.
func AsUser(ctx context.Context, s *engine.StepCtx, name string, vars map[string]string) error {
	return s.ExecCmd(ctx, login(User, Script(name, vars)))
}

// Has diz se o Ubuntu está instalado.
func Has(ctx context.Context, r sys.Runner) bool {
	return r.Has("proot-distro") && sys.OK(ctx, r, sys.C("proot-distro", "login", Distro, "--", "true"))
}

// Version devolve o nome do Ubuntu instalado (ex.: "Ubuntu 26.04 LTS").
func Version(ctx context.Context, r sys.Runner) string {
	out, err := sys.Output(ctx, r, sys.C("proot-distro", "login", Distro, "--", "grep", "PRETTY_NAME", "/etc/os-release"))
	if err != nil {
		return "?"
	}
	v := strings.TrimSpace(out)
	if i := strings.IndexByte(v, '='); i >= 0 {
		v = v[i+1:]
	}
	return strings.Trim(v, `"`)
}

// Install instala a 1ª versão disponível de Versions; avisa se não foi a
// preferida. Se já existir, mantém (não substitui).
func Install(ctx context.Context, s *engine.StepCtx) error {
	if !s.Run.Has("proot-distro") {
		return fmt.Errorf("proot-distro ausente")
	}
	if Has(ctx, s.Run) {
		s.Logf("Ubuntu já instalado: %s. Mantido (não é substituído).", Version(ctx, s.Run))
		return nil
	}
	for i, v := range Versions {
		if s.Exec(ctx, "proot-distro", "install", Distro+":"+v) == nil {
			if i > 0 {
				s.Warn(fmt.Sprintf("Ubuntu %s indisponível no proot-distro; instalado o %s.", Versions[0], v))
			}
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	if err := s.Exec(ctx, "proot-distro", "install", Distro); err != nil {
		return err
	}
	s.Warn(fmt.Sprintf("Nenhuma versão de %v disponível; instalada a padrão: %s.", Versions, Version(ctx, s.Run)))
	return nil
}

// Require falha com mensagem clara se o Ubuntu não existir.
func Require(ctx context.Context, s *engine.StepCtx) error {
	if !Has(ctx, s.Run) {
		return fmt.Errorf("Ubuntu não encontrado (instale pelo menu Ubuntu & IA)")
	}
	return nil
}

// VerifyScript gera um script que imprime "OK nome versão" ou "NO nome"
// para cada comando de versão.
func VerifyScript(cmds []string) string {
	var b strings.Builder
	b.WriteString(`export PATH="$HOME/.local/bin:$HOME/.npm-global/bin:$PATH"` + "\n")
	for _, c := range cmds {
		f := strings.Fields(c)
		if len(f) == 0 {
			continue
		}
		fmt.Fprintf(&b, "if command -v %s >/dev/null 2>&1; then echo \"OK %s $(%s 2>&1 | head -n1)\"; else echo \"NO %s\"; fi\n",
			Quote(f[0]), f[0], c, f[0])
	}
	return b.String()
}

// RunAsUser roda um script solto como o usuário comum (sem common.sh).
func RunAsUser(ctx context.Context, r sys.Runner, script string) (string, error) {
	return sys.Output(ctx, r, login(User, script))
}

// ShellCmd é o comando para abrir um shell interativo no Ubuntu.
func ShellCmd() []string {
	return []string{"proot-distro", "login", Distro, "--user", User, "--", "bash", "-l"}
}
