package mcp

import "aion/ipc"

type toolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	ipcMethod   string
}

func objSchema(props map[string]any, required ...string) map[string]any {
	if props == nil {
		props = map[string]any{}
	}
	schema := map[string]any{
		"type":                 "object",
		"properties":           props,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func strProp(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func enumProp(desc string, values ...string) map[string]any {
	return map[string]any{"type": "string", "description": desc, "enum": values}
}

func boolProp(desc string) map[string]any {
	return map[string]any{"type": "boolean", "description": desc}
}

func intProp(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}

func strArrayProp(desc string) map[string]any {
	return map[string]any{
		"type":        "array",
		"description": desc,
		"items":       map[string]any{"type": "string"},
	}
}

func subtasksProp(desc string) map[string]any {
	return map[string]any{
		"type":        "array",
		"description": desc,
		"items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id":    strProp("Optional existing subtask ID; omit for new subtasks."),
				"title": strProp("Subtask title."),
				"done":  boolProp("Whether the subtask is completed."),
			},
			"required":             []string{"title"},
			"additionalProperties": false,
		},
	}
}

var (
	taskStatuses  = []string{"planned", "in_progress", "done"}
	taskPriority  = []string{"none", "low", "medium", "high", "urgent"}
	jobStages     = []string{"applied", "screening", "assessment", "interview_1", "interview_2", "interview_3", "hr", "offer"}
	jobOutcomes   = []string{"rejected", "offer_accepted", "offer_declined", "withdrawn", "position_cancelled", "other"}
	jobViews      = []string{"all", "active", "offer", "closed"}
	jobSorts      = []string{"applied_desc", "applied_asc", "updated_desc", "updated_asc"}
	trashKindList = []string{"project", "task", "job_application"}
)

func builtinTools() []toolSpec {
	return []toolSpec{
		{
			Name:        "get_workspace_summary",
			Description: "Get a compact overview of the Aion workspace: active projects with task counts, archived project count, job application stats, up to 20 most recently updated active job applications, and trash count.",
			InputSchema: objSchema(nil),
			ipcMethod:   ipc.MethodWorkspaceSummary,
		},

		// Projects
		{
			Name:        "list_projects",
			Description: "List non-deleted Aion projects. Omit archived to list both active and archived projects.",
			InputSchema: objSchema(map[string]any{
				"archived": boolProp("Filter by archived state (true = archived only, false = active only)."),
			}),
			ipcMethod: ipc.MethodProjectsList,
		},
		{
			Name:        "create_project",
			Description: "Create a new project board in Aion.",
			InputSchema: objSchema(map[string]any{
				"name":        strProp("Project display name."),
				"icon":        strProp("Optional icon key."),
				"description": strProp("Optional project description."),
			}, "name"),
			ipcMethod: ipc.MethodProjectsCreate,
		},
		{
			Name:        "update_project",
			Description: "Update an existing project's name, icon, or description. Only provided fields are modified.",
			InputSchema: objSchema(map[string]any{
				"id":          strProp("Project ID."),
				"name":        strProp("New project name."),
				"icon":        strProp("New icon key."),
				"description": strProp("New project description."),
			}, "id"),
			ipcMethod: ipc.MethodProjectsUpdate,
		},
		{
			Name:        "archive_project",
			Description: "Archive or unarchive a project.",
			InputSchema: objSchema(map[string]any{
				"id":       strProp("Project ID."),
				"archived": boolProp("True to archive, false to unarchive."),
			}, "id", "archived"),
			ipcMethod: ipc.MethodProjectsArchive,
		},
		{
			Name:        "trash_project",
			Description: "Soft-delete a project into the Aion Trash (recoverable via restore_trash_item).",
			InputSchema: objSchema(map[string]any{
				"id": strProp("Project ID to move to Trash."),
			}, "id"),
			ipcMethod: ipc.MethodProjectsTrash,
		},

		// Tasks
		{
			Name:        "list_tasks",
			Description: "List non-deleted tasks inside a project, with optional search query, status, priority, and tag filters.",
			InputSchema: objSchema(map[string]any{
				"project_id": strProp("Project ID."),
				"query":      strProp("Optional case-insensitive substring matched against title, description, or tags."),
				"status":     enumProp("Optional board column status filter.", taskStatuses...),
				"priority":   enumProp("Optional priority filter.", taskPriority...),
				"tags":       strArrayProp("Optional exact tags filter (task must match all listed tags)."),
			}, "project_id"),
			ipcMethod: ipc.MethodTasksList,
		},
		{
			Name:        "get_task",
			Description: "Get full details of a single task, including Markdown description, subtasks, tags, and attachment metadata.",
			InputSchema: objSchema(map[string]any{
				"id": strProp("Task ID."),
			}, "id"),
			ipcMethod: ipc.MethodTasksGet,
		},
		{
			Name:        "create_task",
			Description: "Create a new task card in a project board. Defaults status to 'planned' and priority to 'none'.",
			InputSchema: objSchema(map[string]any{
				"project_id":  strProp("Target project ID."),
				"title":       strProp("Task title."),
				"description": strProp("Optional Markdown description."),
				"status":      enumProp("Initial status column.", taskStatuses...),
				"priority":    enumProp("Task priority.", taskPriority...),
				"tags":        strArrayProp("Optional list of tags."),
				"subtasks":    subtasksProp("Optional list of subtasks."),
				"due_date":    strProp("Optional due date in YYYY-MM-DD format."),
			}, "project_id", "title"),
			ipcMethod: ipc.MethodTasksCreate,
		},
		{
			Name:        "update_task",
			Description: "Partially update a task. Only fields explicitly supplied are modified; omitted fields are preserved.",
			InputSchema: objSchema(map[string]any{
				"id":          strProp("Task ID."),
				"title":       strProp("New task title."),
				"description": strProp("New Markdown description."),
				"status":      enumProp("New status column.", taskStatuses...),
				"priority":    enumProp("New priority.", taskPriority...),
				"tags":        strArrayProp("Replacement tag list (pass [] to clear tags)."),
				"subtasks":    subtasksProp("Replacement subtask list (preserve existing IDs when editing)."),
				"due_date":    strProp("New due date in YYYY-MM-DD format, or empty string to clear."),
			}, "id"),
			ipcMethod: ipc.MethodTasksUpdate,
		},
		{
			Name:        "move_task",
			Description: "Move a task to a status column and 0-based position index on the Kanban board.",
			InputSchema: objSchema(map[string]any{
				"id":     strProp("Task ID."),
				"status": enumProp("Target board status column.", taskStatuses...),
				"index":  intProp("0-based target index in the column (0 = top)."),
			}, "id", "status", "index"),
			ipcMethod: ipc.MethodTasksMove,
		},
		{
			Name:        "trash_task",
			Description: "Soft-delete a task into the Aion Trash (recoverable via restore_trash_item).",
			InputSchema: objSchema(map[string]any{
				"id": strProp("Task ID to move to Trash."),
			}, "id"),
			ipcMethod: ipc.MethodTasksTrash,
		},

		// Job Applications
		{
			Name:        "list_job_applications",
			Description: "List non-deleted job applications with aggregate stats and optional view, search query, stage, outcome, date, and sort options.",
			InputSchema: objSchema(map[string]any{
				"view":         enumProp("Slice view: all, active, offer, or closed.", jobViews...),
				"query":        strProp("Case-insensitive substring matched against company or position name."),
				"stage":        enumProp("Optional pipeline stage filter.", jobStages...),
				"outcome":      enumProp("Optional closure outcome filter.", jobOutcomes...),
				"applied_date": strProp("Optional exact applied date filter (YYYY-MM-DD)."),
				"sort":         enumProp("Sort order.", jobSorts...),
			}),
			ipcMethod: ipc.MethodJobsList,
		},
		{
			Name:        "get_job_application",
			Description: "Get a job application and its complete chronological timeline events.",
			InputSchema: objSchema(map[string]any{
				"id": strProp("Job application ID."),
			}, "id"),
			ipcMethod: ipc.MethodJobsGet,
		},
		{
			Name:        "create_job_application",
			Description: "Create a new job application record in the 求职 module.",
			InputSchema: objSchema(map[string]any{
				"company_name":    strProp("Company name."),
				"position_name":   strProp("Job title / position name."),
				"applied_date":    strProp("Applied date in YYYY-MM-DD format (defaults to today if omitted)."),
				"stage":           enumProp("Initial pipeline stage (defaults to applied).", jobStages...),
				"application_url": strProp("Optional http/https job posting or portal URL."),
				"channel":         strProp("Optional application source/channel (e.g. 官网, 内推, Boss直聘)."),
				"notes":           strProp("Optional notes."),
			}, "company_name", "position_name"),
			ipcMethod: ipc.MethodJobsCreate,
		},
		{
			Name:        "update_job_application",
			Description: "Partially update job application metadata without changing workflow stage/lifecycle. Only supplied fields are modified.",
			InputSchema: objSchema(map[string]any{
				"id":              strProp("Job application ID."),
				"company_name":    strProp("New company name."),
				"position_name":   strProp("New position name."),
				"applied_date":    strProp("New applied date (YYYY-MM-DD)."),
				"application_url": strProp("New http/https URL, or empty string to clear."),
				"channel":         strProp("New channel."),
				"notes":           strProp("New notes."),
				"closure_reason":  strProp("Updated closure reason (only valid when lifecycle is closed)."),
			}, "id"),
			ipcMethod: ipc.MethodJobsUpdate,
		},
		{
			Name:        "change_job_stage",
			Description: "Advance or change a job application's current pipeline stage and append a stage_changed timeline event.",
			InputSchema: objSchema(map[string]any{
				"id":       strProp("Job application ID."),
				"to_stage": enumProp("New current pipeline stage.", jobStages...),
				"note":     strProp("Optional note recorded on the stage change event."),
			}, "id", "to_stage"),
			ipcMethod: ipc.MethodJobsChangeStage,
		},
		{
			Name:        "add_job_note",
			Description: "Append a manual note entry to a job application's timeline without modifying current stage or lifecycle.",
			InputSchema: objSchema(map[string]any{
				"application_id": strProp("Job application ID."),
				"note":           strProp("Timeline note content."),
				"occurred_date":  strProp("Optional calendar date (YYYY-MM-DD); defaults to today."),
			}, "application_id", "note"),
			ipcMethod: ipc.MethodJobsAddNote,
		},
		{
			Name:        "backfill_job_stage",
			Description: "Insert a historical stage event into a job application's timeline without overwriting its current stage or lifecycle.",
			InputSchema: objSchema(map[string]any{
				"application_id": strProp("Job application ID."),
				"stage":          enumProp("Historical stage to backfill.", jobStages...),
				"occurred_date":  strProp("Historical occurrence date (YYYY-MM-DD)."),
				"note":           strProp("Optional note for the backfilled stage."),
			}, "application_id", "stage", "occurred_date"),
			ipcMethod: ipc.MethodJobsBackfillStage,
		},
		{
			Name:        "close_job_application",
			Description: "Close an active job application with a closure outcome and optional reason. Offer outcomes require stage='offer' or stage_override='offer'.",
			InputSchema: objSchema(map[string]any{
				"id":             strProp("Job application ID."),
				"outcome":        enumProp("Closure outcome.", jobOutcomes...),
				"reason":         strProp("Optional closure reason."),
				"stage_override": enumProp("Optional stage override applied atomically when closing (e.g. offer).", jobStages...),
			}, "id", "outcome"),
			ipcMethod: ipc.MethodJobsClose,
		},
		{
			Name:        "reopen_job_application",
			Description: "Reopen a previously closed job application and record a reopened timeline event.",
			InputSchema: objSchema(map[string]any{
				"id":   strProp("Job application ID."),
				"note": strProp("Optional note explaining why it was reopened."),
			}, "id"),
			ipcMethod: ipc.MethodJobsReopen,
		},
		{
			Name:        "trash_job_application",
			Description: "Soft-delete a job application into the Aion Trash (recoverable via restore_trash_item).",
			InputSchema: objSchema(map[string]any{
				"id": strProp("Job application ID to move to Trash."),
			}, "id"),
			ipcMethod: ipc.MethodJobsTrash,
		},

		// Trash (read + restore only; never permanent delete)
		{
			Name:        "list_trash",
			Description: "List soft-deleted projects, tasks, and job applications currently in the Aion Trash.",
			InputSchema: objSchema(nil),
			ipcMethod:   ipc.MethodTrashList,
		},
		{
			Name:        "restore_trash_item",
			Description: "Restore a soft-deleted project, task, or job application from the Aion Trash.",
			InputSchema: objSchema(map[string]any{
				"kind": enumProp("Trash item kind.", trashKindList...),
				"id":   strProp("ID of the trashed item to restore."),
			}, "kind", "id"),
			ipcMethod: ipc.MethodTrashRestore,
		},
	}
}
