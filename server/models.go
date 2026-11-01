package main

import "time"

// Flat ownership IDs instead of GORM associations: application services own permissions.
type User struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Username     string    `json:"username" gorm:"uniqueIndex;size:32;not null"`
	Nickname     string    `json:"nickname" gorm:"size:24"`
	AvatarURL    string    `json:"avatarUrl" gorm:"size:160"`
	PasswordHash string    `json:"-" gorm:"not null"`
	Role         string    `json:"role" gorm:"size:16;default:user"`
	CreatedAt    time.Time `json:"createdAt"`
}
type Session struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"index;not null"`
	TokenHash string    `gorm:"uniqueIndex;size:64;not null"`
	ExpiresAt time.Time `gorm:"index;not null"`
	CreatedAt time.Time
}
type Room struct {
	ID         string    `json:"id" gorm:"primaryKey;size:36"`
	OwnerID    uint      `json:"ownerId" gorm:"index;not null"`
	OwnerName  string    `json:"ownerName" gorm:"size:32"`
	Title      string    `json:"title" gorm:"size:80;not null"`
	Visibility string    `json:"visibility" gorm:"size:12;index;not null"` // public/private
	InviteHash string    `json:"-" gorm:"size:64"`
	Closed     bool      `json:"closed" gorm:"index;default:false"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}
type MIDIFile struct {
	ID            string    `json:"id" gorm:"primaryKey;size:36"`
	OwnerID       uint      `json:"ownerId" gorm:"index;not null"`
	OwnerName     string    `json:"ownerName" gorm:"size:32"`
	Title         string    `json:"title" gorm:"size:120;not null"`
	Composer      string    `json:"composer" gorm:"size:120"`
	Description   string    `json:"description" gorm:"size:1500"`
	Tags          string    `json:"tags" gorm:"size:300"` // comma-separated safe normalized tags
	Difficulty    int       `json:"difficulty"`
	Public        bool      `json:"public" gorm:"index"`
	ReviewStatus  string    `json:"reviewStatus" gorm:"size:16;index"` // pending/approved/rejected
	FilePath      string    `json:"-" gorm:"not null"`
	SHA256        string    `json:"sha256" gorm:"size:64"`
	FileSize      int64     `json:"fileSize"`
	ViewCount     int64     `json:"viewCount"`
	PlayCount     int64     `json:"playCount"`
	DownloadCount int64     `json:"downloadCount"`
	NoteCount     int       `json:"noteCount"`
	TrackCount    int       `json:"trackCount"`
	Measures      int       `json:"measures"`
	DurationMs    int64     `json:"durationMs"`
	TempoBPM      float64   `json:"tempoBpm"`
	TimeSignature string    `json:"timeSignature"`
	KeySignature  string    `json:"keySignature"`
	TrackNames    string    `json:"trackNames"` // JSON encoded list; no join needed
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
type PracticeLog struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	UserID          uint       `json:"userId" gorm:"index;not null"`
	MIDIID          string     `json:"midiId" gorm:"column:midi_id;index;size:36;not null"`
	Title           string     `json:"title" gorm:"size:120"`
	Mode            string     `json:"mode" gorm:"size:12"` // play/follow
	StartedAt       time.Time  `json:"startedAt" gorm:"index"`
	LastBeatAt      time.Time  `json:"-"`
	DurationSeconds int        `json:"durationSeconds"`
	EndedAt         *time.Time `json:"endedAt,omitempty"`
}

// Minimal moderation queue; no complex relationships or FK constraints.
type Report struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	ReporterID uint      `json:"reporterId" gorm:"index;not null"`
	Kind       string    `json:"kind" gorm:"size:12;index;not null"` // midi / room
	ResourceID string    `json:"resourceId" gorm:"size:36;index;not null"`
	Reason     string    `json:"reason" gorm:"size:500;not null"`
	Status     string    `json:"status" gorm:"size:16;index;not null"`
	CreatedAt  time.Time `json:"createdAt"`
}

// Each account has exactly one vote and favorite state per song.
type MIDIInteraction struct {
	UserID    uint   `gorm:"primaryKey;autoIncrement:false"`
	MIDIID    string `gorm:"column:midi_id;primaryKey;size:36"`
	Reaction  int    `gorm:"not null;default:0"`
	Favorite  bool   `gorm:"not null;default:false"`
	UpdatedAt time.Time
}

func (MIDIFile) TableName() string        { return "midi_files" }
func (MIDIInteraction) TableName() string { return "midi_interactions" }
