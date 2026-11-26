// Package ipc provides safe local-only RPC over a Unix domain socket so external
// processes (such as `aion mcp`) can share a running Aion GUI instance's Store
// without lock conflicts, and transparently fall back to short-lived direct
// backend access when the GUI is not running.
package ipc

import (
	"encoding/json"
	"errors"
	"fmt"

	"aion/backend/model"
)

// Method names supported by the local IPC surface.
// Destructive operations (permanent delete, clear trash, backup restore, settings writes)
// are intentionally excluded from this contract.
const (
	MethodWorkspaceSummary = "workspace.summary"

	MethodProjectsList    = "projects.list"
	MethodProjectsCreate  = "projects.create"
	MethodProjectsUpdate  = "projects.update"
	MethodProjectsArchive = "projects.archive"
	MethodProjectsTrash   = "projects.trash"

	MethodTasksList   = "tasks.list"
	MethodTasksSearch = "tasks.search"
	MethodTasksGet    = "tasks.get"
	MethodTasksCreate = "tasks.create"
	MethodTasksUpdate = "tasks.update"
	MethodTasksMove   = "tasks.move"
	MethodTasksAttach = "tasks.attach"
	MethodTasksTrash  = "tasks.trash"

	MethodJobsList          = "jobs.list"
	MethodJobsGet           = "jobs.get"
	MethodJobsCreate        = "jobs.create"
	MethodJobsUpdate        = "jobs.update"
	MethodJobsChangeStage   = "jobs.change_stage"
	MethodJobsAddNote       = "jobs.add_note"
	MethodJobsBackfillStage = "jobs.backfill_stage"
	MethodJobsClose         = "jobs.close"
	MethodJobsReopen        = "jobs.reopen"
	MethodJobsTrash         = "jobs.trash"

	MethodTrashList    = "trash.list"
	MethodTrashRestore = "trash.restore"
)

// ChangeScope identifies which UI area should refresh after a committed mutation.
type ChangeScope string

const (
	ScopeProjects ChangeScope = "projects"
	ScopeTasks    ChangeScope = "tasks"
	ScopeJobs     ChangeScope = "job_applications"
	ScopeTrash    ChangeScope = "trash"
)

// ChangeEvent describes a committed external write so the GUI can refresh safely.
type ChangeEvent struct {
	Scope     ChangeScope `json:"scope"`
	ProjectID string      `json:"project_id,omitempty"`
	TaskID    string      `json:"task_id,omitempty"`
	JobID     string      `json:"job_id,omitempty"`
}

// Error codes returned over IPC.
const (
	ErrCodeInvalid       = "invalid"
	ErrCodeNotFound      = "not_found"
	ErrCodeConflict      = "conflict"
	ErrCodeParentDeleted = "parent_deleted"
	ErrCodeUnknownMethod = "unknown_method"
	ErrCodeInternal      = "internal"
)

// RPCError carries a structured error code and human-readable message across IPC.
type RPCError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Is allows callers to match sentinel model errors across the IPC boundary.
func (e *RPCError) Is(target error) bool {
	if e == nil {
		return false
	}
	switch e.Code {
	case ErrCodeNotFound:
		return target == model.ErrNotFound
	case ErrCodeInvalid:
		return target == model.ErrInvalid
	case ErrCodeConflict:
		return target == model.ErrConflict
	case ErrCodeParentDeleted:
		return target == model.ErrParentDeleted
	default:
		return false
	}
}

func encodeError(err error) *RPCError {
	if err == nil {
		return nil
	}
	var rpcErr *RPCError
	if errors.As(err, &rpcErr) {
		return rpcErr
	}
	code := ErrCodeInternal
	switch {
	case errors.Is(err, model.ErrParentDeleted):
		code = ErrCodeParentDeleted
	case errors.Is(err, model.ErrNotFound):
		code = ErrCodeNotFound
	case errors.Is(err, model.ErrConflict):
		code = ErrCodeConflict
	case errors.Is(err, model.ErrInvalid):
		code = ErrCodeInvalid
	}
	return &RPCError{Code: code, Message: err.Error()}
}

type rpcRequest struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result,omitempty"`
	Error  *RPCError       `json:"error,omitempty"`
}

// WorkspaceSummary gives an AI agent a compact snapshot of active projects and job applications.
type WorkspaceSummary struct {
	Projects      []ProjectSummaryItem       `json:"projects"`
	ArchivedCount int                        `json:"archived_projects_count"`
	JobStats      model.JobApplicationStats  `json:"job_stats"`
	ActiveJobs    []model.JobApplication     `json:"active_jobs"`
	TrashCount    int                        `json:"trash_count"`
}

// ProjectSummaryItem pairs a project with its task counts by status.
type ProjectSummaryItem struct {
	Project    model.Project `json:"project"`
	Planned    int           `json:"planned"`
	InProgress int           `json:"in_progress"`
	Done       int           `json:"done"`
	Total      int           `json:"total"`
}

type ListProjectsParams struct {
	Archived *bool `json:"archived,omitempty"`
}

type CreateProjectParams struct {
	Name        string `json:"name"`
	Icon        string `json:"icon,omitempty"`
	Description string `json:"description,omitempty"`
}

type UpdateProjectParams struct {
	ID          string  `json:"id"`
	Name        *string `json:"name,omitempty"`
	Icon        *string `json:"icon,omitempty"`
	Description *string `json:"description,omitempty"`
}

type ArchiveProjectParams struct {
	ID       string `json:"id"`
	Archived bool   `json:"archived"`
}

type IDParams struct {
	ID string `json:"id"`
}

type ListTasksParams struct {
	ProjectID string         `json:"project_id"`
	Query     string         `json:"query,omitempty"`
	Tags      []string       `json:"tags,omitempty"`
	Status    model.Status   `json:"status,omitempty"`
	Priority  model.Priority `json:"priority,omitempty"`
}

type SearchTasksParams struct {
	ProjectID       string         `json:"project_id,omitempty"`
	IncludeArchived bool           `json:"include_archived,omitempty"`
	Query           string         `json:"query,omitempty"`
	Tags            []string       `json:"tags,omitempty"`
	Status          model.Status   `json:"status,omitempty"`
	Priority        model.Priority `json:"priority,omitempty"`
	DueBefore       string         `json:"due_before,omitempty"`
	DueAfter        string         `json:"due_after,omitempty"`
	HasDueDate      *bool          `json:"has_due_date,omitempty"`
	Limit           int            `json:"limit,omitempty"`
}

type TaskSearchItem struct {
	Task        model.Task `json:"task"`
	ProjectName string     `json:"project_name"`
}

type AttachTaskFileParams struct {
	TaskID string `json:"task_id"`
	Path   string `json:"path"`
}

type TaskDetail struct {
	Task        model.Task         `json:"task"`
	Attachments []model.Attachment `json:"attachments"`
}

type CreateTaskParams struct {
	ProjectID   string          `json:"project_id"`
	Title       string          `json:"title"`
	Description string          `json:"description,omitempty"`
	Status      model.Status    `json:"status,omitempty"`
	Priority    model.Priority  `json:"priority,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
	Subtasks    []model.Subtask `json:"subtasks,omitempty"`
	DueDate     string          `json:"due_date,omitempty"`
}

// UpdateTaskParams uses pointer fields so callers can patch individual task fields
// without accidentally wiping description, tags, subtasks, or due_date.
type UpdateTaskParams struct {
	ID          string           `json:"id"`
	Title       *string          `json:"title,omitempty"`
	Description *string          `json:"description,omitempty"`
	Status      *model.Status    `json:"status,omitempty"`
	Priority    *model.Priority  `json:"priority,omitempty"`
	Tags        *[]string        `json:"tags,omitempty"`
	Subtasks    *[]model.Subtask `json:"subtasks,omitempty"`
	DueDate     *string          `json:"due_date,omitempty"`
}

type MoveTaskParams struct {
	ID     string       `json:"id"`
	Status model.Status `json:"status"`
	Index  int          `json:"index"`
}

type ListJobsParams struct {
	View        string             `json:"view,omitempty"`
	Query       string             `json:"query,omitempty"`
	Stage       model.JobStage     `json:"stage,omitempty"`
	Outcome     model.JobOutcome   `json:"outcome,omitempty"`
	AppliedDate string             `json:"applied_date,omitempty"`
	Sort        string             `json:"sort,omitempty"`
}

type ListJobsResult struct {
	Stats        model.JobApplicationStats `json:"stats"`
	Applications []model.JobApplication    `json:"applications"`
}

type JobDetail struct {
	Application model.JobApplication        `json:"application"`
	Events      []model.JobApplicationEvent `json:"events"`
}

type CreateJobParams struct {
	CompanyName    string         `json:"company_name"`
	PositionName   string         `json:"position_name"`
	AppliedDate    string         `json:"applied_date,omitempty"`
	Stage          model.JobStage `json:"stage,omitempty"`
	ApplicationURL string         `json:"application_url,omitempty"`
	Channel        string         `json:"channel,omitempty"`
	Notes          string         `json:"notes,omitempty"`
}

// UpdateJobParams uses pointer fields so callers can update individual job fields safely.
type UpdateJobParams struct {
	ID             string  `json:"id"`
	CompanyName    *string `json:"company_name,omitempty"`
	PositionName   *string `json:"position_name,omitempty"`
	AppliedDate    *string `json:"applied_date,omitempty"`
	ApplicationURL *string `json:"application_url,omitempty"`
	Channel        *string `json:"channel,omitempty"`
	Notes          *string `json:"notes,omitempty"`
	ClosureReason  *string `json:"closure_reason,omitempty"`
}

type ChangeJobStageParams struct {
	ID      string         `json:"id"`
	ToStage model.JobStage `json:"to_stage"`
	Note    string         `json:"note,omitempty"`
}

type AddJobNoteParams struct {
	ApplicationID string `json:"application_id"`
	Note          string `json:"note"`
	OccurredDate  string `json:"occurred_date,omitempty"`
}

type BackfillJobStageParams struct {
	ApplicationID string         `json:"application_id"`
	Stage         model.JobStage `json:"stage"`
	OccurredDate  string         `json:"occurred_date"`
	Note          string         `json:"note,omitempty"`
}

type CloseJobParams struct {
	ID            string           `json:"id"`
	Outcome       model.JobOutcome `json:"outcome"`
	Reason        string           `json:"reason,omitempty"`
	StageOverride model.JobStage   `json:"stage_override,omitempty"`
}

type ReopenJobParams struct {
	ID   string `json:"id"`
	Note string `json:"note,omitempty"`
}

type RestoreTrashParams struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type OKResult struct {
	OK bool `json:"ok"`
}
