package tools

import (
	"os/exec"
	"runtime"
)

const webviewUrl = "https://developer.microsoft.com/en-us/microsoft-edge/webview2/"

func OpenWebViewPage() {
	if runtime.GOOS == "windows" {
		var cmd *exec.Cmd
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", webviewUrl)
		err := cmd.Run()
		if err != nil {
			ThrowErrorBox(err.Error())
			return
		}
	} else if runtime.GOOS == "darwin" {
		cmd := exec.Command("open", webviewUrl)
		err := cmd.Run()
		if err != nil {
			ThrowErrorBox(err.Error())
			return
		}
	}
}
