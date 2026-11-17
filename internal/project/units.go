package project

// HasUnits reports whether any supported ecosystem was found.
func (p *Project) HasUnits() bool {
	return len(p.Pkgs)+len(p.GoModules)+len(p.JavaProjects)+len(p.PyProjects) > 0
}
