# Install guide

This guide starts from a machine with nothing on it and ends with `gclone` working. Skip
any step you have already done.

1. [Install Git](#step-1-install-git)
2. [Install Go](#step-2-install-go)
3. [Install gclone](#step-3-install-gclone)
4. [Check that it works](#step-4-check-that-it-works)

Already have Git and Go 1.24 or newer? Run this and you are done:

```bash
go install github.com/rauni5/gclone@latest
```

Then jump to [step 4](#step-4-check-that-it-works). If `gclone` is "not found" afterwards, see
[Troubleshooting](#troubleshooting).

> The commands for Windows and macOS follow the official instructions of Git, Go and
> Homebrew. They are not all tested on every system by the gclone authors. If one does not
> work for you, please [open an issue](https://github.com/rauni5/gclone/issues) so we can fix
> this page.

---

## Step 1: Install Git

`gclone` uses `git` to clone. Check first whether you have it:

```bash
git --version
```

If that prints a version, go to [step 2](#step-2-install-go).

### Windows

Easiest, in PowerShell:

```powershell
winget install --id Git.Git -e
```

Or download the installer from <https://git-scm.com/download/win> and click through it. The
default options are fine. **Close and reopen PowerShell afterwards**, so it sees the new
`PATH`.

### macOS

```bash
xcode-select --install
```

This installs Apple's command line tools, which include git. Or, if you use
[Homebrew](https://brew.sh):

```bash
brew install git
```

### Linux

| Distribution | Command |
|---|---|
| Debian, Ubuntu, Mint | `sudo apt update && sudo apt install -y git` |
| Fedora | `sudo dnf install -y git` |
| Arch, Manjaro | `sudo pacman -S git` |
| openSUSE | `sudo zypper install git` |

### Tell git who you are (once)

Only needed if you will make commits:

```bash
git config --global user.name "Your Name"
git config --global user.email "you@example.com"
```

Check: `git --version` prints a version.

---

## Step 2: Install Go

gclone needs **Go 1.24 or newer**. Check what you have:

```bash
go version
```

If it prints `go1.24` or higher, go to [step 3](#step-3-install-gclone). If the command is not
found, or the version is older, install Go as below.

> **Prefer not to install Go at all?** You can download a ready-made gclone binary instead
> and skip this step. See [Option B](#option-b-download-a-release-binary).

### Windows

In PowerShell:

```powershell
winget install --id GoLang.Go -e
```

Or download the `.msi` installer for Windows from <https://go.dev/dl/>, open it and click
through it. It adds Go to your `PATH`. **Close and reopen PowerShell**, then check:

```powershell
go version
```

### macOS

Download the `.pkg` installer for your Mac from <https://go.dev/dl/> (Apple Silicon is
`arm64`, Intel is `amd64`) and open it. Or with Homebrew:

```bash
brew install go
```

Open a new terminal and run `go version`.

### Linux

Your package manager often has an old Go. The official tarball is the reliable way. Pick the
file for your machine on <https://go.dev/dl/> (`linux-amd64` for most PCs, `linux-arm64` for
ARM). Replace `1.24.0` below with the version you download:

```bash
curl -LO https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz
```

Add Go to your `PATH`. For bash, put this line in `~/.bashrc` (for zsh, `~/.zshrc`):

```bash
export PATH=$PATH:/usr/local/go/bin
```

Then open a new terminal, or run `source ~/.bashrc`, and check:

```bash
go version
```

### Put Go's install folder on your PATH

`go install` puts programs in a `bin` folder inside your Go folder. For `gclone` to be found
by name, that folder must be on your `PATH`. Find it:

```bash
go env GOPATH
```

The programs are in the `bin` folder inside the printed path (for example
`/home/you/go/bin`, or `C:\Users\you\go\bin` on Windows).

**Linux and macOS**, add to `~/.bashrc` or `~/.zshrc`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

**Windows**: the Go installer usually sets this up. If not, search for "Edit environment
variables for your account", select `Path`, click Edit, add the `bin` folder printed above,
then reopen PowerShell.

Check: `go version` prints `go1.24` or higher.

---

## Step 3: Install gclone

Pick **one** option.

### Option A: `go install` (recommended)

Works on every system that has Go:

```bash
go install github.com/rauni5/gclone@latest
```

For a specific version, replace `latest` with the version, for example `@v0.2.1`.

This builds gclone and puts it in Go's `bin` folder from the previous step.

**Helper scripts** do the same and also fix your `PATH` if needed. From a clone of this
repository:

```bash
# Linux, macOS
scripts/install.sh            # or: scripts/install.sh v0.2.1
```

```powershell
# Windows PowerShell
.\scripts\install.ps1         # or: .\scripts\install.ps1 -Version v0.2.1
```

### Option B: Download a release binary

No Go needed. Go to the [releases page](https://github.com/rauni5/gclone/releases), open the
latest release and download the file for your system:

| System | File |
|---|---|
| Linux, 64-bit PC | `gclone_<version>_linux_amd64.tar.gz` |
| Linux, ARM | `gclone_<version>_linux_arm64.tar.gz` |
| macOS, Intel | `gclone_<version>_darwin_amd64.tar.gz` |
| macOS, Apple Silicon | `gclone_<version>_darwin_arm64.tar.gz` |
| Windows, 64-bit | `gclone_<version>_windows_amd64.zip` |

`<version>` has no leading `v`, so release `v0.2.1` has files named `gclone_0.2.1_...`.

Also download `checksums.txt` from the same release, to verify the file.

**Linux and macOS** (example for 0.2.1 on Linux amd64, change the names to match your file):

```bash
sha256sum -c checksums.txt --ignore-missing      # macOS: shasum -a 256 -c checksums.txt --ignore-missing
tar -xzf gclone_0.2.1_linux_amd64.tar.gz
sudo mv gclone /usr/local/bin/
```

**Windows PowerShell** (example for 0.2.1):

```powershell
# Verify: this hash must match the line for your file in checksums.txt
(Get-FileHash .\gclone_0.2.1_windows_amd64.zip -Algorithm SHA256).Hash
Select-String gclone_0.2.1_windows_amd64.zip checksums.txt

Expand-Archive .\gclone_0.2.1_windows_amd64.zip -DestinationPath $HOME\gclone
```

Then put the `$HOME\gclone` folder on your `PATH` (same "Edit environment variables" screen
as in step 2), and reopen PowerShell.

The binaries are not code-signed yet. Windows SmartScreen or macOS Gatekeeper may warn
about them; see [Troubleshooting](#troubleshooting).

### Option C: Build from source

For contributors, or to try the newest code:

```bash
git clone https://github.com/rauni5/gclone.git
cd gclone
go install .
```

`go build -o gclone .` builds the binary in the current folder instead of installing it. A
build from source prints a version ending in `-dev`. That is expected.

There is no Homebrew, Scoop or winget package yet.

---

## Step 4: Check that it works

Open a **new** terminal and run:

```bash
gclone --version
```

It should print the version you installed. Then ask gclone whether this machine is ready to
run a project (this changes nothing):

```bash
gclone doctor
```

Now try it for real, with `--dry-run` first so nothing is installed:

```bash
gclone --dry-run someone/app
```

Replace `someone/app` with any GitHub repository. Next, read the [usage guide](USAGE.md).

---

## Update

Run the install command again:

```bash
go install github.com/rauni5/gclone@latest
```

If you used a release binary, download the new one and replace the old file.

## Uninstall

Delete the program. With `go install` it is in Go's `bin` folder:

```bash
rm "$(go env GOPATH)/bin/gclone"          # Linux, macOS
```

```powershell
Remove-Item "$(go env GOPATH)\bin\gclone.exe"     # Windows
```

gclone keeps no global settings. In each project it set up it left `SETUP_GUIDE.md` and a
`.gclone/` folder, which you can delete or leave.

---

## Troubleshooting

**`gclone: command not found` (or "not recognized")**
Go's `bin` folder is not on your `PATH`. Run `go env GOPATH`, check the `bin` folder inside
it has `gclone` (`gclone.exe` on Windows), then add that folder to your `PATH` as in
[step 2](#put-gos-install-folder-on-your-path). Open a **new** terminal afterwards.

**`gclone --version` shows an old version, or `gclone doctor` tries to clone something**
An older gclone elsewhere on your machine is found first. Find which one runs:

```bash
which -a gclone          # Linux, macOS
```

```powershell
Get-Command gclone -All  # Windows
```

Remove the old one, or put the folder of the new one earlier on your `PATH`.

**`go: ... requires go >= 1.24` or Go downloads a toolchain**
Your Go is older than gclone needs. Go 1.21 and newer fetch the right toolchain by
themselves when allowed; if that is blocked, install a newer Go (step 2). To see the
setting: `go env GOTOOLCHAIN`.

**`go install` fails with a network or proxy error**
Check your connection. Behind a company proxy, set `HTTPS_PROXY`. If `proxy.golang.org` is
blocked, ask your network admin, or use [Option B](#option-b-download-a-release-binary) or
[Option C](#option-c-build-from-source).

**The version ends in `-dev`**
You built from source or an unreleased version. Install a release with
`go install github.com/rauni5/gclone@latest` or Option B.

**Windows: "Windows protected your PC" (SmartScreen)**
The binary is not signed. Choose *More info*, then *Run anyway*, if you downloaded it from
the official releases page and the checksum matched. Installing with `go install` avoids
this, because you build it yourself.

**macOS: "cannot be opened because the developer cannot be verified"**
Same reason. For a file you downloaded and verified yourself:

```bash
xattr -d com.apple.quarantine /usr/local/bin/gclone
```

**`git: command not found` when running gclone**
Install git ([step 1](#step-1-install-git)) and open a new terminal.

**Still stuck?** See [SUPPORT.md](../SUPPORT.md) and include the output of `gclone --version`,
`go version`, `git --version` and your operating system.
