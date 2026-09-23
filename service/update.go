package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	simpleupdater "github.com/Peiratooo/simple-updater"
)

const updateBaseURL = "https://api.peirato.com"
const maxUpdateBytes = 512 << 20

var updateMu sync.Mutex
var updateHTTP = &http.Client{Timeout: 5 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return fmt.Errorf("更新下载重定向过多")
	}
	if req.URL.Scheme != "https" {
		return fmt.Errorf("更新下载必须使用 HTTPS")
	}
	return nil
}}

type UpdateInfo struct {
	ReleaseID int                  `json:"releaseId"`
	Files     []simpleupdater.File `json:"-"`
	Version   string               `json:"version"`
	Available bool                 `json:"available"`
	Supported bool                 `json:"supported"`
}

// Releases currently use numeric dotted versions. Unknown formats fail closed
// instead of silently downgrading a local build.
func newerVersion(remote, local string) bool {
	parse := func(s string) ([]int, bool) {
		parts := strings.Split(strings.TrimPrefix(s, "v"), ".")
		if len(parts) < 2 || len(parts) > 4 {
			return nil, false
		}
		values := make([]int, 4)
		for i, part := range parts {
			for _, digit := range part {
				if digit < '0' || digit > '9' {
					return nil, false
				}
			}
			n, err := strconv.Atoi(part)
			if err != nil || n < 0 {
				return nil, false
			}
			values[i] = n
		}
		return values, true
	}
	a, okA := parse(remote)
	b, okB := parse(local)
	if !okA || !okB {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}

func updateTarget(exe string) (root, target, restart string) {
	if runtime.GOOS == "darwin" {
		for dir := filepath.Dir(exe); dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			if strings.HasSuffix(strings.ToLower(dir), ".app") {
				rel, _ := filepath.Rel(dir, exe)
				return dir, filepath.ToSlash(rel), dir
			}
		}
	}
	return filepath.Dir(exe), filepath.Base(exe), exe
}

// UpdateState is shared by every app window and survives page navigation.
type UpdateState struct {
	Revision  uint64 `json:"revision"`
	Phase     string `json:"phase"`
	Version   string `json:"version"`
	ReleaseID int    `json:"releaseId"`
	Available bool   `json:"available"`
	Supported bool   `json:"supported"`
	Received  int64  `json:"received"`
	Total     int64  `json:"total"`
	Error     string `json:"error"`
}

var updateStateMu sync.RWMutex
var currentUpdateState = UpdateState{Phase: "idle"}
var preparedUpdate struct {
	Path, SHA256, Exe string
	Info              UpdateInfo
}

func (k *Keyboard) GetUpdateState() UpdateState {
	updateStateMu.RLock()
	defer updateStateMu.RUnlock()
	return currentUpdateState
}
func updateProgress(change func(*UpdateState)) {
	updateStateMu.Lock()
	change(&currentUpdateState)
	currentUpdateState.Revision++
	state := currentUpdateState
	updateStateMu.Unlock()
	if App != nil {
		App.Event.Emit("updateState", state)
	}
}
func updateFailed(err error) {
	updateProgress(func(s *UpdateState) { s.Phase = "error"; s.Error = err.Error() })
}
func clearPreparedUpdate() {
	if preparedUpdate.Path != "" {
		_ = os.Remove(preparedUpdate.Path)
	}
	preparedUpdate.Path, preparedUpdate.SHA256, preparedUpdate.Exe = "", "", ""
}
func cleanupUpdate() {
	if updateMu.TryLock() {
		defer updateMu.Unlock()
		clearPreparedUpdate()
	}
}

// Download and verify without exiting. Installation requires a separate user action.
func (k *Keyboard) DownloadUpdate(releaseID int) (err error) {
	if !updateMu.TryLock() {
		return fmt.Errorf("更新正在进行中")
	}
	defer updateMu.Unlock()
	defer func() {
		if err != nil {
			updateFailed(err)
		}
	}()
	updateProgress(func(s *UpdateState) {
		s.Phase, s.Error = "preparing", ""
		s.Received, s.Total = 0, 0
	})
	if _, err = installedUpdaterPath(); err != nil {
		return err
	}
	info, err := checkReleaseUpdate()
	if err != nil {
		return err
	}
	if !info.Available || info.ReleaseID != releaseID {
		return fmt.Errorf("版本信息已变化，请重新检查更新")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	clearPreparedUpdate()
	updateProgress(func(s *UpdateState) {
		s.Phase, s.Error, s.Version = "preparing", "", info.Version
		s.ReleaseID, s.Available, s.Supported = info.ReleaseID, true, true
		s.Received, s.Total = 0, 0
	})
	return k.downloadRelease(info, exe)
}

// Install only the already downloaded version, after explicit confirmation.
func (k *Keyboard) InstallUpdate() (err error) {
	if !updateMu.TryLock() {
		return fmt.Errorf("更新正在进行中")
	}
	defer updateMu.Unlock()
	defer func() {
		if err != nil {
			updateFailed(err)
		}
	}()
	if preparedUpdate.Path == "" {
		return fmt.Errorf("请先下载更新")
	}
	updaterPath, err := installedUpdaterPath()
	if err != nil {
		return err
	}
	installRoot, _, restart := updateTarget(preparedUpdate.Exe)
	probe, err := os.CreateTemp(installRoot, ".piano-write-check-*")
	if err != nil {
		return fmt.Errorf("安装目录不可写，请以管理员身份运行后更新，或使用完整安装包")
	}
	_ = probe.Close()
	_ = os.Remove(probe.Name())
	updateProgress(func(s *UpdateState) { s.Phase = "installing"; s.Error = "" })
	archive, err := os.Open(preparedUpdate.Path)
	if err != nil {
		clearPreparedUpdate()
		return fmt.Errorf("更新文件已失效，请重新下载")
	}
	defer archive.Close()
	hash := sha256.New()
	if _, err = io.Copy(hash, archive); err != nil {
		return err
	}
	if hex.EncodeToString(hash.Sum(nil)) != preparedUpdate.SHA256 {
		_ = archive.Close()
		clearPreparedUpdate()
		return fmt.Errorf("更新文件校验失败，请重新下载")
	}
	if _, err = archive.Seek(0, io.SeekStart); err != nil {
		return err
	}
	_, err = simpleupdater.StartUpdater(simpleupdater.UpdaterLaunchOptions{
		UpdaterPath: updaterPath, PID: os.Getpid(), InstallRoot: installRoot, RestartPath: restart, Archive: archive,
	})
	if err != nil {
		return err
	}
	_ = archive.Close()
	clearPreparedUpdate()
	k.Quit()
	return nil
}
