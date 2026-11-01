package schola

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractKeywords(t *testing.T) {
	kw := extractKeywords("GLP-1 receptor agonists for weight management in adults")
	if len(kw) < 3 {
		t.Fatalf("expected >=3 keywords, got %v", kw)
	}
	joined := strings.ToLower(strings.Join(kw, " "))
	for _, want := range []string{"glp-1", "receptor", "agonists", "weight"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing keyword %q in %v", want, kw)
		}
	}
	kwCN := extractKeywords("比较司美格鲁肽与二甲双胍在糖尿病治疗中的疗效")
	if len(kwCN) < 3 {
		t.Fatalf("expected >=3 CN keywords, got %v", kwCN)
	}
}

func TestPlan(t *testing.T) {
	p := Plan(ResearchTask{Question: "What are the recent advances in CRISPR gene editing?", Scope: "academic"})
	if len(p.Steps) < 4 {
		t.Fatalf("expected >=4 steps, got %d", len(p.Steps))
	}
	if len(p.Sources) == 0 {
		t.Fatal("expected default sources")
	}
	for _, want := range []string{"pubmed", "openalex", "s2"} {
		found := false
		for _, s := range p.Sources {
			if s == want {
				found = true
			}
		}
		if !found {
			t.Errorf("expected source %q in plan", want)
		}
	}
}

func TestMemoryStore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mem.json")
	s := NewMemoryStore(path)
	if _, err := s.Save(MemoryEntry{Level: "project", Project: "p1", Title: "CRISPR 综述", Content: "证据 3 条"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Save(MemoryEntry{Level: "longterm", Title: "用户偏好：中文优先"}); err != nil {
		t.Fatal(err)
	}
	// 同 title 去重覆盖
	if _, err := s.Save(MemoryEntry{Level: "project", Project: "p1", Title: "CRISPR 综述", Content: "证据 5 条"}); err != nil {
		t.Fatal(err)
	}
	got := s.List("project", "", 10)
	if len(got) != 1 {
		t.Fatalf("expected 1 project entry after dedupe, got %d", len(got))
	}
	if got[0].Content != "证据 5 条" {
		t.Errorf("expected updated content, got %q", got[0].Content)
	}
	c := s.Count()
	if c["project"] != 1 || c["longterm"] != 1 {
		t.Errorf("unexpected counts: %v", c)
	}
	ok, err := s.Forget(got[0].ID)
	if err != nil || !ok {
		t.Fatalf("forget failed: %v %v", ok, err)
	}
	if len(s.List("project", "", 10)) != 0 {
		t.Error("expected project list empty after forget")
	}
	// 重开持久化验证
	s2 := NewMemoryStore(path)
	if len(s2.List("longterm", "", 10)) != 1 {
		t.Error("expected longterm entry persisted")
	}
}

// mockHTTPSource 用 httptest 模拟学术源
func mockHTTPSource(t *testing.T, handler http.HandlerFunc) (map[string]Source, string) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	reg := NewSourceRegistry(srv.Client())
	return reg, srv.URL
}

func TestOpenAlexSource(t *testing.T) {
	reg, base := mockHTTPSource(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"title": "A Study on GLP-1 Agonists", "publication_year": 2025,
					"doi": "10.1000/x", "cited_by_count": 42,
					"authorships": []map[string]interface{}{
						{"author": map[string]interface{}{"display_name": "Alice Chen"}},
					},
					"primary_location": map[string]interface{}{"source": map[string]interface{}{"display_name": "Nature"}},
				},
			},
		})
	})
	src := reg["openalex"]
	if src == nil {
		t.Fatal("openalex source missing")
	}
	src.(*httpSource).base = base
	hits, err := src.Search(context.Background(), "GLP-1", SearchOptions{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	h := hits[0]
	if !strings.Contains(h.Title, "GLP-1") || h.Year != "2025" || h.Citations != 42 {
		t.Errorf("unexpected hit: %+v", h)
	}
}

func TestS2Source(t *testing.T) {
	reg, base := mockHTTPSource(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"title": "Semantic Scholar Paper", "year": 2024,
					"authors": []map[string]interface{}{{"name": "Bob Li"}},
					"venue":   "ACL", "externalIds": map[string]interface{}{"DOI": "10.2000/abc"},
					"citationCount": 7, "abstract": "A short abstract.",
				},
			},
		})
	})
	src := reg["s2"]
	src.(*httpSource).base = base
	hits, err := src.Search(context.Background(), "NLP", SearchOptions{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].DOI != "10.2000/abc" {
		t.Fatalf("unexpected s2 hits: %+v", hits)
	}
}

func TestDedupe(t *testing.T) {
	hits := []SourceHit{
		{Title: "Deep Learning for Science", Source: "openalex", Citations: 100, DOI: "10.1/a"},
		{Title: "Deep Learning  for Science", Source: "s2", Citations: 3, DOI: "10.2/b"},
		{Title: "Another Paper", Source: "arxiv", DOI: "10.3/c"},
	}
	out := Dedupe(hits, 10)
	if len(out) != 2 {
		t.Fatalf("expected 2 after dedupe, got %d: %+v", len(out), out)
	}
	if out[0].Source != "openalex" {
		t.Errorf("expected higher-citation duplicate to win, got %s", out[0].Source)
	}
}

func TestRenderMarkdown(t *testing.T) {
	r := &ResearchReport{Title: "CRISPR 研究", Summary: "摘要", Lang: "zh",
		Evidence: []SourceHit{{Title: "Paper A", Source: "openalex", Year: "2025", Venue: "Nature"}}}
	md := RenderMarkdown(r)
	for _, want := range []string{"CRISPR 研究", "摘要", "Paper A", "via54ScholaBot"} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q", want)
		}
	}
	if !strings.Contains(md, "每一则被证实的证据") {
		t.Errorf("expected poetic quote in zh report")
	}
}

func TestSkills(t *testing.T) {
	s := NewSkills()
	list := s.List()
	if len(list) < 4 {
		t.Fatalf("expected >=4 skills, got %d", len(list))
	}
	if _, ok := s.Get("lit-scan"); !ok {
		t.Error("expected lit-scan skill")
	}
	out, err := s.Run("lit-scan", []SourceHit{{Title: "P", Source: "s", Year: "2025"}})
	if err != nil || !strings.Contains(out, "输入证据 1 条") {
		t.Errorf("skill run failed: %v %q", err, out)
	}
}

func TestModelRouterUnconfigured(t *testing.T) {
	old := os.Getenv("SCHOLA_DEEPSEEK_API_KEY")
	os.Unsetenv("SCHOLA_DEEPSEEK_API_KEY")
	defer os.Setenv("SCHOLA_DEEPSEEK_API_KEY", old)
	r := NewModelRouter("SCHOLA")
	if len(r.Available()) != 0 {
		t.Errorf("expected no available providers without keys, got %v", r.Available())
	}
	_, err := r.Chat(context.Background(), "deepseek", []ChatMessage{{Role: "user", Content: "hi"}}, "zh")
	if err == nil {
		t.Error("expected error when provider unconfigured")
	}
	_ = fmt.Sprint()
}
