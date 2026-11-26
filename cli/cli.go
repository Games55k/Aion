// Package cli implements Aion's lightweight local command-line interface.
// Every command routes through ipc.Router so writes transparently live-sync
// with a running Aion GUI over Unix socket and fall back to short-lived direct
// backend access when the GUI is closed.
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"aion/backend/model"
	"aion/ipc"
)

// IsSubcommand reports whether arg is a recognized CLI subcommand.
func IsSubcommand(arg string) bool {
	switch arg {
	case "summary", "project", "task", "job", "trash", "help", "-h", "--help":
		return true
	default:
		return false
	}
}

// Run executes a CLI subcommand and writes output to stdout/stderr.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stdout)
		return nil
	}
	switch args[0] {
	case "help", "-h", "--help":
		printUsage(stdout)
		return nil
	case "summary":
		return runSummary(ctx, args[1:], stdout, stderr)
	case "project":
		return runProject(ctx, args[1:], stdout, stderr)
	case "task":
		return runTask(ctx, args[1:], stdout, stderr)
	case "job":
		return runJob(ctx, args[1:], stdout, stderr)
	case "trash":
		return runTrash(ctx, args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown command %q (run 'aion help' for usage)", args[0])
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, `Aion Local Workbench CLI

Usage:
  aion                                         Launch the native macOS GUI
  aion mcp [-data-dir <dir>]                   Run stdio MCP server for AI agents
  aion summary [-data-dir <dir>] [-json]       Show workspace summary

  aion project list|add|archive|trash [flags]  Manage projects
  aion task list|add|move|done|attach|trash    Manage tasks across projects
  aion job list|add|stage|note|close|reopen|trash  Manage job applications
  aion trash list|restore [flags]              Inspect or restore soft-deleted items

Global flags accepted by subcommands:
  -data-dir <dir>   Target a custom Aion data directory
  -json             Output formatted JSON instead of human-readable text`)
}

type commonFlags struct {
	dataDir string
	asJSON  bool
}

func newFlagSet(name string, stderr io.Writer, common *commonFlags) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&common.dataDir, "data-dir", "", "custom local data directory")
	fs.BoolVar(&common.asJSON, "json", false, "output JSON")
	return fs
}

func callRouter(ctx context.Context, dataDir, method string, params any) (json.RawMessage, error) {
	router, err := ipc.NewRouter(ipc.RouterOptions{DataDir: dataDir})
	if err != nil {
		return nil, err
	}
	return router.Call(ctx, method, params)
}

func writeJSON(w io.Writer, raw json.RawMessage) error {
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		_, err = fmt.Fprintln(w, string(raw))
		return err
	}
	_, err := fmt.Fprintln(w, buf.String())
	return err
}

func runSummary(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	var cf commonFlags
	fs := newFlagSet("summary", stderr, &cf)
	if err := fs.Parse(args); err != nil {
		return err
	}
	raw, err := callRouter(ctx, cf.dataDir, ipc.MethodWorkspaceSummary, nil)
	if err != nil {
		return err
	}
	if cf.asJSON {
		return writeJSON(stdout, raw)
	}
	var sum ipc.WorkspaceSummary
	if err := json.Unmarshal(raw, &sum); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Projects: %d active (%d archived) | Jobs: %d active, %d offer, %d closed | Trash: %d\n",
		len(sum.Projects), sum.ArchivedCount, sum.JobStats.Active, sum.JobStats.Offered, sum.JobStats.Closed, sum.TrashCount)
	for _, p := range sum.Projects {
		fmt.Fprintf(stdout, "  [project] %s (%s)  planned=%d in_progress=%d done=%d\n",
			p.Project.Name, p.Project.ID, p.Planned, p.InProgress, p.Done)
	}
	for _, j := range sum.ActiveJobs {
		fmt.Fprintf(stdout, "  [job]     %s · %s (%s)  stage=%s applied=%s\n",
			j.CompanyName, j.PositionName, j.ID, j.Stage, j.AppliedDate)
	}
	return nil
}

func runProject(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("project requires a subcommand: list, add, archive, trash")
	}
	sub, rest := args[0], args[1:]
	var cf commonFlags
	switch sub {
	case "list":
		fs := newFlagSet("project list", stderr, &cf)
		archived := fs.String("archived", "", "filter by archived state: true or false")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		var params ipc.ListProjectsParams
		switch strings.ToLower(strings.TrimSpace(*archived)) {
		case "true":
			v := true
			params.Archived = &v
		case "false":
			v := false
			params.Archived = &v
		case "":
		default:
			return fmt.Errorf("invalid -archived value %q (use true or false)", *archived)
		}
		raw, err := callRouter(ctx, cf.dataDir, ipc.MethodProjectsList, params)
		if err != nil {
			return err
		}
		if cf.asJSON {
			return writeJSON(stdout, raw)
		}
		var list []model.Project
		if err := json.Unmarshal(raw, &list); err != nil {
			return err
		}
		for _, p := range list {
			state := "active"
			if p.Archived {
				state = "archived"
			}
			fmt.Fprintf(stdout, "%s  [%s]  %s\n", p.ID, state, p.Name)
		}
		return nil

	case "add":
		fs := newFlagSet("project add", stderr, &cf)
		name := fs.String("name", "", "project name (required)")
		icon := fs.String("icon", "", "optional icon key")
		desc := fs.String("desc", "", "optional description")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		raw, err := callRouter(ctx, cf.dataDir, ipc.MethodProjectsCreate, ipc.CreateProjectParams{
			Name:        *name,
			Icon:        *icon,
			Description: *desc,
		})
		if err != nil {
			return err
		}
		if cf.asJSON {
			return writeJSON(stdout, raw)
		}
		var p model.Project
		if err := json.Unmarshal(raw, &p); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "created project %s (%s)\n", p.ID, p.Name)
		return nil

	case "archive":
		fs := newFlagSet("project archive", stderr, &cf)
		id := fs.String("id", "", "project ID (required)")
		undo := fs.Bool("undo", false, "unarchive instead of archiving")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		raw, err := callRouter(ctx, cf.dataDir, ipc.MethodProjectsArchive, ipc.ArchiveProjectParams{
			ID:       *id,
			Archived: !*undo,
		})
		if err != nil {
			return err
		}
		if cf.asJSON {
			return writeJSON(stdout, raw)
		}
		fmt.Fprintf(stdout, "updated archive=%v for project %s\n", !*undo, *id)
		return nil

	case "trash":
		fs := newFlagSet("project trash", stderr, &cf)
		id := fs.String("id", "", "project ID (required)")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		raw, err := callRouter(ctx, cf.dataDir, ipc.MethodProjectsTrash, ipc.IDParams{ID: *id})
		if err != nil {
			return err
		}
		if cf.asJSON {
			return writeJSON(stdout, raw)
		}
		fmt.Fprintf(stdout, "moved project %s to trash\n", *id)
		return nil

	default:
		return fmt.Errorf("unknown project subcommand %q", sub)
	}
}

func runTask(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("task requires a subcommand: list, add, move, done, attach, trash")
	}
	sub, rest := args[0], args[1:]
	var cf commonFlags
	switch sub {
	case "list":
		fs := newFlagSet("task list", stderr, &cf)
		projectID := fs.String("project", "", "optional project ID")
		query := fs.String("q", "", "search query")
		status := fs.String("status", "", "status filter: planned, in_progress, done")
		priority := fs.String("priority", "", "priority filter: none, low, medium, high, urgent")
		tagsCSV := fs.String("tags", "", "comma-separated required tags")
		dueBefore := fs.String("due-before", "", "inclusive due date upper bound (YYYY-MM-DD)")
		dueAfter := fs.String("due-after", "", "inclusive due date lower bound (YYYY-MM-DD)")
		includeArchived := fs.Bool("archived", false, "include archived projects")
		limit := fs.Int("limit", 50, "max tasks to return")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		raw, err := callRouter(ctx, cf.dataDir, ipc.MethodTasksSearch, ipc.SearchTasksParams{
			ProjectID:       *projectID,
			IncludeArchived: *includeArchived,
			Query:           *query,
			Tags:            splitCSV(*tagsCSV),
			Status:          model.Status(*status),
			Priority:        model.Priority(*priority),
			DueBefore:       *dueBefore,
			DueAfter:        *dueAfter,
			Limit:           *limit,
		})
		if err != nil {
			return err
		}
		if cf.asJSON {
			return writeJSON(stdout, raw)
		}
		var items []ipc.TaskSearchItem
		if err := json.Unmarshal(raw, &items); err != nil {
			return err
		}
		for _, item := range items {
			due := ""
			if item.Task.DueDate != "" {
				due = " due=" + item.Task.DueDate
			}
			fmt.Fprintf(stdout, "%s  [%s/%s]  (%s)  %s%s\n",
				item.Task.ID, item.Task.Status, item.Task.Priority, item.ProjectName, item.Task.Title, due)
		}
		return nil

	case "add":
		fs := newFlagSet("task add", stderr, &cf)
		projectID := fs.String("project", "", "project ID (required)")
		title := fs.String("title", "", "task title (required)")
		desc := fs.String("desc", "", "optional Markdown description")
		status := fs.String("status", "", "initial status: planned, in_progress, done")
		priority := fs.String("priority", "", "priority: none, low, medium, high, urgent")
		tagsCSV := fs.String("tags", "", "comma-separated tags")
		due := fs.String("due", "", "optional due date (YYYY-MM-DD)")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		raw, err := callRouter(ctx, cf.dataDir, ipc.MethodTasksCreate, ipc.CreateTaskParams{
			ProjectID:   *projectID,
			Title:       *title,
			Description: *desc,
			Status:      model.Status(*status),
			Priority:    model.Priority(*priority),
			Tags:        splitCSV(*tagsCSV),
			DueDate:     *due,
		})
		if err != nil {
			return err
		}
		if cf.asJSON {
			return writeJSON(stdout, raw)
		}
		var t model.Task
		if err := json.Unmarshal(raw, &t); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "created task %s (%s)\n", t.ID, t.Title)
		return nil

	case "move":
		fs := newFlagSet("task move", stderr, &cf)
		id := fs.String("id", "", "task ID (required)")
		status := fs.String("status", "", "target status (required)")
		index := fs.Int("index", 0, "0-based target index")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		raw, err := callRouter(ctx, cf.dataDir, ipc.MethodTasksMove, ipc.MoveTaskParams{
			ID:     *id,
			Status: model.Status(*status),
			Index:  *index,
		})
		if err != nil {
			return err
		}
		if cf.asJSON {
			return writeJSON(stdout, raw)
		}
		fmt.Fprintf(stdout, "moved task %s to %s[%d]\n", *id, *status, *index)
		return nil

	case "done":
		fs := newFlagSet("task done", stderr, &cf)
		id := fs.String("id", "", "task ID (required)")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		raw, err := callRouter(ctx, cf.dataDir, ipc.MethodTasksMove, ipc.MoveTaskParams{
			ID:     *id,
			Status: model.Done,
			Index:  0,
		})
		if err != nil {
			return err
		}
		if cf.asJSON {
			return writeJSON(stdout, raw)
		}
		fmt.Fprintf(stdout, "marked task %s done\n", *id)
		return nil

	case "attach":
		fs := newFlagSet("task attach", stderr, &cf)
		id := fs.String("id", "", "task ID (required)")
		file := fs.String("file", "", "local file path (required)")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		raw, err := callRouter(ctx, cf.dataDir, ipc.MethodTasksAttach, ipc.AttachTaskFileParams{
			TaskID: *id,
			Path:   *file,
		})
		if err != nil {
			return err
		}
		if cf.asJSON {
			return writeJSON(stdout, raw)
		}
		var att model.Attachment
		if err := json.Unmarshal(raw, &att); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "attached %s (%s) to task %s\n", att.Name, att.ID, att.TaskID)
		return nil

	case "trash":
		fs := newFlagSet("task trash", stderr, &cf)
		id := fs.String("id", "", "task ID (required)")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		raw, err := callRouter(ctx, cf.dataDir, ipc.MethodTasksTrash, ipc.IDParams{ID: *id})
		if err != nil {
			return err
		}
		if cf.asJSON {
			return writeJSON(stdout, raw)
		}
		fmt.Fprintf(stdout, "moved task %s to trash\n", *id)
		return nil

	default:
		return fmt.Errorf("unknown task subcommand %q", sub)
	}
}

func runJob(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("job requires a subcommand: list, add, stage, note, close, reopen, trash")
	}
	sub, rest := args[0], args[1:]
	var cf commonFlags
	switch sub {
	case "list":
		fs := newFlagSet("job list", stderr, &cf)
		view := fs.String("view", "all", "view: all, active, offer, closed")
		query := fs.String("q", "", "search company or position")
		stage := fs.String("stage", "", "stage filter")
		outcome := fs.String("outcome", "", "outcome filter")
		sortBy := fs.String("sort", "applied_desc", "sort: applied_desc, applied_asc, updated_desc, updated_asc")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		raw, err := callRouter(ctx, cf.dataDir, ipc.MethodJobsList, ipc.ListJobsParams{
			View:    *view,
			Query:   *query,
			Stage:   model.JobStage(*stage),
			Outcome: model.JobOutcome(*outcome),
			Sort:    *sortBy,
		})
		if err != nil {
			return err
		}
		if cf.asJSON {
			return writeJSON(stdout, raw)
		}
		var res ipc.ListJobsResult
		if err := json.Unmarshal(raw, &res); err != nil {
			return err
		}
		for _, a := range res.Applications {
			fmt.Fprintf(stdout, "%s  [%s/%s]  %s · %s  (%s)\n",
				a.ID, a.Lifecycle, a.Stage, a.CompanyName, a.PositionName, a.AppliedDate)
		}
		return nil

	case "add":
		fs := newFlagSet("job add", stderr, &cf)
		company := fs.String("company", "", "company name (required)")
		position := fs.String("position", "", "position name (required)")
		date := fs.String("date", "", "applied date (YYYY-MM-DD, defaults to today)")
		stage := fs.String("stage", "", "initial stage")
		url := fs.String("url", "", "http/https application URL")
		channel := fs.String("channel", "", "application channel")
		notes := fs.String("notes", "", "notes")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		raw, err := callRouter(ctx, cf.dataDir, ipc.MethodJobsCreate, ipc.CreateJobParams{
			CompanyName:    *company,
			PositionName:   *position,
			AppliedDate:    *date,
			Stage:          model.JobStage(*stage),
			ApplicationURL: *url,
			Channel:        *channel,
			Notes:          *notes,
		})
		if err != nil {
			return err
		}
		if cf.asJSON {
			return writeJSON(stdout, raw)
		}
		var a model.JobApplication
		if err := json.Unmarshal(raw, &a); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "created job %s (%s · %s)\n", a.ID, a.CompanyName, a.PositionName)
		return nil

	case "stage":
		fs := newFlagSet("job stage", stderr, &cf)
		id := fs.String("id", "", "job application ID (required)")
		to := fs.String("to", "", "target stage (required)")
		note := fs.String("note", "", "optional timeline note")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		raw, err := callRouter(ctx, cf.dataDir, ipc.MethodJobsChangeStage, ipc.ChangeJobStageParams{
			ID:      *id,
			ToStage: model.JobStage(*to),
			Note:    *note,
		})
		if err != nil {
			return err
		}
		if cf.asJSON {
			return writeJSON(stdout, raw)
		}
		fmt.Fprintf(stdout, "updated job %s stage to %s\n", *id, *to)
		return nil

	case "note":
		fs := newFlagSet("job note", stderr, &cf)
		id := fs.String("id", "", "job application ID (required)")
		note := fs.String("note", "", "timeline note text (required)")
		date := fs.String("date", "", "optional date (YYYY-MM-DD)")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		raw, err := callRouter(ctx, cf.dataDir, ipc.MethodJobsAddNote, ipc.AddJobNoteParams{
			ApplicationID: *id,
			Note:          *note,
			OccurredDate:  *date,
		})
		if err != nil {
			return err
		}
		if cf.asJSON {
			return writeJSON(stdout, raw)
		}
		fmt.Fprintf(stdout, "added timeline note to job %s\n", *id)
		return nil

	case "close":
		fs := newFlagSet("job close", stderr, &cf)
		id := fs.String("id", "", "job application ID (required)")
		outcome := fs.String("outcome", "", "closure outcome (required)")
		reason := fs.String("reason", "", "optional closure reason")
		stageOverride := fs.String("stage-override", "", "optional stage override (e.g. offer)")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		raw, err := callRouter(ctx, cf.dataDir, ipc.MethodJobsClose, ipc.CloseJobParams{
			ID:            *id,
			Outcome:       model.JobOutcome(*outcome),
			Reason:        *reason,
			StageOverride: model.JobStage(*stageOverride),
		})
		if err != nil {
			return err
		}
		if cf.asJSON {
			return writeJSON(stdout, raw)
		}
		fmt.Fprintf(stdout, "closed job %s (%s)\n", *id, *outcome)
		return nil

	case "reopen":
		fs := newFlagSet("job reopen", stderr, &cf)
		id := fs.String("id", "", "job application ID (required)")
		note := fs.String("note", "", "optional reopen note")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		raw, err := callRouter(ctx, cf.dataDir, ipc.MethodJobsReopen, ipc.ReopenJobParams{
			ID:   *id,
			Note: *note,
		})
		if err != nil {
			return err
		}
		if cf.asJSON {
			return writeJSON(stdout, raw)
		}
		fmt.Fprintf(stdout, "reopened job %s\n", *id)
		return nil

	case "trash":
		fs := newFlagSet("job trash", stderr, &cf)
		id := fs.String("id", "", "job application ID (required)")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		raw, err := callRouter(ctx, cf.dataDir, ipc.MethodJobsTrash, ipc.IDParams{ID: *id})
		if err != nil {
			return err
		}
		if cf.asJSON {
			return writeJSON(stdout, raw)
		}
		fmt.Fprintf(stdout, "moved job %s to trash\n", *id)
		return nil

	default:
		return fmt.Errorf("unknown job subcommand %q", sub)
	}
}

func runTrash(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("trash requires a subcommand: list, restore")
	}
	sub, rest := args[0], args[1:]
	var cf commonFlags
	switch sub {
	case "list":
		fs := newFlagSet("trash list", stderr, &cf)
		if err := fs.Parse(rest); err != nil {
			return err
		}
		raw, err := callRouter(ctx, cf.dataDir, ipc.MethodTrashList, nil)
		if err != nil {
			return err
		}
		if cf.asJSON {
			return writeJSON(stdout, raw)
		}
		var items []model.TrashItem
		if err := json.Unmarshal(raw, &items); err != nil {
			return err
		}
		for _, it := range items {
			fmt.Fprintf(stdout, "%s  [%s]  %s\n", it.ID, it.Kind, it.Name)
		}
		return nil

	case "restore":
		fs := newFlagSet("trash restore", stderr, &cf)
		kind := fs.String("kind", "", "item kind: project, task, job_application")
		id := fs.String("id", "", "item ID (required)")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		raw, err := callRouter(ctx, cf.dataDir, ipc.MethodTrashRestore, ipc.RestoreTrashParams{
			Kind: *kind,
			ID:   *id,
		})
		if err != nil {
			return err
		}
		if cf.asJSON {
			return writeJSON(stdout, raw)
		}
		fmt.Fprintf(stdout, "restored %s %s from trash\n", *kind, *id)
		return nil

	default:
		return fmt.Errorf("unknown trash subcommand %q", sub)
	}
}

func splitCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
