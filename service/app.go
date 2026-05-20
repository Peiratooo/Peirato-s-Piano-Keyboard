package service

import (
	"embed"
	"fmt"
	"log"
	"os/exec"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

var (
	App        *application.App
	PianoWin   *application.WebviewWindow
	ControlWin *application.WebviewWindow
	MidiWin    *application.WebviewWindow
	PRODUCTION = false
)

type WindowSize struct {
	Width  int
	Height int
}

func initSize() WindowSize {
	// 这里先使用一个稳定的默认尺寸，避免原来的 Windows User32 调用导致 macOS/Linux 无法编译。
	// 后续如果需要更精准的屏幕自适应，可以按平台拆成 window_size_windows.go / window_size_darwin.go。
	return WindowSize{Width: 1280, Height: 170}
}

func Run(assets embed.FS) {
	size := initSize()
	App = application.New(application.Options{
		Name:        "Peirato's Piano Keyboard",
		Description: "A piano keyboard desktop widget",
		Services: []application.Service{
			application.NewService(&Keyboard{}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID:               "com.peirato.piano",
			OnSecondInstanceLaunch: nil,
			AdditionalData:         nil,
			ExitCode:               0,
		},
		LogLevel:   application.DialogWarning,
		OnShutdown: CloseMidiDevice,
	})

	PianoWin = App.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "Peirato's Piano Keyboard",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			TitleBar:                application.MacTitleBarHidden,
		},
		Windows: application.WindowsWindow{
			BackdropType:                      0,
			WindowMaskDraggable:               true,
			DisableFramelessWindowDecorations: true,
		},
		Frameless:              true,
		Width:                  size.Width,
		Height:                 size.Height,
		MinHeight:              100,
		MinWidth:               800,
		BackgroundType:         application.BackgroundTypeTranslucent,
		BackgroundColour:       application.NewRGBA(0, 0, 0, 0),
		URL:                    "/",
		DevToolsEnabled:        !PRODUCTION,
		OpenInspectorOnStartup: !PRODUCTION,
		EnableFileDrop:         true,
		AlwaysOnTop:            true,
	})
	App.Window.Add(PianoWin)

	// 窗口创建完成后再开始扫描设备，避免 App.Event.Emit 早于 App 初始化。
	go ListenDevices()

	if err := App.Run(); err != nil {
		log.Fatal(err)
	}
}

func (k *Keyboard) OpenUrl(url string) {
	// OpenUrl 只负责打开外部链接。
	// 应用内部动作请使用明确的 Wails 绑定方法，例如 OpenControlCenter / AllNotesOff。
	_ = OpenExternalURL(url)
}

func OpenExternalURL(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

func (k *Keyboard) OpenControlCenter() {
	if App == nil {
		return
	}

	if ControlWin != nil {
		ControlWin.Show()
		ControlWin.Focus()
		return
	}

	ControlWin = App.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:                  "设置中心",
		URL:                    "/#/control",
		Width:                  980,
		Height:                 700,
		MinWidth:               860,
		MinHeight:              600,
		Frameless:              false,
		AlwaysOnTop:            false,
		BackgroundType:         application.BackgroundTypeSolid,
		DevToolsEnabled:        !PRODUCTION,
		OpenInspectorOnStartup: false,
		EnableFileDrop:         true,
	})
	ControlWin.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
		ControlWin = nil
	})
	App.Window.Add(ControlWin)
	ControlWin.Show()
}

func (k *Keyboard) OpenMidiCenter() {
	if App == nil {
		return
	}

	if MidiWin != nil {
		MidiWin.Show()
		MidiWin.Focus()
		emitMidiWindowState(true)
		return
	}
	MidiWin = App.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "MIDI 中心",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 40,
			TitleBar:                application.MacTitleBarHidden,
		},
		Windows: application.WindowsWindow{
			BackdropType:                      0,
			WindowMaskDraggable:               true,
			DisableFramelessWindowDecorations: true,
		},
		Width:                  1120,
		Height:                 360,
		MinWidth:               920,
		MinHeight:              300,
		BackgroundType:         application.BackgroundTypeSolid,
		BackgroundColour:       application.NewRGBA(245, 247, 251, 255),
		URL:                    "/#/midi",
		DevToolsEnabled:        !PRODUCTION,
		OpenInspectorOnStartup: false,
		EnableFileDrop:         true,
		AlwaysOnTop:            false,
	})
	MidiWin.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
		MidiWin = nil
		_ = MidiPlayer.Reset()
		emitMidiWindowState(false)
	})
	App.Window.Add(MidiWin)
	MidiWin.Show()
	emitMidiWindowState(true)
}

func (k *Keyboard) Quit() {
	(&Keyboard{}).AllNotesOff()
	App.Quit()
}

func (k *Keyboard) OpenSoundFontDialog() error {
	if App == nil {
		return fmt.Errorf("应用尚未初始化")
	}

	path, err := App.Dialog.OpenFile().
		SetTitle("选择 SoundFont 音源").
		AddFilter("SoundFont", "*.sf2").
		AddFilter("All Files", "*.*").
		PromptForSingleSelection()

	if err != nil {
		return err
	}
	if path == "" {
		return fmt.Errorf("未选择 SoundFont")
	}

	return AddSoundFontByPath(path)
}

func (k *Keyboard) GetMidiWindowOpen() bool {
	return MidiWin != nil
}

func emitMidiWindowState(open bool) {
	if App != nil {
		App.Event.Emit("midiWindowState", map[string]bool{"open": open})
	}
}
