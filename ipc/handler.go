package ipc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"aion/backend"
	"aion/backend/jobapplications"
	"aion/backend/model"
	"aion/backend/projects"
)

// Handler dispatches safe local RPC methods against an open backend.App and
// emits ChangeEvents after committed mutations.
type Handler struct {
	app      *backend.App
	onChange func(ChangeEvent)
}

// NewHandler creates an IPC handler backed by app. onChange is optional.
func NewHandler(app *backend.App, onChange func(ChangeEvent)) *Handler {
	return &Handler{app: app, onChange: onChange}
}

func (h *Handler) notify(ev ChangeEvent) {
	if h.onChange != nil {
		h.onChange(ev)
	}
}

func decodeParams(raw json.RawMessage, dst any) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("%w: invalid parameters: %v", model.ErrInvalid, err)
	}
	return nil
}

// Call executes one named RPC method against the local backend.
func (h *Handler) Call(ctx context.Context, method string, rawParams json.RawMessage) (any, error) {
	switch method {
	case MethodWorkspaceSummary:
		return h.workspaceSummary(ctx)

	case MethodProjectsList:
		var p ListProjectsParams
		if err := decodeParams(rawParams, &p); err != nil {
			return nil, err
		}
		list, err := h.app.Projects.ListProjects(ctx, projects.ProjectFilter{Archived: p.Archived})
		if err != nil {
			return nil, err
		}
		if list == nil {
			list = []model.Project{}
		}
		return list, nil

	case MethodProjectsCreate:
		var p CreateProjectParams
		if err := decodeParams(rawParams, &p); err != nil {
			return nil, err
		}
		created, err := h.app.Projects.CreateProject(ctx, projects.ProjectInput{
			Name:        p.Name,
			Icon:        p.Icon,
			Description: p.Description,
		})
		if err != nil {
			return nil, err
		}
		h.notify(ChangeEvent{Scope: ScopeProjects, ProjectID: created.ID})
		return created, nil

	case MethodProjectsUpdate:
		var p UpdateProjectParams
		if err := decodeParams(rawParams, &p); err != nil {
			return nil, err
		}
		if strings.TrimSpace(p.ID) == "" {
			return nil, fmt.Errorf("%w: id is required", model.ErrInvalid)
		}
		existing, err := h.findProject(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		in := projects.ProjectInput{
			Name:        existing.Name,
			Icon:        existing.Icon,
			Description: existing.Description,
		}
		if p.Name != nil {
			in.Name = *p.Name
		}
		if p.Icon != nil {
			in.Icon = *p.Icon
		}
		if p.Description != nil {
			in.Description = *p.Description
		}
		updated, err := h.app.Projects.UpdateProject(ctx, p.ID, in)
		if err != nil {
			return nil, err
		}
		h.notify(ChangeEvent{Scope: ScopeProjects, ProjectID: updated.ID})
		return updated, nil

	case MethodProjectsArchive:
		var p ArchiveProjectParams
		if err := decodeParams(rawParams, &p); err != nil {
			return nil, err
		}
		if strings.TrimSpace(p.ID) == "" {
			return nil, fmt.Errorf("%w: id is required", model.ErrInvalid)
		}
		updated, err := h.app.Projects.ArchiveProject(ctx, p.ID, p.Archived)
		if err != nil {
			return nil, err
		}
		h.notify(ChangeEvent{Scope: ScopeProjects, ProjectID: updated.ID})
		return updated, nil

	case MethodProjectsTrash:
		var p IDParams
		if err := decodeParams(rawParams, &p); err != nil {
			return nil, err
		}
		if strings.TrimSpace(p.ID) == "" {
			return nil, fmt.Errorf("%w: id is required", model.ErrInvalid)
		}
		if err := h.app.Projects.DeleteProject(ctx, p.ID); err != nil {
			return nil, err
		}
		h.notify(ChangeEvent{Scope: ScopeTrash, ProjectID: p.ID})
		return OKResult{OK: true}, nil

	case MethodTasksList:
		var p ListTasksParams
		if err := decodeParams(rawParams, &p); err != nil {
			return nil, err
		}
		if strings.TrimSpace(p.ProjectID) == "" {
			return nil, fmt.Errorf("%w: project_id is required", model.ErrInvalid)
		}
		list, err := h.app.Projects.ListTasks(ctx, p.ProjectID, projects.TaskFilter{
			Query:    p.Query,
			Tags:     p.Tags,
			Status:   p.Status,
			Priority: p.Priority,
		})
		if err != nil {
			return nil, err
		}
		if list == nil {
			list = []model.Task{}
		}
		return list, nil

	case MethodTasksSearch:
		var p SearchTasksParams
		if err := decodeParams(rawParams, &p); err != nil {
			return nil, err
		}
		tasks, err := h.app.Projects.SearchTasks(ctx, projects.SearchTasksFilter{
			ProjectID:       p.ProjectID,
			IncludeArchived: p.IncludeArchived,
			Query:           p.Query,
			Tags:            p.Tags,
			Status:          p.Status,
			Priority:        p.Priority,
			DueBefore:       p.DueBefore,
			DueAfter:        p.DueAfter,
			HasDueDate:      p.HasDueDate,
		})
		if err != nil {
			return nil, err
		}
		limit := p.Limit
		if limit <= 0 {
			limit = 50
		} else if limit > 200 {
			limit = 200
		}
		if len(tasks) > limit {
			tasks = tasks[:limit]
		}
		allProjects, err := h.app.Projects.ListProjects(ctx, projects.ProjectFilter{})
		if err != nil {
			return nil, err
		}
		projNames := make(map[string]string, len(allProjects))
		for _, proj := range allProjects {
			projNames[proj.ID] = proj.Name
		}
		out := make([]TaskSearchItem, 0, len(tasks))
		for _, t := range tasks {
			out = append(out, TaskSearchItem{
				Task:        t,
				ProjectName: projNames[t.ProjectID],
			})
		}
		return out, nil

	case MethodTasksAttach:
		var p AttachTaskFileParams
		if err := decodeParams(rawParams, &p); err != nil {
			return nil, err
		}
		if strings.TrimSpace(p.TaskID) == "" || strings.TrimSpace(p.Path) == "" {
			return nil, fmt.Errorf("%w: task_id and path are required", model.ErrInvalid)
		}
		task, err := h.app.Projects.GetTask(ctx, p.TaskID)
		if err != nil {
			return nil, err
		}
		att, err := h.app.Attachments.Import(ctx, p.TaskID, p.Path)
		if err != nil {
			return nil, err
		}
		h.notify(ChangeEvent{Scope: ScopeTasks, ProjectID: task.ProjectID, TaskID: task.ID})
		return att, nil

	case MethodTasksGet:
		var p IDParams
		if err := decodeParams(rawParams, &p); err != nil {
			return nil, err
		}
		if strings.TrimSpace(p.ID) == "" {
			return nil, fmt.Errorf("%w: id is required", model.ErrInvalid)
		}
		task, err := h.app.Projects.GetTask(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		atts, err := h.app.Attachments.List(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		if atts == nil {
			atts = []model.Attachment{}
		}
		return TaskDetail{Task: task, Attachments: atts}, nil

	case MethodTasksCreate:
		var p CreateTaskParams
		if err := decodeParams(rawParams, &p); err != nil {
			return nil, err
		}
		if strings.TrimSpace(p.ProjectID) == "" {
			return nil, fmt.Errorf("%w: project_id is required", model.ErrInvalid)
		}
		created, err := h.app.Projects.CreateTask(ctx, p.ProjectID, projects.TaskInput{
			Title:       p.Title,
			Description: p.Description,
			Status:      p.Status,
			Priority:    p.Priority,
			Tags:        p.Tags,
			Subtasks:    p.Subtasks,
			DueDate:     p.DueDate,
		})
		if err != nil {
			return nil, err
		}
		h.notify(ChangeEvent{Scope: ScopeTasks, ProjectID: created.ProjectID, TaskID: created.ID})
		return created, nil

	case MethodTasksUpdate:
		var p UpdateTaskParams
		if err := decodeParams(rawParams, &p); err != nil {
			return nil, err
		}
		if strings.TrimSpace(p.ID) == "" {
			return nil, fmt.Errorf("%w: id is required", model.ErrInvalid)
		}
		existing, err := h.app.Projects.GetTask(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		in := projects.TaskInput{
			Title:       existing.Title,
			Description: existing.Description,
			Status:      existing.Status,
			Priority:    existing.Priority,
			Tags:        existing.Tags,
			Subtasks:    existing.Subtasks,
			DueDate:     existing.DueDate,
		}
		if p.Title != nil {
			in.Title = *p.Title
		}
		if p.Description != nil {
			in.Description = *p.Description
		}
		if p.Status != nil {
			in.Status = *p.Status
		}
		if p.Priority != nil {
			in.Priority = *p.Priority
		}
		if p.Tags != nil {
			in.Tags = *p.Tags
		}
		if p.Subtasks != nil {
			in.Subtasks = *p.Subtasks
		}
		if p.DueDate != nil {
			in.DueDate = *p.DueDate
		}
		updated, err := h.app.Projects.UpdateTask(ctx, p.ID, in)
		if err != nil {
			return nil, err
		}
		h.notify(ChangeEvent{Scope: ScopeTasks, ProjectID: updated.ProjectID, TaskID: updated.ID})
		return updated, nil

	case MethodTasksMove:
		var p MoveTaskParams
		if err := decodeParams(rawParams, &p); err != nil {
			return nil, err
		}
		if strings.TrimSpace(p.ID) == "" {
			return nil, fmt.Errorf("%w: id is required", model.ErrInvalid)
		}
		moved, err := h.app.Projects.MoveTask(ctx, p.ID, p.Status, p.Index)
		if err != nil {
			return nil, err
		}
		h.notify(ChangeEvent{Scope: ScopeTasks, ProjectID: moved.ProjectID, TaskID: moved.ID})
		return moved, nil

	case MethodTasksTrash:
		var p IDParams
		if err := decodeParams(rawParams, &p); err != nil {
			return nil, err
		}
		if strings.TrimSpace(p.ID) == "" {
			return nil, fmt.Errorf("%w: id is required", model.ErrInvalid)
		}
		var projectID string
		if existing, err := h.app.Projects.GetTask(ctx, p.ID); err == nil {
			projectID = existing.ProjectID
		}
		if err := h.app.Projects.DeleteTask(ctx, p.ID); err != nil {
			return nil, err
		}
		h.notify(ChangeEvent{Scope: ScopeTrash, ProjectID: projectID, TaskID: p.ID})
		return OKResult{OK: true}, nil

	case MethodJobsList:
		var p ListJobsParams
		if err := decodeParams(rawParams, &p); err != nil {
			return nil, err
		}
		all, err := h.app.JobApplications.List(ctx)
		if err != nil {
			return nil, err
		}
		stats := jobapplications.ComputeStats(all)
		filtered := jobapplications.FilterApplications(all, jobapplications.Filter{
			View:        jobapplications.JobView(p.View),
			Query:       p.Query,
			Stage:       p.Stage,
			Outcome:     p.Outcome,
			AppliedDate: p.AppliedDate,
		}, jobapplications.JobSort(p.Sort))
		if filtered == nil {
			filtered = []model.JobApplication{}
		}
		return ListJobsResult{Stats: stats, Applications: filtered}, nil

	case MethodJobsGet:
		var p IDParams
		if err := decodeParams(rawParams, &p); err != nil {
			return nil, err
		}
		if strings.TrimSpace(p.ID) == "" {
			return nil, fmt.Errorf("%w: id is required", model.ErrInvalid)
		}
		app, err := h.app.JobApplications.Get(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		events, err := h.app.JobApplications.ListEvents(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		if events == nil {
			events = []model.JobApplicationEvent{}
		}
		return JobDetail{Application: app, Events: events}, nil

	case MethodJobsCreate:
		var p CreateJobParams
		if err := decodeParams(rawParams, &p); err != nil {
			return nil, err
		}
		created, err := h.app.JobApplications.Create(ctx, jobapplications.CreateInput{
			CompanyName:    p.CompanyName,
			PositionName:   p.PositionName,
			AppliedDate:    p.AppliedDate,
			Stage:          p.Stage,
			ApplicationURL: p.ApplicationURL,
			Channel:        p.Channel,
			Notes:          p.Notes,
		})
		if err != nil {
			return nil, err
		}
		h.notify(ChangeEvent{Scope: ScopeJobs, JobID: created.ID})
		return created, nil

	case MethodJobsUpdate:
		var p UpdateJobParams
		if err := decodeParams(rawParams, &p); err != nil {
			return nil, err
		}
		if strings.TrimSpace(p.ID) == "" {
			return nil, fmt.Errorf("%w: id is required", model.ErrInvalid)
		}
		existing, err := h.app.JobApplications.Get(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		in := jobapplications.UpdateDetailsInput{
			ID:             p.ID,
			CompanyName:    existing.CompanyName,
			PositionName:   existing.PositionName,
			AppliedDate:    existing.AppliedDate,
			ApplicationURL: existing.ApplicationURL,
			Channel:        existing.Channel,
			Notes:          existing.Notes,
			ClosureReason:  p.ClosureReason,
		}
		if p.CompanyName != nil {
			in.CompanyName = *p.CompanyName
		}
		if p.PositionName != nil {
			in.PositionName = *p.PositionName
		}
		if p.AppliedDate != nil {
			in.AppliedDate = *p.AppliedDate
		}
		if p.ApplicationURL != nil {
			in.ApplicationURL = *p.ApplicationURL
		}
		if p.Channel != nil {
			in.Channel = *p.Channel
		}
		if p.Notes != nil {
			in.Notes = *p.Notes
		}
		updated, err := h.app.JobApplications.UpdateDetails(ctx, in)
		if err != nil {
			return nil, err
		}
		h.notify(ChangeEvent{Scope: ScopeJobs, JobID: updated.ID})
		return updated, nil

	case MethodJobsChangeStage:
		var p ChangeJobStageParams
		if err := decodeParams(rawParams, &p); err != nil {
			return nil, err
		}
		updated, err := h.app.JobApplications.ChangeStage(ctx, jobapplications.ChangeStageInput{
			ID:      p.ID,
			ToStage: p.ToStage,
			Note:    p.Note,
		})
		if err != nil {
			return nil, err
		}
		h.notify(ChangeEvent{Scope: ScopeJobs, JobID: updated.ID})
		return updated, nil

	case MethodJobsAddNote:
		var p AddJobNoteParams
		if err := decodeParams(rawParams, &p); err != nil {
			return nil, err
		}
		ev, err := h.app.JobApplications.AddNoteEvent(ctx, jobapplications.AddNoteInput{
			ApplicationID: p.ApplicationID,
			Note:          p.Note,
			OccurredDate:  p.OccurredDate,
		})
		if err != nil {
			return nil, err
		}
		h.notify(ChangeEvent{Scope: ScopeJobs, JobID: p.ApplicationID})
		return ev, nil

	case MethodJobsBackfillStage:
		var p BackfillJobStageParams
		if err := decodeParams(rawParams, &p); err != nil {
			return nil, err
		}
		ev, err := h.app.JobApplications.BackfillStageEvent(ctx, jobapplications.BackfillStageInput{
			ApplicationID: p.ApplicationID,
			Stage:         p.Stage,
			OccurredDate:  p.OccurredDate,
			Note:          p.Note,
		})
		if err != nil {
			return nil, err
		}
		h.notify(ChangeEvent{Scope: ScopeJobs, JobID: p.ApplicationID})
		return ev, nil

	case MethodJobsClose:
		var p CloseJobParams
		if err := decodeParams(rawParams, &p); err != nil {
			return nil, err
		}
		updated, err := h.app.JobApplications.Close(ctx, jobapplications.CloseInput{
			ID:            p.ID,
			Outcome:       p.Outcome,
			Reason:        p.Reason,
			StageOverride: p.StageOverride,
		})
		if err != nil {
			return nil, err
		}
		h.notify(ChangeEvent{Scope: ScopeJobs, JobID: updated.ID})
		return updated, nil

	case MethodJobsReopen:
		var p ReopenJobParams
		if err := decodeParams(rawParams, &p); err != nil {
			return nil, err
		}
		updated, err := h.app.JobApplications.Reopen(ctx, jobapplications.ReopenInput{
			ID:   p.ID,
			Note: p.Note,
		})
		if err != nil {
			return nil, err
		}
		h.notify(ChangeEvent{Scope: ScopeJobs, JobID: updated.ID})
		return updated, nil

	case MethodJobsTrash:
		var p IDParams
		if err := decodeParams(rawParams, &p); err != nil {
			return nil, err
		}
		if strings.TrimSpace(p.ID) == "" {
			return nil, fmt.Errorf("%w: id is required", model.ErrInvalid)
		}
		if err := h.app.JobApplications.Delete(ctx, p.ID); err != nil {
			return nil, err
		}
		h.notify(ChangeEvent{Scope: ScopeTrash, JobID: p.ID})
		return OKResult{OK: true}, nil

	case MethodTrashList:
		items, err := h.app.ListTrash(ctx)
		if err != nil {
			return nil, err
		}
		if items == nil {
			items = []model.TrashItem{}
		}
		return items, nil

	case MethodTrashRestore:
		var p RestoreTrashParams
		if err := decodeParams(rawParams, &p); err != nil {
			return nil, err
		}
		if strings.TrimSpace(p.Kind) == "" || strings.TrimSpace(p.ID) == "" {
			return nil, fmt.Errorf("%w: kind and id are required", model.ErrInvalid)
		}
		if err := h.app.RestoreTrash(ctx, p.Kind, p.ID); err != nil {
			return nil, err
		}
		h.notify(ChangeEvent{Scope: ScopeTrash})
		return OKResult{OK: true}, nil

	default:
		return nil, &RPCError{Code: ErrCodeUnknownMethod, Message: fmt.Sprintf("unknown method %q", method)}
	}
}

func (h *Handler) findProject(ctx context.Context, id string) (model.Project, error) {
	all, err := h.app.Projects.ListProjects(ctx, projects.ProjectFilter{})
	if err != nil {
		return model.Project{}, err
	}
	for _, p := range all {
		if p.ID == id {
			return p, nil
		}
	}
	return model.Project{}, model.ErrNotFound
}

func (h *Handler) workspaceSummary(ctx context.Context) (WorkspaceSummary, error) {
	allProjects, err := h.app.Projects.ListProjects(ctx, projects.ProjectFilter{})
	if err != nil {
		return WorkspaceSummary{}, err
	}
	counts, err := h.app.Projects.CountTasks(ctx)
	if err != nil {
		return WorkspaceSummary{}, err
	}
	activeProjects := make([]ProjectSummaryItem, 0, len(allProjects))
	archivedCount := 0
	for _, p := range allProjects {
		if p.Archived {
			archivedCount++
			continue
		}
		c := counts[p.ID]
		item := ProjectSummaryItem{
			Project:    p,
			Planned:    c[model.Planned],
			InProgress: c[model.InProgress],
			Done:       c[model.Done],
		}
		item.Total = item.Planned + item.InProgress + item.Done
		activeProjects = append(activeProjects, item)
	}

	allJobs, err := h.app.JobApplications.List(ctx)
	if err != nil {
		return WorkspaceSummary{}, err
	}
	stats := jobapplications.ComputeStats(allJobs)
	activeJobs := jobapplications.FilterApplications(allJobs, jobapplications.Filter{
		View: jobapplications.ViewActive,
	}, jobapplications.SortUpdatedDesc)
	if len(activeJobs) > 20 {
		activeJobs = activeJobs[:20]
	}
	if activeJobs == nil {
		activeJobs = []model.JobApplication{}
	}

	trashItems, err := h.app.ListTrash(ctx)
	if err != nil {
		return WorkspaceSummary{}, err
	}

	return WorkspaceSummary{
		Projects:      activeProjects,
		ArchivedCount: archivedCount,
		JobStats:      stats,
		ActiveJobs:    activeJobs,
		TrashCount:    len(trashItems),
	}, nil
}
