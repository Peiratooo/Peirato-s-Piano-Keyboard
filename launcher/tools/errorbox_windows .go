//go:build windows

package tools

import (
	"github.com/Peiratooo/Peirato-s-Piano/launcher/slog"
	"syscall"
	"unsafe"
)

const (
	MB_OK              = 0x00000000
	MB_ICONINFORMATION = 0x00000040
	MB_ICONWARNING     = 0x00000030
	MB_ICONERROR       = 0x00000010
	MB_TOPMOST         = 0x00040000
	MB_SETFOREGROUND   = 0x00010000
)

func ThrowErrorBox(content string) {
	message := "1-请检查程序文件完整性\n2-请检查WebView2是否已安装\n" + content + "\n若问题仍未解决，请尝试重新安装或联系技术支持。"
	ShowMessage("程序运行失败！", message, MB_ICONERROR)
	OpenWebViewPage()
}

// ShowMessage 弹出一个通用的 Windows 对话框
func ShowMessage(title, content string, icon uintptr) {
	user32 := syscall.NewLazyDLL("user32.dll")
	procMessageBoxW := user32.NewProc("MessageBoxW")
	procCreateWindowExW := user32.NewProc("CreateWindowExW")
	procDestroyWindow := user32.NewProc("DestroyWindow")

	titlePtr, _ := syscall.UTF16PtrFromString(title)
	messagePtr, _ := syscall.UTF16PtrFromString(content)

	// 创建一个临时隐藏窗口作为父窗口
	hwnd, _, _ := procCreateWindowExW.Call(
		0,          // dwExStyle
		0,          // lpClassName
		0,          // lpWindowName
		0,          // dwStyle
		0, 0, 0, 0, // x, y, width, height
		0, // hWndParent
		0, // hMenu
		0, // hInstance
		0, // lpParam
	)

	// 调用 MessageBoxW，父窗口为临时窗口
	flags := icon | MB_TOPMOST | MB_SETFOREGROUND
	_, _, err := procMessageBoxW.Call(
		hwnd,
		uintptr(unsafe.Pointer(messagePtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		flags,
	)
	if err != nil {
		slog.P("调用 MessageBoxW 失败:", err)
	}

	// 销毁临时窗口
	procDestroyWindow.Call(hwnd)
}
