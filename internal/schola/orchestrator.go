package schola

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Orchestrator 深度研究流水线编排器
type Orchestrator struct {
	Sources map[string]Source
	Mem     *MemoryStore
	Router  *ModelRouter
}

// NewOrchestrator 构建编排器
func NewOrchestrator(hc HTTPClient, mem *MemoryStore, router *ModelRouter) *Orchestrator {
	if mem == nil {
		mem = NewMemoryStore("")
	}
	if router == nil {
		router = NewModelRouter("")
	}
	return &Orchestrator{Sources: NewSourceRegistry(hc), Mem: mem, Router: router}
}

// Dedupe 跨源去重（按标题规范化）
func Dedupe(hits []SourceHit, limit int) []SourceHit {
	seen := map[string]bool{}
	var out []SourceHit
	norm := func(s string) string {
		return strings.ToLower(strings.TrimSpace(strings.Join(strings.Fields(s), " ")))
	}
	for _, h := range hits {
		key := norm(h.Title)
		if key == "" {
			key = norm(h.DOI)
		}
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, h)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].Citations > out[j].Citations
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// rankScore 简单权威度评分
func rankScore(h SourceHit) float64 {
	s := 0.0
	switch h.Kind {
	case Academic:
		s += 1.0
	case Report:
		s += 0.8
	case Literature:
		s += 0.6
	}
	if h.Citations > 0 {
		s += min(float64(h.Citations)/100, 1.0)
	}
	if h.DOI != "" {
		s += 0.3
	}
	if h.Abstract != "" {
		s += 0.2
	}
	return s
}

// Research 执行一次深度研究：规划 → 并发检索 → 去重排序 → 综合 → 报告
func (o *Orchestrator) Research(ctx context.Context, t ResearchTask) (*ResearchReport, error) {
	if strings.TrimSpace(t.Question) == "" {
		return nil, fmt.Errorf("研究问题不能为空")
	}
	if t.Lang == "" {
		t.Lang = "zh"
	}
	plan := Plan(t)

	// 按计划并发检索
	type result struct {
		source string
		hits   []SourceHit
		err    error
	}
	ch := make(chan result, len(plan.Sources))
	var wg sync.WaitGroup
	for _, name := range plan.Sources {
		src, ok := o.Sources[name]
		if !ok {
			continue
		}
		wg.Add(1)
		go func(s Source) {
			defer wg.Done()
			hits, err := s.Search(ctx, strings.Join(plan.Keywords, " "), SearchOptions{Limit: 5 + t.Depth*3})
			ch <- result{source: s.Name(), hits: hits, err: err}
		}(src)
	}
	wg.Wait()
	close(ch)

	var all []SourceHit
	var failures []string
	for r := range ch {
		if r.err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", r.source, r.err))
			continue
		}
		for i := range r.hits {
			r.hits[i].Score = rankScore(r.hits[i])
		}
		all = append(all, r.hits...)
	}
	evidence := Dedupe(all, 12)

	// 综合摘要（有模型则走模型，否则规则摘要）
	summary := synthesize(ctx, o.Router, t.Question, evidence, t.Lang)
	sections := buildSections(t.Question, evidence, failures, t.Lang)

	// 沉淀项目记忆
	o.Mem.Save(MemoryEntry{
		Level:   "project",
		Project: "via54ScholaBot",
		Title:   t.Question,
		Content: fmt.Sprintf("研究产出 %d 条证据，报告已生成", len(evidence)),
	})

	return &ResearchReport{
		Title:       t.Question,
		Summary:     summary,
		Sections:    sections,
		Evidence:    evidence,
		GeneratedAt: time.Now(),
		Lang:        t.Lang,
	}, nil
}

// synthesize 生成研究摘要
func synthesize(ctx context.Context, r *ModelRouter, question string, ev []SourceHit, lang string) string {
	if len(ev) == 0 {
		if lang == "en" {
			return "No reliable evidence was retrieved. The question remains open to further exploration."
		}
		return "未能从可靠来源检索到足够证据，该问题仍有待深入探索。"
	}
	prompt := fmt.Sprintf(
		"你是资深科研助理。基于以下%d条证据，用%s为研究问题生成一段不超过200字的客观摘要，并指出证据强度。\n问题：%s\n证据：\n",
		len(ev), langName(lang), question)
	for i, h := range ev {
		prompt += fmt.Sprintf("%d. [%s] %s (%s, %s)\n", i+1, h.Source, h.Title, h.Year, h.Venue)
	}
	if r != nil {
		if out, err := r.Chat(ctx, "摘要生成", []ChatMessage{{Role: "user", Content: prompt}}, lang); err == nil && out != "" {
			return out
		}
	}
	// 规则摘要兜底
	titles := make([]string, 0, min(len(ev), 5))
	for _, h := range ev[:min(len(ev), 5)] {
		titles = append(titles, h.Title)
	}
	if lang == "en" {
		return fmt.Sprintf("Evidence scan identified %d unique sources. Key items: %s.", len(ev), strings.Join(titles, "; "))
	}
	return fmt.Sprintf("证据扫描共获得 %d 条去重来源。关键条目：%s。", len(ev), strings.Join(titles, "；"))
}

func langName(lang string) string {
	if lang == "en" {
		return "English"
	}
	return "中文"
}

// buildSections 构建报告章节
func buildSections(question string, ev []SourceHit, failures []string, lang string) []ReportSection {
	var sections []ReportSection
	if len(ev) > 0 {
		head := "证据清单"
		if lang == "en" {
			head = "Evidence List"
		}
		var body strings.Builder
		for i, h := range ev {
			body.WriteString(fmt.Sprintf("%d. **%s** — %s (%s, %s)\n", i+1, h.Title, h.Source, h.Year, h.Venue))
			if h.DOI != "" {
				body.WriteString(fmt.Sprintf("   DOI: %s\n", h.DOI))
			}
			if len(h.Authors) > 0 {
				body.WriteString(fmt.Sprintf("   作者: %s\n", strings.Join(h.Authors, ", ")))
			}
		}
		sections = append(sections, ReportSection{Heading: head, Body: body.String()})
	}
	if len(failures) > 0 {
		head := "数据源状态"
		if lang == "en" {
			head = "Source Status"
		}
		sections = append(sections, ReportSection{Heading: head, Body: strings.Join(failures, "\n")})
	}
	head := "研究建议"
	if lang == "en" {
		head = "Research Suggestions"
	}
	sections = append(sections, ReportSection{Heading: head, Body: researchSuggestions(question, lang)})
	return sections
}

func researchSuggestions(question string, lang string) string {
	if lang == "en" {
		return "1. Broaden the search with synonyms and translated keywords.\n2. Cross-verify key claims against primary sources.\n3. Consider a follow-up deep research round to refine the question."
	}
	return "1. 使用同义词与翻译关键词扩大检索范围。\n2. 关键论断需回溯一手来源交叉验证。\n3. 可发起新一轮深度研究以细化问题。"
}
