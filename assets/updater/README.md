# Precompiled updater

Source: https://github.com/Peiratooo/simple-updater

These binaries are packaging inputs, not embedded in the application.
Build once using `go run ./cmd/updater-builder` when upgrading the pinned updater dependency.
Normal application builds only copy the existing binaries.

The `release` task stages `updater-windows-amd64.bin` as `bin/updater.exe`
for Wails3 Manager's Windows asset list. The macOS app-bundle task copies the
universal `updater-darwin.bin` as `Contents/MacOS/updater` before signing.

The application uses this sibling executable for release updates.
The upstream StartUpdater API makes a temporary execution copy so the installed
updater can itself be replaced during an update. This does not compile any code.

Licenses and notices are available in the pinned upstream module.
