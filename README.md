# `debloat++`
[![Go Report Card](https://goreportcard.com/badge/github.com/lavafroth/debloatplusplus)](https://goreportcard.com/report/github.com/lavafroth/debloatplusplus)

Minimalist yet functional android debloat tool

### Installation

This project needs `adb` as a prerequisite. Make sure to get it
installed and available in the `PATH` environment variable
before proceeding. To install `debloat++` one can either

```bash
go install github.com/lavafroth/debloatplusplus@latest
```

or clone the repository and in the project directory run

```bash
go run .
```

### Usage
* Just `go run .` and start typing to search for an app.
* Use the up / down arrow keys to select a package.
* Hit enter to wipe it off the face of your device.

[![asciicast](https://asciinema.org/a/511427.svg)](https://asciinema.org/a/511427)
