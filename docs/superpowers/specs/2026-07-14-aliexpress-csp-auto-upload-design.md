# 速卖通全托管（CSP）独立上架 MVP — 设计说明

**日期**：2026-07-14  
**状态**：待用户审阅  
**仓库（建议）**：`D:\dev\workspace\aliexpress-csp-auto-upload`  
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
| 发品回跳 | `https://csp.aliexpress.com/ait/cn_ams/item_choice/product_publish?...`（以配置中完整 URL 为准） |
| 主域名 | `csp.aliexpress.com` 及相关网关（抓包确认） |

默认登录 URL 写入 `conf.py`（即用户提供的完整链接，含 `return_url`）。

---

## 2. 非目标

- 调用 `erp.91miaoshou.com` / 店小秘等第三方 ERP
- 经典海外自营卖家中心（`seller.aliexpress.com`）完整发品（旧 Scheme B 方向）
- 自动破解滑块 / 短信 / 二次验证
- 多账号并发展品、生产级调度队列
- 在本仓直接改 Commander Web/Server 路由

---

## 3. 方案选型

| 方案 | 结论 |
|------|------|
| SAU 风格新仓 + 抓包回放（Python / Patchright） | **采用** |
| 继续扩展 `aliexpress-seller-api-mvp`（TS） | 否；旧仓仅参考 |
| UI RPA 填表为主 | 否；仅登录/兜底 |

技术栈倾向：

- **Python 3.11+**
- **Patchright**（与 `social-auto-upload` 对齐；Playwright API 兼容）
- CLI 安装后命令名：**`ae-csp`**
- HTTP 回放：基于浏览器 `storage_state` 导出的 Cookie + 抓包得到的 CSRF/业务头

---

## 4. 架构

```text
CLI (ae-csp)
  ├─ login / check     → Auth（有头浏览器 + storage_state）
  ├─ capture           → 录制 CSP 相关请求 → captures/
  └─ publish           → 按 endpoints 清单回放 API
         ↑
cookies/csp_<account>.json
captures/<session>/
products/*.json
```

| 层 | 职责 |
|----|------|
| CLI | 子命令、参数、退出码、日志 |
| Auth | `login` / `cookie_auth` / `setup`；UI 仅登录与失效兜底 |
| Capture | 打开 `product_publish`，过滤并落盘请求 |
| Publisher | schema 校验 → 上传媒体（若需要）→ 创建/编辑 → SKU → 提交 → 刷新 cookie |
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
├── cookies/                    # gitignore
├── captures/                   # gitignore
├── products/
│   └── demo.json
├── uploader/
│   └── csp_uploader/
│       ├── auth.py
│       ├── capture.py
│       ├── publish.py
│       ├── product_schema.py
│       └── endpoints.py        # 抓包后填写；初版可占位
├── utils/                      # stealth、可见 Chrome/CDP、日志
└── docs/
    ├── CLI.md
    └── capture-notes.md
```

---

## 6. CLI 契约

```text
ae-csp login   --account <name> [--headless]
ae-csp check   --account <name>
ae-csp capture --account <name> [--out captures/<session>] [--har]
ae-csp publish --account <name> --from products/demo.json [--dry-run]
```

| 命令 | 行为 |
|------|------|
| `login` | 打开 CSP 登录 URL → 人工完成登录 → 写 `cookies/csp_<name>.json` |
| `check` | 用 storage_state 校验会话（发品页或轻量鉴权 API）；失败提示先 login |
| `capture` | 有头进入发品页；记录请求；默认 JSONL + 过滤摘要；`--har` 可选完整 HAR |
| `publish` | 读 JSON → 校验 → 回放最小接口链；`--dry-run` 只打印将发请求 |

退出码约定（实现时遵守）：

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
| CheckSession | 与 `check` 同逻辑；失败即退出 |
| LoadProduct | 校验 schema |
| UploadMedia | 若抓到上传/STS 类接口则先传图；否则使用已有 URL |
| CreateOrEdit | 建品或编辑草稿，拿到 `productId`/`draftId`（名称以抓包为准） |
| FillSku | 写入 SKU/价库/规格；可与 Create 合并为同一步（以真实接口为准，代码做成可插拔步骤） |
| Submit | 提交审核或上架 |
| Confirm | 以响应 JSON 为准；可选再 GET 详情确认 |
| RefreshCookie | 成功后回写 `storage_state` |

错误分级：

- 鉴权失败 → 停止，要求重新 `login`
- 网络 / 5xx → 有限次重试
- 业务校验失败 → 记日志，不盲目重试
- `--dry-run` → 只组装请求，不 POST

---

## 9. 抓包产物约定

每次 `capture` 生成 session 目录，例如 `captures/20260714_135500/`：

| 文件 | 内容 |
|------|------|
| `meta.json` | account、起始 URL、时间、浏览器信息 |
| `requests.jsonl` | method、url、status、关键头、body 摘要（大体截断） |
| `filtered.json` | 发品相关候选请求（host/path 关键词可配置） |
| `*.har` | 可选；需 `--har` |

**人工必做（至少一次）**：有头登录后进入 `product_publish`，手点完整创建并提交一件商品；根据 `filtered.json` 将最小接口链写入 `endpoints.py` 与 `docs/capture-notes.md`。  
自动化负责录制与过滤，**不自动推断业务语义**。

过滤关键词建议（可配置）：`product`、`item`、`sku`、`upload`、`publish`、`draft`、`choice`、`ams` 等。

---

## 10. 安全与风控

| 项 | 约定 |
|----|------|
| 敏感文件 | `cookies/`、`captures/` 默认 gitignore；禁止提交 cookie/token |
| 有头策略 | `login` / `capture` 默认有头；纯回放可 headless；鉴权异常时建议有头 `check`/`login` |
| 人机协同 | 滑块/短信/2FA 不自动过，暂停等人完成 |
| 限速 | publish 请求间隔可配置（默认约 1–3s） |
| 旧 MVP | 可借鉴 CSRF/cookie 头拼装思路，无运行时依赖 |

---

## 11. 验收标准（MVP「跑通」）

1. `ae-csp login --account demo` → 生成有效 `cookies/csp_demo.json`
2. `ae-csp check --account demo` → 通过
3. `ae-csp capture` → 录到与创建/发布相关的成功请求，并能整理出最小接口清单写入 docs
4. `ae-csp publish --from products/demo.json` → 在全托管后台可见新建商品（草稿或已提交，以接口语义为准）
5. cookie 失效时失败信息明确；重新 `login` 可恢复
6. 全程不调用妙手 / 店小秘

---

## 12. 与 Commander 的衔接边界（二期备忘）

| 点 | 约定 |
|----|------|
| Web | 仍用「速卖通自动上架」入口，本 MVP 不改 |
| Agent | 未来以本仓稳定回放客户端替换 `AliExpressFactory` 妙手实现；可先 feature-flag 旁路 |
| 数据 | 二期 Excel→JSON；字段映射独立模块，不绑妙手模板语义 |
| 进程 | 独立 MVP 可用子进程/HTTP 包装供 Agent 调用，具体形态二期定 |

---

## 13. 里程碑（实现顺序提示）

1. 脚手架：`pyproject`、CLI 骨架、`conf`、gitignore
2. `login` + `check` + cookie 落盘
3. `capture` + 过滤 + 文档模板
4. **用户配合**：手工发品一轮，固化 `endpoints.py`
5. `product_schema` + `publish`（含 dry-run）
6. 用真实 `demo.json` 验收创建/发布
7. （另开任务）Excel 适配 → Agent 接入

---

## 14. 开放问题（实现期消化，不阻塞开干）

- 全托管上传媒体是直传 OSS 还是 CSP 中转接口（抓包定）
- 「发布」是「提交审核」还是「直接上架」（以接口与后台状态为准，验收文案跟实际语义）
- CSRF / token 头具体名称与刷新方式（抓包定）

以上开放问题 **不改变** §1–§11 的架构与验收口径；仅影响 `endpoints.py` 与 upload 步骤细节。
