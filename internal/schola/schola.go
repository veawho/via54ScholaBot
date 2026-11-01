// Package schola implements the via54ScholaBot deep-research engine.
//
// [CN] via54ScholaBot 深度研究引擎：集学术检索及研究、报告检索及研究、
// 文献检索及研究于一体，整合三层记忆、技能库与多模型路由。
// [EN] Deep research engine uniting academic search, report research,
// literature research with three-tier memory, skills registry and model routing.
package schola

import (
	"context"
	"time"
)

// SourceKind 数据源类别
type SourceKind int

const (
	// Academic 学术文献源
	Academic SourceKind = iota
	// Report 研究报告/公开申报源
	Report
	// Literature 文献全文源
	Literature
)

func (k SourceKind) String() string {
	switch k {
	case Academic:
		return "academic"
	case Report:
		return "report"
	case Literature:
		return "literature"
	default:
		return "unknown"
	}
}

// SearchOptions 检索选项
type SearchOptions struct {
	Limit int
	Lang  string
	From  string
	To    string
}

// SourceHit 单条检索命中
type SourceHit struct {
	Source    string     `json:"source"`
	Kind      SourceKind `json:"kind"`
	Title     string     `json:"title"`
	Authors   []string   `json:"authors"`
	Year      string     `json:"year"`
	Venue     string     `json:"venue"`
	DOI       string     `json:"doi"`
	URL       string     `json:"url"`
	Abstract  string     `json:"abstract"`
	Citations int        `json:"citations"`
	Score     float64    `json:"score"`
}

// Source 研究数据源接口
type Source interface {
	Name() string
	Kind() SourceKind
	Search(ctx context.Context, q string, opts SearchOptions) ([]SourceHit, error)
}

// MemoryEntry 记忆条目（会话/项目/长期三层）
type MemoryEntry struct {
	ID        string `json:"id"`
	Level     string `json:"level"` // session | project | longterm
	Project   string `json:"project"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// Skill 科研技能
type Skill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Kind        string `json:"kind"`
}

// ResearchTask 深度研究任务
type ResearchTask struct {
	Question string   `json:"question"`
	Scope    string   `json:"scope"`     // academic | report | literature | all
	Depth    int      `json:"depth"`     // 1..3
	Lang     string   `json:"lang"`      // zh | en
	Sources  []string `json:"sources"`   // 空=全源
	MemoryID string   `json:"memory_id"` // 关联项目记忆
}

// PlanStep 研究计划步骤
type PlanStep struct {
	Order  int    `json:"order"`
	Action string `json:"action"`
	Detail string `json:"detail"`
}

// ResearchPlan 研究计划
type ResearchPlan struct {
	Question string     `json:"question"`
	Keywords []string   `json:"keywords"`
	Sources  []string   `json:"sources"`
	Steps    []PlanStep `json:"steps"`
}

// ReportSection 报告章节
type ReportSection struct {
	Heading string `json:"heading"`
	Body    string `json:"body"`
}

// ResearchReport 深度研究报告
type ResearchReport struct {
	Title       string          `json:"title"`
	Summary     string          `json:"summary"`
	Sections    []ReportSection `json:"sections"`
	Evidence    []SourceHit     `json:"evidence"`
	GeneratedAt time.Time       `json:"generated_at"`
	Lang        string          `json:"lang"`
}
