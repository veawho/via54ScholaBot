# via54ScholaBot · 学术深研引擎 / The Deep-Research Engine

> **人类群星闪耀时 · Moments of Starlight in Human Scholarship**
>
> 每一篇论文，都是一颗星；每一份报告，都是一座灯塔。
> 当检索的晨光穿透数据库的云层，当证据的潮水涌向思考的堤岸——
> 此间的引擎，愿做你案头那盏不灭的灯。
>
> Every paper is a star; every report, a lighthouse.
> When the dawn of retrieval pierces the clouds of databases, when tides of evidence wash upon the shore of thought —
> this engine shall be the undying lamp upon your desk.

via54ScholaBot 是集**学术检索及研究、报告检索及研究、文献检索及研究**于一体的深度研究引擎。它由 `via54Medit` 与「攻玉学术：科研 AI Harness」之理念深度融合而生，以 Go 构建，模块清晰、接口优雅，可在本地、服务器或 CI 中一键运行。

via54ScholaBot is a deep-research engine uniting **academic search & research, report search & research, and literature search & research**. Born from the fusion of `via54Medit` with the philosophy of "Jadense: A Research AI Harness", it is written in Go — modular, elegant, and ready to run locally, on servers, or in CI.

## ✦ 星图 / The Star Map

| 能力 / Capability | 中文释义 | 数据源 / Sources |
|---|---|---|
| **Academic** | 学术检索及研究：多源并发，去重排序，证据综合 | OpenAlex, Semantic Scholar, PubMed, arXiv, Crossref |
| **Report** | 报告检索及研究：企业披露、监管数据、药物警戒 | SEC EDGAR, OpenFDA (FAERS), Crossref |
| **Literature** | 文献检索及研究：DOI 全文开放获取定位 | Unpaywall, PubMed, Crossref |
| **Memory** | 三层记忆：会话 / 项目 / 长期，JSON 持久化 | local store |
| **Router** | BYOK 多模型路由：自带 Key，随插随用 | DeepSeek, MiniMax, GLM, OpenAI-compatible |
| **Skills** | 科研技能库：综述大纲、证据评级、引用核查、报告起草、文献速读 | built-in |

## ✦ 启程 / Setting Sail

```bash
# 构建 / Build
go build -o build/schola ./cmd/schola

# 深度研究 / Deep research
./schola research "CRISPR gene editing advances" --scope all --depth 2 --out research.md

# 学术检索及研究 / Academic
./schola academic "GLP-1 receptor agonists"

# 报告检索及研究 / Report
./schola report "Tesla"

# 文献检索及研究 / Literature (by DOI)
./schola lit "10.1038/s41586-020-2649-2"

# 记忆与技能 / Memory & Skills
./schola memory save "CRISPR-task" "collect 2025 evidence"
./schola skill list
./schola doctor
```

### 环境变量 / Environment

| 变量 | 用途 |
|---|---|
| `SCHOLA_DEEPSEEK_API_KEY` | DeepSeek 模型路由（BYOK） |
| `SCHOLA_MINIMAX_API_KEY` | MiniMax 模型路由 |
| `SCHOLA_GLM_API_KEY` | 智谱 GLM 模型路由 |
| `SCHOLA_OPENAI_API_KEY` / `SCHOLA_OPENAI_BASE_URL` | 任意 OpenAI 兼容端点 |

未配置任何 Key 时，引擎仍可完成多源检索、去重、记忆与报告骨架生成（离线模式）。

## ✦ 架构 / Architecture

```
cmd/schola          CLI 入口（cobra）
internal/schola     —— 深度研究核心
    ├─ schola.go        领域模型（ResearchTask / Report / Evidence）
    ├─ planner.go       研究规划与中文关键词提取
    ├─ sources.go       8 大数据源适配器（JSON / XML）
    ├─ orchestrator.go  并发检索 → 去重 → 综合 → 报告
    ├─ memory.go        三层记忆（会话 / 项目 / 长期）
    ├─ router.go        BYOK 多模型路由（OpenAI 兼容）
    ├─ skills.go        科研技能库
    └─ report.go        中英双语诗意化渲染（Markdown / HTML）
```

## ✦ 测试 / Trials of the Engine

```bash
go test ./internal/... -count=1
go vet ./internal/... ./cmd/...
```

内置 9 项单元测试覆盖：关键词提取、研究规划、三层记忆（含去重与持久化）、OpenAlex / Semantic Scholar 检索适配（httptest mock）、跨源去重、诗意化渲染、技能执行、模型路由降级。

## ✦ 协议 / License

[AGPL v3.0](LICENSE) — 自由软件，永续闪耀。

此项目的灵魂属于所有愿意分享知识的人。你的修改与分发，同样须以 AGPL v3.0 向世界敞开。

The soul of this project belongs to all who share knowledge. Your modifications and distributions must likewise be opened to the world under AGPL v3.0.

## ✦ 致谢 / Gratitude

致 via54Medit 的先行者们，致「攻玉学术」——他山之石，可以攻玉；致所有在深夜点击"搜索"按钮的研究者。

To the pioneers of via54Medit, to Jadense — *stones from other hills may serve to polish the jade*; and to every researcher who clicks "Search" deep into the night.

---

*通过 antigravity 开发 · Crafted with antigravity*
*Go · AGPL-3.0 · Built for the curious*
