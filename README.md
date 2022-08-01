# `debloat++`
Your Android device. Cleaner.

[![Go Report Card](https://goreportcard.com/badge/github.com/lavafroth/debloatplusplus)](https://goreportcard.com/report/github.com/lavafroth/debloatplusplus)

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

- Download the binary for your operating system from the [releases](https://github.com/lavafroth/debloatplusplus/releases).

- If feeling adventurous, try
  ```bash
  go install github.com/lavafroth/debloatplusplus@latest
  ```

- If you're a skeptic willing to inspect the code, clone the repository and in the project directory run
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
