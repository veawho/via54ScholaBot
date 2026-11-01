package schola

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPClient 可替换的 HTTP 客户端（便于测试注入 mock）
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type httpSource struct {
	name string
	kind SourceKind
	base string
	hc   HTTPClient
	// build 将查询转换为 HTTP 请求并解析响应
	build func(ctx context.Context, base, q string, opts SearchOptions, hc HTTPClient) ([]SourceHit, error)
}

func (s *httpSource) Name() string     { return s.name }
func (s *httpSource) Kind() SourceKind { return s.kind }

func (s *httpSource) Search(ctx context.Context, q string, opts SearchOptions) ([]SourceHit, error) {
	if opts.Limit <= 0 {
		opts.Limit = 10
	}
	return s.build(ctx, s.base, q, opts, s.hc)
}

func defaultClient() HTTPClient {
	return &http.Client{Timeout: 20 * time.Second}
}

func getJSON(ctx context.Context, hc HTTPClient, rawURL string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "via54ScholaBot/0.1 (research engine; mailto:via54@example.com)")
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: HTTP %d", rawURL, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func getXML(ctx context.Context, hc HTTPClient, rawURL string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "via54ScholaBot/0.1")
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: HTTP %d", rawURL, resp.StatusCode)
	}
	return xml.NewDecoder(resp.Body).Decode(out)
}

func getText(ctx context.Context, hc HTTPClient, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "via54ScholaBot/0.1")
	resp, err := hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ---- OpenAlex ----

type openalexResp struct {
	Results []struct {
		Title       string `json:"title"`
		Year        int    `json:"publication_year"`
		Doi         string `json:"doi"`
		CitedBy     int    `json:"cited_by_count"`
		Authorships []struct {
			Author struct {
				DisplayName string `json:"display_name"`
			} `json:"author"`
		} `json:"authorships"`
		PrimaryLocation struct {
			Source struct {
				DisplayName string `json:"display_name"`
			} `json:"source"`
		} `json:"primary_location"`
	} `json:"results"`
}

func buildOpenAlex(ctx context.Context, base, q string, opts SearchOptions, hc HTTPClient) ([]SourceHit, error) {
	u := base + "/works?search=" + url.QueryEscape(q) + fmt.Sprintf("&per-page=%d", opts.Limit)
	var resp openalexResp
	if err := getJSON(ctx, hc, u, &resp); err != nil {
		return nil, err
	}
	hits := make([]SourceHit, 0, len(resp.Results))
	for _, r := range resp.Results {
		h := SourceHit{Source: "openalex", Kind: Academic, Title: r.Title,
			Year: fmt.Sprintf("%d", r.Year), DOI: r.Doi, Citations: r.CitedBy}
		if r.PrimaryLocation.Source.DisplayName != "" {
			h.Venue = r.PrimaryLocation.Source.DisplayName
		}
		for _, a := range r.Authorships {
			if a.Author.DisplayName != "" {
				h.Authors = append(h.Authors, a.Author.DisplayName)
			}
		}
		if len(h.Authors) > 5 {
			h.Authors = h.Authors[:5]
		}
		hits = append(hits, h)
	}
	return hits, nil
}

// ---- Semantic Scholar ----

type s2Resp struct {
	Data []struct {
		Title   string `json:"title"`
		Year    int    `json:"year"`
		Authors []struct {
			Name string `json:"name"`
		} `json:"authors"`
		Venue       string `json:"venue"`
		ExternalIds struct {
			DOI string `json:"DOI"`
		} `json:"externalIds"`
		CitationCount int    `json:"citationCount"`
		Abstract      string `json:"abstract"`
	} `json:"data"`
}

func buildS2(ctx context.Context, base, q string, opts SearchOptions, hc HTTPClient) ([]SourceHit, error) {
	u := base + "/graph/v1/paper/search?query=" + url.QueryEscape(q) +
		fmt.Sprintf("&limit=%d&fields=title,year,authors,venue,externalIds,citationCount,abstract", opts.Limit)
	var resp s2Resp
	if err := getJSON(ctx, hc, u, &resp); err != nil {
		return nil, err
	}
	hits := make([]SourceHit, 0, len(resp.Data))
	for _, r := range resp.Data {
		h := SourceHit{Source: "s2", Kind: Academic, Title: r.Title,
			Year: fmt.Sprintf("%d", r.Year), Venue: r.Venue, DOI: r.ExternalIds.DOI,
			Citations: r.CitationCount, Abstract: r.Abstract}
		for _, a := range r.Authors {
			h.Authors = append(h.Authors, a.Name)
		}
		if len(h.Authors) > 5 {
			h.Authors = h.Authors[:5]
		}
		hits = append(hits, h)
	}
	return hits, nil
}

// ---- PubMed E-utilities ----

type pubmedESearch struct {
	IDList []string `xml:"IdList>Id"`
}

type pubmedESummary struct {
	DocSum []struct {
		ID      string `xml:"Id"`
		Title   string `xml:"Item Name=Title"`
		PubDate string `xml:"Item Name=PubDate"`
		Authors []struct {
			Name string `xml:"Name"`
		} `xml:"Item Name=AuthorList>Item"`
		Source string `xml:"Item Name=Source"`
		DOI    string `xml:"Item Name=DOI"`
	} `xml:"DocSum"`
}

func buildPubMed(ctx context.Context, base, q string, opts SearchOptions, hc HTTPClient) ([]SourceHit, error) {
	esURL := base + "/entrez/eutils/esearch.fcgi?db=pubmed&retmode=json&retmax=" +
		fmt.Sprintf("%d", opts.Limit) + "&term=" + url.QueryEscape(q)
	var es struct {
		EsSearchResult struct {
			IDList []string `json:"IdList"`
		} `json:"esearchresult"`
	}
	if err := getJSON(ctx, hc, esURL, &es); err != nil {
		return nil, err
	}
	if len(es.EsSearchResult.IDList) == 0 {
		return nil, nil
	}
	ids := strings.Join(es.EsSearchResult.IDList, ",")
	suURL := base + "/entrez/eutils/esummary.fcgi?db=pubmed&retmode=xml&id=" + ids
	var su pubmedESummary
	if err := getXML(ctx, hc, suURL, &su); err != nil {
		return nil, err
	}
	hits := make([]SourceHit, 0, len(su.DocSum))
	for _, d := range su.DocSum {
		h := SourceHit{Source: "pubmed", Kind: Academic, Title: d.Title,
			Venue: d.Source, Year: firstYear(d.PubDate), DOI: d.DOI}
		for _, a := range d.Authors {
			if a.Name != "" {
				h.Authors = append(h.Authors, a.Name)
			}
		}
		if len(h.Authors) > 5 {
			h.Authors = h.Authors[:5]
		}
		hits = append(hits, h)
	}
	return hits, nil
}

func firstYear(s string) string {
	for _, c := range strings.FieldsFunc(s, func(r rune) bool { return r < '0' || r > '9' }) {
		if len(c) == 4 {
			return c
		}
	}
	if len(s) >= 4 {
		return s[:4]
	}
	return s
}

// ---- arXiv ----

type arxivResp struct {
	Entries []struct {
		Title     string `xml:"title"`
		Summary   string `xml:"summary"`
		ID        string `xml:"id"`
		Published string `xml:"published"`
		Authors   []struct {
			Name string `xml:"name"`
		} `xml:"author"`
	} `xml:"entry"`
}

func buildArxiv(ctx context.Context, base, q string, opts SearchOptions, hc HTTPClient) ([]SourceHit, error) {
	u := base + "/api/query?search_query=all:" + url.QueryEscape(q) +
		fmt.Sprintf("&start=0&max_results=%d", opts.Limit)
	var resp arxivResp
	if err := getXML(ctx, hc, u, &resp); err != nil {
		return nil, err
	}
	hits := make([]SourceHit, 0, len(resp.Entries))
	for _, e := range resp.Entries {
		h := SourceHit{Source: "arxiv", Kind: Academic, Title: strings.TrimSpace(e.Title),
			Abstract: strings.TrimSpace(e.Summary), Year: firstYear(e.Published), URL: strings.TrimSpace(e.ID)}
		for _, a := range e.Authors {
			if a.Name != "" {
				h.Authors = append(h.Authors, a.Name)
			}
		}
		if len(h.Authors) > 5 {
			h.Authors = h.Authors[:5]
		}
		hits = append(hits, h)
	}
	return hits, nil
}

// ---- Crossref ----

type crossrefResp struct {
	Message struct {
		Items []struct {
			Title          []string `json:"title"`
			ContainerTitle []string `json:"container-title"`
			Published      struct {
				DateParts [][]int `json:"date-parts"`
			} `json:"published"`
			DOI    string `json:"DOI"`
			Author []struct {
				Given  string `json:"given"`
				Family string `json:"family"`
			} `json:"author"`
			IsReference bool `json:"is-referenced-by-count"`
		} `json:"items"`
	} `json:"message"`
}

func buildCrossref(ctx context.Context, base, q string, opts SearchOptions, hc HTTPClient) ([]SourceHit, error) {
	u := base + "/works?query=" + url.QueryEscape(q) + fmt.Sprintf("&rows=%d", opts.Limit)
	var resp crossrefResp
	if err := getJSON(ctx, hc, u, &resp); err != nil {
		return nil, err
	}
	hits := make([]SourceHit, 0, len(resp.Message.Items))
	for _, r := range resp.Message.Items {
		h := SourceHit{Source: "crossref", Kind: Academic, DOI: r.DOI, Year: firstYear(fmt.Sprintf("%v", r.Published.DateParts))}
		if len(r.Title) > 0 {
			h.Title = r.Title[0]
		}
		if len(r.ContainerTitle) > 0 {
			h.Venue = r.ContainerTitle[0]
		}
		for _, a := range r.Author {
			name := strings.TrimSpace(a.Given + " " + a.Family)
			if name != "" {
				h.Authors = append(h.Authors, name)
			}
		}
		if len(h.Authors) > 5 {
			h.Authors = h.Authors[:5]
		}
		hits = append(hits, h)
	}
	return hits, nil
}

// ---- SEC EDGAR（公司提交检索）----

func buildSEC(ctx context.Context, base, q string, opts SearchOptions, hc HTTPClient) ([]SourceHit, error) {
	// SEC 检索接口按公司名近似匹配（示例：搜索公司简称）
	u := base + "/cgi-bin/browse-edgar?action=getcompany&company=" +
		url.QueryEscape(q) + "&type=10-K&output=atom"
	var r struct {
		Results []struct {
			CompanyName string `xml:"companyName"`
			FormType    string `xml:"formType"`
			FilingDate  string `xml:"filingDate"`
			FilingHref  string `xml:"filingHref"`
		} `xml:"results>result"`
	}
	if err := getXML(ctx, hc, u, &r); err != nil {
		return nil, err
	}
	hits := make([]SourceHit, 0, len(r.Results))
	for _, x := range r.Results {
		h := SourceHit{Source: "sec", Kind: Report, Title: x.CompanyName + " " + x.FormType,
			Venue: "SEC EDGAR", Year: firstYear(x.FilingDate), URL: x.FilingHref}
		hits = append(hits, h)
	}
	return hits, nil
}

// ---- OpenFDA（药品报告检索）----

func buildOpenFDA(ctx context.Context, base, q string, opts SearchOptions, hc HTTPClient) ([]SourceHit, error) {
	u := base + "/drug/event.json?search=patient.drug.openfda.brand_name:" +
		url.QueryEscape(q) + "&limit=" + fmt.Sprintf("%d", opts.Limit)
	var r struct {
		Results []struct {
			Receivedate      string `json:"receivedate"`
			Seriousnessdeath int    `json:"seriousnessdeath"`
			Patient          struct {
				Patientonsetage int `json:"patientonsetage"`
			} `json:"patient"`
		} `json:"results"`
	}
	if err := getJSON(ctx, hc, u, &r); err != nil {
		return nil, err
	}
	hits := make([]SourceHit, 0, len(r.Results))
	for i, x := range r.Results {
		h := SourceHit{Source: "openfda", Kind: Report,
			Title: fmt.Sprintf("FAERS 不良事件报告 #%d (%s)", i+1, x.Receivedate),
			Venue: "FDA FAERS", Year: firstYear(x.Receivedate)}
		hits = append(hits, h)
	}
	return hits, nil
}

// ---- Unpaywall（文献全文开放获取）----

func buildUnpaywall(ctx context.Context, base, q string, opts SearchOptions, hc HTTPClient) ([]SourceHit, error) {
	// Unpaywall 按 DOI 查询开放获取全文
	doi := q
	if strings.Contains(q, "10.") && strings.Contains(q, "/") {
		doi = q
	} else {
		return nil, fmt.Errorf("unpaywall 需要 DOI 作为查询（收到 %q）", q)
	}
	u := base + "/v2/" + url.PathEscape(doi) + "?email=via54@example.com"
	var r struct {
		Title  string `json:"title"`
		IsOA   bool   `json:"is_oa"`
		BestOA struct {
			URL  string `json:"url_for_pdf"`
			Host string `json:"host_type"`
		} `json:"best_oa_location"`
	}
	if err := getJSON(ctx, hc, u, &r); err != nil {
		return nil, err
	}
	h := SourceHit{Source: "unpaywall", Kind: Literature, Title: r.Title, DOI: doi, URL: r.BestOA.URL}
	if r.IsOA {
		h.Score = 1
	}
	return []SourceHit{h}, nil
}

// NewSourceRegistry 注册全部内置源
func NewSourceRegistry(hc HTTPClient) map[string]Source {
	if hc == nil {
		hc = defaultClient()
	}
	reg := map[string]Source{
		"openalex":  &httpSource{name: "openalex", kind: Academic, base: "https://api.openalex.org", hc: hc, build: buildOpenAlex},
		"s2":        &httpSource{name: "s2", kind: Academic, base: "https://api.semanticscholar.org", hc: hc, build: buildS2},
		"pubmed":    &httpSource{name: "pubmed", kind: Academic, base: "https://eutils.ncbi.nlm.nih.gov", hc: hc, build: buildPubMed},
		"arxiv":     &httpSource{name: "arxiv", kind: Academic, base: "http://export.arxiv.org", hc: hc, build: buildArxiv},
		"crossref":  &httpSource{name: "crossref", kind: Academic, base: "https://api.crossref.org", hc: hc, build: buildCrossref},
		"sec":       &httpSource{name: "sec", kind: Report, base: "https://www.sec.gov", hc: hc, build: buildSEC},
		"openfda":   &httpSource{name: "openfda", kind: Report, base: "https://api.fda.gov", hc: hc, build: buildOpenFDA},
		"unpaywall": &httpSource{name: "unpaywall", kind: Literature, base: "https://api.unpaywall.org", hc: hc, build: buildUnpaywall},
	}
	return reg
}
