package model

import "time"

// Project represents the top-level boundary for work tracking, documentation, and RBAC.
type Project struct {
	ID          string           `firestore:"id" json:"id"`
	Name        string           `firestore:"name" json:"name"`
	KeyPrefix   string           `firestore:"key_prefix" json:"key_prefix"` // e.g., "JYNKO"
	Description string           `firestore:"description" json:"description"`
	ItemSeq     int              `firestore:"item_seq" json:"item_seq"`     // Atomic sequence counter for WorkItem keys
	MemberIDs   []string         `firestore:"member_ids" json:"member_ids"` // Indexed array of user IDs for fast project listing
	Settings    *ProjectSettings `firestore:"settings,omitempty" json:"settings,omitempty"`
	CreatedBy   string           `firestore:"created_by" json:"created_by"`
	CreatedAt   time.Time        `firestore:"created_at" json:"created_at"`
	UpdatedAt   time.Time        `firestore:"updated_at" json:"updated_at"`
	// DeletedAt marks a soft-deleted project (PLAN-13). Deleted projects are hidden from listings and
	// project-scoped routes until restored, and are permanently purged by the cleanup API.
	DeletedAt *time.Time `firestore:"deleted_at,omitempty" json:"deleted_at,omitempty"`
	DeletedBy string     `firestore:"deleted_by,omitempty" json:"deleted_by,omitempty"`
}

// IsDeleted reports whether the project has been soft-deleted.
func (p *Project) IsDeleted() bool {
	return p != nil && p.DeletedAt != nil && !p.DeletedAt.IsZero()
}

// ProjectSettings stores configurable project-level workflows, board view defaults, and documentation policies.
type ProjectSettings struct {
	DefaultNewItemStatus             string   `firestore:"default_new_item_status" json:"default_new_item_status"`
	DefaultNewItemPriority           string   `firestore:"default_new_item_priority" json:"default_new_item_priority"`
	EstimationScale                  string   `firestore:"estimation_scale" json:"estimation_scale"`
	CycleDurationWeeks               string   `firestore:"cycle_duration_weeks" json:"cycle_duration_weeks"`
	VisibleBoardColumns              []string `firestore:"visible_board_columns" json:"visible_board_columns"`
	AutoAssignToLead                 bool     `firestore:"auto_assign_to_lead" json:"auto_assign_to_lead"`
	AutoRolloverUnfinishedCycleItems bool     `firestore:"auto_rollover_unfinished_cycle_items" json:"auto_rollover_unfinished_cycle_items"`
	RequireEstimateBeforeInProgress  bool     `firestore:"require_estimate_before_in_progress" json:"require_estimate_before_in_progress"`
	AutoLockPublishedRunbooks        bool     `firestore:"auto_lock_published_runbooks" json:"auto_lock_published_runbooks"`
	CreateVersionOnEverySave         bool     `firestore:"create_version_on_every_save" json:"create_version_on_every_save"`
	// BackgroundColor is a palette key (see ProjectBackgroundColors) used to tint project pages so
	// users can tell projects apart at a glance. Empty or "DEFAULT" means the standard slate-50 canvas.
	BackgroundColor string `firestore:"background_color,omitempty" json:"background_color,omitempty"`

	// Scion AI Agent & Slash-Command Integration Settings
	ScionEnabled            bool   `firestore:"scion_enabled" json:"scion_enabled"`
	DefaultGitRepoURL       string `firestore:"default_git_repo_url,omitempty" json:"default_git_repo_url"`
	DefaultBaseBranch       string `firestore:"default_base_branch,omitempty" json:"default_base_branch"`
	ScionProjectSlug        string `firestore:"scion_project_slug,omitempty" json:"scion_project_slug"`
	ScionTemplate           string `firestore:"scion_template,omitempty" json:"scion_template"`
	ImplementHarness        string `firestore:"implement_harness,omitempty" json:"implement_harness"`
	ReviewHarness           string `firestore:"review_harness,omitempty" json:"review_harness"`
	ImplementPromptTemplate string `firestore:"implement_prompt_template,omitempty" json:"implement_prompt_template"`
	ReviewPromptTemplate    string `firestore:"review_prompt_template,omitempty" json:"review_prompt_template"`
	FollowupPromptTemplate  string `firestore:"followup_prompt_template,omitempty" json:"followup_prompt_template"`
}

// ProjectBackgroundColors lists the allowed palette keys for ProjectSettings.BackgroundColor.
var ProjectBackgroundColors = []string{
	"DEFAULT", "SKY", "INDIGO", "VIOLET", "ROSE", "AMBER", "LIME", "EMERALD", "TEAL", "STONE",
}

// IsValidProjectBackgroundColor reports whether c is an allowed project background palette key.
// An empty value is accepted and treated as "DEFAULT".
func IsValidProjectBackgroundColor(c string) bool {
	if c == "" {
		return true
	}
	for _, v := range ProjectBackgroundColors {
		if v == c {
			return true
		}
	}
	return false
}

// DefaultProjectSettings returns sensible defaults for a project's settings.
func DefaultProjectSettings() *ProjectSettings {
	return &ProjectSettings{
		DefaultNewItemStatus:             "TODO",
		DefaultNewItemPriority:           "MEDIUM",
		EstimationScale:                  "FIBONACCI",
		CycleDurationWeeks:               "2",
		VisibleBoardColumns:              []string{"BACKLOG", "TODO", "IN_PROGRESS", "IN_REVIEW", "DONE"},
		AutoAssignToLead:                 false,
		AutoRolloverUnfinishedCycleItems: true,
		RequireEstimateBeforeInProgress:  false,
		AutoLockPublishedRunbooks:        false,
		CreateVersionOnEverySave:         true,
		BackgroundColor:                  "DEFAULT",
		ScionEnabled:                     false,
		DefaultBaseBranch:                "dev",
		ScionProjectSlug:                 "global",
		ScionTemplate:                    "jynko-dev",
		ImplementHarness:                 "claude",
		ReviewHarness:                    "antigravity",
	}
}

// Module represents a persistent functional component or feature domain.
type Module struct {
	ID          string    `firestore:"id" json:"id"`
	ProjectID   string    `firestore:"project_id" json:"project_id"`
	Name        string    `firestore:"name" json:"name"`
	Description string    `firestore:"description" json:"description"`
	LeadUserID  string    `firestore:"lead_user_id" json:"lead_user_id"`
	GitRepoURL  string    `firestore:"git_repo_url,omitempty" json:"git_repo_url,omitempty"`
	BaseBranch  string    `firestore:"base_branch,omitempty" json:"base_branch,omitempty"`
	CreatedAt   time.Time `firestore:"created_at" json:"created_at"`
	UpdatedAt   time.Time `firestore:"updated_at" json:"updated_at"`
}

// Epic Status constants.
const (
	EpicStatusPlanned    = "PLANNED"
	EpicStatusInProgress = "IN_PROGRESS"
	EpicStatusCompleted  = "COMPLETED"
	EpicStatusCancelled  = "CANCELLED"
)

// Epic represents a strategic milestone or major initiative.
type Epic struct {
	ID          string     `firestore:"id" json:"id"`
	ProjectID   string     `firestore:"project_id" json:"project_id"`
	Name        string     `firestore:"name" json:"name"`
	Description string     `firestore:"description" json:"description"`
	Status      string     `firestore:"status" json:"status"` // PLANNED, IN_PROGRESS, COMPLETED, CANCELLED
	StartDate   *time.Time `firestore:"start_date,omitempty" json:"start_date,omitempty"`
	TargetDate  *time.Time `firestore:"target_date,omitempty" json:"target_date,omitempty"`
	CreatedAt   time.Time  `firestore:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `firestore:"updated_at" json:"updated_at"`
}

// Cycle represents a sprint period or iteration.
type Cycle struct {
	ID          string    `firestore:"id" json:"id"`
	ProjectID   string    `firestore:"project_id" json:"project_id"`
	Name        string    `firestore:"name" json:"name"`
	Description string    `firestore:"description" json:"description"`
	StartDate   time.Time `firestore:"start_date" json:"start_date"`
	EndDate     time.Time `firestore:"end_date" json:"end_date"`
	IsActive    bool      `firestore:"is_active" json:"is_active"`
	CreatedAt   time.Time `firestore:"created_at" json:"created_at"`
	UpdatedAt   time.Time `firestore:"updated_at" json:"updated_at"`
}

// Label represents a categorical metadata tag within a project.
type Label struct {
	ID        string    `firestore:"id" json:"id"`
	ProjectID string    `firestore:"project_id" json:"project_id"`
	Name      string    `firestore:"name" json:"name"`
	Color     string    `firestore:"color" json:"color"` // Hex color e.g. "#0284c7"
	CreatedAt time.Time `firestore:"created_at" json:"created_at"`
}

// WorkItemStatus & WorkItemPriority constants.
const (
	StatusBacklog    = "BACKLOG"
	StatusTodo       = "TODO"
	StatusInProgress = "IN_PROGRESS"
	StatusInReview   = "IN_REVIEW"
	StatusDone       = "DONE"
	StatusCancelled  = "CANCELLED"

	PriorityNone   = "NONE"
	PriorityLow    = "LOW"
	PriorityMedium = "MEDIUM"
	PriorityHigh   = "HIGH"
	PriorityUrgent = "URGENT"
)

// WorkItem represents an actionable task.
type WorkItem struct {
	ID                string     `firestore:"id" json:"id"`
	ProjectID         string     `firestore:"project_id" json:"project_id"`
	Key               string     `firestore:"key" json:"key"`                 // e.g., "JYNKO-12"
	SequenceID        int        `firestore:"sequence_id" json:"sequence_id"` // e.g., 12
	ModuleID          string     `firestore:"module_id,omitempty" json:"module_id,omitempty"`
	EpicID            string     `firestore:"epic_id,omitempty" json:"epic_id,omitempty"`
	CycleID           string     `firestore:"cycle_id,omitempty" json:"cycle_id,omitempty"`
	Title             string     `firestore:"title" json:"title"`
	Description       string     `firestore:"description" json:"description"`
	Status            string     `firestore:"status" json:"status"`     // BACKLOG, TODO, IN_PROGRESS, IN_REVIEW, DONE, CANCELLED
	Priority          string     `firestore:"priority" json:"priority"` // NONE, LOW, MEDIUM, HIGH, URGENT
	AssigneeID        string     `firestore:"assignee_id,omitempty" json:"assignee_id,omitempty"`
	ReporterID        string     `firestore:"reporter_id" json:"reporter_id"`
	LabelIDs          []string   `firestore:"label_ids" json:"label_ids"`
	Estimate          *int       `firestore:"estimate,omitempty" json:"estimate,omitempty"` // Story points / effort estimate
	StartDate         *time.Time `firestore:"start_date,omitempty" json:"start_date,omitempty"`
	DueDate           *time.Time `firestore:"due_date,omitempty" json:"due_date,omitempty"`
	SubItemCount       int        `firestore:"sub_item_count" json:"sub_item_count"`
	CompletedSubCount  int        `firestore:"completed_sub_count" json:"completed_sub_count"`
	ImplAgentID        string     `firestore:"impl_agent_id,omitempty" json:"impl_agent_id,omitempty"`
	ImplAgentName      string     `firestore:"impl_agent_name,omitempty" json:"impl_agent_name,omitempty"`
	ImplAgentHarness   string     `firestore:"impl_agent_harness,omitempty" json:"impl_agent_harness,omitempty"`
	ReviewAgentID      string     `firestore:"review_agent_id,omitempty" json:"review_agent_id,omitempty"`
	ReviewAgentName    string     `firestore:"review_agent_name,omitempty" json:"review_agent_name,omitempty"`
	ReviewAgentHarness string     `firestore:"review_agent_harness,omitempty" json:"review_agent_harness,omitempty"`
	CreatedAt          time.Time  `firestore:"created_at" json:"created_at"`
	UpdatedAt          time.Time  `firestore:"updated_at" json:"updated_at"`
}

// WorkItemFilter holds query filter parameters for listing work items.
type WorkItemFilter struct {
	Status     string
	Priority   string
	ModuleID   string
	EpicID     string
	CycleID    string
	AssigneeID string
	LabelID    string
	Search     string
}

// SubWorkItem represents a child task under a parent WorkItem.
type SubWorkItem struct {
	ID           string    `firestore:"id" json:"id"`
	ParentItemID string    `firestore:"parent_item_id" json:"parent_item_id"`
	Title        string    `firestore:"title" json:"title"`
	IsCompleted  bool      `firestore:"is_completed" json:"is_completed"`
	AssigneeID   string    `firestore:"assignee_id,omitempty" json:"assignee_id,omitempty"`
	CreatedAt    time.Time `firestore:"created_at" json:"created_at"`
	UpdatedAt    time.Time `firestore:"updated_at" json:"updated_at"`
}

// Comment represents a message on a WorkItem.
type Comment struct {
	ID           string    `firestore:"id" json:"id"`
	WorkItemID   string    `firestore:"work_item_id" json:"work_item_id"`
	AuthorID     string    `firestore:"author_id" json:"author_id"`
	AuthorName   string    `firestore:"author_name" json:"author_name"`
	Body         string    `firestore:"body" json:"body"`
	IsSystem     bool      `firestore:"is_system,omitempty" json:"is_system,omitempty"`
	SlashCommand string    `firestore:"slash_command,omitempty" json:"slash_command,omitempty"`
	AgentID      string    `firestore:"agent_id,omitempty" json:"agent_id,omitempty"`
	AgentURL     string    `firestore:"agent_url,omitempty" json:"agent_url,omitempty"`
	CreatedAt    time.Time `firestore:"created_at" json:"created_at"`
	UpdatedAt    time.Time `firestore:"updated_at" json:"updated_at"`
}

// StickyColor constants for playful sticky notes.
const (
	StickyColorBanana   = "BANANA"
	StickyColorFlamingo = "FLAMINGO"
	StickyColorMint     = "MINT"
	StickyColorLagoon   = "LAGOON"
	StickyColorLavender = "LAVENDER"
	StickyColorPeach    = "PEACH"
)

// Sticky represents a colorful quick scratchpad note under a project.
type Sticky struct {
	ID        string    `firestore:"id" json:"id"`
	ProjectID string    `firestore:"project_id" json:"project_id"`
	Content   string    `firestore:"content" json:"content"`
	Color     string    `firestore:"color" json:"color"` // BANANA, FLAMINGO, MINT, LAGOON, LAVENDER, PEACH
	IsPinned  bool      `firestore:"is_pinned" json:"is_pinned"`
	AuthorID  string    `firestore:"author_id" json:"author_id"`
	CreatedAt time.Time `firestore:"created_at" json:"created_at"`
	UpdatedAt time.Time `firestore:"updated_at" json:"updated_at"`
}

// Attachment represents an uploaded file attached to a Project (and optionally a WorkItem).
type Attachment struct {
	ID           string    `firestore:"id" json:"id"`
	ProjectID    string    `firestore:"project_id" json:"project_id"`
	WorkItemID   string    `firestore:"work_item_id,omitempty" json:"work_item_id,omitempty"`
	FileName     string    `firestore:"file_name" json:"file_name"`
	ContentType  string    `firestore:"content_type" json:"content_type"`
	SizeBytes    int64     `firestore:"size_bytes" json:"size_bytes"`
	StoragePath  string    `firestore:"storage_path" json:"storage_path"`
	UploadedBy   string    `firestore:"uploaded_by" json:"uploaded_by"`
	UploadedName string    `firestore:"uploaded_name" json:"uploaded_name"`
	CreatedAt    time.Time `firestore:"created_at" json:"created_at"`
	UpdatedAt    time.Time `firestore:"updated_at" json:"updated_at"`
	URL          string    `firestore:"-" json:"url"`
	Markdown     string    `firestore:"-" json:"markdown"`
}
