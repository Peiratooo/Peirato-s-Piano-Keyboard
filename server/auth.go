package main

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var validUsername = regexp.MustCompile(`^[a-zA-Z0-9_]{3,24}$`)

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) bootstrapAdmin() error {
	name := config("ADMIN_USERNAME", "")
	pass := config("ADMIN_PASSWORD", "")
	if name == "" || pass == "" {
		return nil
	}
	if !validUsername.MatchString(name) || len(pass) < 12 {
		return &configError{"ADMIN_USERNAME/password invalid (minimum 12 characters)"}
	}
	var u User
	err := s.db.Where("username = ?", name).First(&u).Error
	if err == nil {
		if u.Role != "admin" {
			return &configError{"admin bootstrap username already belongs to a regular account"}
		}
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	h, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.db.Create(&User{Username: name, Nickname: name, PasswordHash: string(h), Role: "admin"}).Error
}

type configError struct{ message string }

func (e *configError) Error() string { return e.message }
func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	if !s.limiter.allow("register:"+clientIP(r), 8, 15*time.Minute) {
		fail(w, 429, "操作过于频繁")
		return
	}
	var c credentials
	if !decode(w, r, &c) {
		return
	}
	c.Username = strings.TrimSpace(c.Username)
	if !validUsername.MatchString(c.Username) || len(c.Password) < 10 || len(c.Password) > 72 {
		fail(w, 400, "账号仅支持 3-24 位字母、数字、下划线；密码为 10-72 位")
		return
	}
	if c.Username == config("ADMIN_USERNAME", "") {
		fail(w, 409, "账号名不可用")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(w, 500, "无法创建账号")
		return
	}
	u := User{Username: c.Username, Nickname: c.Username, PasswordHash: string(hash), Role: "user"}
	if err = s.db.Create(&u).Error; err != nil {
		fail(w, 409, "账号名已存在")
		return
	}
	s.issueToken(w, u)
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if !s.limiter.allow("login:"+clientIP(r), 30, 15*time.Minute) {
		fail(w, 429, "登录次数过多")
		return
	}
	var c credentials
	if !decode(w, r, &c) {
		return
	}
	var u User
	if err := s.db.Where("username = ?", strings.TrimSpace(c.Username)).First(&u).Error; err != nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(c.Password)) != nil {
		fail(w, 401, "账号或密码错误")
		return
	}
	s.issueToken(w, u)
}
func (s *Server) issueToken(w http.ResponseWriter, u User) {
	token, expires, err := s.newSession(u)
	if err != nil {
		fail(w, 500, "登录会话创建失败")
		return
	}
	writeJSON(w, 200, map[string]any{"token": token, "user": u, "expiresAt": expires})
}
func (s *Server) newSession(u User) (string, time.Time, error) {
	token := secret(32)
	ses := Session{UserID: u.ID, TokenHash: hashToken(token), ExpiresAt: time.Now().Add(7 * 24 * time.Hour)}
	if err := s.db.Create(&ses).Error; err != nil {
		return "", time.Time{}, err
	}
	return token, ses.ExpiresAt, nil
}
func (s *Server) userByToken(token string) (*User, error) {
	if len(token) != 64 {
		return nil, gorm.ErrRecordNotFound
	}
	var ses Session
	if err := s.db.Where("token_hash = ? AND expires_at > ?", hashToken(token), time.Now()).First(&ses).Error; err != nil {
		return nil, err
	}
	var u User
	if err := s.db.First(&u, ses.UserID).Error; err != nil {
		return nil, err
	}
	return &u, nil
}
func (s *Server) optionalUser(r *http.Request) *User {
	head := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(head, "Bearer ") {
		return nil
	}
	u, _ := s.userByToken(strings.TrimSpace(strings.TrimPrefix(head, "Bearer ")))
	return u
}
func (s *Server) requireUser(w http.ResponseWriter, r *http.Request) *User {
	u := s.optionalUser(r)
	if u == nil {
		fail(w, 401, "请先登录")
		return nil
	}
	return u
}
func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) *User {
	u := s.requireUser(w, r)
	if u == nil {
		return nil
	}
	if u.Role != "admin" {
		fail(w, 403, "无管理权限")
		return nil
	}
	return u
}
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u != nil {
		writeJSON(w, 200, u)
	}
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if len(token) == 64 {
		s.db.Where("token_hash = ?", hashToken(token)).Delete(&Session{})
	}
	w.WriteHeader(204)
}
