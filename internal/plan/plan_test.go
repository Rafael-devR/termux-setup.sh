package plan_test

import (
	"slices"
	"testing"

	"github.com/Rafael-devR/termux-setup.sh/internal/catalog"
	"github.com/Rafael-devR/termux-setup.sh/internal/plan"
	"github.com/Rafael-devR/termux-setup.sh/modules"
)

func actions(p *plan.Plan) []string {
	var a []string
	for _, s := range p.Steps {
		a = append(a, s.Action)
	}
	return a
}

func count(list []string, v string) int {
	n := 0
	for _, x := range list {
		if x == v {
			n++
		}
	}
	return n
}

func TestInstallDev(t *testing.T) {
	c, _ := catalog.Load(modules.FS)
	p, err := plan.Install(c, c.Preset("dev"), plan.Options{GitName: "Ana", KeysPreset: 2}, "aarch64", 100)
	if err != nil {
		t.Fatal(err)
	}
	a := actions(p)
	if !slices.Equal(a[:4], []string{"mirror-probe", "mirror-best", "apt-update", "apt-upgrade"}) {
		t.Fatalf("preparo: %v", a[:4])
	}
	if a[len(a)-1] != "apt-cleanup" {
		t.Fatal("limpeza deve ser o último passo")
	}
	if count(a, "reload-settings") != 1 {
		t.Fatalf("'Aplicar aparência' deve aparecer uma vez: %v", a)
	}
	for _, want := range []string{"git-config", "termux-commands", "termux-pyvenv", "font"} {
		if !slices.Contains(a, want) {
			t.Errorf("faltou %s", want)
		}
	}
	for _, s := range p.Steps {
		if s.Action == "keys" && s.Args[0] != "2" {
			t.Fatalf("preset de teclas: %v", s.Args)
		}
	}
	if p.Est.DownloadMB < 100 || p.Est.Seconds != p.Seconds() {
		t.Fatalf("estimativa: %+v", p.Est)
	}
}

func TestInstallIAOrdersUbuntuFirst(t *testing.T) {
	c, _ := catalog.Load(modules.FS)
	p, _ := plan.Install(c, []string{"gemini"}, plan.Options{}, "aarch64", 100)
	a := actions(p)
	i, j := slices.Index(a, "ubuntu-install"), slices.Index(a, "ub-npm")
	if i < 0 || j < 0 || i > j || !slices.Contains(a, "ubuntu-node") {
		t.Fatalf("ordem: %v", a)
	}
	if count(a, "ub-commands") != 1 {
		t.Fatal("ub-commands deve aparecer uma vez")
	}
}

func TestUpdate(t *testing.T) {
	if a := actions(plan.Update(true)); !slices.Contains(a, "ubuntu-upgrade") || !slices.Contains(a, "ia-update") {
		t.Fatal(a)
	}
	if a := actions(plan.Update(false)); slices.Contains(a, "ubuntu-upgrade") {
		t.Fatal(a)
	}
}
