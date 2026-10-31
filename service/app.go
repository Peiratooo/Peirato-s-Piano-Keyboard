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
	PRODUCTION = false
)

type WindowSize struct {
	Width  int
	Height int
}

func initSize() WindowSize {
	layout := GetUserConfig().MainWindowLayouts.Waterfall
	return WindowSize{Width: layout.Width, Height: layout.Height}
}

func Run(assets embed.FS) {
	size := initSize()
	App = application.New(application.Options{
		Name:        appDisplayName,
		Description: "Piano performance and waterfall practice",
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
		OnShutdown: func() { stopRecordingForShutdown(); cleanupUpdate(); CloseMidiDevice(); finishCommunityPractice() },
	})

	PianoWin = App.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: appDisplayName,
		Mac: application.MacWindow{
			// The frontend marks its blank title area as draggable so toolbar
			// buttons keep receiving clicks, including in compact mode.
			TitleBar: application.MacTitleBarHiddenInset,
			Backdrop: application.MacBackdropTransparent,
		},
		Windows: application.WindowsWindow{
			BackdropType:                      0,
			WindowMaskDraggable:               true,
			DisableFramelessWindowDecorations: true,
		},
		// Keep native macOS window controls; the frontend reserves their space.
		Frameless:              runtime.GOOS != "darwin",
		Width:                  size.Width,
		Height:                 size.Height,
		MinHeight:              600,
		MinWidth:               960,
		BackgroundType:         application.BackgroundTypeTranslucent,
		BackgroundColour:       application.NewRGBA(28, 28, 28, 255),
		URL:                    "/",
		DevToolsEnabled:        !PRODUCTION,
		OpenInspectorOnStartup: false,
		EnableFileDrop:         true,
		AlwaysOnTop:            false,
		// Apply the screen work-area clamp before the first visible frame.
		Hidden: true,
	})
	App.Window.Add(PianoWin)
	installMainWindowPresentation()
	installMainWindowCloseHandler()

	// 窗口创建完成后再开始扫描设备，避免 App.Event.Emit 早于 App 初始化。
	go ListenDevices()

	if err := App.Run(); err != nil {
		log.Fatal(err)
	}
}

// Closing the main window on macOS quits the app, including any settings window.
// Native close requests must first preserve recording drafts, just like the
// frontend Quit action. Disk work and shutdown stay off the UI thread.
func installMainWindowCloseHandler() {
	quitOnClose := runtime.GOOS == "darwin"
	pending := false // protected by inputEventsMu
	PianoWin.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		inputEventsMu.Lock()
		needsBackup := recording.status == "recording" || (recording.status == "draft" && !recording.backupSaved)
		if !needsBackup && !quitOnClose {
			inputEventsMu.Unlock()
			return
		}
		event.Cancel()
		if pending {
			inputEventsMu.Unlock()
			return
		}
		pending = true
		inputEventsMu.Unlock()
		go func() {
			defer func() {
				inputEventsMu.Lock()
				pending = false
				inputEventsMu.Unlock()
			}()
			if needsBackup {
				state, err := (&Keyboard{}).StopRecording()
				if err != nil || (state.Status == "draft" && !state.BackupSaved) {
					if state.Error == "" {
						state.Error = "录制草稿尚未备份，请先保存 MIDI 后退出"
					}
					App.Event.Emit("recordingExitBlocked", state)
					return
				}
			}
			if quitOnClose {
				(&Keyboard{}).Quit()
			} else {
				PianoWin.Close()
			}
		}()
	})
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
		Title:                  appDisplayName + " · 设置",
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

// OpenMidiCenter preserves the old entry point while MIDI controls now live in
// the main window. No second window or playback-reset lifecycle is created.
func (k *Keyboard) OpenMidiCenter() error {
	if App == nil || PianoWin == nil {
		return fmt.Errorf("主窗口尚未就绪")
	}
	if _, err := k.SetMainDisplayMode(mainDisplayWaterfall); err != nil {
		return fmt.Errorf("打开曲目播放面板失败：%w", err)
	}
	PianoWin.UnMinimise()
	PianoWin.Show()
	PianoWin.Focus()
	App.Event.Emit("mainMidiModeRequested", map[string]string{"mode": "play"})
	return nil
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
	// Retained for older frontend clients; MIDI is part of the main window.
	return false
}
