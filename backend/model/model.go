// Package model defines the UI-independent Aion backend contract.
package model

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrInvalid       = errors.New("invalid input")
	ErrConflict      = errors.New("conflict")
	ErrParentDeleted = errors.New("parent project is deleted")
)

type Status string

const (
	Planned    Status = "planned"
	InProgress Status = "in_progress"
	Done       Status = "done"
)

func (s Status) Valid() bool { return s == Planned || s == InProgress || s == Done }

type Priority string

const (
	NoPriority Priority = "none"
	Low        Priority = "low"
	Medium     Priority = "medium"
	High       Priority = "high"
	Urgent     Priority = "urgent"
)

func (p Priority) Valid() bool {
	return p == NoPriority || p == Low || p == Medium || p == High || p == Urgent
}

type Project struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Icon        string     `json:"icon"`
	Description string     `json:"description"`
	Archived    bool       `json:"archived"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}
type Subtask struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}
type Task struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"project_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      Status     `json:"status"`
	Priority    Priority   `json:"priority"`
	Tags        []string   `json:"tags"`
	Subtasks    []Subtask  `json:"subtasks"`
	DueDate     string     `json:"due_date,omitempty"`
	Position    int        `json:"position"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}
type Attachment struct {
	ID          string    `json:"id"`
	TaskID      string    `json:"task_id"`
	Name        string    `json:"name"`
	StorageName string    `json:"storage_name"`
	MediaType   string    `json:"media_type"`
	Size        int64     `json:"size"`
	SHA256      string    `json:"sha256"`
	CreatedAt   time.Time `json:"created_at"`
}
type TrashItem struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Name      string    `json:"name"`
	ProjectID string    `json:"project_id,omitempty"`
	DeletedAt time.Time `json:"deleted_at"`
}

func NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
func Timestamp() string { return time.Now().UTC().Format(time.RFC3339Nano) }
