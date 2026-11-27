---
name: aion-workbench
description: Work with the user's local Aion desktop workbench (macOS / Windows) via Aion MCP tools (projects, Kanban tasks, attachments, job applications, timelines, and trash). Use when the user asks to plan tasks, inspect today's focus, log or update job applications from a JD or interview notes, attach local files to tasks, or review progress in Aion.
---

# Aion Local Workbench Skill

Use Aion's local MCP server (`aion`) as the structured source of truth for the user's projects, tasks, and job applications.

## Operating Rules

1. **Start with context**: Call `get_workspace_summary` before creating or editing items so you reuse existing project IDs and avoid duplicate records.
2. **Use partial updates safely**: `update_project`, `update_task`, and `update_job_application` merge only the fields you pass. Never resend empty strings/arrays unless you explicitly intend to clear that field. When editing `subtasks`, read `get_task` first and preserve existing subtask `id` values.
3. **Date format**: Always use `YYYY-MM-DD` (local calendar date) for `due_date`, `applied_date`, and `occurred_date`.
4. **Soft-delete safety**: `trash_project`, `trash_task`, and `trash_job_application` move items to Aion's Trash (never permanent deletion). If anything was removed by mistake, recover it with `restore_trash_item`.

## Common Workflows

### 1. Daily Focus & Review (每日聚焦 / 早晚复盘)
1. Call `get_workspace_summary` to inspect active projects, task counts, and active job applications.
2. Call `search_tasks` with `due_before` (e.g., today + 7 days) and/or `status: "in_progress"` to pull urgent, overdue, and in-progress items across all active projects.
3. Present a concise, scannable brief:
   - Top 3 highest-leverage tasks for today (highlighting overdue/due-soon items)
   - Active job applications needing follow-up or upcoming interview prep
   - Offer to update statuses, reschedule due dates, or add follow-up tasks.

### 2. Log a Job Application from a JD & Break Down Prep Tasks
1. Call `list_job_applications(query="<company>")` first to check for duplicate active applications.
2. Call `create_job_application` with `company_name`, `position_name`, `applied_date` (omit for today), `application_url` (valid `http`/`https`), `channel`, and concise `notes` (key stack, compensation/location hints, JD summary).
3. If the user wants preparation tasks, find or create a relevant project (`list_projects` / `create_project`) and add structured tasks via `create_task` with `subtasks`, `priority`, and `due_date`.
4. If the user has a local resume, JD PDF, or notes file, attach it directly using `attach_file_to_task`.

### 3. Post-Interview Debrief & Timeline Maintenance
Choose the exact timeline operation matching the user's intent:
- **Stage moved forward**: `change_job_stage` (updates current stage and appends a `stage_changed` event with optional `note`).
- **Notes without stage change**: `add_job_note` (appends a `note_added` event; supports historical `occurred_date`).
- **Missing historical stage**: `backfill_job_stage` (inserts a past stage event at `occurred_date` without overwriting the current stage/lifecycle).
- **Process ended**: `close_job_application` (for `offer_accepted` / `offer_declined`, set `stage_override: "offer"` if not already at `offer`).
- **Re-engaged later**: `reopen_job_application`.
