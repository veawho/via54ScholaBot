package schola

import (
	"regexp"
	"strings"
)

// defaultSources 按研究范围返回默认数据源
func defaultSources(scope string) []string {
	switch scope {
	case "report":
		return []string{"sec", "openfda", "crossref"}
	case "literature":
		return []string{"unpaywall", "crossref", "openalex"}
	case "academic":
		return []string{"pubmed", "openalex", "s2", "arxiv", "crossref"}
	default: // all
		return []string{"pubmed", "openalex", "s2", "arxiv", "crossref", "sec", "openfda", "unpaywall"}
	}
}

var stopTokens = map[string]bool{
	"的": true, "了": true, "和": true, "是": true, "在": true, "与": true,
	"及": true, "对": true, "中": true, "为": true, "the": true, "a": true,
	"an": true, "of": true, "in": true, "on": true, "for": true, "to": true,
	"and": true, "or": true, "with": true, "by": true, "from": true,
}

var punctRe = regexp.MustCompile(`[^\p{L}\p{N}\s\-]`)

// extractKeywords 从研究问题中提取关键词（中英文；过滤停用词与标点）
func extractKeywords(q string) []string {
	// 中文无空格分词：以中文停用词/连词为分隔符切分
	cnSeps := []string{"比较", "以及", "与", "和", "在", "中", "的", "及", "对", "为", "关于", "如何", "是否", "哪些", "什么", "怎样"}
	clean := punctRe.ReplaceAllString(q, " ")
	parts := strings.Fields(clean)
	seen := map[string]bool{}
	var kw []string
	add := func(p string) {
		lower := strings.ToLower(p)
		if stopTokens[lower] || len(p) <= 1 {
			return
		}
		if !seen[lower] {
			seen[lower] = true
			kw = append(kw, p)
		}
	}
	for _, p := range parts {
		if isCJK(p) {
			segs := []string{p}
			for _, sep := range cnSeps {
				var next []string
				for _, s := range segs {
					chunks := strings.Split(s, sep)
					for _, c := range chunks {
						if c != "" {
							next = append(next, c)
						}
					}
				}
				segs = next
			}
			for _, s := range segs {
				add(s)
			}
			continue
		}
		add(p)
	}
	if len(kw) > 8 {
		kw = kw[:8]
	}
	return kw
}

func isCJK(s string) bool {
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}

// Plan 为研究任务生成多步研究计划
func Plan(t ResearchTask) ResearchPlan {
	scope := t.Scope
	if scope == "" {
		scope = "all"
	}
	kw := extractKeywords(t.Question)
	if len(kw) == 0 {
		kw = []string{strings.TrimSpace(t.Question)}
	}
	srcs := t.Sources
	if len(srcs) == 0 {
		srcs = defaultSources(scope)
	}
	steps := []PlanStep{
		{Order: 1, Action: "search", Detail: "多源并发检索：获取候选文献/报告，覆盖 " + strings.Join(srcs, ", ")},
		{Order: 2, Action: "dedupe", Detail: "跨源去重与权威度排序"},
		{Order: 3, Action: "read", Detail: "全文获取与关键论据抽取"},
		{Order: 4, Action: "synthesize", Detail: "证据综合与研究结论生成"},
		{Order: 5, Action: "report", Detail: "研究报告渲染（中英双语模板）"},
	}
	return ResearchPlan{
		Question: t.Question,
		Keywords: kw,
		Sources:  srcs,
		Steps:    steps,
	}
}
