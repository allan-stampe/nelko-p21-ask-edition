# Changelog

## v1.0.1 — 2026-11-16

### Hardware confirmation

- Project owner confirmed successful printing from ASK Edition v1.0.1 on a physical NELKO P21 under Windows 11, using **COM4, 115200 baud, 8N1**.
- The remaining failure was selecting the wrong COM port. The earlier PowerShell report used COM5; the confirmed ASK configuration uses COM4 on that PC. Port numbers vary by PC.
- The timing changes have not been independently isolated as the cause. The earlier version has not been reported as retested on COM4. Battery replies and the complete acceptance checklist are not claimed as hardware-verified.

### Added

- Diagnostics tab with timestamped actual serial TX/RX in escaped ASCII and HEX, battery test, Cancel Pause, reference print and log export.
- Fixed 14x40 mm, density 15, one-copy reference job with 3408 binary bitmap bytes and separate command/header/payload/CRLF/PRINT writes. Its test pattern is new; the original PowerShell bitmap was not supplied.
- Saved opt-in automatic battery polling; off by default to keep extra requests out of the reference test.
- Regression coverage for exact bytes, timing, partial/error writes, transaction exclusion, binary battery replies, communication logs and restoring the last COM port across a fresh app instance.
- Official GitHub Actions Windows x64 release assets: standalone EXE, ZIP with MIT license/credits/docs, and SHA256 checksums.
- Project slogan and a separate-project roadmap for ASK Mobile Edition. Mobile development has not started.

### Changed and fixed

- Printing waits 500 ms after Cancel Pause and 3 seconds after the print job under one serial lock.
- Cancel Pause failures abort a job; short writes are completed or reported as errors. Failed jobs are never automatically replayed.
- Binary battery data is no longer trimmed as whitespace; malformed or unavailable replies display unknown.
- The saved serial port remains selectable even when temporarily missing from enumeration. Startup reconnection uses the saved target; COM4 is not hardcoded.
- README and diagnostic/test guides document the successful hardware result and correct port selection.

### Preserved

Designer, label/template library, Danish/English UI, saved preferences, font/size, label dimensions, density, text, reconnection, Diagnostics, Bluetooth backends and 115200/8N1 settings. Original MIT license and Tyler C attribution remain unchanged.

## v1.0.0 — 2026-11-16

Initial ASK Edition development build: editable multiline Designer text, bundled Go fonts, persistent settings/design, named labels/templates with edit/rename/delete, Danish/English UI, reconnection, version/battery display and Windows build workflow. Preserved the original Fyne-P21-Print MIT license and attribution. Superseded by v1.0.1; no official v1.0.0 release is claimed here.
