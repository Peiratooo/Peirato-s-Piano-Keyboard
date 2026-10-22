package service

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

const (
	mainDisplayWaterfall = "waterfall"
	mainDisplaySimple    = "simple"
)

type MainWindowLayout struct {
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	Maximized bool `json:"maximized"`
}

type MainWindowLayouts struct {
	Waterfall MainWindowLayout `json:"waterfall"`
	Simple    MainWindowLayout `json:"simple"`
}

type MainWindowPresentation struct {
	Mode      string `json:"mode"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Maximized bool   `json:"maximized"`
}

func defaultMainWindowLayouts() MainWindowLayouts {
	return MainWindowLayouts{
		Waterfall: MainWindowLayout{Width: 1280, Height: 800},
		Simple:    MainWindowLayout{Width: 1280, Height: 170},
	}
}

func normalizeMainWindowLayouts(layouts MainWindowLayouts) MainWindowLayouts {
	defaults := defaultMainWindowLayouts()
	for _, entry := range []struct{ value, fallback *MainWindowLayout }{
		{&layouts.Waterfall, &defaults.Waterfall},
		{&layouts.Simple, &defaults.Simple},
	} {
		if entry.value.Width <= 0 || entry.value.Width > 32768 {
			entry.value.Width = entry.fallback.Width
		}
		if entry.value.Height <= 0 || entry.value.Height > 32768 {
			entry.value.Height = entry.fallback.Height
		}
	}
	layouts.Simple.Maximized = false
	return layouts
}

// Native operations are serialized separately from state. Shutdown runs on the
// UI thread and must never wait for a goroutine holding a lock across InvokeSync.
type mainWindowController struct {
	operationMu sync.Mutex
	stateMu     sync.Mutex
	persistMu   sync.Mutex
	mode        string
	layouts     MainWindowLayouts
	initialized bool
	closed      bool
	saveTimer   *time.Timer
}

var mainWindowPresentation = &mainWindowController{mode: mainDisplayWaterfall}

func installMainWindowPresentation() {
	c := mainWindowPresentation
	c.stateMu.Lock()
	c.layouts = GetUserConfig().MainWindowLayouts
	c.stateMu.Unlock()

	PianoWin.OnWindowEvent(events.Common.WindowRuntimeReady, func(_ *application.WindowEvent) {
		if _, err := (&Keyboard{}).GetMainWindowPresentation(); err != nil {
			log.Printf("初始化主窗口失败: %v", err)
			PianoWin.Show()
		}
	})
	for _, event := range []events.WindowEventType{
		events.Common.WindowDidResize, events.Common.WindowMaximise,
		events.Common.WindowUnMaximise, events.Common.WindowRestore,
	} {
		PianoWin.OnWindowEvent(event, func(_ *application.WindowEvent) {
			c.captureNativeChange()
		})
	}
	App.OnShutdown(c.shutdown)
}

func (k *Keyboard) GetMainWindowPresentation() (MainWindowPresentation, error) {
	c := mainWindowPresentation
	c.operationMu.Lock()
	defer c.operationMu.Unlock()
	if err := c.initialize(); err != nil {
		return MainWindowPresentation{}, err
	}
	return c.capture(), nil
}

func (k *Keyboard) SetMainDisplayMode(mode string) (MainWindowPresentation, error) {
	if mode != mainDisplayWaterfall && mode != mainDisplaySimple {
		return MainWindowPresentation{}, fmt.Errorf("未知的主窗口模式：%s", mode)
	}
	c := mainWindowPresentation
	c.operationMu.Lock()
	defer c.operationMu.Unlock()
	if err := c.initialize(); err != nil {
		return MainWindowPresentation{}, err
	}
	current := c.capture()
	if current.Mode == mode {
		return current, nil
	}
	area, err := mainWindowWorkArea()
	if err != nil {
		return current, err
	}
	c.stateMu.Lock()
	layout := c.layouts.Waterfall
	if mode == mainDisplaySimple {
		layout = c.layouts.Simple
	}
	c.stateMu.Unlock()
	c.apply(mode, layout, area, false)
	c.stateMu.Lock()
	c.mode = mode
	c.stateMu.Unlock()
	presentation := c.capture()
	c.scheduleSave()
	App.Event.Emit("mainWindowPresentation", presentation)
	return presentation, nil
}

// Called with operationMu held. Wails' supported Window APIs marshal each
// native call to the UI thread themselves.
func (c *mainWindowController) initialize() error {
	c.stateMu.Lock()
	initialized, closed := c.initialized, c.closed
	layout := c.layouts.Waterfall
	c.stateMu.Unlock()
	if App == nil || PianoWin == nil || closed {
		return fmt.Errorf("主窗口尚未就绪")
	}
	if initialized {
		return nil
	}
	area, err := mainWindowWorkArea()
	if err != nil {
		return err
	}
	c.apply(mainDisplayWaterfall, layout, area, true)
	c.stateMu.Lock()
	c.mode = mainDisplayWaterfall
	c.initialized = true
	c.stateMu.Unlock()
	c.capture()
	PianoWin.Show()
	c.scheduleSave()
	return nil
}

func mainWindowWorkArea() (application.Rect, error) {
	screen, err := PianoWin.GetScreen()
	if err != nil {
		return application.Rect{}, fmt.Errorf("读取主窗口所在屏幕失败：%w", err)
	}
	if screen == nil {
		screen = App.Screen.GetPrimary()
	}
	if screen == nil || screen.WorkArea.Width <= 0 || screen.WorkArea.Height <= 0 {
		return application.Rect{}, fmt.Errorf("无法取得可用屏幕区域")
	}
	return screen.WorkArea, nil
}

func (c *mainWindowController) apply(mode string, layout MainWindowLayout, area application.Rect, center bool) {
	minWidth, minHeight := 960, 600
	if mode == mainDisplaySimple {
		minWidth, minHeight = 800, 100
	}
	minWidth, minHeight = min(minWidth, area.Width), min(minHeight, area.Height)
	width := max(minWidth, min(layout.Width, area.Width))
	height := max(minHeight, min(layout.Height, area.Height))
	PianoWin.UnMinimise()
	PianoWin.UnFullscreen()
	PianoWin.UnMaximise()
	PianoWin.SetMinSize(minWidth, minHeight)
	PianoWin.SetAlwaysOnTop(mode == mainDisplaySimple)
	if mode == mainDisplaySimple {
		PianoWin.SetBackgroundColour(application.NewRGBA(0, 0, 0, 0))
		PianoWin.SetMaximiseButtonState(application.ButtonDisabled)
	} else {
		PianoWin.SetBackgroundColour(application.NewRGBA(28, 28, 28, 255))
		PianoWin.SetMaximiseButtonState(application.ButtonEnabled)
	}
	PianoWin.SetSize(width, height)
	x, y := PianoWin.Position()
	if center {
		x, y = area.X+(area.Width-width)/2, area.Y+(area.Height-height)/2
	}
	PianoWin.SetPosition(max(area.X, min(x, area.X+area.Width-width)), max(area.Y, min(y, area.Y+area.Height-height)))
	// Preserve the normal dimensions while maximized, so restoring and switching
	// back from the keyboard widget does not turn the normal window into a screen.
	c.stateMu.Lock()
	if mode == mainDisplaySimple {
		c.layouts.Simple = MainWindowLayout{Width: width, Height: height}
	} else {
		c.layouts.Waterfall = MainWindowLayout{Width: width, Height: height, Maximized: layout.Maximized}
	}
	c.stateMu.Unlock()
	if mode == mainDisplayWaterfall && layout.Maximized {
		PianoWin.Maximise()
	}
}

// Called with operationMu held; stateMu never spans a native/UI-thread call.
func (c *mainWindowController) capture() MainWindowPresentation {
	width, height := PianoWin.Size()
	maximized := PianoWin.IsMaximised()
	minimized, fullscreen := PianoWin.IsMinimised(), PianoWin.IsFullscreen()
	canStoreSize := !maximized && !minimized && !fullscreen
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	layout := &c.layouts.Waterfall
	if c.mode == mainDisplaySimple {
		layout = &c.layouts.Simple
	}
	if canStoreSize && width > 0 && height > 0 {
		layout.Width, layout.Height = width, height
	}
	if c.mode == mainDisplayWaterfall && !minimized && !fullscreen {
		layout.Maximized = maximized
	}
	return MainWindowPresentation{Mode: c.mode, Width: width, Height: height, Maximized: maximized}
}

func (c *mainWindowController) captureNativeChange() {
	c.operationMu.Lock()
	defer c.operationMu.Unlock()
	c.stateMu.Lock()
	ready := c.initialized && !c.closed
	c.stateMu.Unlock()
	if !ready {
		return
	}
	presentation := c.capture()
	c.scheduleSave()
	App.Event.Emit("mainWindowPresentation", presentation)
}

func (c *mainWindowController) scheduleSave() {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	if c.closed {
		return
	}
	if c.saveTimer != nil {
		c.saveTimer.Stop()
	}
	c.saveTimer = time.AfterFunc(400*time.Millisecond, c.persist)
}

func (c *mainWindowController) persist() {
	c.persistMu.Lock()
	defer c.persistMu.Unlock()
	c.stateMu.Lock()
	layouts := c.layouts
	c.stateMu.Unlock()
	if err := saveMainWindowLayouts(layouts); err != nil {
		log.Printf("主窗口布局未保存: %v", err)
		if App != nil {
			App.Event.Emit("mainWindowPresentationSaveError", map[string]string{"error": err.Error()})
		}
	}
}

func (c *mainWindowController) shutdown() {
	// Do not read native dimensions here: shutdown executes on the UI thread.
	c.stateMu.Lock()
	c.closed = true
	initialized := c.initialized
	if c.saveTimer != nil {
		c.saveTimer.Stop()
	}
	c.stateMu.Unlock()
	if initialized {
		c.persist()
	}
}
