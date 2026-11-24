# Módulos: como adicionar uma ferramenta

Cada ferramenta é um arquivo TOML em `modules/`. O menu, a lista de marcar,
as estimativas, o plano de instalação e a tela "Verificar" saem desses
arquivos. Para a maioria das ferramentas, **não é preciso escrever Go**.

## Exemplo mínimo (pacotes do Termux)

`modules/rust.toml`:

```toml
id = "rust"
name = "Rust"
description = "Compilador e cargo."
group = "linguagens"
order = 55
packages = ["rust"]
presets = ["tudo"]
verify = ["rustc --version", "cargo --version"]

[estimate]
seconds = 70
download_mb = 150
disk_mb = 900
```

Depois disso:

```bash
go test ./...                       # valida o catálogo inteiro
go run ./cmd/termux-setup modulos --teste
go run ./cmd/termux-setup instalar rust --teste --sim
```

## Campos

| Campo            | Obrigatório | O que é                                                                 |
|------------------|-------------|-------------------------------------------------------------------------|
| `id`             | não         | Identificador (padrão: o nome do arquivo)                               |
| `name`           | sim         | Nome nos menus                                                          |
| `description`    | não         | Aparece embaixo quando o item está selecionado                          |
| `group`          | sim         | Seção do menu (veja `modules/groups.toml`)                              |
| `order`          | não         | Ordem no menu e na instalação (menor primeiro)                          |
| `target`         | não         | `termux` (padrão), `ubuntu` (apt dentro do Ubuntu) ou `none` (só passos) |
| `packages`       | não         | Pacotes do apt. `"a\|b"` usa o primeiro que existir no espelho          |
| `packages_label` | não         | Nome do passo de pacotes (padrão: `name`)                               |
| `requires`       | não         | Outros módulos necessários (entram sozinhos, por dependência)           |
| `arch`           | não         | Arquiteturas suportadas (`uname -m`); vazio = todas                     |
| `presets`        | não         | Conjuntos onde o módulo entra: `dev`, `ia`, `ubuntu`, `tudo`            |
| `hidden`         | não         | Não aparece nos menus (módulo auxiliar)                                 |
| `verify`         | não         | Comandos de versão para a tela "Verificar"                              |
| `verify_in`      | não         | `termux` ou `ubuntu` (padrão: o `target`)                               |
| `[estimate]`     | não         | `seconds`, `download_mb`, `disk_mb` (aproximados, base arm64)           |
| `[[steps]]`      | não         | Passos extras depois dos pacotes: `label`, `action`, `args`, `seconds`  |
| `[[finalize]]`   | não         | Passos no fim do plano, sem repetir (ex.: "Aplicar aparência")          |

Regras conferidas pelos testes (`go test ./...` falha se alguma quebrar):

- campos desconhecidos são erro (pega erro de digitação);
- `group`, `requires` e `action` precisam existir;
- módulo com `target = "ubuntu"` precisa de `requires = ["ubuntu"]`;
- não pode haver dependência circular.

## Ações disponíveis

Uma ação é uma função Go registrada em `internal/actions/actions.go`.

| Ação              | Argumentos   | O que faz                                          |
|-------------------|--------------|----------------------------------------------------|
| `npm-global`      | pacotes      | `npm install -g` no Termux                         |
| `termux-pyvenv`   |              | Cria `~/venv` com requests e rich                  |
| `ub-npm`          | pacote       | `npm install -g` no Ubuntu (usuário comum)         |
| `ubuntu-node`     |              | Node.js LTS (NodeSource) no Ubuntu                 |
| `ubuntu-pyvenv`   |              | `~/venv` no Ubuntu                                 |
| `claude-install`  |              | Instalador oficial do Claude Code                  |
| `ub-commands`     |              | Cria `claude`, `codex`, `gemini`, `opencode`, `ubuntu`… em `~/bin` |
| `theme` `font` `keys` `hush` |   | Aparência                                          |
| `reload-settings` |              | `termux-reload-settings`                           |
| `root-tsu`        |              | root-repo + tsu                                    |

Para uma ação nova, escreva a função em `internal/actions` (recebe um
`*engine.StepCtx` com executor de comandos, ambiente, log e argumentos),
registre em `Registry()` e use o nome no TOML. Ações que rodam no Ubuntu
devem usar um script em `internal/ubuntu/scripts/` (o CI passa shellcheck
neles).
