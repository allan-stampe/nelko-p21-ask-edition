# NELKO P21 – ASK Edition v1.0.0

ASK Edition of **Fyne-P21-Print**, the cross-platform Go/Fyne label maker for NELKO P21.

- Editable multiline Designer text with a scrollable property editor.
- Automatic preferences: bundled font (Go Regular / Bold / Mono), size, label dimensions, darkness, text, orientation and last printer.
- Automatic connection to the last printer at startup and retries after detected connection errors. Explicit Disconnect cancels retries for the current session. Failed print jobs are never replayed automatically.
- Named label and template library: save, open/edit, replace, rename and delete. Includes two editable starter templates. Text and complete Designer documents are supported; add images in Designer to include them in the library.
- Danish (default) and English, selectable in **Language / Sprog**.
- Visible version and battery level, or “unknown” when the existing battery query is unsupported.

### Windows 11 download and testing

The **[ASK Edition - Windows x64](https://github.com/allan-stampe/nelko-p21-ask-edition/actions/workflows/windows-build.yml)** workflow runs tests and builds a standalone 64-bit GUI EXE on every branch push, pull request and manual run. Download the ZIP under **Artifacts** on a successful run. It includes the EXE, original MIT license and test guide. No merge or release tag is needed to build the development branch.

See [the Danish Windows 11 test guide](docs/WINDOWS-11-TEST.md) for physical printer acceptance checks and connection-detection limitations. The existing Bluetooth backends, serial settings, battery command and TSPL output have not been changed.

Preferences and the library are stored locally through Fyne under the stable app id `dk.ask.nelko-p21`. Designer `.nlbl` files remain compatible; older labels without a font field use Go Regular. The last standalone image is not restored on restart; image elements inside saved Designer labels are retained.

### Credits and license

Original Fyne-P21-Print: **Tyler C**, Copyright (c) 2026, MIT. The original [LICENSE](LICENSE) is preserved unchanged. Printer protocol research: [merlinschumacher/nelko-p21-print](https://github.com/merlinschumacher/nelko-p21-print). ASK Edition is developed in [allan-stampe/nelko-p21-ask-edition](https://github.com/allan-stampe/nelko-p21-ask-edition).

### Development checks

```sh
go test -count=1 ./...
go test -tags ci -race -count=1 ./...
go vet -tags ci ./...
go build ./cmd/nelko-print
```

Fyne requires CGO and a C compiler for desktop builds. The `ci` build tag uses Fyne's software driver for tests without desktop system libraries. The original release workflow remains available for tagged cross-platform builds.

---

## Original project documentation

Cross-platform label maker app for the Nelko P21 thermal printer. Works on Linux and Windows!

Written in Go using Fyne

<img width="691" height="607" alt="image" src="https://github.com/user-attachments/assets/d9cfd7af-1121-44f5-845a-ff8512c2498d" />


Based on the reverse engineering work from [merlinschumacher/nelko-p21-print](https://github.com/merlinschumacher/nelko-p21-print).

Here are some vibe docs, I really don't expect anyone to use it but if you stumble upon this and think it's useful, create a new issue and I'll help you get it working and get better docs. Just call the issue "Software missing a good onboarding process" or some shit and we'll work it out. I would be down to package for Flatpak, Snap, and AppImage (plus deb/rpm) if there was genuine interest.

## Quick Start

### Linux
```bash
# One-time setup
./setup.sh

# Run the app
./nelko-print
```

### Windows
1. Download `nelko-print.exe` from releases (or build from source)
2. Pair your Nelko P21 via Windows Bluetooth settings
3. Run the app and select your printer's COM port
4. Print!

### macOS
```bash
# Build (requires Go 1.22+ and Xcode Command Line Tools: xcode-select --install)
make build-macos

# Run
./nelko-print
```
1. Pair your Nelko P21 via System Settings > Bluetooth
2. Run the app and select your printer (shows up as a `/dev/cu.*` port)
3. Print!

## Supported Platforms

| Platform | Status | Notes |
|----------|--------|-------|
| Linux (x64) | ✅ Full support | Auto Bluetooth connection via rfcomm |
| Windows (x64) | ✅ Full support | Uses COM ports (auto-detected when BT paired) |
| macOS (Apple Silicon / Intel) | ✅ Supported | Paired BT printer appears as a `/dev/cu.*` serial port |

## Prerequisites

### Linux (Ubuntu/Debian/Zorin)
```bash
# Fyne deps
sudo apt install -y libgl1-mesa-dev xorg-dev

# Bluetooth
sudo apt install -y bluez

# Add yourself to dialout group (one-time, then logout/login)
sudo usermod -aG dialout $USER
```

### Windows
- Go 1.22+ (for building from source)
- No additional dependencies for running pre-built exe

### macOS
- Go 1.22+
- Xcode Command Line Tools (`xcode-select --install`) - needed by Fyne (CGO)
- No serial driver needed; macOS exposes the paired BT printer as `/dev/cu.*`

## Building

### Linux
```bash
make build
# or
go build -o nelko-print ./cmd/nelko-print
```

### Windows (native)
```powershell
go build -o nelko-print.exe ./cmd/nelko-print
```

### Cross-compile Windows from Linux
```bash
# Install cross-compiler first
make install-cross-deps

# Build Windows exe
make build-windows
```

### Build all platforms
```bash
make build-all
```

## Usage

### Linux - The Easy Way

1. **Pair the printer** via your system's Bluetooth settings (one-time setup)
2. **Run the app**: `./nelko-print`
3. **Select your printer** from the dropdown (auto-detects Nelko devices)
4. **Click Connect** - the app will prompt for your password via `pkexec` to establish the Bluetooth connection
5. **Load an image or type text**, then print!

The app automatically handles the RFCOMM connection that previously required manual terminal commands.

### Windows

1. **Pair the printer** via Windows Settings → Bluetooth & devices
2. **Note the COM port** assigned (check Device Manager → Ports if unsure)
3. **Run the app**: `nelko-print.exe`
4. **Select your COM port** from the dropdown
5. **Click Connect** and print!

On Windows, paired Bluetooth SPP devices appear as COM ports automatically - no special setup needed.

### Manual/Advanced Method (Linux)

If you prefer manual control or the auto-connect doesn't work:

```bash
# Pair the printer first (one-time)
bluetoothctl
> scan on
> pair XX:XX:XX:XX:XX:XX
> trust XX:XX:XX:XX:XX:XX
> quit

# Create RFCOMM device manually
sudo rfcomm connect /dev/rfcomm0 XX:XX:XX:XX:XX:XX 1

# Run the app and use "Advanced" section to connect to /dev/rfcomm0
./nelko-print
```

## Features

- **Designer mode (WYSIWYG)**: Place and arrange multiple elements on a visual
  label - text, B/W graphics, barcodes (Aztec/QR/DataMatrix/Code128), and
  shapes. Drag to move, drag the corner handle to resize, drag the top handle
  to rotate in 45-degree steps. Save/load designs as `.nlbl` files and export
  the composed label as PNG.
- **Image printing**: Load PNG, JPG, GIF, BMP, WebP images
- **Text labels**: Type text directly with adjustable font size
- **Orientation**: Horizontal or Vertical text layout
- **Invert**: White-on-black or black-on-white
- **Word wrap options**: Break anywhere or only on spaces
- **Multiple copies**: Print multiple labels at once
- **Density control**: Adjust print darkness

## Supported Label Sizes

- 12x40mm
- 14x40mm (default)
- 14x50mm
- 14x75mm
- 15x30mm

## How It Works

The Nelko P21 uses Bluetooth Serial Port Profile (SPP/RFCOMM) for communication.

**On Linux:** The `rfcomm` tool creates a virtual serial device (`/dev/rfcommN`) that the app can write print commands to. The app handles this automatically with privilege escalation.

**On Windows:** Paired Bluetooth SPP devices appear as COM ports automatically. Just select the right COM port and connect.

Both platforms then communicate using TSPL2 print commands over the serial connection.

## Troubleshooting

### Linux

#### "Permission denied" on /dev/rfcomm0
Add yourself to the `dialout` group:
```bash
sudo usermod -aG dialout $USER
```
Then log out and back in.

#### rfcomm command not found
```bash
sudo apt install bluez
```

#### pkexec not working / No authentication agent
Install PolicyKit agent for your desktop:
```bash
# GNOME/Zorin
sudo apt install policykit-1-gnome

# KDE
sudo apt install polkit-kde-agent-1
```

### Windows

#### Can't find COM port
1. Open Device Manager
2. Expand "Ports (COM & LPT)"
3. Look for a Bluetooth-related COM port (may say "Standard Serial over Bluetooth link")
4. Note the COM number (e.g., COM3)

#### Printer paired but no COM port appears
Some Windows versions need you to manually add the Serial Port service:
1. Settings → Bluetooth & devices → Devices
2. Click on your printer → More Bluetooth options
3. COM Ports tab → Add → Outgoing → Select your printer

### General

#### Printer not showing in dropdown
Make sure the printer is paired in your system's Bluetooth settings first. The app only shows devices that are already paired.

#### Connection timeout
1. Make sure the printer is powered on and in range
2. Try re-pairing the printer
3. On Linux, use the manual method with `rfcomm connect` to see detailed error messages

## License

MIT
