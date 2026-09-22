// Regenerate the precompiled updater binaries from the version pinned in go.mod.
package main

import (
	"context"
	"github.com/Peiratooo/simple-updater/updaterstub"
	"log"
	"os"
)

func main() {
	const pkg = "github.com/Peiratooo/simple-updater/updaterstub/cmd/updater"
	const dir = "assets/updater/"
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatal(err)
	}
	if _, err := updaterstub.Build(context.Background(), updaterstub.BuildOptions{System: "windows", Arch: "amd64", Package: pkg, Output: dir + "updater-windows-amd64.bin"}); err != nil {
		log.Fatal(err)
	}
	if _, err := updaterstub.BuildUniversalDarwin(context.Background(), updaterstub.UniversalBuildOptions{Package: pkg, Output: dir + "updater-darwin.bin"}); err != nil {
		log.Fatal(err)
	}
}
