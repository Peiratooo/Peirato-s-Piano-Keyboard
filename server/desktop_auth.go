package main

import (
	"net/http"
	"net/url"
	"time"
)

type DesktopGrant struct {
	Code      string
	ExpiresAt time.Time
	UserID    uint
	Denied    bool
}

func (s *Server) startDesktopAuth(w http.ResponseWriter, r *http.Request) {
	if !s.limiter.allow("desktop:"+clientIP(r), 20, 15*time.Minute) {
		fail(w, 429, "请稍后重新登录")
		return
	}
	device := secret(32)
	code := secret(4)
	expires := time.Now().Add(5 * time.Minute)
	s.desktopMu.Lock()
	for key, g := range s.desktopGrants {
		if time.Now().After(g.ExpiresAt) {
			delete(s.desktopGrants, key)
		}
	}
	s.desktopGrants[hashToken(device)] = &DesktopGrant{Code: code, ExpiresAt: expires}
	s.desktopMu.Unlock()
	target, err := url.Parse(config("TOY_URL", "http://127.0.0.1:5173/"))
	if err != nil {
		fail(w, 500, "Toy 地址未配置")
		return
	}
	query := target.Query()
	query.Set("desktopCode", code)
	target.RawQuery = query.Encode()
	writeJSON(w, 200, map[string]any{"deviceCode": device, "userCode": code, "verificationUrl": target.String(), "expiresAt": expires, "interval": 5})
}
func (s *Server) confirmDesktopAuth(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	var input struct {
		UserCode string `json:"userCode"`
		Approve  bool   `json:"approve"`
	}
	if !decode(w, r, &input) {
		return
	}
	s.desktopMu.Lock()
	defer s.desktopMu.Unlock()
	for _, g := range s.desktopGrants {
		if g.Code == input.UserCode && time.Now().Before(g.ExpiresAt) && g.UserID == 0 && !g.Denied {
			if input.Approve {
				g.UserID = u.ID
			} else {
				g.Denied = true
			}
			w.WriteHeader(204)
			return
		}
	}
	fail(w, 410, "授权已过期或已处理，请在软件中重新发起")
}
func (s *Server) pollDesktopAuth(w http.ResponseWriter, r *http.Request) {
	var input struct {
		DeviceCode string `json:"deviceCode"`
	}
	if !decode(w, r, &input) {
		return
	}
	if len(input.DeviceCode) != 64 {
		fail(w, 400, "无效授权请求")
		return
	}
	key := hashToken(input.DeviceCode)
	s.desktopMu.Lock()
	defer s.desktopMu.Unlock()
	g := s.desktopGrants[key]
	if g == nil || time.Now().After(g.ExpiresAt) {
		delete(s.desktopGrants, key)
		fail(w, 410, "授权已过期，请重新登录")
		return
	}
	if g.Denied {
		delete(s.desktopGrants, key)
		fail(w, 403, "已取消桌面登录")
		return
	}
	if g.UserID == 0 {
		writeJSON(w, 202, map[string]any{"pending": true})
		return
	}
	var u User
	if err := s.db.First(&u, g.UserID).Error; err != nil {
		fail(w, 401, "账号不可用")
		return
	}
	token, expires, err := s.newSession(u)
	if err != nil {
		fail(w, 500, "无法创建会话")
		return
	}
	delete(s.desktopGrants, key)
	writeJSON(w, 200, map[string]any{"token": token, "user": u, "expiresAt": expires})
}
