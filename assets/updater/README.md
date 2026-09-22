# Precompiled updater

Source: https://github.com/Peiratooo/simple-updater

These binaries are packaging inputs, not embedded in the application.
Build once using `go run ./cmd/updater-builder` when upgrading the pinned updater dependency.
Normal application builds only copy the existing binaries.

Windows packaging copies `updater-windows-amd64.bin` beside the application as
`updater.exe`. macOS packaging copies the universal `updater-darwin.bin`
as `Contents/MacOS/updater`, before signing the bundle.

The application uses this sibling executable for release updates.
The upstream StartUpdater API makes a temporary execution copy so the installed
updater can itself be replaced during an update. This does not compile any code.

Licenses: `docs/licenses/simple-updater-LICENSE` and accompanying notices.
