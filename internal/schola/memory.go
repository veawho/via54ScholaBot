package schola

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryStore 三层记忆存储（会话/项目/长期），JSON 文件持久化
type MemoryStore struct {
	mu      sync.Mutex
	path    string
	entries map[string]MemoryEntry
}

// DefaultMemoryPath 返回默认记忆文件路径
func DefaultMemoryPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".via54schola", "memory.json")
}

// NewMemoryStore 打开（或创建）记忆库；path 为空时使用默认路径
func NewMemoryStore(path string) *MemoryStore {
	if path == "" {
		path = DefaultMemoryPath()
	}
	s := &MemoryStore{path: path, entries: map[string]MemoryEntry{}}
	s.load()
	return s
}

func (s *MemoryStore) load() {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var list []MemoryEntry
	if err := json.Unmarshal(b, &list); err != nil {
		return
	}
	for _, e := range list {
		s.entries[e.ID] = e
	}
}

func (s *MemoryStore) persist() error {
	// 调用方需持有 mu；此处不再加锁，避免与 Save/Forget 死锁
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	list := make([]MemoryEntry, 0, len(s.entries))
	for _, e := range s.entries {
		list = append(list, e)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt > list[j].CreatedAt })
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0o644)
}

// Save 保存一条记忆；同 (level,title) 自动去重覆盖
func (s *MemoryStore) Save(e MemoryEntry) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.ID == "" {
		e.ID = newID(e.Level + "|" + e.Title)
	}
	if e.CreatedAt == "" {
		e.CreatedAt = time.Now().Format(time.RFC3339)
	}
	if e.Level == "" {
		e.Level = "session"
	}
	for id, old := range s.entries {
		if old.Level == e.Level && old.Title == e.Title {
			e.ID = id
			delete(s.entries, id)
			break
		}
	}
	s.entries[e.ID] = e
	return e.ID, s.persist()
}

// List 列出记忆（可按层级/项目过滤）
func (s *MemoryStore) List(level, project string, limit int) []MemoryEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []MemoryEntry
	for _, e := range s.entries {
		if level != "" && e.Level != level {
			continue
		}
		if project != "" && !strings.Contains(strings.ToLower(e.Project), strings.ToLower(project)) {
			continue
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// Forget 删除指定 ID 的记忆
func (s *MemoryStore) Forget(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.entries[id]; !ok {
		return false, nil
	}
	delete(s.entries, id)
	return true, s.persist()
}

func newID(seed string) string {
	sum := sha256.Sum256([]byte(seed + time.Now().String()))
	return "mem_" + hex.EncodeToString(sum[:8])
}

// Count 返回各层级条目数
func (s *MemoryStore) Count() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]int{"session": 0, "project": 0, "longterm": 0}
	for _, e := range s.entries {
		out[e.Level]++
	}
	return out
}

// String 打印存储位置与条目数
func (s *MemoryStore) String() string {
	c := s.Count()
	return fmt.Sprintf("memory store: %s (session=%d project=%d longterm=%d)",
		s.path, c["session"], c["project"], c["longterm"])
}
