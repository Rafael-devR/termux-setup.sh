package ui

import (
	"context"
	"os"
	"os/exec"
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/Rafael-devR/termux-setup.sh/internal/catalog"
	"github.com/Rafael-devR/termux-setup.sh/internal/plan"
	"github.com/Rafael-devR/termux-setup.sh/internal/ubuntu"
	"github.com/Rafael-devR/termux-setup.sh/internal/ui/fx"
)

// MainMenu é o menu principal.
func MainMenu() *Menu {
	return &Menu{
		Title: "Menu principal",
		Pre: func(c *Ctx) []string {
			var pre []string
			if c.H >= 28 && c.W >= 38 {
				for _, l := range fx.HalfBlocks(fx.Bitmap("TERMUX")) {
					pre = append(pre, grad(l, 0, false))
				}
				pre = append(pre, "")
			}
			i := c.S.Info
			pre = append(pre,
				paint(fit("Android "+i.Android+" · "+i.Model+" · "+i.Arch, c.W-2), Dim),
				paint(fit(humanMB(c.S.FreeMB())+" livres · "+c.S.Mirror(), c.W-2), Faint))
			return pre
		},
		OnBack: func(*Ctx) tea.Cmd { return Push(&Outro{}) },
		Items: []MenuItem{
			{Key: "1", Label: "Instalação completa (tudo + IA)", Hint: "Todos os módulos, Ubuntu e as 3 IAs.",
				Run: func(c *Ctx) tea.Cmd { return StartInstall(c, c.S.Cat.Preset("tudo")) }},
			{Key: "2", Label: "Dev padrão (recomendado)", Hint: "Linguagens, ferramentas e aparência. Sem Ubuntu.",
				Run: func(c *Ctx) tea.Cmd { return StartInstall(c, c.S.Cat.Preset("dev")) }},
			{Key: "3", Label: "Ubuntu & IA (linguagens + IAs)", Hint: "Ubuntu real no Termux, linguagens e Claude/Codex/Gemini.",
				Run: func(*Ctx) tea.Cmd { return Push(UbuntuMenu()) }},
			{Key: "4", Label: "Aparência (teclas, tema, fonte)", Hint: "Tema, fonte JetBrains Mono e barra de teclas.",
				Run: func(c *Ctx) tea.Cmd {
					return Push(NewChecklist("Aparência", []string{"aparencia"}, c.S.Cat.InGroup("aparencia")))
				}},
			{Key: "5", Label: "Personalizada (escolher módulos)", Hint: "Marque o que quiser, módulo por módulo.",
				Run: func(c *Ctx) tea.Cmd {
					return Push(NewChecklist("Personalizada", groupIDs(c.S.Cat), c.S.Cat.Preset("dev")))
				}},
			{Key: "6", Label: "GitHub (entrar com token)", Hint: "Login do gh por token (o do navegador trava no Termux).",
				Run: func(*Ctx) tea.Cmd { return Push(&GitHub{}) }},
			{Key: "7", Label: "Atualizar tudo (Termux+Ubuntu+IA)", Hint: "Espelho, pacotes, Ubuntu e IAs.",
				Run: func(c *Ctx) tea.Cmd { return updateAll(c) }},
			{Key: "8", Label: "Espelhos: testar / trocar", Hint: "Mede os 12 espelhos oficiais e troca o atual.",
				Run: func(*Ctx) tea.Cmd { return Push(&Mirrors{}) }},
			{Key: "9", Label: "Reparar pacotes quebrados", Hint: "dpkg --configure -a, fix-broken e atualização.",
				Run: func(c *Ctx) tea.Cmd { return Push(NewConfirm(plan.Repair(), nil)) }},
			{Key: "v", Label: "Verificar instalação", Hint: "Versões de tudo que está instalado (Termux e Ubuntu).",
				Run: func(*Ctx) tea.Cmd { return Push(&Verify{}) }},
			{Key: "l", Label: "Ver último log", Hint: "Resumo da última execução.",
				Run: func(*Ctx) tea.Cmd { return Push(NewLogView()) }},
			{Key: "d", Label: "Desfazer (restaurar original)", Hint: "Volta espelho, aparência e .bashrc. Não remove pacotes.",
				Run: func(c *Ctx) tea.Cmd {
					return Push(NewConfirm(plan.Undo(), []string{"Volta ao estado de antes do 1º uso:",
						"espelho, tema/fonte/teclas e aliases.", "Pacotes e o Ubuntu NÃO são removidos."}))
				}},
			{Key: "0", Label: "Sair", Run: func(*Ctx) tea.Cmd { return Push(&Outro{}) }},
		},
	}
}

// UbuntuMenu é o submenu do Ubuntu e das IAs.
func UbuntuMenu() *Menu {
	return &Menu{
		Title: "Ubuntu & IA",
		Pre: func(c *Ctx) []string {
			return []string{paint("Ubuntu real dentro do Termux (proot)", Dim),
				paint("Python · Node · Java · C/C++ · IAs", Dim),
				paint("login das IAs: navegador, no 1º uso", Faint)}
		},
		Items: []MenuItem{
			{Key: "1", Label: "Instalar tudo (linguagens + IAs)", Hint: "Ubuntu, todas as linguagens e as 3 IAs.",
				Run: func(c *Ctx) tea.Cmd {
					ids := append(c.S.Cat.InGroup("ubuntu"), c.S.Cat.InGroup("ia")...)
					return StartInstall(c, append(ids, "base", "api"))
				}},
			{Key: "2", Label: "Escolher o que instalar", Hint: "Marque linguagens e IAs.",
				Run: func(c *Ctx) tea.Cmd {
					return Push(NewChecklist("Ubuntu & IA", []string{"ubuntu", "ia"}, []string{"ubuntu", "ubtools"}))
				}},
			{Key: "3", Label: "Atualizar Ubuntu (+ limpeza)", Hint: "full-upgrade, reparo, autoremove e clean.",
				Run: func(c *Ctx) tea.Cmd { return needUbuntu(c, plan.Ubuntu(true, false)) }},
			{Key: "4", Label: "Atualizar só as IAs", Hint: "Claude, Codex, Gemini, pip e pipx.",
				Run: func(c *Ctx) tea.Cmd { return needUbuntu(c, plan.Ubuntu(false, true)) }},
			{Key: "5", Label: "Atualizar Ubuntu + IAs",
				Run: func(c *Ctx) tea.Cmd { return needUbuntu(c, plan.Ubuntu(true, true)) }},
			{Key: "6", Label: "Entrar no Ubuntu (shell)", Hint: "Shell do usuário " + ubuntu.User + ". Digite exit para voltar.",
				Run: func(c *Ctx) tea.Cmd {
					return Push(NewLoading("Ubuntu", "procurando o Ubuntu", func() tea.Cmd {
						if !c.S.HasUbuntu(context.Background()) {
							return Replace(NewMessage("Ubuntu", Yellow, "Ubuntu ainda não instalado.", "Use \"Instalar tudo\" ou \"Escolher\"."))
						}
						if c.S.Env.DryRun {
							return Replace(NewMessage("Ubuntu", Yellow, "Modo teste: o shell não é aberto."))
						}
						args := ubuntu.ShellCmd()
						cmd := exec.Command(args[0], args[1:]...)
						cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
						return tea.Sequence(Pop, tea.ExecProcess(cmd, func(error) tea.Msg { return nil }))
					}))
				}},
			{Key: "0", Label: "Voltar", Run: func(*Ctx) tea.Cmd { return Pop }},
		},
	}
}

func groupIDs(cat *catalog.Catalog) []string {
	var ids []string
	for _, g := range cat.Groups() {
		ids = append(ids, g.ID)
	}
	return ids
}

// needUbuntu só segue com o plano se o Ubuntu existir.
func needUbuntu(c *Ctx, p *plan.Plan) tea.Cmd {
	return Push(NewLoading("Ubuntu", "procurando o Ubuntu", func() tea.Cmd {
		if !c.S.HasUbuntu(context.Background()) {
			return Replace(NewMessage("Ubuntu", Yellow, "Ubuntu ainda não instalado."))
		}
		return Replace(NewConfirm(p, nil))
	}))
}

func updateAll(c *Ctx) tea.Cmd {
	return Push(NewLoading("Atualizar", "verificando o Ubuntu", func() tea.Cmd {
		return Replace(NewConfirm(plan.Update(c.S.HasUbuntu(context.Background())), nil))
	}))
}

// StartInstall começa o fluxo: perguntas (se precisar) → confirmação → instalação.
func StartInstall(c *Ctx, ids []string) tea.Cmd {
	slices.Sort(ids)
	ids = slices.Compact(ids)
	if c.Auto {
		p, err := plan.Install(c.S.Cat, ids, plan.Options{KeysPreset: 1}, c.S.Info.Arch, c.S.Info.SizeFactor())
		if err != nil {
			return Push(NewMessage("Erro", Red, err.Error()))
		}
		return Push(NewInstall(p))
	}
	return Push(NewOptions(ids))
}
