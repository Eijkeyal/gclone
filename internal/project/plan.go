package project

// Planned is one step of the setup plan: what to show, and how to run it.
type Planned struct {
	Where        string
	Name         string
	Cmd          string
	Run          func() error
	Fallback     func() error
	FallbackNote string
	Resumable    bool // safe to skip on `setup --resume` when it already succeeded
}

// UnitLabel names a folder for people: "." becomes "repo root".
func UnitLabel(l string) string {
	if l == "" || l == "." {
		return "repo root"
	}
	return l
}
