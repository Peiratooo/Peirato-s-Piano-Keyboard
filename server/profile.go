package main

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	var input struct {
		Nickname string `json:"nickname"`
	}
	if !decode(w, r, &input) {
		return
	}
	name := cleanText(input.Nickname, 24)
	if len([]rune(name)) < 2 {
		fail(w, 400, "昵称须为 2–24 个字")
		return
	}
	if err := s.db.Model(u).Update("nickname", name).Error; err != nil {
		fail(w, 500, "保存昵称失败")
		return
	}
	u.Nickname = name
	writeJSON(w, 200, u)
}
func (s *Server) uploadAvatar(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, (2<<20)+32768)
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		fail(w, 400, "头像须小于 2MB")
		return
	}
	defer r.MultipartForm.RemoveAll()
	file, _, err := r.FormFile("file")
	if err != nil {
		fail(w, 400, "请选择头像")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (2<<20)+1))
	if err != nil || len(data) > 2<<20 {
		fail(w, 400, "头像须小于 2MB")
		return
	}
	_, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "png" && format != "jpeg") {
		fail(w, 400, "请选择有效的 PNG/JPEG 图片")
		return
	}
	name := secret(16) + "." + format
	dir := filepath.Join(s.dataDir, "avatars")
	if err = os.MkdirAll(dir, 0700); err != nil {
		fail(w, 500, "无法保存头像")
		return
	}
	if err = os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
		fail(w, 500, "无法保存头像")
		return
	}
	old := u.AvatarURL
	u.AvatarURL = "/api/avatars/" + name
	if err = s.db.Model(u).Update("avatar_url", u.AvatarURL).Error; err != nil {
		os.Remove(filepath.Join(dir, name))
		fail(w, 500, "无法保存头像")
		return
	}
	if old != "" {
		os.Remove(filepath.Join(dir, filepath.Base(old)))
	}
	writeJSON(w, 200, u)
}

var avatarName = regexp.MustCompile(`^[a-f0-9]{32}\.(png|jpeg)$`)

func (s *Server) getAvatar(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !avatarName.MatchString(name) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, filepath.Join(s.dataDir, "avatars", name))
}
