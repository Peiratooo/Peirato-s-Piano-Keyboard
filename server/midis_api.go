package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const maxMIDISize = 8 << 20

func midiVisible(m MIDIFile, u *User) bool {
	return (u != nil && (u.ID == m.OwnerID || (u.Role == "admin" && (m.Public || m.ReviewStatus == "blocked")))) ||
		(m.Public && m.ReviewStatus == "approved")
}

type MIDIView struct {
	MIDIFile
	OwnerAvatarURL string `json:"ownerAvatarUrl"`
	LikeCount      int64  `json:"likeCount"`
	DislikeCount   int64  `json:"dislikeCount"`
	FavoriteCount  int64  `json:"favoriteCount"`
	MyReaction     int    `json:"myReaction"`
	MyFavorite     bool   `json:"myFavorite"`
}

func (s *Server) midiViews(u *User) *gorm.DB {
	uid := uint(0)
	if u != nil {
		uid = u.ID
	}
	totals := s.db.Model(&MIDIInteraction{}).Select("midi_id, SUM(CASE WHEN reaction=1 THEN 1 ELSE 0 END) AS likes, SUM(CASE WHEN reaction=-1 THEN 1 ELSE 0 END) AS dislikes, SUM(CASE WHEN favorite THEN 1 ELSE 0 END) AS favorites").Group("midi_id")
	return s.db.Table("midi_files AS m").Select("m.*, COALESCE(NULLIF(u.nickname,''), u.username) AS owner_name, u.avatar_url AS owner_avatar_url, COALESCE(t.likes,0) AS like_count, COALESCE(t.dislikes,0) AS dislike_count, COALESCE(t.favorites,0) AS favorite_count, COALESCE(mine.reaction,0) AS my_reaction, COALESCE(mine.favorite,false) AS my_favorite").Joins("LEFT JOIN users u ON u.id=m.owner_id").Joins("LEFT JOIN (?) t ON t.midi_id=m.id", totals).Joins("LEFT JOIN midi_interactions mine ON mine.midi_id=m.id AND mine.user_id=?", uid)
}
func (s *Server) midiView(id string, u *User) (MIDIView, error) {
	var out MIDIView
	err := s.midiViews(u).Where("m.id=?", id).Take(&out).Error
	return out, err
}
func (s *Server) listMIDIs(w http.ResponseWriter, r *http.Request) {
	u := s.optionalUser(r)
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "public"
	}
	q := s.midiViews(u)
	switch scope {
	case "public":
		q = q.Where("m.public = true AND m.review_status = 'approved'")
	case "mine":
		if u == nil {
			fail(w, 401, "请先登录")
			return
		}
		q = q.Where("m.owner_id=?", u.ID)
	case "favorites":
		if u == nil {
			fail(w, 401, "请先登录")
			return
		}
		q = q.Where("mine.favorite = true AND (m.owner_id=? OR (m.public = true AND m.review_status = 'approved'))", u.ID)
	default:
		fail(w, 400, "无效曲库范围")
		return
	}
	if text := cleanText(r.URL.Query().Get("q"), 60); text != "" {
		pattern := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(text) + "%"
		q = q.Where("(m.title ILIKE ? OR m.composer ILIKE ? OR u.nickname ILIKE ? OR m.tags ILIKE ?)", pattern, pattern, pattern, pattern)
	}
	if raw := r.URL.Query().Get("difficulty"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 5 {
			fail(w, 400, "难度必须 1–5")
			return
		}
		q = q.Where("m.difficulty=?", n)
	}
	if tag := cleanText(r.URL.Query().Get("tag"), 20); tag != "" {
		q = q.Where("? = ANY(string_to_array(m.tags, ','))", strings.ToLower(tag))
	}
	orders := map[string]string{"latest": "m.created_at DESC", "likes": "like_count DESC", "favorites": "favorite_count DESC", "views": "m.view_count DESC", "plays": "m.play_count DESC", "downloads": "m.download_count DESC"}
	sort := r.URL.Query().Get("sort")
	if sort == "" {
		sort = "latest"
	}
	order, ok := orders[sort]
	if !ok {
		fail(w, 400, "无效排序方式")
		return
	}
	page, size := 1, 20
	if raw := r.URL.Query().Get("page"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100000 {
			fail(w, 400, "无效页码")
			return
		}
		page = n
	}
	if raw := r.URL.Query().Get("pageSize"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 50 {
			fail(w, 400, "每页须为 1–50 首")
			return
		}
		size = n
	}
	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		fail(w, 500, "读取曲库失败")
		return
	}
	list := []MIDIView{}
	if err := q.Order(order).Order("m.id DESC").Limit(size).Offset((page - 1) * size).Scan(&list).Error; err != nil {
		fail(w, 500, "读取曲库失败")
		return
	}
	writeJSON(w, 200, map[string]any{"items": list, "total": total, "page": page, "pageSize": size})
}
func (s *Server) uploadMIDI(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	if !s.limiter.allow("upload:"+stringID(u.ID), 20, time.Hour) {
		fail(w, 429, "上传太频繁")
		return
	}
	var count int64
	s.db.Model(&MIDIFile{}).Where("owner_id = ?", u.ID).Count(&count)
	if count >= 100 {
		fail(w, 409, "每个账号最多保存 100 个 MIDI")
		return
	}
	var used int64
	s.db.Model(&MIDIFile{}).Where("owner_id = ?", u.ID).Select("COALESCE(SUM(file_size), 0)").Scan(&used)
	if used >= 100<<20 {
		fail(w, 409, "个人 MIDI 云存储已达到 100MB 上限")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxMIDISize+32768)
	if err := r.ParseMultipartForm(maxMIDISize); err != nil {
		fail(w, 400, "上传数据或文件过大")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	src, head, err := r.FormFile("file")
	if err != nil {
		fail(w, 400, "缺少 MIDI 文件")
		return
	}
	defer src.Close()
	if head.Size > maxMIDISize {
		fail(w, 413, "MIDI 文件不得超过 8MB")
		return
	}
	raw, err := io.ReadAll(io.LimitReader(src, maxMIDISize+1))
	if err != nil || len(raw) > maxMIDISize {
		fail(w, 413, "MIDI 文件不得超过 8MB")
		return
	}
	if used+int64(len(raw)) > 100<<20 {
		fail(w, 413, "该文件将超出个人 100MB 配额")
		return
	}
	info, err := parseMIDIInfo(raw)
	if err != nil {
		fail(w, 422, "解析 MIDI 失败："+err.Error())
		return
	}
	title := cleanText(r.FormValue("title"), 120)
	if title == "" {
		title = cleanText(strings.TrimSuffix(filepath.Base(head.Filename), filepath.Ext(head.Filename)), 120)
	}
	if title == "" {
		title = "未命名 MIDI"
	}
	diff, _ := strconv.Atoi(r.FormValue("difficulty"))
	if diff < 1 || diff > 5 {
		diff = 3
	}
	pub := r.FormValue("public") == "true"
	if pub && r.FormValue("rightsConfirmed") != "true" {
		fail(w, 400, "公开发布前请确认已获得必要分享权利")
		return
	}
	sha := sha256.Sum256(raw)
	id := secret(16)
	p := filepath.Join(s.dataDir, "midi", id+".mid")
	if err := os.WriteFile(p, raw, 0600); err != nil {
		fail(w, 500, "保存文件失败")
		return
	}
	entry := MIDIFile{ID: id, OwnerID: u.ID, OwnerName: u.Username, Title: title, Composer: cleanText(r.FormValue("composer"), 120), Description: cleanText(r.FormValue("description"), 1500), Tags: sanitizeTags(r.FormValue("tags")), Difficulty: diff, Public: pub, ReviewStatus: "approved", FilePath: p, SHA256: hex.EncodeToString(sha[:]), FileSize: int64(len(raw)), NoteCount: info.NoteCount, TrackCount: info.TrackCount, Measures: info.Measures, DurationMs: info.DurationMs, TempoBPM: info.TempoBPM, TimeSignature: info.TimeSignature, KeySignature: info.KeySignature, TrackNames: info.TrackNames}
	if pub {
		entry.ReviewStatus = "pending"
	}
	if err := s.db.Create(&entry).Error; err != nil {
		_ = os.Remove(p)
		fail(w, 500, "保存元数据失败")
		return
	}
	view, _ := s.midiView(entry.ID, u)
	writeJSON(w, 201, view)
}
func (s *Server) loadMIDI(id string) (MIDIFile, error) {
	var item MIDIFile
	err := s.db.Where("id = ?", id).First(&item).Error
	return item, err
}
func (s *Server) getMIDI(w http.ResponseWriter, r *http.Request) {
	item, err := s.loadMIDI(r.PathValue("id"))
	if err != nil {
		fail(w, 404, "曲目不存在")
		return
	}
	if !midiVisible(item, s.optionalUser(r)) {
		fail(w, 404, "曲目不存在")
		return
	}
	view, err := s.midiView(item.ID, s.optionalUser(r))
	if err != nil {
		fail(w, 500, "读取曲目失败")
		return
	}
	writeJSON(w, 200, view)
}
func (s *Server) downloadMIDI(w http.ResponseWriter, r *http.Request) {
	item, err := s.loadMIDI(r.PathValue("id"))
	if err != nil || !midiVisible(item, s.optionalUser(r)) {
		fail(w, 404, "曲目不存在")
		return
	}
	file, err := os.Open(item.FilePath)
	if err != nil {
		fail(w, 500, "文件暂不可用")
		return
	}
	defer file.Close()
	w.Header().Set("Content-Type", "audio/midi")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", item.ID+".mid"))
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = io.Copy(w, file)
}
func (s *Server) updateMIDI(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	item, err := s.loadMIDI(r.PathValue("id"))
	if err != nil || item.OwnerID != u.ID {
		fail(w, 404, "曲目不存在")
		return
	}
	var input struct {
		Title           string `json:"title"`
		Composer        string `json:"composer"`
		Description     string `json:"description"`
		Tags            string `json:"tags"`
		Difficulty      int    `json:"difficulty"`
		Public          bool   `json:"public"`
		RightsConfirmed bool   `json:"rightsConfirmed"`
	}
	if !decode(w, r, &input) {
		return
	}
	title := cleanText(input.Title, 120)
	if title == "" {
		fail(w, 400, "标题不能为空")
		return
	}
	if input.Difficulty < 1 || input.Difficulty > 5 {
		fail(w, 400, "难度必须 1-5")
		return
	}
	// A moderated takedown cannot be bypassed by toggling the same file public.
	if input.Public && item.ReviewStatus == "blocked" {
		fail(w, 403, "此 MIDI 已被管理员下架，不能再次公开")
		return
	}
	if input.Public && !item.Public && !input.RightsConfirmed {
		fail(w, 400, "公开分享前请确认拥有必要权利")
		return
	}
	updates := map[string]any{"title": title, "composer": cleanText(input.Composer, 120), "description": cleanText(input.Description, 1500), "tags": sanitizeTags(input.Tags), "difficulty": input.Difficulty, "public": input.Public}
	changed := title != item.Title || updates["composer"] != item.Composer || updates["description"] != item.Description || updates["tags"] != item.Tags || input.Difficulty != item.Difficulty
	switch {
	case item.ReviewStatus == "blocked":
		updates["review_status"] = "blocked"
	case input.Public && (!item.Public || changed || item.ReviewStatus == "rejected"):
		updates["review_status"] = "pending"
	case !input.Public:
		updates["review_status"] = "approved"
	}
	if err := s.db.Model(&item).Updates(updates).Error; err != nil {
		fail(w, 500, "保存失败")
		return
	}
	updated, _ := s.midiView(item.ID, u)
	writeJSON(w, 200, updated)
}
func (s *Server) deleteMIDI(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	item, err := s.loadMIDI(r.PathValue("id"))
	if err != nil || item.OwnerID != u.ID {
		fail(w, 404, "曲目不存在")
		return
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("midi_id=?", item.ID).Delete(&MIDIInteraction{}).Error; err != nil {
			return err
		}
		return tx.Delete(&item).Error
	}); err != nil {
		fail(w, 500, "删除失败")
		return
	}
	_ = os.Remove(item.FilePath)
	w.WriteHeader(204)
}
func (s *Server) pendingMIDIs(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}
	if status != "pending" && status != "blocked" && status != "rejected" {
		fail(w, 400, "无效审核队列")
		return
	}
	var list []MIDIView
	if err := s.midiViews(s.optionalUser(r)).Where("m.review_status = ?", status).Order("m.created_at asc").Limit(50).Scan(&list).Error; err != nil {
		fail(w, 500, "获取失败")
		return
	}
	writeJSON(w, 200, map[string]any{"items": list})
}
func (s *Server) reviewMIDI(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.Status != "approved" && req.Status != "rejected" {
		fail(w, 400, "无效状态")
		return
	}
	result := s.db.Model(&MIDIFile{}).Where("id = ? AND ((public = true AND review_status IN ?) OR review_status = ?)", r.PathValue("id"), []string{"pending", "rejected"}, "blocked").Updates(map[string]any{"review_status": req.Status, "public": true})
	if result.Error != nil {
		fail(w, 500, "审核失败")
		return
	}
	if result.RowsAffected == 0 {
		fail(w, 404, "曲目不存在")
		return
	}
	w.WriteHeader(204)
}

func (s *Server) setMIDIInteraction(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	item, err := s.loadMIDI(r.PathValue("id"))
	if err != nil || !midiVisible(item, u) {
		fail(w, 404, "曲目不存在")
		return
	}
	var input struct {
		Reaction int  `json:"reaction"`
		Favorite bool `json:"favorite"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.Reaction < -1 || input.Reaction > 1 {
		fail(w, 400, "无效评价")
		return
	}
	row := MIDIInteraction{UserID: u.ID, MIDIID: item.ID, Reaction: input.Reaction, Favorite: input.Favorite, UpdatedAt: time.Now()}
	if err := s.db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "midi_id"}}, DoUpdates: clause.AssignmentColumns([]string{"reaction", "favorite", "updated_at"})}).Create(&row).Error; err != nil {
		fail(w, 500, "保存互动失败")
		return
	}
	view, err := s.midiView(item.ID, u)
	if err != nil {
		fail(w, 500, "读取互动失败")
		return
	}
	writeJSON(w, 200, view)
}
func (s *Server) recordMIDIEvent(w http.ResponseWriter, r *http.Request) {
	u := s.optionalUser(r)
	item, err := s.loadMIDI(r.PathValue("id"))
	if err != nil || !midiVisible(item, u) {
		fail(w, 404, "曲目不存在")
		return
	}
	var input struct {
		Type string `json:"type"`
	}
	if !decode(w, r, &input) {
		return
	}
	column, ok := map[string]string{"view": "view_count", "play": "play_count", "download": "download_count"}[input.Type]
	if !ok {
		fail(w, 400, "无效曲目事件")
		return
	}
	if err := s.db.Model(&item).UpdateColumn(column, gorm.Expr(column+" + 1")).Error; err != nil {
		fail(w, 500, "记录失败")
		return
	}
	w.WriteHeader(204)
}
