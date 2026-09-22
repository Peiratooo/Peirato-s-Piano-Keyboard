// The launcher only starts the app; release updates are owned by the app.
package main

import (
	"github.com/Peiratooo/Peirato-s-Piano/launcher/tools"
	"runtime"
)

func main() {
	app := "pp_keyboard"
	if runtime.GOOS == "windows" {
		app += ".exe"
	}
	tools.RunApp(app)
}
