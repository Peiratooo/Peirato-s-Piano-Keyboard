package main

import (
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"time"
)

func (s *Server) createReport(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	if !s.limiter.allow("report:"+stringID(u.ID), 10, 24*time.Hour) {
		fail(w, 429, "举报次数过多")
		return
	}
	var req struct {
		Kind       string `json:"kind"`
		ResourceID string `json:"resourceId"`
		Reason     string `json:"reason"`
	}
	if !decode(w, r, &req) {
		return
	}
	reason := cleanText(req.Reason, 500)
	if len([]rune(reason)) < 5 {
		fail(w, 400, "请描述至少 5 个字的举报原因")
		return
	}
	switch req.Kind {
	case "midi":
		item, err := s.loadMIDI(req.ResourceID)
		if err != nil || !item.Public {
			fail(w, 404, "曲目不存在")
			return
		}
	case "room":
		var room Room
		if err := s.db.Where("id = ? AND visibility = ?", req.ResourceID, "public").First(&room).Error; err != nil {
			fail(w, 404, "房间不存在")
			return
		}
	default:
		fail(w, 400, "无效举报对象")
		return
	}
	report := Report{ReporterID: u.ID, Kind: req.Kind, ResourceID: req.ResourceID, Reason: reason, Status: "pending"}
	if err := s.db.Create(&report).Error; err != nil {
		fail(w, 500, "举报失败")
		return
	}
	writeJSON(w, 201, report)
}
func (s *Server) listReports(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}
	var list []Report
	if err := s.db.Where("status = ?", "pending").Order("created_at asc").Limit(100).Find(&list).Error; err != nil {
		fail(w, 500, "获取举报失败")
		return
	}
	writeJSON(w, 200, map[string]any{"items": list})
}

// Moderation actions are explicit and scoped to the reported resource.
// "resolved" without an action records a completed review; closing a room
// or hiding a MIDI additionally removes it from public discovery immediately.
func (s *Server) resolveReport(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		fail(w, 400, "无效举报 ID")
		return
	}
	var req struct {
		Status string `json:"status"`
		Action string `json:"action"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.Status != "resolved" && req.Status != "dismissed" {
		fail(w, 400, "无效处理状态")
		return
	}
	if req.Action != "" && (req.Status != "resolved" || (req.Action != "hide_midi" && req.Action != "close_room")) {
		fail(w, 400, "无效审核操作")
		return
	}
	var report Report
	if err := s.db.Where("id = ? AND status = ?", id, "pending").First(&report).Error; err != nil {
		fail(w, 404, "待处理举报不存在")
		return
	}
	if (req.Action == "hide_midi" && report.Kind != "midi") || (req.Action == "close_room" && report.Kind != "room") {
		fail(w, 400, "处理操作与举报对象不匹配")
		return
	}
	// Keep changes to a resource and its moderation record consistent.
	err = s.db.Transaction(func(tx *gorm.DB) error {
		switch req.Action {
		case "hide_midi":
			result := tx.Model(&MIDIFile{}).Where("id = ? AND public = ?", report.ResourceID, true).
				Updates(map[string]any{"public": false, "review_status": "blocked"})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return gorm.ErrRecordNotFound
			}
		case "close_room":
			result := tx.Model(&Room{}).Where("id = ? AND closed = ?", report.ResourceID, false).Update("closed", true)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return gorm.ErrRecordNotFound
			}
		}
		result := tx.Model(&Report{}).Where("id = ? AND status = ?", id, "pending").Update("status", req.Status)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			fail(w, 409, "举报或资源已发生变化，请刷新后重试")
		} else {
			fail(w, 500, "处理失败")
		}
		return
	}
	if req.Action == "close_room" {
		s.hub.closeRoom(report.ResourceID)
	}
	w.WriteHeader(http.StatusNoContent)
}
