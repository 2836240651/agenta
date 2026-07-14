# 速卖通全托管（CSP）独立上架 MVP — 设计说明

**日期**：2026-07-14  
**修订**：2026-07-14（审查修订 v2）  
**状态**：待用户审阅  
**仓库路径（建议）**：`D:\dev\workspace\aliexpress-csp-auto-upload`  
**Git**：独立 git 仓库（自有 `origin`）；**不是** Commander / Agent 子工程目录，也不并入其 mono 提交树  
**参考**：`D:\multiPlaformUpLoad\social-auto-upload`（抖音上传流水线）、`D:\dev\workspace\aliexpress-seller-api-mvp`（仅思路参考，不作主代码树）

---

## 1. 背景与目标

### 1.1 背景

Commander Agent 当前速卖通上架依赖 **妙手 ERP**（生成导入 Excel → OSS → `processImportCopyV2`）。目标改为：**不依赖妙手**，对官方 **全托管 CSP** 卖家后台做 **页面登录 + 抓包 + Cookie/CSRF API 回放**。

### 1.2 本阶段目标（独立 MVP）

在独立新仓库中跑通：

`登录 → 会话校验 →（人工发品抓包）→ Cookie/CSRF API 回放创建/发布 → 刷新 cookie`

- 输入：最少字段的 `product.json`
- UI：仅用于登录、鉴权失效兜底、（可选）人工过滑块/短信
- **不在本阶段**：Excel 适配、接入 Commander Agent / Web、UI 全自动填表发品

### 1.3 二期（本设计仅预留，不实现）

1. Excel → JSON 适配层（对齐 Commander「速卖通自动上架」模板列）
2. 替换/旁路 Agent `AliExpressFactory` 中的妙手路径；Web 入口保持不变

### 1.4 院线与入口

| 项 | 值 |
|----|-----|
| 院线 | 速卖通 **全托管**（CSP / Choice Seller Platform） |
| 登录 | `https://login.aliexpress.com/user/seller/login?bizSegment=CSP&return_url=...` |
| 发品回跳 | `https://csp.aliexpress.com/ait/cn_ams/item_choice/product_publish?...` |
| 主域名 | `csp.aliexpress.com` 及相关网关（抓包确认） |

**按账号配置（硬性）**：

- `conf.py` 可放**默认**完整登录 URL（含 `return_url`）。
- `return_url` 中的 `switchId`、`channelId` 等参数通常与**店铺/渠道**绑定，**禁止**假定全局唯一。
- 每个账号允许覆盖：`accounts/<account>.yaml`（或旁路配置）提供 `login_url` / `publish_url` / shop 相关参数；解析顺序：账号配置 > `conf` 默认。

---

## 2. 非目标

- 调用 `erp.91miaoshou.com` / 店小秘等第三方 ERP
- 经典海外自营卖家中心（`seller.aliexpress.com`）完整发品（旧 Scheme B 范围）
- 自动破解滑块 / 短信 / 二次验证
- 多账号并发展品、生产级调度队列
- 在本仓直接改 Commander Web/Server 路由
- 把本仓源码嵌进 Commander Agent 仓库树（二期以进程/包调用接入，不 mono 混交）

---

## 3. 方案选型

| 方案 | 结论 |
|------|------|
| SAU 风格新仓 + 抓包回放（Python / Patchright） | **采用** |
| 继续扩展 `aliexpress-seller-api-mvp`（TS） | 否；旧仓仅参考 |
| UI RPA 填表为主 | 否；仅登录/兜底 |

技术栈：

- **Python 3.11+**
- **Patchright**（与 `social-auto-upload` 对齐；Playwright API 兼容）
- CLI 安装后命令名：**`ae-csp`**
- **回放 HTTP 客户端（MVP 指定）**：Patchright / Playwright **`APIRequestContext`**，从 `storage_state` 创建 `browser.new_context(storage_state=...)` 后取 `context.request`（或等价）发业务请求。便于携带浏览器 Cookie、处理 `Set-Cookie`，并在结束后 `context.storage_state(path=...)` 回写。
- **不采用**纯 httpx/requests 作为 MVP 默认回放通道（二期若验证等价可再评估）。

---

## 4. 架构

```text
CLI (ae-csp)
  ├─ login / check     → Auth（有头浏览器 + storage_state）
  ├─ capture           → 录制 CSP 相关请求 → captures/
  └─ publish           → APIRequestContext 按 endpoints 回放
         ↑
cookies/csp_<account>.json
accounts/<account>.yaml      # 可选；login_url / publish_url 覆盖
captures/<session>/
products/*.json
```

| 层 | 职责 |
|----|------|
| CLI | 子命令、参数、退出码、日志 |
| Auth | `login` / `cookie_auth` / `setup`；UI 仅登录与失效兜底 |
| Capture | 打开 `product_publish`，过滤并落盘请求 |
| Publisher | schema 校验 → 上传媒体（若需要）→ 创建/编辑 → SKU → 提交 → 刷新 `storage_state` |
| Product schema | MVP 用 JSON；`extras` 承接未建模字段 |
| endpoints | 抓包确认后固化的最小接口表 |

---

## 5. 目录结构

```text
aliexpress-csp-auto-upload/
├── pyproject.toml
├── README.md
├── conf.py
├── ae_cli.py
├── .gitignore                 # cookies/、captures/、accounts/*.yaml 敏感项
├── cookies/                   # gitignore；csp_<account>.json
├── accounts/                  # gitignore 或仅提交 *.example.yaml
│   └── demo.example.yaml
├── captures/                  # gitignore
├── products/
│   └── demo.json
├── uploader/
│   └── csp_uploader/
│       ├── auth.py
│       ├── capture.py
│       ├── publish.py
│       ├── product_schema.py
│       └── endpoints.py       # 抓包后填写；初版可占位
├── utils/                     # stealth、可见 Chrome/CDP、日志
└── docs/
    ├── CLI.md
    └── capture-notes.md
```

---

## 6. CLI 契约

```text
ae-csp login   --account <name>
ae-csp check   --account <name>
ae-csp capture --account <name> [--out captures/<session>] [--har]
ae-csp publish --account <name> --from products/demo.json [--dry-run] [--headless]
```

| 命令 | 行为 |
|------|------|
| `login` | **强制有头**。打开账号对应 CSP 登录 URL → 人工完成登录 → 写 `cookies/csp_<name>.json`。不提供 `--headless`。 |
| `check` | **优先有头**（可用配置覆盖）。用 `storage_state` 打开发品页；判定见下。失败提示先 `login`。 |
| `capture` | **强制有头**。进入发品页；记录请求；JSONL + filtered；`--har` 可选。不提供 `--headless`。 |
| `publish` | 读 JSON → 校验 → `APIRequestContext` 回放最小接口链。默认可无头（`--headless`）；鉴权失败时提示有头 `check`/`login`。`--dry-run` 只打印将发请求。 |

**`check` 通过标准（MVP 写死，对齐抖音门闸）**：

1. 使用账号的 `publish_url`（或 conf 默认发品 URL）导航；
2. 最终 URL 仍在 `csp.aliexpress.com` 且路径仍与发品/后台相关（含 `product_publish` 或抓包确认的等价路径）；
3. 页面上**不出现**明显登录态文案（如「登录」「请登录」等，具体列表实现时按抓包/实页微调）；
4. 满足以上则判定 cookie 有效。  
   MVP **不做**「或轻量鉴权 API」双轨，避免实现摇摆。

退出码：

- `0` 成功
- `2` 鉴权失败
- `3` 业务校验失败
- `4` 网络/未知错误

Cookie 路径：`cookies/csp_<account>.json`（Playwright `storage_state` 格式）。

---

## 7. product.json 最小 schema

```json
{
  "title": "Demo Product Title",
  "currency": "CNY",
  "category_id": "",
  "main_images": ["D:/path/or/https://...jpg"],
  "detail_images": [],
  "description": "plain text or html snippet",
  "skus": [
    {
      "outer_sku_id": "SKU-001",
      "price": 19.9,
      "stock": 100,
      "weight_kg": 0.2,
      "image": "",
      "attrs": { "Color": "Black", "Size": "M" }
    }
  ],
  "extras": {}
}
```

| 规则 | 说明 |
|------|------|
| 代码级必填 | `title`、`main_images`（≥1）、`skus`（≥1，且含 `price`/`stock`） |
| 可空占位 | `category_id`、详情图、重量/尺寸等 — 以抓包后 `capture-notes.md` 收紧 |
| `extras` | 未建模平台字段，原样合并进回放 payload |
| 图片 | 本地路径或 URL；若接口要求先上传，则在 UploadMedia 步骤替换为平台 URL |

字段名在抓包后可微调，但须保持「必填 / extras 扩展」原则，避免 schema 锁死。

---

## 8. 发布状态机

```text
CheckSession → LoadProduct → UploadMedia → CreateOrEdit → FillSku → Submit → Confirm → RefreshCookie
```

| 阶段 | 行为 |
|------|------|
| CheckSession | 与 `check` 同判定；失败即退出码 `2` |
| LoadProduct | 校验 schema |
| UploadMedia | 若抓到上传/STS 类接口则先传图；否则使用已有 URL |
| CreateOrEdit | 建品或编辑草稿，拿到 `productId`/`draftId`（名称以抓包为准） |
| FillSku | 写入 SKU/价库/规格；可与 Create 合并为同一步（可插拔） |
| Submit | 提交审核或上架 |
| Confirm | 以响应 JSON 为准；可选再 GET 详情确认 |
| RefreshCookie | 在**同一** Playwright `BrowserContext` 内回放结束后执行 `storage_state` 写回 cookie 文件（MVP 指定路径，见 §3） |

错误分级：

- 鉴权失败 → 停止，要求重新 `login`
- 网络 / 5xx → 有限次重试
- 业务校验失败 → 记日志，不盲目重试
- `--dry-run` → 只组装请求，不 POST；不写回 cookie

---

## 9. 抓包产物约定

每次 `capture` 生成 session 目录，例如 `captures/20260714_135500/`：

| 文件 | 内容 |
|------|------|
| `meta.json` | account、起始 URL、时间、浏览器信息 |
| `requests.jsonl` | 全量流水：method、url、status、关键头、**body 可截断摘要**（控体积） |
| `filtered.json` | 发品相关候选；**request/response body 必须完整保留**（含 JSON/form），供回放与固化 `endpoints.py` |
| `critical/`（可选） | 若单文件过大，按接口拆成完整 body 文件，并由 `filtered.json` 索引 |
| `*.har` | 可选；需 `--har` |

**硬性规则**：凡进入 `filtered` / `critical` 的候选请求，**禁止**只保留摘要；截断仅允许出现在 `requests.jsonl`。

**人工必做（至少一次，硬门闸）**：有头登录后进入 `product_publish`，手点完整创建并提交一件商品；根据 `filtered.json` 将最小接口链写入 `endpoints.py` 与 `docs/capture-notes.md`。  
自动化负责录制与过滤，**不自动推断业务语义**。

过滤关键词建议（可配置）：`product`、`item`、`sku`、`upload`、`publish`、`draft`、`choice`、`ams` 等。

---

## 10. 安全与风控

| 项 | 约定 |
|----|------|
| 敏感文件 | `cookies/`、`captures/`、含真实 URL 的 `accounts/*.yaml` 默认 gitignore；禁止提交 cookie/token |
| 有头策略 | `login` / `capture` **强制有头**；`check` 默认有头；`publish` 允许 `--headless` |
| 人机协同 | 滑块/短信/2FA 不自动过，暂停等人完成 |
| 限速 | publish 请求间隔可配置（默认约 1–3s） |
| 旧 MVP | 可借鉴 CSRF/cookie 头拼装思路，无运行时依赖 |

---

## 11. 验收标准（MVP「跑通」）

1. `ae-csp login --account demo` → 生成有效 `cookies/csp_demo.json`
2. `ae-csp check --account demo` → 按 §6 写死标准通过
3. `ae-csp capture` → 录到与创建/发布相关的成功请求，且 `filtered.json` 含**完整 body**；并整理出最小接口清单写入 `docs/capture-notes.md` + `endpoints.py`
4. **依赖验收 3 / 里程碑 4 完成之后**：`ae-csp publish --from products/demo.json` → 在全托管后台可见新建商品（草稿或已提交，以接口语义为准）
5. cookie 失效时失败信息明确；重新 `login` 可恢复
6. 全程不调用妙手 / 店小秘

未完成里程碑 4（人工发品 + 固化 endpoints）时，**不得**宣称验收 4 通过。

---

## 12. 与 Commander 的衔接边界（二期备忘）

| 点 | 约定 |
|----|------|
| Web | 仍用「速卖通自动上架」入口，本 MVP 不改 |
| Agent | 未来以本仓稳定回放客户端替换 `AliExpressFactory` 妙手实现；可先 feature-flag 旁路 |
| 数据 | 二期 Excel→JSON；字段映射独立模块，不绑妙手模板语义 |
| 进程 | 独立仓保持独立 git；Agent 侧以子进程 CLI 或后续 packaging 调用，具体形态二期定 |

---

## 13. 里程碑（实现顺序）

1. 脚手架：`pyproject`、CLI 骨架、`conf`、gitignore、独立 git init
2. `login` + `check` + cookie 落盘 + 按账号 URL 覆盖
3. `capture` + 过滤（filtered **完整 body**）+ 文档模板
4. **硬门闸 / 用户配合**：手工发品一轮，固化 `endpoints.py` + `capture-notes.md`  
   → **未完成则不可进入验收 4**
5. `product_schema` + `publish`（`APIRequestContext` + dry-run + RefreshCookie）
6. 用真实 `demo.json` 验收创建/发布
7. （另开任务）Excel 适配 → Agent 接入

---

## 14. 开放问题（实现期消化，不阻塞脚手架与 login/capture）

- 全托管上传媒体是直传 OSS 还是 CSP 中转接口（抓包定）
- 「发布」是「提交审核」还是「直接上架」（以接口与后台状态为准，验收文案跟实际语义）
- CSRF / token 头具体名称与刷新方式（抓包定；回放时从 filtered 请求头复制策略写入 `endpoints.py`）

以上开放问题 **不改变** §1–§13 的架构与验收口径；仅影响 `endpoints.py` 与 upload 步骤细节。

---

## 15. 审查修订摘要（v2）

相对初版已闭合：

1. `filtered` / `critical` 必须保留完整 body；截断仅限 `requests.jsonl`
2. RefreshCookie 绑定 Playwright `BrowserContext` + `APIRequestContext` 回放路径，禁止 MVP 默认纯 httpx 却声称回写 `storage_state`
3. `login`/`capture` 强制有头；`--headless` 仅 `publish`
4. `switchId`/`channelId` 等按账号覆盖，禁止全局写死唯一 URL
5. `check` 写死发品页门闸；里程碑 4 为 publish 验收硬依赖
6. 标明独立 git 仓库，不并入 Commander 子工程树
