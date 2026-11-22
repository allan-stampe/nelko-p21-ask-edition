# NELKO P21 – ASK Edition v1.0.2 for Windows

New Windows program icon using the owner's supplied NELKO P21 / Windows / ASK Edition artwork. The ICO contains 16, 32, 48 and 256 pixel images; it is compiled into the Windows x64 GUI EXE. The application and main window explicitly use the same embedded PNG for the window/taskbar icon. No external icon file is required at runtime. Windows version metadata is 1.0.2.0; the window and About version are v1.0.2.

Bluetooth Classic SPP, outgoing COM-port selection/reconnection, serial configuration, printer protocol, bitmap generation, diagnostics and all other functions are unchanged from the physically tested v1.0.1. Original MIT license and Tyler C credits remain; the mobile project is untouched.

Download the standalone Windows x64 EXE or the ZIP containing EXE, ICO and documentation. SHA256SUMS.txt covers both downloads. Windows may cache pinned shortcut icons: remove an old pin and pin the newly downloaded EXE if the old icon persists.

CI runs Linux race tests/vet and native Windows tests. It verifies the exact four ICO payloads inside the final EXE and asks Windows APIs to load each size, as well as checking Windows file/product version metadata. Those checks do not exercise a physical Bluetooth radio/printer. On your Windows 11 PC, use the same paired printer's **outgoing Bluetooth Classic SPP COM port** (previously COM4; the current assigned port may differ) and confirm one reference print. No new physical v1.0.2 result is claimed.

Your printer. Your labels. Your data.
No cloud. No account. Just print.
