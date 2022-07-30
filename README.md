# `debloat++`
> Your Android device. Cleaner.

[![Go Report Card](https://goreportcard.com/badge/github.com/lavafroth/debloatplusplus)](https://goreportcard.com/report/github.com/lavafroth/debloatplusplus)

### Installation

- Install ADB (see the intructions by clicking on your OS below):
  <p>
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
  </p>

  <p>
  <details>
  <summary>macOS</summary>

  - Install [Homebrew](https://brew.sh/)
  - Install *Android platform tools*
    ```bash
    brew install android-platform-tools
    ```
  </details>
  </p>

  <p>
  <details>
  <summary>Windows</summary>

  - Download and extract [android platform tools](https://dl.google.com/android/repository/platform-tools-latest-windows.zip).
  - [Add the extracted folder to your PATH](https://www.architectryan.com/2018/03/17/add-to-the-path-on-windows-10/).
  - [Install USB drivers for your device](https://developer.android.com/studio/run/oem-usb#Drivers)
  - Check if your device is detected:
  ```batch
  adb devices
  ```
  </details>
  </p>

- Install `debloat++`

One can either install the binary by running

```bash
go install github.com/lavafroth/debloatplusplus@latest
```

or clone the repository and in the project directory run

```bash
go run .
```

### Usage
- Kill any previously running adb servers on the host.
```
adb kill-server
```
- Backup the data on your device before you accidentally screw up.
- [Enable Developer Options and USB debugging on your device.](https://developer.android.com/studio/debug/dev-options#enable)
- From the settings, disconnect from any OEM / vendor accounts (deleting an OEM account package could lock you on the lockscreen because the device can no longer associate your identity)
- Run `debloatplusplus` and start typing to search for an app.
- Use the up / down arrow keys to select an app.
- Hit enter to wipe it off the face of your device.
- If an essential system app gets accidentally removed and `debloatplusplus` is still running, select the app (now marked in red) and hit enter again to restore it.

[![asciicast](https://asciinema.org/a/511427.svg)](https://asciinema.org/a/511427)
