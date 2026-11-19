package model

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// JobStage identifies a built-in stage of a job application pipeline.
type JobStage string

const (
	JobStageApplied    JobStage = "applied"
	JobStageScreening  JobStage = "screening"
	JobStageAssessment JobStage = "assessment"
	JobStageInterview1 JobStage = "interview_1"
	JobStageInterview2 JobStage = "interview_2"
	JobStageInterview3 JobStage = "interview_3"
	JobStageHR         JobStage = "hr"
	JobStageOffer      JobStage = "offer"
)

type jobStageMeta struct {
	Stage JobStage
	Label string
	Order int
}

var jobStageTable = []jobStageMeta{
	{Stage: JobStageApplied, Label: "已投递", Order: 1},
	{Stage: JobStageScreening, Label: "简历筛选中", Order: 2},
	{Stage: JobStageAssessment, Label: "笔试 / 测评", Order: 3},
	{Stage: JobStageInterview1, Label: "一面", Order: 4},
	{Stage: JobStageInterview2, Label: "二面", Order: 5},
	{Stage: JobStageInterview3, Label: "三面", Order: 6},
	{Stage: JobStageHR, Label: "HR 面", Order: 7},
	{Stage: JobStageOffer, Label: "Offer", Order: 8},
}

func (s JobStage) meta() (jobStageMeta, bool) {
	for _, m := range jobStageTable {
		if m.Stage == s {
			return m, true
		}
	}
	return jobStageMeta{}, false
}

// Valid reports whether s is a supported job application stage.
func (s JobStage) Valid() bool {
	_, ok := s.meta()
	return ok
}

// Label returns the localized display name for s.
func (s JobStage) Label() string {
	if m, ok := s.meta(); ok {
		return m.Label
	}
	return string(s)
}

// Order returns the display order (1-based) for s, or 0 if unknown.
func (s JobStage) Order() int {
	if m, ok := s.meta(); ok {
		return m.Order
	}
	return 0
}

// AllJobStages returns all built-in job application stages in display order.
func AllJobStages() []JobStage {
	out := make([]JobStage, len(jobStageTable))
	for i, m := range jobStageTable {
		out[i] = m.Stage
	}
	return out
}

// JobLifecycle distinguishes ongoing vs closed job applications independently of stage.
type JobLifecycle string

const (
	JobLifecycleActive JobLifecycle = "active"
	JobLifecycleClosed JobLifecycle = "closed"
)

// Valid reports whether l is a supported lifecycle state.
func (l JobLifecycle) Valid() bool {
	return l == JobLifecycleActive || l == JobLifecycleClosed
}

// Label returns the localized display name for l.
func (l JobLifecycle) Label() string {
	switch l {
	case JobLifecycleActive:
		return "进行中"
	case JobLifecycleClosed:
		return "已结束"
	default:
		return string(l)
	}
}

// JobOutcome records why a closed job application ended.
type JobOutcome string

const (
	JobOutcomeRejected          JobOutcome = "rejected"
	JobOutcomeOfferAccepted     JobOutcome = "offer_accepted"
	JobOutcomeOfferDeclined     JobOutcome = "offer_declined"
	JobOutcomeWithdrawn         JobOutcome = "withdrawn"
	JobOutcomePositionCancelled JobOutcome = "position_cancelled"
	JobOutcomeOther             JobOutcome = "other"
)

type jobOutcomeMeta struct {
	Outcome            JobOutcome
	Label              string
	RequiresOfferStage bool
}

var jobOutcomeTable = []jobOutcomeMeta{
	{Outcome: JobOutcomeRejected, Label: "未通过"},
	{Outcome: JobOutcomeOfferAccepted, Label: "接受 Offer", RequiresOfferStage: true},
	{Outcome: JobOutcomeOfferDeclined, Label: "拒绝 Offer", RequiresOfferStage: true},
	{Outcome: JobOutcomeWithdrawn, Label: "主动放弃"},
	{Outcome: JobOutcomePositionCancelled, Label: "岗位取消"},
	{Outcome: JobOutcomeOther, Label: "其他"},
}

func (o JobOutcome) meta() (jobOutcomeMeta, bool) {
	for _, m := range jobOutcomeTable {
		if m.Outcome == o {
			return m, true
		}
	}
	return jobOutcomeMeta{}, false
}

// Valid reports whether o is a supported closure outcome.
func (o JobOutcome) Valid() bool {
	_, ok := o.meta()
	return ok
}

// Label returns the localized display name for o.
func (o JobOutcome) Label() string {
	if o == "" {
		return ""
	}
	if m, ok := o.meta(); ok {
		return m.Label
	}
	return string(o)
}

// RequiresOfferStage reports whether o requires the application stage to be Offer.
func (o JobOutcome) RequiresOfferStage() bool {
	m, ok := o.meta()
	return ok && m.RequiresOfferStage
}

// AllJobOutcomes returns all built-in closure outcomes in display order.
func AllJobOutcomes() []JobOutcome {
	out := make([]JobOutcome, len(jobOutcomeTable))
	for i, m := range jobOutcomeTable {
		out[i] = m.Outcome
	}
	return out
}

// JobEventType identifies the kind of entry in an application timeline.
type JobEventType string

const (
	JobEventCreated         JobEventType = "created"
	JobEventStageChanged    JobEventType = "stage_changed"
	JobEventStageBackfilled JobEventType = "stage_backfilled"
	JobEventClosed          JobEventType = "closed"
	JobEventReopened        JobEventType = "reopened"
	JobEventNoteAdded       JobEventType = "note_added"
)

var jobEventLabels = map[JobEventType]string{
	JobEventCreated:         "创建投递",
	JobEventStageChanged:    "阶段变更",
	JobEventStageBackfilled: "补录阶段",
	JobEventClosed:          "流程结束",
	JobEventReopened:        "重新开启",
	JobEventNoteAdded:       "添加备注",
}

// Valid reports whether t is a supported timeline event type.
func (t JobEventType) Valid() bool {
	_, ok := jobEventLabels[t]
	return ok
}

// Label returns the localized display name for t.
func (t JobEventType) Label() string {
	if label, ok := jobEventLabels[t]; ok {
		return label
	}
	return string(t)
}

// JobApplication represents one job application entry tracked in the 求职 module.
type JobApplication struct {
	ID             string       `json:"id"`
	CompanyName    string       `json:"company_name"`
	PositionName   string       `json:"position_name"`
	AppliedDate    string       `json:"applied_date"`
	Stage          JobStage     `json:"stage"`
	Lifecycle      JobLifecycle `json:"lifecycle"`
	Outcome        JobOutcome   `json:"outcome,omitempty"`
	ClosureReason  string       `json:"closure_reason,omitempty"`
	ClosedAt       *time.Time   `json:"closed_at,omitempty"`
	ApplicationURL string       `json:"application_url,omitempty"`
	Channel        string       `json:"channel,omitempty"`
	Notes          string       `json:"notes,omitempty"`
	ReachedOffer   bool         `json:"reached_offer"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
	DeletedAt      *time.Time   `json:"deleted_at,omitempty"`
}

// JobApplicationEvent records one immutable workflow or note entry for an application.
type JobApplicationEvent struct {
	ID            string       `json:"id"`
	ApplicationID string       `json:"application_id"`
	EventType     JobEventType `json:"event_type"`
	FromStage     JobStage     `json:"from_stage,omitempty"`
	ToStage       JobStage     `json:"to_stage,omitempty"`
	Outcome       JobOutcome   `json:"outcome,omitempty"`
	Reason        string       `json:"reason,omitempty"`
	Note          string       `json:"note,omitempty"`
	OccurredAt    time.Time    `json:"occurred_at"`
	CreatedAt     time.Time    `json:"created_at"`
}

// JobApplicationStats summarizes non-deleted application counts across the whole module.
type JobApplicationStats struct {
	Total   int `json:"total"`
	Active  int `json:"active"`
	Offered int `json:"offered"`
	Closed  int `json:"closed"`
}

// ParseCalendarDate parses a YYYY-MM-DD local calendar date string without timezone shifts.
func ParseCalendarDate(raw string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) != len("2006-01-02") {
		return time.Time{}, fmt.Errorf("%w: date must use YYYY-MM-DD format", ErrInvalid)
	}
	parsed, err := time.ParseInLocation("2006-01-02", trimmed, time.Local)
	if err != nil || parsed.Format("2006-01-02") != trimmed {
		return time.Time{}, fmt.Errorf("%w: invalid calendar date %q", ErrInvalid, raw)
	}
	return parsed, nil
}

// ValidateJobURL verifies that raw is a valid HTTP or HTTPS URL with a host.
func ValidateJobURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	if strings.ContainsAny(trimmed, " \t\r\n") {
		return "", fmt.Errorf("%w: application URL cannot contain whitespace", ErrInvalid)
	}
	parsed, err := url.ParseRequestURI(trimmed)
	if err != nil {
		return "", fmt.Errorf("%w: invalid application URL", ErrInvalid)
	}
	scheme := strings.ToLower(parsed.Scheme)
	if (scheme != "http" && scheme != "https") || parsed.Hostname() == "" {
		return "", fmt.Errorf("%w: application URL must use http or https with a valid host", ErrInvalid)
	}
	return trimmed, nil
}
