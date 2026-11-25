# Design system (cores)

Toda cor da interface vem de um **tema** (`internal/ui/theme.go`). Quem desenha usa o
*papel* da cor (`Panel`, `Dim`, `Accent`...) e nunca um valor solto; trocar de tema troca
todos os papéis de uma vez.

| Grupo | Papéis | Uso |
|---|---|---|
| Superfícies | `BG` < `Panel` < `Raised` | fundo da tela, painel, cartão |
| Texto | `White` > `Text` > `Dim` > `Faint` > `Ghost` | título, corpo, apoio, dica, linha apagada |
| Seleção | `Select` + `Ink` | linha selecionada (fundo claro, texto escuro) |
| Acentos | `Accent` (primário), `Cyan` (secundário), `Purple` (títulos de seção) | destaque da marca, barras e cartões, grupos |
| Estado | `Green`, `Yellow`, `Red` | ok, aviso, erro |
| Marca | `Grad`, `Matrix` | logo/barra de progresso, chuva da abertura |

## Temas

`meia-noite` (padrão), `floresta`, `brasa` e `carvao`. Escolha em **Tema da interface**
(tecla `c` no menu principal; a tela muda enquanto você navega) ou force com
`TSETUP_TEMA=floresta termux-setup`. A escolha fica em `~/.config/termux-setup/tema`.

## Regras (testadas em `theme_test.go`)

Para todo tema: `White/Panel` ≥ 12:1, `Text/Panel` ≥ 7:1, `Dim/Panel` ≥ 4.5:1, acentos e
estados ≥ 5:1 sobre o painel, `Ink/Select` ≥ 9:1, e as camadas `BG < Panel < Raised` ficam
em ordem de luminosidade. Para criar um tema novo, copie um de `Themes` e rode os testes.
