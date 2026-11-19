## What this changes

<!-- One or two sentences. Link the issue: "Fixes #123". -->

## Why

<!-- The problem this solves, or the behaviour it improves. -->

## How it was tested

<!-- Commands you ran. For anything that touches Docker, package managers, Windows or macOS,
say whether you tested on a real system or only with a fake. -->

## Did you run it?

<!-- Not only the tests. Which project or folder did you run gclone on, and what did it print? -->

## AI assistance

- [ ] No AI tool wrote a substantial part of this
- [ ] An AI tool helped (which one, and what you checked by hand): 

## Checklist

- [ ] `gofmt -l .` prints nothing, `go vet ./...` and `go test ./...` pass
- [ ] I added or updated tests
- [ ] I updated the README or docs if behaviour changed
- [ ] I added a line to `CHANGELOG.md` under **Unreleased**
- [ ] Anything that installs software, starts a service, clones a repository or runs project
      code asks first (or is behind a flag)
- [ ] No secret values are printed or written anywhere
- [ ] I read my own diff, and I can explain every change
- [ ] I only claim what I checked, and I say what I could not test
