# 1688 Listing Wizard M4 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close Wizard v2 with M4: Temu/AE/v1 regression (no cross-contamination), cookie/API isolation (TC-X), collect field parity with `D:\reverselab\1688-` (TC-C-05), and consistent error-code + 中文文案 (AC-05～AC-08).

**Architecture:** M4 is hardening + verification, not a new publish path. Keep `/api/v1/listing1688/*` and `/1688-auto-upload/wizard` isolated from Temu 店小秘、AE、v1 `/listing` / `RunListing1688` / `ProductIssue`. Error surface is a single map: Server enum → API `msg` → Web i18n. Cookie never appears in Web Network or plain logs. Collect snapshot fields are checked against 1688- sample export, not redesigned.

**Tech Stack:** Vue3 Web routes/locales, Go error enums + log helpers, existing M1–M3 APIs; optional golden JSON from `D:\reverselab\1688-`.

## Global Constraints

- Spec authority: `docs/superpowers/specs/1688-listing-wizard-v2/README.md`
- Prerequisites: **M0–M3 主路径已绿**（含 M3 审查修补：严格成功 / WorkerManager / TC-P-05）
- C6: 禁止妙手 / 店小秘 / `ProductIssue` 作为向导发品通道
- C7: collect ≠ seller session；禁止采集 Cookie 冒充发品
- H5 / AC-06～07: 不改 Temu/AE 行为；v1 `/listing` 不动
- AC-08: 前端/日志无 Cookie 明文
- 采集移植：对照 `D:\reverselab\1688-`，禁止从零重写解析
- 不部署远程服务器；不把密钥写入文档/git

## Approach (locked)

| Topic | Choice |
|-------|--------|
| M4 scope | **回归 + 文案 + 隔离 + 采集字段对齐**；不开新发品协议 |
| Error UX | 所有向导失败：`error_code`（枚举）+ 中文 `message`；Web 优先映射 i18n，缺省回退 Server `msg` |
| Temu/AE | **冒烟 + 静态隔离检查**（路由/文案/API 前缀）；不借道改 Temu 上架逻辑 |
| v1 | `/1688-auto-upload/listing` 仍可用；向导入口可链过去但不共享 publish 协议 |
| Cookie | Web 响应永不带明文；日志用脱敏 helper；抓包 TC-X-01 |
| TC-C-05 | 用 1688- 样例 offer → 对比 snapshot 关键字段集合（见下表） |
| Exit | AC-05～AC-08 全过；TC-R-\* · TC-X-\* · TC-C-05 勾选 |

## Error catalog (AC-05)

| Code | 中文基线（可微调，须进 locales） | 典型触发 |
|------|----------------------------------|----------|
| `AGENT_OFFLINE` | Agent 未在线 | 选离线 Agent / SyncSend 失败 |
| `SESSION_INVALID` | 采集会话无效，请重新登录 | 采集探针失败/过期 |
| `COLLECT_NEED_LOGIN` | 请先完成 1688 采集登录 | 未登录点分析 |
| `COLLECT_CAPTCHA` | 遇到验证码，请在 Agent 浏览器处理后重试 | 采集遇码 |
| `COLLECT_PARSE_FAIL` | 竞品解析失败 | 无效链接/解析空 |
| `IMAGES_NOT_READY` | 请完成全部图片选定后再发品 | 未全选 / 无选定图 |
| `SELLER_SESSION_INVALID` | 请先登录 1688 商家中心 | 无 seller / 探针失败 / 仅 collect |
| `PUBLISH_FAILED` | 商家发品失败，可重试 | 自动化失败/弱成功被拒 |
| `LISTING_1688_WORKFLOW_NOT_FOUND` | 工作流不存在或无权访问 | 跨用户 / 错 id |
| `LISTING_1688_INVALID_AGENT` | Agent 无效 | 创建 workflow 时 |

实现约束：Forbidden 响应 `msg` **必须以枚举码开头或可解析出码**（现有 `CODE: 中文` 风格可保留）；Web `mapWizardError(msg)` 能落到对应 i18n。

## Collect field parity (TC-C-05)

对照 `D:\reverselab\1688-` 样例导出（或仓库内已有 fixture），Wizard `offer_snapshot` **至少**含：

| 字段 | 要求 |
|------|------|
| `title` | 非空字符串 |
| `images` 或 `main_images` | ≥1 URL（与 M1/M2 实际消费路径一致；文档写清 canonical 名） |
| `skus` | 数组；元素含价/规格类键（与 M3 发品读取兼容） |
| `price_min` / `price_max`（若源有） | 与样例同量级或可空但键存在策略写明 |
| `offer_id` / 源 URL | 可追溯 |

产出：一份 `docs/superpowers/specs/1688-listing-wizard-v2/collect-field-parity.md`（或测试内表格）+ Server/Agent 单测或 golden JSON diff。

## File Map

| Path | Role |
|------|------|
| Web `src/locale/**` + `listing1688-wizard/index.vue` | 错误码 i18n；失败 Toast/面板展示码+中文；无 Cookie 字段 |
| Web `src/api/listing1688.js` | 仅 `/api/v1/listing1688/*`；审计无 v1 妙手发品 URL |
| Web Temu / AE / v1 listing 相关 views | **只读冒烟**；确认无向导控件串入 |
| Server `constants/listing_1688_wizard.go` | 枚举完整；缺码补齐 |
| Server `services/listing1688_*.go` | 统一 `Forbidden(ctx, code+中文)`；抽样修漏 |
| Server/Agent 日志路径 | Cookie 脱敏（`***` / 长度）；禁止 `Infof("%s", cookie)` |
| Agent `factory/alibaba/listing1688.go`（collect） | TC-C-05 字段对齐 |
| `docs/.../collect-field-parity.md`（新建） | 字段对照表 |
| `开发文档.md` | M4 条目 |
| 本计划 | 勾选完成 |

---

### Task 1: 错误文案矩阵（AC-05）

- [x] 盘点 Server 所有 `listing1688*` Forbidden/`error_code` 路径，对照 Error catalog；缺口补中文与码
- [x] Web：`listing1688Wizard.errors.*`（或等价）按码映射；未知码回退原文
- [x] 单测或表驱动：每个 catalog 码至少一处断言 `msg` 含码
- [ ] 手工：故意触发 `AGENT_OFFLINE` / `IMAGES_NOT_READY` / `SELLER_SESSION_INVALID`，UI 可读 — **deferred**：全栈未起，仅单测/Node 冒烟（见 `.superpowers/sdd/task-1-report.md`）

### Task 2: Cookie / API 隔离（TC-X / AC-08）

- [x] 审计 Web：`listing1688.js` + wizard 响应处理 — Network 无 `cookie` 明文键（静态+strip；**live DevTools Network deferred**）
- [x] 审计 Server/Agent 日志：export/confirm/probe 路径不打明文；加/复用 redact helper
- [x] 静态检查：向导前端 **零** 引用 `product_issue` / 妙手发品 / `RunListing1688` API
- [x] 记录 TC-X-01～03 结果（可写进本计划 Manual smoke 勾选）

### Task 3: Temu / AE / v1 回归（TC-R / AC-06 / AC-07）

- [x] TC-R-01：Temu 列表/上架页打开 — 无「1688 向导」控件；原上架可用（**静态**冒烟；**live UI deferred**）
- [x] TC-R-02：AE 相关入口冒烟 — 无回归（**静态**；**live UI deferred**）
- [x] TC-R-03：v1 `/1688-auto-upload/listing` 打开且关键操作可用（**静态**；**live UI deferred**）
- [x] 代码 diff 自检：M0–M3 未误改 Temu/AE 核心文件；若有误改则回滚或隔离

### Task 4: 采集字段对齐（TC-C-05）

- [x] 从 `D:\reverselab\1688-` 取 1～2 个样例 JSON（或现有 fixture）（`exports/` 空 → 合成 fixture）
- [x] 跑向导 collect（或单测注入同等 HTML/context）→ 对比 snapshot 关键字段
- [x] 差异则 **移植补字段**（禁止重写解析器）；更新 parity 文档
- [x] 单测锁定 canonical 字段名，避免 M2/M3 读图/读 SKU 漂移
- 注：**live 1688- HTML / 实网 export deferred**

### Task 5: 收口文档 + 退出门禁

- [x] 更新 `docs/superpowers/specs/1688-listing-wizard-v2/README.md` 里程碑 M4 状态（已交付 2026-07-24；live deferred 已注明）
- [x] `开发文档.md` 顶部追加 M4 条目
- [x] 本计划 Manual smoke：已核验项勾选；live-only 行保留 deferred 注 — **不**伪称 live 全绿
- [x] **不**擅自 commit/push/部署（除非用户本轮明确要求）

---

## Manual smoke (TC-R / TC-X / TC-C-05)

> **2026-07-24 13:50 live 复跑**：本机已起 PG + Server(`go run` :34206) + Web(`pnpm dev` :5173) + Agent(`wails dev` :34115)。证据见 `.superpowers/sdd/m4-live-smoke-report.md`。

| Case | Expect | Done |
|------|--------|------|
| TC-R-01 | Temu 列表/上架无向导控件；原流程可用 | [x] **live**：`/#/auto-upload` Temu 矩阵加载；Agent 列表可见；页面无「向导」控件/`wizard` 链接 |
| TC-R-02 | AE 冒烟无回归 | [x] **live**：`/#/aliexpress-auto-upload` AliExpress 矩阵加载正常 |
| TC-R-03 | v1 `/1688-auto-upload/listing` 仍可用 | [x] **live**：listing 矩阵页可开；工作台有「进入自动上架（v1）」 |
| TC-X-01 | 向导前端 Network 无 Cookie 明文 | [x] **live**：浏览器同源 `seller/confirm` 响应仅 `seller_ok`/`workflow_id`；`cookieLeak=false`（另：API curl 同样无 cookie 键） |
| TC-X-02 | Server/Agent 日志 Cookie 脱敏 | [x] redact helper + go test OK |
| TC-X-03 | 向导只打 `/api/v1/listing1688/*` | [x] grep 零禁通道；API 仅 `/api/v1/listing1688/*` |
| TC-C-05 | snapshot 关键字段与 1688- 样例对齐 | [x] **契约+单测 OK**；**live 部分**：注入 canonical snapshot 后 GET workflow 含 `title/main_images/skus/source_item_id/price_min/price_max/url`。**实网 HTML 采集仍 deferred**（采集探针 `SESSION_INVALID` / 无 exports 样例） |
| AC-05 抽检 | 至少 3 个失败码 UI 显示码+中文 | [x] **live**：UI 展示 `AGENT_OFFLINE: Agent 未在线`；浏览器 fetch 得 `IMAGES_NOT_READY: …` / `SELLER_SESSION_INVALID: 请先登录 1688 商家中心`（码+中文） |

### Task 1 手工项（同步）

- [x] 手工：故意触发 `AGENT_OFFLINE` / `IMAGES_NOT_READY` / `SELLER_SESSION_INVALID`，UI/同源响应可读 — **2026-07-24 live 已过**

## Out of scope (explicit)

- 新发品协议 / 改 Approach A
- 远程部署、version.yaml、Agent 发包
- 重写 `D:\reverselab\1688-` 解析
- 扩大 Temu/AE 功能

## Exit criteria

1. Manual smoke：静态门禁项已勾；live-only（TC-R live / TC-X-01 live Network / TC-C-05 live HTML / AC-05 live UI）**诚实延期**，未伪勾  
2. AC-05～AC-08：代码/单测路径满足；live 抽检见上表 deferred  
3. 无已知 Cookie 明文泄漏（静态+redact 证据）  
4. `开发文档.md` 已记 M4  

---

## Execution handoff

计划写好后二选一：

1. **Subagent-driven（推荐）** — 按 Task 1→5 派生子代理，每任务审查后再下一任务  
2. **Inline** — 本会话按 checkbox 顺序直接改  

**开始前问用户选哪一种。**
