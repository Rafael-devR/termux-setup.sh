// Package plan transforma uma escolha (módulos + opções) numa lista de
// passos. Não executa nada: o motor (engine) é que roda os passos.
package plan

import (
	"slices"

	"github.com/Rafael-devR/termux-setup.sh/internal/catalog"
)

// Options são as respostas do usuário antes de instalar.
type Options struct {
	GitName    string
	GitEmail   string
	Storage    bool // pedir acesso ao armazenamento
	KeysPreset int  // 1 compacta, 2 grande, 3 padrão do Termux
}

// Step é um passo do plano: uma ação do motor com argumentos.
type Step struct {
	Label   string
	Action  string
	Args    []string
	Seconds int    // peso na barra de progresso
	Module  string // módulo de origem ("" = passo do sistema)
}

// Kind é o tipo do plano.
type Kind int

const (
	KindInstall Kind = iota
	KindUpdate
	KindRepair
	KindUndo
	KindMirror
)

// Plan é a lista de passos e o que se sabe sobre ela.
type Plan struct {
	Title    string
	Kind     Kind
	Steps    []Step
	Res      catalog.Resolution
	Est      catalog.Estimate
	Options  Options
	Selected []string
}

// Seconds é o tempo estimado total.
func (p *Plan) Seconds() int {
	n := 0
	for _, s := range p.Steps {
		n += s.Seconds
	}
	return n
}

// Has diz se o plano inclui o módulo.
func (p *Plan) Has(module string) bool {
	return slices.ContainsFunc(p.Res.Modules, func(m *catalog.Manifest) bool { return m.ID == module })
}

func prepare() []Step {
	return []Step{
		{Label: "Testar espelhos", Action: "mirror-probe", Seconds: 4},
		{Label: "Aplicar melhor espelho", Action: "mirror-best", Seconds: 1},
		{Label: "Atualizar listas (apt)", Action: "apt-update", Seconds: 5},
		{Label: "Atualizar pacotes", Action: "apt-upgrade", Seconds: 35},
	}
}

var cleanup = Step{Label: "Limpeza", Action: "apt-cleanup", Seconds: 3}

// Install monta o plano de instalação dos módulos escolhidos.
// factorPct ajusta tamanhos à arquitetura (100 = arm64).
func Install(c *catalog.Catalog, selected []string, opt Options, arch string, factorPct int) (*Plan, error) {
	res, err := c.Resolve(selected, arch)
	if err != nil {
		return nil, err
	}
	p := &Plan{Title: "Instalando", Kind: KindInstall, Res: res, Options: opt, Selected: selected}
	p.Steps = prepare()
	var final []Step
	for _, m := range res.Modules {
		extra := 0
		for _, s := range m.Steps {
			extra += s.Seconds
		}
		if len(m.Packages) > 0 {
			label := m.PackagesLabel
			if label == "" {
				label = m.Name
			}
			action := "termux-install"
			if m.Target == catalog.TargetUbuntu {
				action = "ubuntu-apt"
			}
			p.Steps = append(p.Steps, Step{Label: label, Action: action, Args: m.Packages,
				Seconds: max(1, m.Estimate.Seconds-extra), Module: m.ID})
		}
		for _, s := range m.Steps {
			st := Step{Label: s.Label, Action: s.Action, Args: s.Args, Seconds: max(1, s.Seconds), Module: m.ID}
			if s.Action == "keys" {
				st.Args = []string{presetArg(opt.KeysPreset)}
			}
			p.Steps = append(p.Steps, st)
		}
		for _, s := range m.Finalize {
			if !slices.ContainsFunc(final, func(f Step) bool { return f.Action == s.Action }) {
				final = append(final, Step{Label: s.Label, Action: s.Action, Args: s.Args, Seconds: max(1, s.Seconds), Module: m.ID})
			}
		}
	}
	if opt.GitName != "" || opt.GitEmail != "" {
		p.Steps = append(p.Steps, Step{Label: "Configurar Git", Action: "git-config", Args: []string{opt.GitName, opt.GitEmail}, Seconds: 1})
	}
	if opt.Storage {
		p.Steps = append(p.Steps, Step{Label: "Acesso ao armazenamento", Action: "storage", Seconds: 3})
	}
	p.Steps = append(p.Steps, Step{Label: "Criar comandos e aliases", Action: "termux-commands", Seconds: 1})
	p.Steps = append(p.Steps, final...)
	p.Steps = append(p.Steps, cleanup)
	p.Est = catalog.Total(res.Modules, factorPct)
	p.Est.DownloadMB += 30 // listas e atualização do sistema
	p.Est.DiskMB += 60
	p.Est.Seconds = p.Seconds()
	return p, nil
}

func presetArg(n int) string {
	switch n {
	case 2:
		return "2"
	case 3:
		return "3"
	}
	return "1"
}

// Update atualiza o Termux e, se existir, o Ubuntu e as IAs.
func Update(withUbuntu bool) *Plan {
	p := &Plan{Title: "Atualizando", Kind: KindUpdate, Steps: prepare()}
	if withUbuntu {
		p.Steps = append(p.Steps,
			Step{Label: "Ubuntu: atualizar tudo", Action: "ubuntu-upgrade", Seconds: 40},
			Step{Label: "Atualizar as IAs", Action: "ia-update", Seconds: 40})
	}
	p.Steps = append(p.Steps, cleanup)
	p.Est.Seconds = p.Seconds()
	return p
}

// Ubuntu atualiza só o Ubuntu, só as IAs ou os dois.
func Ubuntu(upgrade, ia bool) *Plan {
	p := &Plan{Title: "Ubuntu", Kind: KindUpdate}
	if upgrade {
		p.Steps = append(p.Steps, Step{Label: "Ubuntu: atualizar tudo", Action: "ubuntu-upgrade", Seconds: 40})
	}
	if ia {
		p.Steps = append(p.Steps, Step{Label: "Atualizar as IAs", Action: "ia-update", Seconds: 40})
	}
	p.Est.Seconds = p.Seconds()
	return p
}

// Repair conserta pacotes quebrados.
func Repair() *Plan {
	p := &Plan{Title: "Reparando", Kind: KindRepair, Steps: []Step{
		{Label: "Reparar pacotes quebrados", Action: "apt-repair", Seconds: 30}, cleanup}}
	p.Est.Seconds = p.Seconds()
	return p
}

// Undo volta espelho, aparência e .bashrc ao estado de antes do 1º uso.
func Undo() *Plan {
	p := &Plan{Title: "Desfazendo", Kind: KindUndo, Steps: []Step{
		{Label: "Restaurar espelho original", Action: "undo-apt", Seconds: 5},
		{Label: "Restaurar aparência", Action: "undo-look", Seconds: 1},
		{Label: "Limpar ~/.bashrc", Action: "undo-bashrc", Seconds: 1},
	}}
	p.Est.Seconds = p.Seconds()
	return p
}

// Mirror aplica um espelho escolhido e atualiza as listas.
func Mirror(url string) *Plan {
	p := &Plan{Title: "Espelho", Kind: KindMirror, Steps: []Step{
		{Label: "Aplicar espelho", Action: "mirror-apply", Args: []string{url}, Seconds: 1},
		{Label: "Atualizar listas (apt)", Action: "apt-update", Seconds: 5},
	}}
	p.Est.Seconds = p.Seconds()
	return p
}
