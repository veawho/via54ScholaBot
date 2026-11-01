package schola

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ChatMessage 对话消息
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Provider 模型提供商
type Provider struct {
	Name  string
	Base  string
	Key   string // 从环境变量读取
	Model string
}

// ModelRouter 多模型路由与 BYOK 配置
// 支持 OpenAI 兼容协议（GLM/MiniMax/DeepSeek/OpenAI 兼容端点）与 Anthropic 协议。
type ModelRouter struct {
	hc        HTTPClient
	providers map[string]Provider
}

// NewModelRouter 构建模型路由；envPrefix 为环境变量前缀（默认 SCHOLA）
func NewModelRouter(envPrefix string) *ModelRouter {
	if envPrefix == "" {
		envPrefix = "SCHOLA"
	}
	pf := func(name, envKey, base, model string) Provider {
		return Provider{Name: name, Base: base, Key: os.Getenv(envKey), Model: model}
	}
	r := &ModelRouter{hc: defaultClient(), providers: map[string]Provider{}}
	r.register(pf("deepseek", envPrefix+"_DEEPSEEK_API_KEY", "https://api.deepseek.com/v1", "deepseek-chat"))
	r.register(pf("glm", envPrefix+"_GLM_API_KEY", "https://open.bigmodel.cn/api/paas/v4", "glm-4-flash"))
	r.register(pf("minimax", envPrefix+"_MINIMAX_API_KEY", "https://api.minimax.chat/v1", "MiniMax-Text-01"))
	r.register(pf("openai", envPrefix+"_OPENAI_API_KEY", "https://api.openai.com/v1", "gpt-4o-mini"))
	r.register(pf("anthropic", envPrefix+"_ANTHROPIC_API_KEY", "https://api.anthropic.com", "claude-3-5-haiku-latest"))
	return r
}

func (r *ModelRouter) register(p Provider) {
	r.providers[p.Name] = p
}

// Available 返回已配置 API Key 的可用提供商
func (r *ModelRouter) Available() []Provider {
	var out []Provider
	for _, p := range r.providers {
		if p.Key != "" {
			out = append(out, p)
		}
	}
	return out
}

// Chat 调用指定提供商（或自动选择第一个可用）
func (r *ModelRouter) Chat(ctx context.Context, provider string, msgs []ChatMessage, lang string) (string, error) {
	var p Provider
	var ok bool
	if provider != "" {
		p, ok = r.providers[provider]
	} else {
		for _, prov := range r.providers {
			if prov.Key != "" {
				p, ok = prov, true
				break
			}
		}
	}
	if !ok || p.Key == "" {
		return "", fmt.Errorf("provider %q 未配置 API Key（环境变量 %s_*_API_KEY）", provider, "SCHOLA")
	}
	if strings.Contains(p.Base, "anthropic.com") {
		return r.chatAnthropic(ctx, p, msgs)
	}
	return r.chatOpenAICompat(ctx, p, msgs)
}

func (r *ModelRouter) chatOpenAICompat(ctx context.Context, p Provider, msgs []ChatMessage) (string, error) {
	payload := map[string]interface{}{
		"model":    p.Model,
		"messages": msgs,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(p.Base, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.Key)
	resp, err := r.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s chat/completions HTTP %d: %s", p.Name, resp.StatusCode, truncate(string(b), 200))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}
	return out.Choices[0].Message.Content, nil
}

func (r *ModelRouter) chatAnthropic(ctx context.Context, p Provider, msgs []ChatMessage) (string, error) {
	var system string
	var conv []map[string]string
	for _, m := range msgs {
		if m.Role == "system" {
			system = m.Content
			continue
		}
		role := m.Role
		if role == "assistant" {
			role = "assistant"
		} else {
			role = "user"
		}
		conv = append(conv, map[string]string{"role": role, "content": m.Content})
	}
	payload := map[string]interface{}{
		"model":      p.Model,
		"max_tokens": 2048,
		"messages":   conv,
	}
	if system != "" {
		payload["system"] = system
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(p.Base, "/")+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.Key)
	req.Header.Set("anthropic-version", "2023-06-01")
	resp, err := r.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s /v1/messages HTTP %d: %s", p.Name, resp.StatusCode, truncate(string(b), 200))
	}
	var out struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return "", err
	}
	if len(out.Content) == 0 {
		return "", fmt.Errorf("no content returned")
	}
	return out.Content[0].Text, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

var _ = time.Now
