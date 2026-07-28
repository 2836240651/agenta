# 1688 Listing Wizard M3 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver M3 seller-backend publish for the 1688 wizard (Step4): separate seller login (Agent opens 商家页供人工登录 → 记 Cookie，等同店小秘开页模式) → validate seller session → full-auto fill/submit publish page → `published` on own 1688 shop (AC-04 / C6 / C7 / C9 / C10 / TC-P-*).

**Architecture:** Step4 never uses collect Cookie. Web calls seller session + publish APIs under `/api/v1/listing1688/`. Agent opens **work.1688.com** (+ login Done→工作台) tabs and **keeps them open** (same pattern as startup opening 店小秘 / `work.1688.com`). Confirm exports seller Cookie → Server AES-GCM into `listing_1688_sessions` with `session_type=seller`. Publish creates `system_tasks` with dedicated protocol `listing1688_seller_publish` (not `product_issue` / Miaoshou); Worker SyncSends Agent; Agent Rod-automates 发品页 then returns offerId/success.

**Tech Stack:** Go/Gin/GORM `system_tasks`, Agent Rod CDP, Vue3 Step4 UI.

## Global Constraints

- Spec authority: `docs/superpowers/specs/1688-listing-wizard-v2/README.md`
- **Approach A (user-locked 2026-07-24):** 尽量全自动提交到自家店
- Seller login UX (user-locked): Agent 打开商家相关页供登录（类比开店小秘）；**不关页**；confirm 记 Cookie；再跑自动化
- C6: 禁止妙手 / 店小秘 / `AlibabaFactory.ProductIssue` / `alibaba_collect_box` / `RunListing1688` 整链
- C7: `session_type=collect` ≠ `seller`；发品前必须商家探针通过；**禁止**用采集 Cookie 冒充
- C9: 仅 Publish 走 `system_tasks`
- C5/M2: publish 前 `listing1688AssertImgReady`
- v1 `/listing` untouched
- Selectors centralized in Agent config/constants（改版只改一处）

## Approach (locked)

| Topic | Choice |
|-------|--------|
| Seller open pages | `https://login.1688.com/member/signin.htm?Done=https://work.1688.com/` **and** `https://work.1688.com/`（可多开）；页面保持打开 |
| Seller cookie | Export work/login 相关域合并；加密入库 `session_type=seller` |
| Seller probe | Agent 请求商家工作台态接口（见协议）；`ok` 才允许 publish |
| Publish transport | `WorkerManager.CreateNewTask` + protocol `listing1688_seller_publish` |
| Automation | Rod：进发布入口 → 填标题 → 上传/填入选定图 URL → 填 SKU/价（来自 snapshot，能填尽填）→ 点发布；失败 `PUBLISH_FAILED` 可重试 |
| Success | Agent 回传 `{ok, offer_id?, message}`；Server → workflow `published`；失败 → `failed` + 枚举，**可重试**（回 `img_ready` 或保持可 publish） |
| Collect reuse | **禁止**；seller 协议与 collect 协议分离 |

## Protocols (platform=`1688`)

| Protocol | Send | Receive |
|----------|------|---------|
| `listing1688_seller_open_login` | — | `{login_urls[], opened}` 打开商家登录/工作台页，**不关页** |
| `listing1688_seller_export_cookie` | — | `{cookie}` 明文一次 |
| `listing1688_seller_probe` | `{cookie?}` | `{ok, code, message}` |
| `listing1688_seller_publish` | `{title, images[], skus, price_min, price_max, offer_url?, category_hint?}` | `{ok, offer_id?, message, captcha?}` |

## API contracts

| Method | Behavior |
|--------|----------|
| `POST .../session/seller/open-login` | SyncSend seller_open_login；不改采集 status；可记 message「请在 Agent 浏览器登录商家中心」 |
| `POST .../session/seller/confirm` | export → encrypt → seller probe → 写/更新 seller session；失败 `SELLER_SESSION_INVALID` |
| `POST .../session/seller/validate` | 用已存 seller cookie 再探针 |
| `POST .../publish` | body `{title?}`；assert img_ready + seller valid + agent online → `publishing` + CreateNewTask |
| `GET .../publish/status` | 查 system_task + workflow status |

## File Map

| Path | Role |
|------|------|
| Server `constants/protocol.go` + wizard errors | 新协议；`PUBLISH_FAILED` 向导码（可复用 v1 码或加 wizard 别名） |
| Server `services/listing1688_seller_session.go` | seller open/confirm/validate |
| Server `services/listing1688_publish.go` | publish + status；组装载荷；CreateNewTask |
| Server `router/register.go` | seller session + 已有 publish 路由接线 |
| Server `manager/WorkerManager.go` | case `listing1688_seller_publish` → SyncSend Agent 长超时 |
| Agent `factory/alibaba/listing1688_seller.go` | open/export/probe/publish |
| Agent `listing1688_seller_selectors.go` | URL + 选择器常量 |
| Agent dispatcher + protocol consts | 路由四协议 |
| Web wizard Step4 + `listing1688.js` | 商家登录按钮 + 发布 + 轮询 |
| `开发文档.md` | M3 条目 |

---

### Task 1: Seller session (Server + Agent)

- [ ] Protocols + Agent open work.1688.com pages (**keep open**), export work cookies, seller probe
- [ ] Server seller open-login / confirm / validate；cookie `session_type=seller`；与 collect 隔离单测（TC-P-05）
- [ ] Web Step4：打开商家登录 / 我已登录商家中心

### Task 2: Publish task + Agent automation

- [ ] `Listing1688Publish`：assert ready + seller → CreateNewTask；workflow `publishing`；存 `task_id`（workflow.message 或扩展字段 — **lock: 写入 workflows.message 前缀 `task:` + taskId，或 JSON error_code 旁；优先在 Update 里加可选列 `publish_task_id` 若易改，否则 message 约定 `publish_task_id=<id>`**）
- [ ] WorkerManager 调度 SyncSend `listing1688_seller_publish`（超时 ≥ 180s）
- [ ] Agent：导航发品入口 → 填标题/图/SKU → 提交；选择器集中；失败码 `PUBLISH_FAILED` / captcha
- [ ] 成功/失败回写 workflow；`GET publish/status`；失败可重试

### Task 3: Web Step4 UX + docs

- [ ] 标题可改；选定图预览；「发布到自家 1688 商家店」；无妙手文案
- [ ] 轮询 status；成功提示；失败引导登录商家中心
- [ ] 开发文档 + 计划勾选

---

## Manual smoke (TC-P)

| Case | Expect |
|------|--------|
| TC-P-01 | 商家已登录 + img_ready → 发布 → 自家店可见 / 有 offer_id |
| TC-P-02 | 未商家登录 → `SELLER_SESSION_INVALID` |
| TC-P-03 | 发品页失败 → `PUBLISH_FAILED` 可重试 |
| TC-P-04 | 调用链无 ProductIssue / 妙手 / 店小秘 |
| TC-P-05 | 仅采集登录不能发品成功 |

---

## Self-Review

1. Approach A + Agent 开商家页记 Cookie — locked
2. No Miaoshou publish path
3. Dual session C7
4. system_tasks for publish only
5. Out of scope M3: 多店切换、官方 ERP 替代、完美对抗任意发品页改版（选择器可后续热更）
