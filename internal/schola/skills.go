package schola

import (
	"fmt"
	"sort"
	"strings"
)

// Skills 技能库：为深度研究提供可复用的科研工作流
type Skills struct {
	skills map[string]Skill
}

// NewSkills 注册内置科研技能
func NewSkills() *Skills {
	s := &Skills{skills: map[string]Skill{}}
	s.register(Skill{
		Name:        "lit-scan",
		Description: "文献速读：对一批文献生成结构化摘要与关键结论",
		Kind:        "literature",
	})
	s.register(Skill{
		Name:        "review-outline",
		Description: "综述大纲：将证据清单组织为系统综述章节大纲",
		Kind:        "academic",
	})
	s.register(Skill{
		Name:        "evidence-grade",
		Description: "证据评级：按来源与引用情况给出证据强度评级",
		Kind:        "academic",
	})
	s.register(Skill{
		Name:        "report-draft",
		Description: "报告起草：基于检索证据生成中英双语研究报告",
		Kind:        "report",
	})
	s.register(Skill{
		Name:        "citation-check",
		Description: "引用核查：核对文献引用真实性与一致性",
		Kind:        "literature",
	})
	return s
}

func (s *Skills) register(sk Skill) { s.skills[sk.Name] = sk }

// List 列出全部技能
func (s *Skills) List() []Skill {
	out := make([]Skill, 0, len(s.skills))
	for _, sk := range s.skills {
		out = append(out, sk)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Get 获取单个技能
func (s *Skills) Get(name string) (Skill, bool) {
	sk, ok := s.skills[name]
	return sk, ok
}

// Run 执行技能（当前为结构化提示词模板引擎，供上层接入模型后调用）
func (s *Skills) Run(name string, evidence []SourceHit) (string, error) {
	sk, ok := s.skills[name]
	if !ok {
		return "", fmt.Errorf("未知技能: %s（可用: %s）", name, strings.Join(skillNames(s), ", "))
	}
	var b strings.Builder
	b.WriteString("【技能】" + sk.Name + "\n")
	b.WriteString(sk.Description + "\n\n")
	b.WriteString(fmt.Sprintf("输入证据 %d 条：\n", len(evidence)))
	for i, h := range evidence {
		b.WriteString(fmt.Sprintf("%d. %s — %s (%s)\n", i+1, h.Title, h.Source, h.Year))
	}
	b.WriteString("\n请按该技能要求输出结构化结果。")
	return b.String(), nil
}

func skillNames(s *Skills) []string {
	var names []string
	for _, sk := range s.skills {
		names = append(names, sk.Name)
	}
	sort.Strings(names)
	return names
}
