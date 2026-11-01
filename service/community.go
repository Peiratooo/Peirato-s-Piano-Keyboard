package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Override these URLs with runtime environment variables or build -ldflags.
var CommunityAPIURL = "http://127.0.0.1:8080"
var communityHTTP = &http.Client{Timeout: 15 * time.Second}
var communityMu sync.Mutex
var communityAccount communityCredentials
var communityLoaded bool
var desktopDeviceCode string
var desktopRemember bool
var communityPracticeID uint

type CommunityUser struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl"`
	Role      string `json:"role"`
}
type communityCredentials struct {
	Token     string         `json:"token"`
	User      *CommunityUser `json:"user"`
	ExpiresAt time.Time      `json:"expiresAt"`
}
type CommunitySession struct {
	API     string         `json:"api"`
	User    *CommunityUser `json:"user"`
	Pending bool           `json:"pending"`
	Offline bool           `json:"offline"`
}
type CommunityDesktopGrant struct {
	VerificationURL string    `json:"verificationUrl"`
	UserCode        string    `json:"userCode"`
	ExpiresAt       time.Time `json:"expiresAt"`
	Interval        int       `json:"interval"`
}

func communityBase() string {
	if value := os.Getenv("PP_COMMUNITY_API_URL"); value != "" {
		return strings.TrimRight(value, "/")
	}
	return strings.TrimRight(CommunityAPIURL, "/")
}
func communitySessionPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configDirectoryName, "community-session.json"), nil
}
func communityCurrentUserID() uint {
	communityMu.Lock()
	defer communityMu.Unlock()
	if communityAccount.User == nil {
		return 0
	}
	return communityAccount.User.ID
}
func communityToken() string {
	communityMu.Lock()
	defer communityMu.Unlock()
	return communityAccount.Token
}
func communityFetch(path, method string, body io.Reader, contentType, token string) ([]byte, int, error) {
	req, err := http.NewRequest(method, communityBase()+path, body)
	if err != nil {
		return nil, 0, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	res, err := communityHTTP.Do(req)
	if err != nil {
		return nil, 0, errors.New("社区服务连接失败，请检查网络或服务地址")
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, (9<<20)+1))
	if err != nil {
		return nil, res.StatusCode, err
	}
	if len(data) > 9<<20 {
		return nil, res.StatusCode, errors.New("社区响应过大")
	}
	if res.StatusCode >= 400 {
		var message struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(data, &message) == nil && message.Error != "" {
			return nil, res.StatusCode, errors.New(message.Error)
		}
		return nil, res.StatusCode, fmt.Errorf("社区请求失败（%d）", res.StatusCode)
	}
	return data, res.StatusCode, nil
}
func storeCommunityCredentials(account communityCredentials, remember bool) error {
	path, err := communitySessionPath()
	if err != nil {
		return err
	}
	if remember {
		data, err := json.Marshal(account)
		if err != nil {
			return err
		}
		if err = writeAtomicFile(path, data, 0600); err != nil {
			return err
		}
	} else if err = os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	communityMu.Lock()
	communityAccount = account
	communityLoaded = true
	desktopDeviceCode = ""
	communityMu.Unlock()
	return nil
}
func (k *Keyboard) GetCommunitySession() (CommunitySession, error) {
	communityMu.Lock()
	if !communityLoaded {
		path, err := communitySessionPath()
		if err != nil {
			communityMu.Unlock()
			return CommunitySession{}, err
		}
		data, err := os.ReadFile(path)
		if err == nil {
			if err = json.Unmarshal(data, &communityAccount); err != nil {
				communityMu.Unlock()
				return CommunitySession{}, errors.New("本地登录信息无法读取，请重新登录")
			}
		} else if !os.IsNotExist(err) {
			communityMu.Unlock()
			return CommunitySession{}, err
		}
		communityLoaded = true
	}
	saved := communityAccount
	communityMu.Unlock()
	out := CommunitySession{API: communityBase()}
	if saved.Token == "" {
		return out, nil
	}
	if time.Now().After(saved.ExpiresAt) {
		return out, storeCommunityCredentials(communityCredentials{}, false)
	}
	data, status, err := communityFetch("/api/auth/me", "GET", nil, "", saved.Token)
	if status == 401 {
		return out, storeCommunityCredentials(communityCredentials{}, false)
	}
	if err != nil {
		out.User = saved.User
		out.Offline = true
		return out, nil
	}
	var user CommunityUser
	if err = json.Unmarshal(data, &user); err != nil {
		return out, err
	}
	communityMu.Lock()
	communityAccount.User = &user
	communityMu.Unlock()
	out.User = &user
	return out, nil
}
func (k *Keyboard) CommunityLogin(action, username, password string, remember bool) (CommunitySession, error) {
	if action != "login" && action != "register" {
		return CommunitySession{}, errors.New("无效登录操作")
	}
	payload, _ := json.Marshal(map[string]string{"username": username, "password": password})
	data, _, err := communityFetch("/api/auth/"+action, "POST", bytes.NewReader(payload), "application/json", "")
	if err != nil {
		return CommunitySession{}, err
	}
	var account communityCredentials
	if err = json.Unmarshal(data, &account); err != nil {
		return CommunitySession{}, err
	}
	if err = storeCommunityCredentials(account, remember); err != nil {
		return CommunitySession{}, err
	}
	return CommunitySession{API: communityBase(), User: account.User}, nil
}
func (k *Keyboard) CommunityLogout() error {
	item, _ := FindUserMidiByID(MidiPlayer.State().MidiID)
	token := communityToken()
	var remoteErr error
	if token != "" {
		_, status, err := communityFetch("/api/auth/logout", "POST", nil, "", token)
		if err != nil && status != 401 {
			remoteErr = err
		}
	}
	if err := storeCommunityCredentials(communityCredentials{}, false); err != nil {
		return err
	}
	if item.CommunityAccountID != 0 {
		MidiPlayer.Stop()
	}
	return remoteErr
}

// JSON requests stay in Go so desktop bearer sessions never enter WebView storage.
func (k *Keyboard) CommunityRequest(path, method, payload string) (string, error) {
	parsed, err := url.Parse(path)
	if err != nil || parsed.IsAbs() || !strings.HasPrefix(parsed.Path, "/api/") || strings.HasPrefix(parsed.Path, "/api/auth/desktop") || parsed.Path == "/api/auth/login" || parsed.Path == "/api/auth/register" {
		return "", errors.New("无效社区请求")
	}
	if method != "GET" && method != "POST" && method != "PUT" && method != "PATCH" && method != "DELETE" {
		return "", errors.New("无效请求方法")
	}
	data, status, err := communityFetch(path, method, strings.NewReader(payload), "application/json", communityToken())
	if status == 401 {
		_ = storeCommunityCredentials(communityCredentials{}, false)
		if App != nil {
			App.Event.Emit("communitySessionExpired", nil)
		}
	}
	if err != nil {
		return "", err
	}
	if parsed.Path == "/api/practice/start" {
		var record struct {
			ID uint `json:"id"`
		}
		if err = json.Unmarshal(data, &record); err != nil {
			return "", err
		}
		communityMu.Lock()
		communityPracticeID = record.ID
		communityMu.Unlock()
	}
	if parsed.Path == "/api/practice/finish" {
		communityMu.Lock()
		communityPracticeID = 0
		communityMu.Unlock()
	}
	if parsed.Path == "/api/auth/me" && method == "PATCH" {
		var user CommunityUser
		if err = json.Unmarshal(data, &user); err != nil {
			return "", err
		}
		communityMu.Lock()
		communityAccount.User = &user
		account := communityAccount
		communityMu.Unlock()
		path, _ := communitySessionPath()
		if _, err := os.Stat(path); err == nil {
			if err = storeCommunityCredentials(account, true); err != nil {
				return "", err
			}
		}
	}
	if len(data) == 0 {
		return "null", nil
	}
	return string(data), nil
}
func (k *Keyboard) BeginCommunityToyLogin(remember bool) (CommunityDesktopGrant, error) {
	data, _, err := communityFetch("/api/auth/desktop/start", "POST", strings.NewReader("{}"), "application/json", "")
	if err != nil {
		return CommunityDesktopGrant{}, err
	}
	var response struct {
		CommunityDesktopGrant
		DeviceCode string `json:"deviceCode"`
	}
	if err = json.Unmarshal(data, &response); err != nil {
		return CommunityDesktopGrant{}, err
	}
	communityMu.Lock()
	desktopDeviceCode = response.DeviceCode
	desktopRemember = remember
	communityMu.Unlock()
	return response.CommunityDesktopGrant, nil
}
func (k *Keyboard) PollCommunityToyLogin() (CommunitySession, error) {
	communityMu.Lock()
	code := desktopDeviceCode
	remember := desktopRemember
	communityMu.Unlock()
	if code == "" {
		return CommunitySession{}, errors.New("桌面授权已取消")
	}
	payload, _ := json.Marshal(map[string]string{"deviceCode": code})
	data, status, err := communityFetch("/api/auth/desktop/poll", "POST", bytes.NewReader(payload), "application/json", "")
	if err != nil {
		return CommunitySession{}, err
	}
	if status == 202 {
		return CommunitySession{API: communityBase(), Pending: true}, nil
	}
	var account communityCredentials
	if err = json.Unmarshal(data, &account); err != nil {
		return CommunitySession{}, err
	}
	if err = storeCommunityCredentials(account, remember); err != nil {
		return CommunitySession{}, err
	}
	if PianoWin != nil {
		PianoWin.UnMinimise()
		PianoWin.Show()
		PianoWin.Focus()
	}
	return CommunitySession{API: communityBase(), User: account.User}, nil
}
func (k *Keyboard) CancelCommunityToyLogin() {
	communityMu.Lock()
	desktopDeviceCode = ""
	communityMu.Unlock()
}

type CommunityLocalFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

func (k *Keyboard) ChooseCommunityMIDI() (*CommunityLocalFile, error) {
	path, err := App.Dialog.OpenFile().SetTitle("选择要上传的 MIDI").AddFilter("MIDI", "*.mid;*.midi").PromptForSingleSelection()
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > 8<<20 {
		return nil, errors.New("MIDI 文件不得超过 8MB")
	}
	return &CommunityLocalFile{Name: filepath.Base(path), Path: path, Size: info.Size()}, nil
}
func communityUpload(path, endpoint string, fields map[string]any, max int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > max {
		return nil, errors.New("文件超出大小限制")
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return nil, err
	}
	if _, err = io.Copy(part, io.LimitReader(file, max+1)); err != nil {
		return nil, err
	}
	for key, value := range fields {
		if err = writer.WriteField(key, fmt.Sprint(value)); err != nil {
			return nil, err
		}
	}
	if err = writer.Close(); err != nil {
		return nil, err
	}
	data, _, err := communityFetch(endpoint, "POST", &body, writer.FormDataContentType(), communityToken())
	return data, err
}
func (k *Keyboard) UploadCommunityMIDI(path, fieldsJSON string) (string, error) {
	var fields map[string]any
	if err := json.Unmarshal([]byte(fieldsJSON), &fields); err != nil {
		return "", err
	}
	data, err := communityUpload(path, "/api/midis", fields, 8<<20)
	return string(data), err
}
func (k *Keyboard) SetCommunityAvatar() (*CommunityUser, error) {
	path, err := App.Dialog.OpenFile().SetTitle("设置头像").AddFilter("PNG / JPEG", "*.png;*.jpg;*.jpeg").PromptForSingleSelection()
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil
	}
	data, err := communityUpload(path, "/api/auth/avatar", nil, 2<<20)
	if err != nil {
		return nil, err
	}
	var user CommunityUser
	if err = json.Unmarshal(data, &user); err != nil {
		return nil, err
	}
	communityMu.Lock()
	communityAccount.User = &user
	account := communityAccount
	communityMu.Unlock()
	sessionPath, err := communitySessionPath()
	if err != nil {
		return nil, err
	}
	if _, err = os.Stat(sessionPath); err == nil {
		if err = storeCommunityCredentials(account, true); err != nil {
			return nil, err
		}
	}
	return &user, nil
}

func finishCommunityPractice() {
	communityMu.Lock()
	id := communityPracticeID
	token := communityAccount.Token
	communityPracticeID = 0
	communityMu.Unlock()
	if id == 0 || token == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	payload, _ := json.Marshal(map[string]uint{"sessionId": id})
	req, err := http.NewRequestWithContext(ctx, "POST", communityBase()+"/api/practice/finish", bytes.NewReader(payload))
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	if res, err := communityHTTP.Do(req); err == nil {
		res.Body.Close()
	}
}
