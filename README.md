# `debloat++`
Your Android device. Cleaner.

[![Go Report Card](https://goreportcard.com/badge/github.com/lavafroth/debloatplusplus)](https://goreportcard.com/report/github.com/lavafroth/debloatplusplus)

## Introduction

### What?

`debloat++` is a cross-platform TUI application which utilizes `adb` and `aapt` to help the user
remove unwanted system apps ([bloatware](https://en.wikipedia.org/wiki/Software_bloat)) from
their android device without root access. The tool aids removal of such apps on all versions of
Android with x86 or ARM processors. This subsequently increases storage space, reduces
power consumption and hardens the user's privacy against vendor distributed spyware.

### Why?

Despite the existence of projects like [the UAD project](https://github.com/0x192/Universal-Android-Debloater),
there has recently been a lot of trouble correlating package names with an app's label.
Manufacturers including Oppo, Xiaomi and the like use obscure package names for their spyware
apps which make it difficult if not impossible to remove them without playing Russian roulette
and risking a [bootloop](https://en.wikipedia.org/wiki/Bootloop).

### How?

`deboat++` solves the aforementioned issue by fetching the package names as well as app labels
using the `aapt` binaries. This can help the user get better insights on whether an app is
safe to get rid of.

With that being said, a user still runs the risk of hitting a bootloop if they have absolutely
no idea of what they're doing.

## Installation

### Install ADB
Choose an OS:
<details>
  <summary>Linux</summary>
  
  Debian:
  ```bash
  sudo apt install android-sdk-platform-tools
  ```

  Arch:
  ```bash
  sudo pacman -S android-tools
  ```

  Red Hat:
  ```bash
  sudo yum install android-tools
  ```

  OpenSUSE:
  ```bash
  sudo zypper install android-tools
  ```

  </details>

  <details>
  <summary>macOS</summary>

  - Install [Homebrew](https://brew.sh/#install)
  - Install *Android platform tools*
    ```bash
    brew install android-platform-tools
    ```
  </details>
  <details>
  <summary>Windows</summary>

  - Install [Chocolatey](https://chocolatey.org/install#install-step2)
  - Install adb
    ```powershell
    choco install adb
    ```
  </details>

### Install `debloat++`

`debloat++` can be installed in either of the following ways:

#### Using precompiled binaries
This is what most users will use since it does not involve setting up a development environment. Download the binary for your operating system from the [releases](https://github.com/lavafroth/debloatplusplus/releases).

#### Using `go install`
If feeling adventurous, try the following to fetch and compile the bleeding edge version of the code.

```bash
go install github.com/lavafroth/debloatplusplus@latest
```

#### For developers and tinkers
If you're a skeptic willing to inspect or tinker with the code, clone the repository

```bash
git clone https://github.com/lavafroth/debloatplusplus.git
```

and in the project directory run

```bash
go run .
```

## Usage
- Kill any previously running adb servers on the host
```
adb kill-server
```
- Backup the data on your device before you accidentally screw up
- [Enable Developer Options and USB debugging on your device](https://developer.android.com/studio/debug/dev-options#enable)
- From the settings, disconnect from any OEM / vendor accounts (deleting an OEM account package could lock you on the lockscreen because the device can no longer associate your identity)
- Run `debloatplusplus` and start typing to search for an app
- Use the up / down arrow keys to select an app
- Hit enter to wipe it off the face of your device
- If an essential system app gets accidentally removed and `debloatplusplus` is still running, select the app (now marked in red) and hit enter again to restore it

[![asciicast](https://asciinema.org/a/511965.svg)](https://asciinema.org/a/511965)
