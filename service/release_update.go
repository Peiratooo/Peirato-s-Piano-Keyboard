package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	simpleupdater "github.com/Peiratooo/simple-updater"
)

func releaseAppID() string { return "com.peirato.piano" }

func releaseURL(endpoint string) string { return updateBaseURL + "/release/" + endpoint }

// Protected by updateMu; a 204 confirms disk files, not the running version.
var filesCurrentReleaseID int

func (k *Keyboard) CheckUpdate() (info UpdateInfo, err error) {
	if !updateMu.TryLock() {
		state := k.GetUpdateState()
		return UpdateInfo{Version: state.Version, ReleaseID: state.ReleaseID, Available: state.Available, Supported: state.Supported}, nil
	}
	defer updateMu.Unlock()
	if preparedUpdate.Path != "" {
		updateProgress(func(s *UpdateState) { s.Phase = "ready"; s.Error = "" })
		return preparedUpdate.Info, nil
	}
	updateProgress(func(s *UpdateState) { s.Phase = "checking"; s.Error = "" })
	info, err = checkReleaseUpdate()
	if err != nil {
		updateFailed(err)
		return info, err
	}
	_, helperErr := installedUpdaterPath()
	info.Supported = helperErr == nil
	filesCurrent := info.Available && info.ReleaseID == filesCurrentReleaseID
	if filesCurrent {
		info.Available = false
	}
	updateProgress(func(s *UpdateState) {
		s.Version, s.ReleaseID, s.Available, s.Supported = info.Version, info.ReleaseID, info.Available, info.Supported
		s.Phase = "latest"
		if info.Available {
			s.Phase = "available"
		} else if filesCurrent {
			s.Phase = "files-current"
		}
	})
	return info, nil
}

func checkReleaseUpdate() (UpdateInfo, error) {
	info := UpdateInfo{Supported: true}
	params := url.Values{"system": {runtime.GOOS}, "app_id": {releaseAppID()}}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseURL("get_info")+"?"+params.Encode(), nil)
	if err != nil {
		return info, err
	}
	resp, err := updateHTTP.Do(request)
	if err != nil {
		return info, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 || resp.StatusCode == 410 {
		return info, fmt.Errorf("暂时没有可用的更新信息，请稍后重试（HTTP %d）", resp.StatusCode)
	}
	if resp.StatusCode != 200 {
		return info, fmt.Errorf("release 服务返回 HTTP %d", resp.StatusCode)
	}
	var result struct {
		Code int                   `json:"code"`
		Data simpleupdater.Product `json:"data"`
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&result); err != nil {
		return info, err
	}
	p := result.Data
	if result.Code != 100 || p.ID <= 0 || p.Version == "" || p.System != runtime.GOOS || strings.ToLower(strings.Trim(p.AppID, "{}")) != releaseAppID() || len(p.Files) == 0 {
		return info, fmt.Errorf("release 版本信息无效")
	}
	if _, err = simpleupdater.GenerateUpdateScript(runtime.GOOS, p.Files); err != nil {
		return info, err
	}
	info.Version, info.ReleaseID, info.Files = p.Version, p.ID, p.Files
	info.Available = newerVersion(p.Version, GetUserConfig().Version)
	return info, nil
}

// Hash only release-owned files, not user data, cached packages or other apps.
func localReleaseManifest(root string, manifest []simpleupdater.File) ([]simpleupdater.File, error) {
	result := make([]simpleupdater.File, 0, len(manifest))
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	for _, expected := range manifest {
		path := filepath.Join(root, filepath.FromSlash(expected.Path))
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		parent, err := filepath.EvalSymlinks(filepath.Dir(path))
		if err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(root, parent)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("更新文件父目录越界: %s", expected.Path)
		}
		file := simpleupdater.File{Path: expected.Path, Mode: uint32(info.Mode().Perm()), Size: uint64(info.Size()), Type: simpleupdater.FileTypeRegular}
		if info.Mode()&os.ModeSymlink != 0 {
			file.Type = simpleupdater.FileTypeSymlink
			file.LinkTarget, err = os.Readlink(path)
			if err != nil {
				return nil, err
			}
		} else {
			if !info.Mode().IsRegular() {
				continue
			}
			src, err := os.Open(path)
			if err != nil {
				return nil, err
			}
			hash := sha256.New()
			_, err = io.Copy(hash, src)
			closeErr := src.Close()
			if err != nil {
				return nil, err
			}
			if closeErr != nil {
				return nil, closeErr
			}
			file.SHA256 = hex.EncodeToString(hash.Sum(nil))
		}
		result = append(result, file)
	}
	return result, nil
}

func (k *Keyboard) downloadRelease(info UpdateInfo, exe string) error {
	root, _, _ := updateTarget(exe)
	files, err := localReleaseManifest(root, info.Files)
	if err != nil {
		return err
	}
	request := struct {
		System    string               `json:"system"`
		AppID     string               `json:"app_id"`
		ReleaseID int                  `json:"release_id"`
		Files     []simpleupdater.File `json:"files"`
	}{runtime.GOOS, releaseAppID(), info.ReleaseID, files}
	body, err := json.Marshal(request)
	if err != nil {
		return err
	}
	resp, err := updateHTTP.Post(releaseURL("download"), "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		clearPreparedUpdate()
		filesCurrentReleaseID = info.ReleaseID
		updateProgress(func(s *UpdateState) {
			s.Phase, s.Error = "files-current", ""
			s.Version, s.ReleaseID = info.Version, info.ReleaseID
			s.Available = false
			s.Received, s.Total = 0, 0
		})
		return nil
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("release 下载返回 HTTP %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Release-ID") != strconv.Itoa(info.ReleaseID) {
		return fmt.Errorf("更新版本不一致")
	}
	digest := resp.Header.Get("X-Patch-SHA256")
	expected, err := hex.DecodeString(digest)
	if err != nil || len(expected) != sha256.Size {
		return fmt.Errorf("差分包缺少 SHA-256 校验值")
	}
	archive, err := os.CreateTemp("", "piano-release-*.tar.gz")
	if err != nil {
		return err
	}
	keep := false
	defer func() {
		if !keep {
			_ = os.Remove(archive.Name())
		}
	}()
	defer archive.Close()
	hash := sha256.New()
	progress := &updateDownloadProgress{Total: resp.ContentLength}
	updateProgress(func(s *UpdateState) { s.Phase = "downloading"; s.Total = resp.ContentLength })
	n, err := io.Copy(io.MultiWriter(archive, hash, progress), io.LimitReader(resp.Body, maxUpdateBytes+1))
	if err != nil {
		return err
	}
	if n > maxUpdateBytes || !bytes.Equal(hash.Sum(nil), expected) {
		return fmt.Errorf("差分包校验失败")
	}
	updateProgress(func(s *UpdateState) { s.Phase = "verifying"; s.Received = n })
	if err = archive.Sync(); err != nil {
		return err
	}
	if err = archive.Close(); err != nil {
		return err
	}
	preparedUpdate.Path, preparedUpdate.SHA256, preparedUpdate.Exe = archive.Name(), hex.EncodeToString(expected), exe
	preparedUpdate.Info = info
	keep = true
	updateProgress(func(s *UpdateState) { s.Phase = "ready"; s.Received, s.Total = n, n })
	return nil
}

type updateDownloadProgress struct {
	Total, Received int64
	last            time.Time
}

func (p *updateDownloadProgress) Write(data []byte) (int, error) {
	p.Received += int64(len(data))
	if time.Since(p.last) >= 100*time.Millisecond {
		updateProgress(func(s *UpdateState) { s.Received, s.Total = p.Received, p.Total })
		p.last = time.Now()
	}
	return len(data), nil
}

// Use the precompiled updater shipped beside the application executable.
func installedUpdaterPath() (string, error) {
	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		return "", fmt.Errorf("当前平台不支持自动更新")
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	name := "updater"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(filepath.Dir(exe), name)
	info, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("找不到随软件安装的 updater: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return "", fmt.Errorf("updater 文件无效: %s", path)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0111 == 0 {
		return "", fmt.Errorf("updater 没有执行权限: %s", path)
	}
	return path, nil
}
