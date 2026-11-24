package project

// RustProject is one Cargo package or workspace (a folder with a Cargo.toml).
type RustProject struct {
	Dir        string
	Rel        string
	Name       string   // package name, or the folder name for a workspace without a package
	Edition    string   // "2021", empty when unknown
	RustWanted string   // rust-version (minimum supported Rust) from Cargo.toml, empty when unset
	Toolchain  string   // channel from rust-toolchain(.toml): "stable", "nightly" or "1.75.0"
	Installed  string   // installed rustc version such as "1.82.0", empty when Rust is missing
	Workspace  bool     // Cargo.toml has a [workspace] table
	HasBin     bool     // builds a program (src/main.rs, src/bin or a [[bin]] table)
	HasLib     bool     // builds a library (src/lib.rs)
	HasLock    bool     // Cargo.lock exists
	Framework  string   // Axum, Actix Web, Rocket, ... empty when none was found
	DBHints    []string // database crates found (sqlx, diesel, ...)
	Warnings   []string
}
