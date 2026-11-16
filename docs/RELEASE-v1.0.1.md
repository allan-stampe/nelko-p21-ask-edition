# NELKO P21 – ASK Edition v1.0.1

**Your printer. Your labels. Your data.**<br>
**No cloud. No account. Just print.**

## Hardware-confirmed on Windows 11

The project owner confirmed printing from this version on a physical NELKO P21 via **COM4, 115200 baud, 8N1**, on 16 November 2026. The final problem was selecting the wrong COM port. Use your printer's actual outgoing Bluetooth port: COM numbers vary between PCs. The prior timing changes were not independently isolated as the cause.

## Download and run

- **Windows 64-bit EXE:** download `NELKO-P21-ASK-Edition-v1.0.1-windows-amd64.exe` and run it. No Go installation or GitHub account is required for this public download.
- **ZIP package:** the same EXE with original MIT license, credits, README, changelog, roadmap and diagnostic/test guides.
- **SHA256SUMS:** checksums of the EXE and ZIP.

Pair the P21 in Windows Bluetooth settings. Close PowerShell or other programs holding its COM port, select the correct port in ASK Edition, and connect. The port is saved locally and restored at restart; Auto reconnect controls automatic reopening. The EXE is not code-signed.

## Included

Editable Designer text, bundled fonts, persistent preferences, local named labels/templates with edit/rename/delete, Danish/English UI, version/battery display, reconnection and Diagnostics. Diagnostics provides timestamped serial ASCII/HEX TX/RX, battery test, Cancel Pause, fixed reference print and saveable logs. Battery polling is optional and off by default. The existing 500 ms / 3 s print timing, partial-write handling and command serialization are retained.

A successful write is not a hardware acknowledgement. Battery support and additional device/firmware combinations are not asserted by the confirmed printing test. See the included guides for testing details.

## ASK Mobile Edition

A separate open-source project is on the roadmap: Android first, iOS planned for a later phase subject to transport feasibility, direct local Bluetooth, no cloud, no account, no subscriptions and no advertising. **Mobile development has not started.**

## Credits and license

Based on **Fyne-P21-Print by Tyler C**, Copyright (c) 2026, under the original **MIT License**, preserved unchanged in the ZIP and source tree. Printer protocol research: [merlinschumacher/nelko-p21-print](https://github.com/merlinschumacher/nelko-p21-print). Built with Fyne and Go.
