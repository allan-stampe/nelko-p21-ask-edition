# Windows icon and packaging — v1.0.2

## Artwork and generated assets

`assets/windows-icon-source.png` is the original PNG supplied by the owner, extracted from their ZIP without modification. No artwork is redrawn or generated. `tools/prepare_windows_icon.py` uses Pillow at development time only: EXIF orientation is respected, the complete image is resized with Lanczos to a 256-pixel square (transparent padding only if a future source is not square), and ICO frames are 16/32/48/256 pixels.

```sh
python -m pip install Pillow==13.0.0
python tools/prepare_windows_icon.py assets/windows-icon-source.png
python tools/verify_windows_icon.py
```

The generated `cmd/nelko-print/assets/app-icon.png` and `.ico` are committed. The PNG is embedded by Go; the ICO is compiled by MinGW `windres` through `app.rc` into a generated `app_windows_amd64.syso`. Do not commit that object. The object is rebuilt before the native Windows tests/build in GitHub Actions and by `make build-windows`. The resource includes file/product version 1.0.2.0. Manual native build:

```sh
(cd cmd/nelko-print && windres --codepage=65001 --target=pe-x86-64 -O coff -i app.rc -o app_windows_amd64.syso)
go build -mod=readonly -trimpath -ldflags="-s -w -H=windowsgui" -o NELKO-P21-ASK-Edition-v1.0.2-windows-amd64.exe ./cmd/nelko-print
```

Fyne receives `a.SetIcon(appIcon)` before main-window creation and `w.SetIcon(appIcon)` before showing it. Fyne's desktop driver passes the PNG to GLFW, whose Windows backend sets window icons; the EXE resources additionally serve Explorer, shortcuts and taskbar pinning. The ICO/PNG are part of the executable, so copying the standalone EXE is sufficient.

## Automated checks and limits

- The Go icon test decodes the embedded 256×256 PNG and checks Fyne application/window resource assignment using the CI driver.
- `tools/verify_windows_icon.py` checks the four ICO frames, window-PNG equality, PE32+ AMD64 GUI headers, and exact RT_ICON payloads referenced by RT_GROUP_ICON in the actual EXE, plus presence of VERSIONINFO.
- `tools/verify_windows_icon.ps1` uses native Windows icon extraction/GetIconInfo for all four sizes and validates file/product version strings. This exercises Windows' resource loader without needing OpenGL or a Bluetooth printer.
- All existing application/printer tests run on Linux (race detector) and Windows; vet runs on Linux. These are not physical radio or printer tests and do not certify Windows' cached/pinned-shell appearance on the owner's desktop.

## Windows 11 check for the owner

Start the newly downloaded EXE and check the Explorer icon, window icon and taskbar icon. If Windows still shows an old cached icon, remove the old pin and pin the new EXE. Existing preferences/app ID are preserved. Check the same printer's **outgoing Bluetooth Classic SPP COM port** in Windows settings, reconnect and print the reference diagnostic label once. v1.0.1 was physically confirmed using COM4 at 115200/8N1; COM numbering can change, and the current outgoing port is the authoritative choice. The serial/SPP/protocol implementation is byte-for-byte unchanged. No physical v1.0.2 test has yet been reported.
