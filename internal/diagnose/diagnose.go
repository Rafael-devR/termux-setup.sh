// Package diagnose lê o log de uma execução e explica erros conhecidos em
// português simples. Para ensinar um erro novo, adicione uma regra.
package diagnose

import (
	"regexp"
	"strings"
)

type rule struct {
	re   *regexp.Regexp
	hint string
}

var rules = []rule{
	{regexp.MustCompile(`NO_PUBKEY|is not signed`), "Chave GPG ausente: provável Termux da Play Store. Use o app do F-Droid/GitHub."},
	{regexp.MustCompile(`unmet dependencies`), "Dependências conflitantes: espelho dessincronizado. Use Espelhos e Reparar."},
	{regexp.MustCompile(`Could not resolve|Temporary failure|Failed to fetch|timed out|Err:[0-9]`), "Falha de rede/espelho: tente outro espelho ou outra rede."},
	{regexp.MustCompile(`No space left`), "Sem espaço no armazenamento."},
	{regexp.MustCompile(`dpkg was interrupted`), "dpkg interrompido: use o menu Reparar (ou: dpkg --configure -a)."},
	{regexp.MustCompile(`Unable to locate package`), "Algum pacote não existe neste espelho/arquitetura."},
	{regexp.MustCompile(`\bKilled\b|signal 9`), "O Android matou um processo (memória). Feche apps; Opções do desenvolvedor > limite de processos."},
	{regexp.MustCompile(`(?i)permission denied`), "Erro de permissão em algum arquivo."},
	{regexp.MustCompile(`EBADPLATFORM`), "O npm recusou a plataforma (o Node do Ubuntu se acha \"android\"). Atualize o termux-setup (v6.0.2+): ele já faz o Node dizer \"linux\"."},
	{regexp.MustCompile(`postinstall script was not run`), "Um pacote npm ficou sem o postinstall. Rode: termux-setup instalar opencode (ou o módulo do pacote) de novo."},
}

// Hints devolve as explicações para os erros encontrados no log.
func Hints(log string) []string {
	var out []string
	for _, r := range rules {
		if r.re.MatchString(log) {
			out = append(out, r.hint)
		}
	}
	return out
}

var errLine = regexp.MustCompile(`(?i)(^E:|^Err:|error|failed|unable|cannot|denied|unmet)`)

// ErrorLines devolve as últimas n linhas que parecem erro.
func ErrorLines(log string, n int) []string {
	var out []string
	for _, l := range strings.Split(log, "\n") {
		if errLine.MatchString(l) {
			if len(l) > 110 {
				l = l[:110]
			}
			out = append(out, l)
		}
	}
	if len(out) > n {
		out = out[len(out)-n:]
	}
	return out
}
