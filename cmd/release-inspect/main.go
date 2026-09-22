// Inspect an installer using the exact parser used by /release/upload_file.
package main

import (
	"encoding/json"
	simpleupdater "github.com/Peiratooo/simple-updater"
	"log"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: release-inspect <setup.exe|app.dmg>")
	}
	file, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	system, typ, err := simpleupdater.AnalyzePackage(file)
	if err != nil {
		log.Fatal(err)
	}
	if _, err = file.Seek(0, 0); err != nil {
		log.Fatal(err)
	}
	var product *simpleupdater.Product
	if typ == simpleupdater.PackageTypeInno {
		product, err = simpleupdater.AnalyzeInnoSetupEXE(file)
	} else {
		product, err = simpleupdater.AnalyzeSetupDMG(file)
	}
	if err != nil {
		log.Fatal(err)
	}
	if _, err = simpleupdater.GenerateUpdateScript(system, product.Files); err != nil {
		log.Fatal(err)
	}
	product.System = system
	product.PackageType = typ
	if err = json.NewEncoder(os.Stdout).Encode(product); err != nil {
		log.Fatal(err)
	}
}
