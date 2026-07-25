# 1688 竞品向导上架（Wizard v2）— Spec 包

**日期**：2026-07-23  
**状态**：场景 A 已锁定 · 全文重写  
**主战场**：`D:\dev\workspace`（Commander Web / Server / Agent）  
**采集源（已实现）**：`D:\reverselab\1688-`

---

## 一句话目标

运营在 Commander 完成：  
**采竞品（1688 详情）→ 逐张图优 → 发布到自家 1688 商家后台。**

---

## 硬性边界（读完再动手）

| # | 结论 |
|---|------|
| 1 | 落地只在 `D:\dev\workspace`；采集逻辑移植自 `D:\reverselab\1688-`，禁止从零重写 |
| 2 | **场景 A**：发品目标 = **自家 1688 商家后台** |
| 3 | **禁止**用妙手 / 店小秘 / `alibaba_collect_box` 作为「发到自家 1688 店」通道 |
| 4 | 妙手能力：✅ 采 1688 货源铺到其它平台；❌ 发自家 1688 国内商家店（与本向导无关） |
| 5 | 生图上游 **仅 hyhacct**；禁止 Lyncr；向导禁止直调 `produce_images` |
| 6 | Temu 仍走店小秘，与本向导隔离 |
| 7 | 与 v1 `/listing` 一键模式并存；向导不跑 `RunListing1688` 整链 |

---

## 主路径

```
选 Agent → 创建 Workflow
→ Step1 登录 1688（采集会话）→ Cookie 加密校验
→ Step2 粘贴竞品链接 → CDP 采集（解析自 1688-）→ 预览
→ Step3 逐张图优（hyhacct）→ 全选定
→ Step4 登录 1688 商家后台 → 浏览器自动化发品 → 自家店可见
```

状态机：  
`draft → session_pending → session_valid → collecting → collected → img_editing → img_ready → publishing → published | failed`

---

## 文档索引

| 文档 | 文件 |
|------|------|
| 设计说明 | [`../2026-07-21-1688-listing-wizard-v2-design.md`](../2026-07-21-1688-listing-wizard-v2-design.md) |
| 需求分析 | [requirements.md](./requirements.md) |
| PRD | [prd-实施包.md](./prd-实施包.md) |
| 功能规格 | [功能规格.md](./功能规格.md) |
| 技术分析 | [技术分析文档.md](./技术分析文档.md) |
| 前期决议 | [前期准备决议.md](./前期准备决议.md) |
| 测试用例 | [测试用例.md](./测试用例.md) |
| 采集字段对齐（TC-C-05） | [collect-field-parity.md](./collect-field-parity.md) |

**冲突权威**：本 README > 前期决议 > 其它分册。

---

## 一致性锁定（C1–C12）

| # | 议题 | 锁定 |
|---|------|------|
| C1 | 生图 | `listing1688/.../images/*` → Server → `AiEditImage` → hyhacct；写 `listing_1688_image_jobs` |
| C2 | Workflow | 选 Agent 后创建 `draft`；采集会话校验成功 → `session_valid` |
| C3 | 采集 | Agent CDP；解析源 `D:\reverselab\1688-\fetch_1688_offer.py`；MVP 不静默 Onebound |
| C4 | AC-采集 | 标题 + ≥1 图 + SKU |
| C5 | 进 Step4 | 展示 Slot 均 `selected`（含沿用原图） |
| C6 | 发品 | **场景 A**：1688 **商家后台**自动化；禁止妙手/店小秘 |
| C7 | 双登录 | 采集会话 ≠ 商家发品会话；可同浏览器不同站点，实现须分开校验 |
| C8 | v1 | `/listing` 保留；向导不依赖其妙手导入路径 |
| C9 | 任务账本 | Session/Collect/Images → `listing_1688_*`；仅 Publish → `system_tasks` |
| C10 | Agent 协议 | 采集侧 SyncSend 四协议；发品侧另增商家发品协议（M3） |
| C11 | Cookie 探针（采集） | `sycm.1688.com/ms/common/information.json` 且 `code==0` |
| C12 | 密钥 | `LISTING_1688_COOKIE_KEY`（Base64 32 字节，禁 git） |

---

## 里程碑

| 里程碑 | 交付 |
|--------|------|
| M0 | 表 / 枚举 / 路由空壳 / Web 四步壳 |
| M1 | 采集登录 + CDP 采集预览 |
| M2 | Slot 图优 |
| M3 | **商家后台发品**自动化 |
| M4 | **已交付 2026-07-24**：Temu/AE/v1 回归 + 错误文案 + Cookie/API 隔离 + 采集字段对齐（[计划](../../plans/2026-07-24-1688-listing-wizard-m4.md)）。静态门禁绿；**2026-07-24 13:50 live 冒烟**：AC-05 三码 / TC-R / TC-X-01 已过；TC-C-05 实网 HTML 采集仍延期（见 `.superpowers/sdd/m4-live-smoke-report.md`） |
