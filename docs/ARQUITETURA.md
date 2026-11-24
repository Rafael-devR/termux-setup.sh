# Arquitetura

```
cmd/termux-setup/        main: acerta o fuso e chama a CLI
modules/*.toml           catálogo: uma ferramenta por arquivo (embutido no binário)
internal/
  cli/                   comandos (Cobra); escolhe interface ou modo texto
  app/                   monta os serviços: real ou --teste
  ui/                    interface (Bubble Tea v2): telas e navegação
    fx/                  efeitos visuais puros (canvas, logo, chuva, ondas…)
  plan/                  escolha → lista de passos (não executa nada)
  engine/                executa passos, log, resumo, observadores
  actions/               nome da ação (usado nos .toml) → função
  apt/ mirror/ termux/ ubuntu/   domínio: o que cada ação faz de verdade
  catalog/               carrega e valida modules/*.toml, resolve dependências
  diagnose/              explica erros conhecidos do log
  sys/                   tudo que toca o sistema: caminhos, comandos, aparelho
legacy/                  script bash v5 (congelado)
```

## Fluxo de uma instalação

```
menu/CLI ─► catalog.Resolve (dependências, arquitetura)
         ─► plan.Install (passos: espelhos, apt, módulos, finalização)
         ─► engine.Run ──► actions[nome](ctx, StepCtx) ──► sys.Runner
                 │
                 └─► Observers: ui.Install (painel), TextReporter, Notifier
```

- **O motor não conhece a interface.** Ele avisa observadores. O painel
  animado lê um retrato (`engine.Progress.Snapshot`) a cada quadro, em vez
  de processar cada linha do apt.
- **Todo acesso ao sistema passa por `sys.Runner` e `sys.Fetcher`.**
  Existem três implementações: a real, a do modo `--teste` (finge e
  espera um pouco, para a interface ter o que animar) e a falsa dos
  testes (respostas programadas e registro de chamadas).
- **Os downloads usam o curl do Termux**, não a rede do Go: binários Go no
  Android não acham o DNS nem os certificados do sistema.
- **Scripts que rodam dentro do Ubuntu** são arquivos `.sh` reais em
  `internal/ubuntu/scripts/` (variáveis com aspas seguras + `common.sh`).
- **Arquivos que o app cria no Termux** (`.bashrc`, `~/bin`, tema, barras
  de teclas) ficam em `internal/termux/assets/`.

## Interface

- `ui.Model` é a raiz: uma pilha de `Screen` (`Push`, `Pop`, `Replace`,
  `Root`).
- O relógio de animação (30 fps) **só roda enquanto algo anima**: na
  entrada de uma tela, no item que se decifra ou no painel de instalação.
  Parado no menu, o app não gasta CPU.
- Os efeitos são funções puras do número do quadro (`ui/fx`), testadas sem
  terminal.
- Cores em 24 bits. A Bubble Tea converte para 256 cores se o terminal
  não suportar.

## Testes

```bash
go test ./...          # tudo
go test -race ./...    # o CI roda assim
```

- `catalog`: o catálogo embutido é válido; dependências e arquitetura.
- `engine`: sucesso, falha que não para a execução, panic numa ação,
  cancelamento (Ctrl+C), resumo e diagnóstico.
- `actions`: toda ação usada nos `.toml` existe; instala **tudo** no modo
  teste e depois desfaz, conferindo que o espelho original voltou.
- `mirror`, `apt`, `termux`, `ubuntu`: cada regra de domínio (backup feito
  uma vez só, deb822, alternativas de pacote, aspas nos scripts…).
- `ui`: toda tela cabe em 40×24; navegação; o painel roda o motor até o
  resultado; Ctrl+C interrompe.
- `cli`: ajuda, códigos de saída e modo texto.

## Publicação

`scripts/build-release.sh` compila 4 alvos:

| Alvo            | Para quem                         | Observação                         |
|-----------------|-----------------------------------|------------------------------------|
| `android/arm64` | quase todos os celulares          | nativo do Android                  |
| `linux/arm`     | celulares 32 bits                 | Android sem CGO não compila em 32 bits |
| `linux/amd64`   | emuladores x86_64, Chromebooks    |                                    |
| `linux/386`     | emuladores antigos                |                                    |

Nos alvos `linux`, `sys.FixTimezone` lê o fuso do Android (`getprop`) e
usa a base de fusos embutida.
