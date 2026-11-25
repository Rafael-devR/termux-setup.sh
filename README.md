# termux-setup

Configura o Termux do zero (linguagens, ferramentas, um Ubuntu de verdade,
Claude Code, Codex, Gemini e OpenCode, tema, fonte e barra de teclas) com uma
interface animada no terminal.

Escrito em **Go** com **[Bubble Tea v2](https://github.com/charmbracelet/bubbletea)**
(a mesma família de bibliotecas de CLIs modernos), compilado pelo GitHub Actions.

## Instalar

No Termux:

```bash
curl -fsSL https://raw.githubusercontent.com/Rafael-devR/termux-setup.sh/main/install.sh | bash
```

O instalador baixa o binário da última versão publicada, confere o SHA256 e
instala em `$PREFIX/bin/termux-setup`. Se ainda não houver versão
publicada (ou com `--fonte`), ele compila do código-fonte com o Go do
Termux.

## Usar

```bash
termux-setup                 # abre o menu
termux-setup --teste         # simulação: não instala nada
termux-setup dev             # dev padrão (com perguntas e confirmação)
termux-setup dev --sim       # dev padrão, sem perguntas
termux-setup instalar python node ubpy
termux-setup atualizar       # Termux + Ubuntu + IAs
termux-setup doctor           # diagnostica o ambiente e diz como corrigir
termux-setup atualizar-app   # atualiza o próprio termux-setup (o app também avisa no menu quando há versão nova; tecla u)
termux-setup verificar       # versões instaladas
termux-setup desfazer        # volta espelho, aparência e .bashrc
termux-setup modulos         # lista tudo que dá para instalar
termux-setup --help
```

| Opção            | O que faz                                                        |
|------------------|------------------------------------------------------------------|
| `--teste`        | Simula tudo num Termux de mentira (pasta temporária)             |
| `--sim`, `-y`    | Não faz perguntas                                                 |
| `--sem-animacao` | Desliga as animações (ou `TSETUP_ANIM=0`)                         |

No modo teste, `TSETUP_FALHAR=texto` faz falhar qualquer comando que
contenha o texto (para ver como a interface mostra erros), e
`TSETUP_MANTER=1` mantém a pasta temporária para inspecionar.

Sem terminal interativo (saída redirecionada, CI), os comandos funcionam em
modo texto com `--sim`. O código de saída é 0 (completo), 1 (com falhas)
ou 2 (incompleto).

## O que tem dentro

- **Termux:** base, compiladores, Python (com `~/venv`), Node.js, Java,
  Ruby/Perl, SQLite/Redis, rede, extras (tmux, jq, rg…), eza/zoxide,
  gh/lazygit/neovim e Termux:API.
- **Ubuntu (proot):** ferramentas, C/C++, Python com venv, Node.js LTS e
  Java 21. Atualização completa com reparo e limpeza.
- **IAs:** Claude Code (instalador oficial), Codex CLI, Gemini CLI e
  OpenCode, dentro do Ubuntu, com os comandos `claude`, `codex`, `gemini`
  e `opencode` no Termux. A pasta atual do Termux aparece no Ubuntu como `/work/<pasta>`.
- **Aparência:** tema do VS Code, JetBrains Mono (conferida por SHA256),
  barra de teclas com atalhos e sem a mensagem de boas-vindas.
- **Espelhos:** mede os 12 espelhos oficiais e usa o melhor. Se o
  `apt update` falhar, troca de espelho sozinho.
- **Segurança:** os instaladores externos são baixados e conferidos antes
  de rodar, e todo arquivo alterado ganha backup (é o que o desfazer usa).
- **Logs:** cada execução gera `~/termux-setup-logs/setup-*.log` e um
  resumo com diagnóstico automático dos erros conhecidos.

## Desenvolvimento

Precisa de Go 1.26+.

```bash
make test     # todos os testes
make teste    # abre a interface no modo simulação
make lint     # gofmt, go vet e shellcheck
make release V=v6.0.0   # binários em dist/
```

- **Adicionar uma ferramenta:** crie um arquivo em `modules/`. Veja
  [docs/MODULOS.md](docs/MODULOS.md).
- **Entender o código:** [docs/ARQUITETURA.md](docs/ARQUITETURA.md).
- **Publicar uma versão:** `git tag v6.0.0 && git push origin v6.0.0`.
  O workflow de release roda os testes, compila os 4 alvos e publica.

O script bash anterior (v5) está em [`legacy/`](legacy/termux-setup.sh),
congelado, só como referência.

## Ferramentas (menu `t`)

Utilitários dentro do app, no menu **Ferramentas**:

| Grupo | Ferramenta |
|---|---|
| Monitor (ao vivo) | bateria, temperatura, memória, CPU e armazenamento · processos (com opção de encerrar) · Wi-Fi |
| Testes | velocidade (ping/download/upload) · internet (mostra onde quebra) · ping contínuo · estresse de 30 s com temperatura |
| Manutenção | doctor · limpeza (cache do apt, caches, logs antigos) · informações do aparelho (copiáveis) · backup/restauração da configuração |

Bateria, temperatura e Wi-Fi usam o app **Termux:API**; sem ele a tela avisa o que falta. O Android
bloqueia a CPU (`/proc/stat`) em algumas versões, e então ela aparece como "indisponível".
A lógica fica em `internal/tools` (com testes) e cada tela é uma `Tool` em `internal/ui`.
