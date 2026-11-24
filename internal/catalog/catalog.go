// Package catalog carrega as ferramentas (módulos) a partir de arquivos
// TOML em modules/. Cada arquivo descreve uma ferramenta: nome, pacotes,
// dependências, estimativas e passos extras. Menus, estimativas, plano de
// instalação e a tela "Verificar" saem todos daqui: para adicionar uma
// ferramenta basta criar um arquivo (veja docs/MODULOS.md).
package catalog

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Onde os pacotes do módulo são instalados.
const (
	TargetTermux = "termux" // apt do Termux (padrão)
	TargetUbuntu = "ubuntu" // apt do Ubuntu (proot)
	TargetNone   = "none"   // sem pacotes: só passos
)

// Step é um passo extra do módulo: uma ação registrada no motor.
type Step struct {
	Label   string   `toml:"label"`
	Action  string   `toml:"action"`
	Args    []string `toml:"args"`
	Seconds int      `toml:"seconds"`
}

// Estimate são valores aproximados (base arm64) para tela de confirmação
// e barra de progresso.
type Estimate struct {
	Seconds    int `toml:"seconds"`
	DownloadMB int `toml:"download_mb"`
	DiskMB     int `toml:"disk_mb"`
}

// Manifest é um arquivo de modules/*.toml.
type Manifest struct {
	ID            string   `toml:"id"`
	Name          string   `toml:"name"`
	Description   string   `toml:"description"`
	Group         string   `toml:"group"`
	Order         int      `toml:"order"`
	Target        string   `toml:"target"`
	Packages      []string `toml:"packages"` // "a|b" = o primeiro que existir
	PackagesLabel string   `toml:"packages_label"`
	Requires      []string `toml:"requires"`
	Arch          []string `toml:"arch"` // vazio = todas
	Presets       []string `toml:"presets"`
	Hidden        bool     `toml:"hidden"`
	Estimate      Estimate `toml:"estimate"`
	Steps         []Step   `toml:"steps"`
	Finalize      []Step   `toml:"finalize"`  // no fim do plano, sem repetir
	Verify        []string `toml:"verify"`    // comandos de versão p/ "Verificar"
	VerifyIn      string   `toml:"verify_in"` // "termux" ou "ubuntu" (padrão: o target)

	file string
}

// SupportsArch diz se o módulo roda na arquitetura (uname -m).
func (m *Manifest) SupportsArch(arch string) bool {
	if len(m.Arch) == 0 {
		return true
	}
	return slices.Contains(m.Arch, arch)
}

// Group é uma seção dos menus.
type Group struct {
	ID    string `toml:"id"`
	Title string `toml:"title"`
}

// Catalog é o conjunto de módulos carregado.
type Catalog struct {
	mods   []*Manifest
	byID   map[string]*Manifest
	groups []Group
}

// Load lê groups.toml e todos os *.toml de fsys. Campos desconhecidos são
// erro (pega erro de digitação nos manifestos).
func Load(fsys fs.FS) (*Catalog, error) {
	c := &Catalog{byID: map[string]*Manifest{}}
	files, err := fs.Glob(fsys, "*.toml")
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	for _, f := range files {
		data, err := fs.ReadFile(fsys, f)
		if err != nil {
			return nil, err
		}
		if f == "groups.toml" {
			var g struct {
				Group []Group `toml:"group"`
			}
			if err := decode(data, &g); err != nil {
				return nil, fmt.Errorf("%s: %w", f, err)
			}
			c.groups = g.Group
			continue
		}
		m := &Manifest{file: f}
		if err := decode(data, m); err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		if m.Target == "" {
			m.Target = TargetTermux
		}
		if m.VerifyIn == "" {
			m.VerifyIn = TargetTermux
			if m.Target == TargetUbuntu {
				m.VerifyIn = TargetUbuntu
			}
		}
		if m.ID == "" {
			m.ID = strings.TrimSuffix(path.Base(f), ".toml")
		}
		if _, dup := c.byID[m.ID]; dup {
			return nil, fmt.Errorf("%s: id %q repetido", f, m.ID)
		}
		c.byID[m.ID] = m
		c.mods = append(c.mods, m)
	}
	sort.SliceStable(c.mods, func(i, j int) bool { return c.mods[i].Order < c.mods[j].Order })
	return c, nil
}

func decode(data []byte, v any) error {
	d := toml.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	return d.Decode(v)
}

// All devolve os módulos na ordem do catálogo.
func (c *Catalog) All() []*Manifest { return c.mods }

// Visible devolve os módulos que aparecem nos menus.
func (c *Catalog) Visible() []*Manifest {
	var out []*Manifest
	for _, m := range c.mods {
		if !m.Hidden {
			out = append(out, m)
		}
	}
	return out
}

// Groups devolve as seções dos menus.
func (c *Catalog) Groups() []Group { return c.groups }

// GroupTitle devolve o título de uma seção.
func (c *Catalog) GroupTitle(id string) string {
	for _, g := range c.groups {
		if g.ID == id {
			return g.Title
		}
	}
	return id
}

// Get busca um módulo pelo id.
func (c *Catalog) Get(id string) (*Manifest, bool) { m, ok := c.byID[id]; return m, ok }

// InGroup devolve os ids visíveis de uma seção.
func (c *Catalog) InGroup(group string) []string {
	var ids []string
	for _, m := range c.Visible() {
		if m.Group == group {
			ids = append(ids, m.ID)
		}
	}
	return ids
}

// Preset devolve os ids marcados com o preset (ex.: "dev", "tudo").
func (c *Catalog) Preset(name string) []string {
	var ids []string
	for _, m := range c.mods {
		if slices.Contains(m.Presets, name) {
			ids = append(ids, m.ID)
		}
	}
	return ids
}

// Validate confere o catálogo inteiro: grupos e dependências existentes,
// ações conhecidas pelo motor e ausência de dependência circular.
func (c *Catalog) Validate(knownAction func(string) bool) error {
	var errs []error
	groups := map[string]bool{}
	for _, g := range c.groups {
		groups[g.ID] = true
	}
	for _, m := range c.mods {
		where := m.file
		if m.Name == "" {
			errs = append(errs, fmt.Errorf("%s: falta name", where))
		}
		if !groups[m.Group] {
			errs = append(errs, fmt.Errorf("%s: grupo %q não existe em groups.toml", where, m.Group))
		}
		switch m.Target {
		case TargetTermux, TargetUbuntu, TargetNone:
		default:
			errs = append(errs, fmt.Errorf("%s: target %q inválido", where, m.Target))
		}
		if m.Target == TargetNone && len(m.Packages) > 0 {
			errs = append(errs, fmt.Errorf("%s: target none não pode ter packages", where))
		}
		if m.Target == TargetUbuntu && !slices.Contains(m.Requires, "ubuntu") {
			errs = append(errs, fmt.Errorf("%s: módulo do Ubuntu precisa de requires = [\"ubuntu\"]", where))
		}
		for _, r := range m.Requires {
			if _, ok := c.byID[r]; !ok {
				errs = append(errs, fmt.Errorf("%s: requires %q não existe", where, r))
			}
		}
		for _, s := range append(slices.Clone(m.Steps), m.Finalize...) {
			if s.Label == "" || s.Action == "" {
				errs = append(errs, fmt.Errorf("%s: passo sem label/action", where))
			} else if knownAction != nil && !knownAction(s.Action) {
				errs = append(errs, fmt.Errorf("%s: ação %q não existe no motor", where, s.Action))
			}
		}
	}
	for _, m := range c.mods {
		if cyc := c.cycle(m.ID, map[string]int{}); cyc != "" {
			errs = append(errs, fmt.Errorf("dependência circular: %s", cyc))
			break
		}
	}
	return errors.Join(errs...)
}

func (c *Catalog) cycle(id string, state map[string]int) string {
	switch state[id] {
	case 1:
		return id
	case 2:
		return ""
	}
	state[id] = 1
	for _, r := range c.byID[id].Requires {
		if _, ok := c.byID[r]; !ok {
			continue
		}
		if cyc := c.cycle(r, state); cyc != "" {
			return id + " → " + cyc
		}
	}
	state[id] = 2
	return ""
}

// Resolution é o resultado de Resolve.
type Resolution struct {
	Modules []*Manifest // na ordem de instalação (dependências antes)
	Added   []string    // incluídos por dependência
	Dropped []string    // tirados: não rodam nesta arquitetura
}

// Resolve completa a seleção com as dependências, tira o que não roda na
// arquitetura (e o que depende disso) e ordena: dependências primeiro,
// depois a ordem do catálogo.
func (c *Catalog) Resolve(selected []string, arch string) (Resolution, error) {
	var res Resolution
	want := map[string]bool{}
	for _, id := range selected {
		if _, ok := c.byID[id]; !ok {
			return res, fmt.Errorf("módulo desconhecido: %q", id)
		}
		want[id] = true
	}
	// dependências (transitivas)
	for changed := true; changed; {
		changed = false
		for _, m := range c.mods {
			if !want[m.ID] {
				continue
			}
			for _, r := range m.Requires {
				if !want[r] {
					want[r] = true
					res.Added = append(res.Added, r)
					changed = true
				}
			}
		}
	}
	// arquitetura: tira o módulo e tudo que depende dele
	for changed := true; changed; {
		changed = false
		for _, m := range c.mods {
			if !want[m.ID] {
				continue
			}
			bad := !m.SupportsArch(arch)
			for _, r := range m.Requires {
				bad = bad || !want[r]
			}
			if bad {
				delete(want, m.ID)
				res.Dropped = append(res.Dropped, m.ID)
				changed = true
			}
		}
	}
	// ordem topológica estável
	done := map[string]bool{}
	var visit func(m *Manifest)
	visit = func(m *Manifest) {
		if done[m.ID] {
			return
		}
		done[m.ID] = true
		for _, r := range m.Requires {
			visit(c.byID[r])
		}
		res.Modules = append(res.Modules, m)
	}
	for _, m := range c.mods {
		if want[m.ID] {
			visit(m)
		}
	}
	res.Added = slices.DeleteFunc(res.Added, func(id string) bool { return !want[id] })
	return res, nil
}

// Total soma as estimativas dos módulos (com o fator da arquitetura, em %).
func Total(mods []*Manifest, factorPct int) Estimate {
	var e Estimate
	for _, m := range mods {
		e.Seconds += m.Estimate.Seconds
		e.DownloadMB += m.Estimate.DownloadMB * factorPct / 100
		e.DiskMB += m.Estimate.DiskMB * factorPct / 100
	}
	return e
}
