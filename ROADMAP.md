# Roadmap

> **Your printer. Your labels. Your data.**<br>
> **No cloud. No account. Just print.**

## Desktop ASK Edition

- Maintain the hardware-confirmed Windows 11 / NELKO P21 workflow and retain Diagnostics.
- Collect reproducible compatibility reports using the real outgoing COM port, OS version and exported logs.
- Investigate additional printer/firmware/label combinations and battery response formats before claiming support.
- Continue improving local label design, template management and port-selection guidance.

## ASK Mobile Edition

**Status: planned only. Development has not started.**

ASK Mobile Edition will be a **separate open-source project and repository**, with its own code, tests and releases. This desktop repository contains the roadmap only. No mobile app, mobile scaffold, separate repository or implementation is created as part of v1.0.1.

### Principles

- **Android first**, with **iOS planned for a later phase**, subject to transport feasibility.
- **Direct local Bluetooth communication** between phone and printer.
- **No cloud**: labels and printing stay local.
- **No user account**.
- **No subscriptions**.
- **No advertising**.
- **Open source** with third-party licenses and original developer credits respected.

### Planned stages

1. Define supported printer models and investigate their local Bluetooth transports, permissions and protocol behavior on Android.
2. Plan an Android app with local label creation/storage, printer selection, printing and diagnostics.
3. Validate actual hardware and privacy/offline behavior before any release.
4. Investigate iOS transport feasibility and platform restrictions before committing to an implementation or support claim. Compatibility with the P21's desktop Bluetooth serial transport must not be assumed on iOS.

These are future goals, not implemented capabilities or delivery dates. There is no mobile development commitment or start in this release task.
