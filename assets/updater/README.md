# Precompiled updater

Source: https://github.com/Peiratooo/simple-updater

Pinned component: v0.2.8. Both binaries are built from that module using Go 1.27.3.

These binaries are packaging inputs, not embedded in the application.
Build once using `go run ./cmd/helpergen` when upgrading the pinned updater dependency.
Normal application builds only copy the existing binaries.

Regenerate both binaries when changing the component or Go toolchain version.
Use the Go version required by go.mod (currently 1.27.3). Updating go.mod or
rebuilding the main application does not rebuild these assets.

The `release` task stages `updater-windows-amd64.bin` as `bin/updater.exe`
for Wails3 Manager's Windows asset list. The macOS app-bundle task copies the
universal `updater-darwin.bin` as `Contents/MacOS/updater` before signing.

The application uses this sibling executable for release updates.
The upstream StartUpdater API makes a temporary execution copy so the installed
updater can itself be replaced during an update. This does not compile any code.

Licenses and notices are available in the pinned upstream module.
