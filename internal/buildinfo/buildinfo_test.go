package buildinfo

import "testing"

func TestPick(t *testing.T) {
	cases := []struct{ set, module, want string }{
		{"0.2.1-dev", "v0.2.0", "0.2.0"},                               // go install ...@v0.2.0
		{"0.2.1-dev", "", "0.2.1-dev"},                                 // go build in a checkout
		{"0.2.0", "v0.2.0", "0.2.0"},                                   // release script: ldflags wins
		{"dev", "v0.2.0", "dev"},                                       // make build with the default VERSION
		{"0.2.1-dev", "v0.2.1-0.20261120-abc", "0.2.1-0.20261120-abc"}, // go install @main
	}
	for _, c := range cases {
		if got := Pick(c.set, c.module); got != c.want {
			t.Errorf("Pick(%q,%q)=%q, want %q", c.set, c.module, got, c.want)
		}
	}
}

func TestModuleIgnoresDevel(t *testing.T) {
	if module("(devel)") != "" || module("") != "" || module("v1.2.3") != "v1.2.3" {
		t.Error("module")
	}
}
