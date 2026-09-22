package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	simpleupdater "github.com/Peiratooo/simple-updater"
)

func releaseAppID() string { return "com.peirato.piano" }

func releaseURL(endpoint string) string { return updateBaseURL + "/release/" + endpoint }

func (k *Keyboard) CheckUpdate() (UpdateInfo, error) {
	if _, err := installedUpdaterPath(); err != nil {
		return UpdateInfo{}, err
	}
	return checkReleaseUpdate()
}

func checkReleaseUpdate() (UpdateInfo, error) {
	info := UpdateInfo{Supported: true}
	params := url.Values{"system": {runtime.GOOS}, "app_id": {releaseAppID()}}
	resp, err := updateHTTP.Get(releaseURL("get_info") + "?" + params.Encode())
	if err != nil {
		return info, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 || resp.StatusCode == 410 {
		return info, fmt.Errorf("release 更新服务尚未部署或没有可用版本（HTTP %d）", resp.StatusCode)
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

func (k *Keyboard) installRelease(info UpdateInfo, exe string) error {
	root, _, restart := updateTarget(exe)
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
	if resp.StatusCode == 204 {
		return fmt.Errorf("文件已是最新版本")
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
	defer os.Remove(archive.Name())
	defer archive.Close()
	hash := sha256.New()
	n, err := io.Copy(io.MultiWriter(archive, hash), io.LimitReader(resp.Body, maxUpdateBytes+1))
	if err != nil {
		return err
	}
	if n > maxUpdateBytes || !bytes.Equal(hash.Sum(nil), expected) {
		return fmt.Errorf("差分包校验失败")
	}
	if _, err = archive.Seek(0, io.SeekStart); err != nil {
		return err
	}
	updaterPath, err := installedUpdaterPath()
	if err != nil {
		return err
	}
	_, err = simpleupdater.StartUpdater(simpleupdater.UpdaterLaunchOptions{UpdaterPath: updaterPath, PID: os.Getpid(), InstallRoot: root, RestartPath: restart, Archive: archive})
	if err != nil {
		return err
	}
	k.Quit()
	return nil
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
