//go:build windows

package tools

import (
	"github.com/Peiratooo/Peirato-s-Piano/launcher/slog"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/shirou/gopsutil/process"
	"golang.org/x/sys/windows/registry"
)

func isInProgramFiles() bool {
	exePath, err := os.Executable()
	if err != nil {
		return false
	}
	// 转换为小写以便比较
	exePath = strings.ToLower(exePath)
	return strings.HasPrefix(exePath, strings.ToLower(`C:\Program Files (x86)`)) || strings.HasPrefix(exePath, strings.ToLower(`C:\Program Files`))
}

func appPath(app string) string {
	exePath, err := os.Executable()
	if err != nil {
		return app
	}
	return filepath.Join(filepath.Dir(exePath), app)
}

func CheckExeName(name, app string) bool {
	return strings.Contains(name, app)
}
func IsRunning(app string) bool {

	processes, err := process.Processes()
	if err != nil {
		return false
	}
	j := len(processes)
	res := false
	for i := 0; i < j-1; i++ {
		j--
		name, _ := processes[i].Exe()
		if CheckExeName(name, app) {
			res = true
		}
		name, _ = processes[j].Exe()
		if CheckExeName(name, app) {
			res = true
		}
		if res {
			return true
		}
	}
	return false
}

func runAppInternal(app string, retried bool) {
	if IsRunning(app) {
		return
	}

	slog.P("启动程序:", app)

	var cmd *exec.Cmd
	target := appPath(app)
	if isInProgramFiles() {
		powershellCmd := `Start-Process -FilePath $env:PIANO_LAUNCH_TARGET -WorkingDirectory $env:PIANO_LAUNCH_DIR -Verb RunAs`
		cmd = exec.Command("powershell", "-Command", powershellCmd)
		cmd.Env = append(os.Environ(), "PIANO_LAUNCH_TARGET="+target, "PIANO_LAUNCH_DIR="+filepath.Dir(target))
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow: true,
		}
	} else {
		cmd = exec.Command(target)
		cmd.Dir = filepath.Dir(target)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		slog.P("启动失败:", err)

		if !retried {
			slog.P("启动失败，尝试修复 WebView2 后重试")
			EnsureWebView2()
			runAppInternal(app, true)
			return
		}

		ThrowErrorBox("启动失败：" + err.Error())
		return
	}

	// Release updates may replace the launcher: detach after starting the app.
	if isInProgramFiles() {
		_ = cmd.Wait()
	} else {
		_ = cmd.Process.Release()
	}
}

func RunApp(app string) {
	runAppInternal(app, false)
}

func IsWebView2Installed() bool {
	const guid = `{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`

	type check struct {
		root registry.Key
		path string
	}

	checks := []check{
		// 系统级（最常见）
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\` + guid},
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\EdgeUpdate\Clients\` + guid},
		// 用户级
		{registry.CURRENT_USER, `SOFTWARE\Microsoft\EdgeUpdate\Clients\` + guid},
	}

	for _, c := range checks {
		k, err := registry.OpenKey(c.root, c.path, registry.READ)
		if err != nil {
			continue
		}

		location, _, _ := k.GetStringValue("location")
		k.Close()

		if location == "" {
			continue
		}

		if hasValidWebView2Exe(location) {
			return true
		}
	}

	return false
}

func hasValidWebView2Exe(base string) bool {
	entries, err := os.ReadDir(base)
	if err != nil {
		return false
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		exe := filepath.Join(base, e.Name(), "msedgewebview2.exe")
		if _, err := os.Stat(exe); err == nil {
			return true
		}
	}
	return false
}

func DownloadWebView2Installer(path string) error {
	resp, err := http.Get("https://go.microsoft.com/fwlink/p/?LinkId=2124703")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func InstallWebView2(installerPath string) error {
	cmd := exec.Command(installerPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: false,
	}
	return cmd.Run()
}

func EnsureWebView2() {
	if runtime.GOOS != "windows" {
		return
	}
	if IsWebView2Installed() {
		return
	}

	// 2. 如果没装，尝试下载
	tmp := filepath.Join(os.TempDir(), "webview2.exe")
	err := DownloadWebView2Installer(tmp)
	if err != nil {
		ThrowErrorBox("依赖组件下载失败，请检查网络: " + err.Error())
		os.Exit(1)
	}

	//3. 尝试安装
	err = InstallWebView2(tmp)
	os.Remove(tmp)
	if err != nil {
		msg := "您的系统缺少必要组件 WebView2，且自动安装失败。\n\n" +
			"解决办法：\n" +
			"请关闭程序，然后【右键点击此启动器】选择【以管理员身份运行】。"
		ShowMessage("环境初始化提示", msg, MB_ICONWARNING)
		os.Exit(1)
	}
	//ShowMessage("安装成功", "WebView2 组件已安装完成，点击确定开始运行应用。", MB_OK|MB_ICONINFORMATION)
}
