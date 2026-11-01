package main

import (
	"gorm.io/gorm"
	"net/http"
	"time"
)

func (s *Server) startPractice(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	var req struct {
		MIDIID string `json:"midiId"`
		Mode   string `json:"mode"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.Mode != "play" && req.Mode != "follow" {
		fail(w, 400, "mode 只能为 play/follow")
		return
	}
	item, err := s.loadMIDI(req.MIDIID)
	if err != nil || !midiVisible(item, u) {
		fail(w, 404, "MIDI 不可用")
		return
	}
	now := time.Now().UTC()
	// Keep one active practice session per user.
	s.db.Model(&PracticeLog{}).Where("user_id = ? AND ended_at IS NULL", u.ID).Update("ended_at", now)
	log := PracticeLog{UserID: u.ID, MIDIID: item.ID, Title: item.Title, Mode: req.Mode, StartedAt: now, LastBeatAt: now}
	if err := s.db.Create(&log).Error; err != nil {
		fail(w, 500, "无法建立练习记录")
		return
	}
	writeJSON(w, 201, log)
}
func (s *Server) beatPractice(w http.ResponseWriter, r *http.Request, finish bool) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	var req struct {
		SessionID uint `json:"sessionId"`
	}
	if !decode(w, r, &req) {
		return
	}
	var out PracticeLog
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var log PracticeLog
		if err := tx.Where("id = ? AND user_id = ?", req.SessionID, u.ID).First(&log).Error; err != nil {
			return err
		}
		if log.EndedAt != nil {
			out = log
			return nil
		}
		now := time.Now().UTC()
		delta := now.Sub(log.LastBeatAt)
		// Count only recent server-observed activity; no client-supplied seconds.
		// An absent heartbeat cannot inflate a session indefinitely.
		seconds := int(delta.Seconds())
		if seconds < 0 {
			seconds = 0
		}
		if seconds > 40 {
			seconds = 40
		}
		log.DurationSeconds += seconds
		log.LastBeatAt = now
		if finish {
			log.EndedAt = &now
		}
		if err := tx.Save(&log).Error; err != nil {
			return err
		}
		out = log
		return nil
	})
	if err != nil {
		fail(w, 404, "练习记录不存在")
		return
	}
	writeJSON(w, 200, out)
}
func (s *Server) heartbeatPractice(w http.ResponseWriter, r *http.Request) {
	s.beatPractice(w, r, false)
}
func (s *Server) finishPractice(w http.ResponseWriter, r *http.Request) { s.beatPractice(w, r, true) }
func (s *Server) practiceHistory(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	limit, offset := pager(r)
	var list []PracticeLog
	q := s.db.Where("user_id = ?", u.ID)
	if id := r.URL.Query().Get("midiId"); id != "" {
		q = q.Where("midi_id = ?", id)
	}
	if err := q.Order("started_at desc").Limit(limit).Offset(offset).Find(&list).Error; err != nil {
		fail(w, 500, "读取失败")
		return
	}
	writeJSON(w, 200, map[string]any{"items": list})
}
func (s *Server) practiceStats(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	type summary struct {
		MIDIID        string    `json:"midiId" gorm:"column:midi_id"`
		Title         string    `json:"title"`
		Sessions      int       `json:"sessions"`
		Seconds       int       `json:"seconds"`
		LastPracticed time.Time `json:"lastPracticed"`
		LastMode      string    `json:"lastMode"`
	}
	var list []summary
	q := s.db.Model(&PracticeLog{}).Select("midi_id as midi_id, MAX(title) as title, COUNT(*) as sessions, COALESCE(SUM(duration_seconds),0) as seconds, MAX(COALESCE(ended_at,started_at)) as last_practiced, (array_agg(mode ORDER BY started_at DESC))[1] as last_mode").Where("user_id = ?", u.ID)
	if id := r.URL.Query().Get("midiId"); id != "" {
		q = q.Where("midi_id = ?", id)
	}
	if err := q.Group("midi_id").Order("last_practiced DESC").Limit(100).Scan(&list).Error; err != nil {
		fail(w, 500, "读取统计失败")
		return
	}
	writeJSON(w, 200, map[string]any{"items": list})
}
