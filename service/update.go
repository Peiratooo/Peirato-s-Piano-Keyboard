package service

import (
	"fmt"
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

func (k *Keyboard) InstallUpdate() error {
	if !updateMu.TryLock() {
		return fmt.Errorf("更新正在进行中")
	}
	defer updateMu.Unlock()
	info, err := k.CheckUpdate()
	if err != nil {
		return err
	}
	if !info.Supported {
		return fmt.Errorf("未找到可用的 updater，请重新安装完整软件包")
	}
	if !info.Available {
		return fmt.Errorf("没有可安装的更新")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	installRoot, _, _ := updateTarget(exe)
	probe, err := os.CreateTemp(installRoot, ".piano-write-check-*")
	if err != nil {
		return fmt.Errorf("安装目录不可写，请以管理员身份运行后更新，或使用安装包: %w", err)
	}
	_ = probe.Close()
	_ = os.Remove(probe.Name())
	return k.installRelease(info, exe)
}
