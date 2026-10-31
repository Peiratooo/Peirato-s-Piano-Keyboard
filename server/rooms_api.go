package main

import (
	"net/http"
	"time"
)

func (s *Server) listRooms(w http.ResponseWriter, r *http.Request) {
	limit, offset := pager(r)
	live := s.hub.publicLiveRooms()
	if len(live) == 0 {
		writeJSON(w, 200, map[string]any{"rooms": []Room{}, "lobby": s.hub.lobbySnapshot()})
		return
	}
	var list []Room
	if err := s.db.Where("id IN ? AND visibility = ? AND closed = ?", live, "public", false).Order("updated_at desc").Limit(limit).Offset(offset).Find(&list).Error; err != nil {
		fail(w, 500, "获取房间失败")
		return
	}
	writeJSON(w, 200, map[string]any{"rooms": list, "lobby": s.hub.lobbySnapshot()})
}
func (s *Server) createRoom(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	if !s.limiter.allow("room:"+stringID(u.ID), 10, time.Hour) {
		fail(w, 429, "创建房间过于频繁")
		return
	}
	var req struct {
		Title      string `json:"title"`
		Visibility string `json:"visibility"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.Visibility != "public" && req.Visibility != "private" {
		fail(w, 400, "visibility 只能是 public/private")
		return
	}
	title := cleanText(req.Title, 80)
	if title == "" {
		title = u.Username + "的琴房"
	}
	id := secret(16)
	invite := secret(20)
	room := Room{ID: id, OwnerID: u.ID, OwnerName: u.Username, Title: title, Visibility: req.Visibility, InviteHash: hashToken(invite)}
	// One active personal room per user; expired rooms are retained as records.
	if err := s.db.Model(&Room{}).Where("owner_id = ? AND closed = ?", u.ID, false).Updates(map[string]any{"closed": true, "updated_at": time.Now()}).Error; err != nil {
		fail(w, 500, "更新旧房间失败")
		return
	}
	s.hub.closeOwnerRooms(u.ID)
	if err := s.db.Create(&room).Error; err != nil {
		fail(w, 500, "创建房间失败")
		return
	}
	writeJSON(w, 201, map[string]any{"room": room, "inviteCode": invite})
}
func (s *Server) updateRoom(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	var room Room
	if err := s.db.Where("id = ? AND owner_id = ? AND closed = ?", r.PathValue("id"), u.ID, false).First(&room).Error; err != nil {
		fail(w, 404, "房间不存在")
		return
	}
	var req struct {
		Title        string `json:"title"`
		Visibility   string `json:"visibility"`
		RotateInvite bool   `json:"rotateInvite"`
	}
	if !decode(w, r, &req) {
		return
	}
	changes := map[string]any{}
	if req.Title != "" {
		changes["title"] = cleanText(req.Title, 80)
	}
	if req.Visibility != "" {
		if req.Visibility != "public" && req.Visibility != "private" {
			fail(w, 400, "无效可见性")
			return
		}
		changes["visibility"] = req.Visibility
	}
	invite := ""
	if req.RotateInvite {
		invite = secret(20)
		changes["invite_hash"] = hashToken(invite)
	}
	if len(changes) > 0 {
		if err := s.db.Model(&room).Updates(changes).Error; err != nil {
			fail(w, 500, "保存房间失败")
			return
		}
	}
	s.db.First(&room, "id = ?", room.ID)
	s.hub.refreshRoom(room)
	writeJSON(w, 200, map[string]any{"room": room, "inviteCode": invite})
}
func (s *Server) closeRoom(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	result := s.db.Model(&Room{}).Where("id = ? AND owner_id = ? AND closed = ?", r.PathValue("id"), u.ID, false).Update("closed", true)
	if result.Error != nil {
		fail(w, 500, "关闭房间失败")
		return
	}
	if result.RowsAffected == 0 {
		fail(w, 404, "房间不存在")
		return
	}
	s.hub.closeRoom(r.PathValue("id"))
	w.WriteHeader(204)
}
func (s *Server) randomRoom(w http.ResponseWriter, r *http.Request) {
	ids := s.hub.publicLiveRooms()
	if len(ids) == 0 {
		fail(w, 404, "暂无正在直播练琴的公开琴房")
		return
	}
	// Rotate by server time; avoid a global database random scan.
	selected := ids[int(time.Now().UnixNano()%int64(len(ids)))]
	writeJSON(w, 200, map[string]string{"roomId": selected})
}
func stringID(id uint) string { return fmtUint(uint64(id)) }
