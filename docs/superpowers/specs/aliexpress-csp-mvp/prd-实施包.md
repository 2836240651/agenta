# PRD 实施包：速卖通全托管（CSP）独立上架 MVP

**日期**：2026-07-14  
**对齐设计**：[`../2026-07-14-aliexpress-csp-auto-upload-design.md`](../2026-07-14-aliexpress-csp-auto-upload-design.md)  
**对齐需求**：[`./requirements.md`](./requirements.md)  
**目标仓路径**：`D:\dev\workspace\aliexpress-csp-auto-upload`（独立 git）

---

## 1. 文档目标

指导独立 MVP **落地与验收**，明确：业务目标、功能需求、里程碑、交付物、门闸与二期边界。  
实现细节见技术分析；用例见测试文档；架构决策以设计 spec v2 为准。

---

## 2. 业务目标

### 2.1 核心目标

1. 有头完成 CSP 全托管登录，持久化 `storage_state`
2. 稳定校验会话（发品页门闸）
3. 抓包录得发品相关接口，且 **filtered 含完整 body**
4. 人工固化最小接口链后，用 `product.json` **API 回放**创建/提交商品
5. 全程不经过妙手 / 店小秘

### 2.2 非目标

- 批量调度与多店并发
- UI 自动填表发品
- Excel / Commander 接入
- 验证码自动破解
- 经典 Seller Center 海外自营发品

---

## 3. 目标用户

| 用户 | 用法 |
|------|------|
| 开发 | 建仓、实现 CLI、固化 endpoints、排障 |
| 产品/项目 | 看里程碑与验收，判断能否进二期接系统 |
| 测试 | 执行测试用例，签署 MVP 是否跑通 |
| 运营（偶发） | 在工程指导下 login / 配合手点发品抓包 |

---

## 4. 用户场景

### 场景 A：建立登录态

操作者执行 `ae-csp login --account <name>`，在有头浏览器打开账号专属 CSP 登录 URL，人工完成登录/验证。系统写入 `cookies/csp_<name>.json`。

### 场景 B：校验登录态

执行 `ae-csp check --account <name>`。按发品页 URL + 「非登录页」文案判定。失败退出码 2，提示重新 login。

### 场景 C：手点发品抓包（硬门闸）

执行 `ae-csp capture --account <name>`，在发品页**完整手建一品并提交**。系统写出 session；工程人员整理 `filtered.json` → `endpoints.py` + `docs/capture-notes.md`。  
**未完成本场景，不得进入场景 E 的正式验收。**

### 场景 D：Dry-run 组装请求

在 endpoints 初稿就绪后，执行 `publish --dry-run`，确认字段映射与将发 URL/体，不改后台。

### 场景 E：真实回放发品

执行 `ae-csp publish --account <name> --from products/demo.json`（可选 `--headless`）。经 CheckSession → … → Submit → RefreshCookie；后台可见新商品。

---

## 5. 功能需求

### FR-1 登录态获取

- 命令：`ae-csp login --account <name>`
- **强制有头**；禁止 `--headless`
- 登录 URL：账号 yaml > conf 默认
- 成功写 `cookies/csp_<name>.json`；可覆盖旧态

**验收**：文件存在且 `check` 可通过。

### FR-2 会话校验

- 命令：`ae-csp check --account <name>`
- 门闸：设计 §6 四条写死标准
- 失败退出码 `2`，提示 login

**验收**：有效态通过；伪造/过期态失败且信息明确。

### FR-3 请求捕获

- 命令：`ae-csp capture --account <name> [--out ...] [--har]`
- **强制有头**
- **结束方式**：终端提示后操作者按 Enter（或文档约定的退出键）结束；异常 Ctrl+C 亦须 flush 已录数据并写 `meta.json` 结束原因
- 产出：`meta.json`、`requests.jsonl`（可截断）、`filtered.json`（**完整 body**）、可选 `critical/`、`--har`

**验收**：手点发品后 filtered 含创建/上传/提交相关成功请求及完整 body；异常退出不丢已写入部分。

### FR-4 接口固化（人工+文档）

- 人工根据 filtered 填写 `endpoints.py` 与 `capture-notes.md`
- 程序**不**自动推断业务语义

**验收**：文档列出最小调用顺序、关键头、成功判据；代码可 dry-run。

### FR-5 JSON 发布回放

- 命令：`ae-csp publish --account <name> --from <json> [--dry-run] [--headless]`
- 客户端：Playwright `APIRequestContext`（BrowserContext + storage_state）
- 状态机：CheckSession → LoadProduct → UploadMedia → CreateOrEdit → FillSku → Submit → Confirm → RefreshCookie
- schema 最少字段见设计 §7；`extras` 可扩展
- 退出码：0 / 2 / 3 / 4
- **RefreshCookie**：仅整体成功（退出码 0）时回写 `storage_state`；失败或 `--dry-run` **不**回写

**验收**：依赖 FR-4 完成后，后台可见新商品；`--dry-run` 不写后台、不回写 cookie。

### FR-5.1 单接口诊断（推荐，非 MVP 签署必项）

- 命令：`ae-csp api --account <name> --method <M> --url <URL> [--body-file ...]`
- 用途：在 M4 前后用登录态探活/核对 CSRF，避免「只有整链 publish 才能证明 H2」
- **不计入** MVP 完成签署的必测项；但 M5 排障时强烈建议具备

**验收（可选）**：对一需登录的 GET/POST 返回可解释的 HTTP/业务结果与请求摘要。

### FR-6 按账号店铺参数

- 支持 `accounts/<account>.yaml` 覆盖 `login_url` / `publish_url`
- 禁止全局唯一写死 `switchId`/`channelId`

**验收**：不同账号配置互不覆盖；错误账号 URL 可被 check/login 暴露。

### FR-7 安全落盘

- `.gitignore`：cookies、captures、含真实 URL 的 accounts

**验收**：`git status` 不出现敏感态文件。

---

## 6. 非功能需求

| ID | 要求 |
|----|------|
| NFR-1 | 敏感文件不进 git |
| NFR-2 | 风控人工介入；程序可等待 |
| NFR-3 | publish 限速可配（默认 1–3s） |
| NFR-4 | 日志可定位鉴权 / 业务 / 网络失败 |
| NFR-5 | Windows 本机可跑；Python 3.11+；Patchright |
| NFR-6 | 独立 git 仓，不并入 Commander mono 树 |

---

## 7. 交付范围

### 7.1 本次交付（MVP）

| 交付物 | 说明 |
|--------|------|
| 独立仓库 | `aliexpress-csp-auto-upload` |
| 文档 | 本 PRD、需求、技术分析、测试用例（先落在 workspace `docs/superpowers/specs/aliexpress-csp-mvp/`，**建仓后复制进**目标仓 `docs/`）；设计 spec；CLI.md |
| CLI | `login` / `check` / `capture` / `publish` |
| 示例 | `products/demo.json`、`accounts/*.example.yaml` |
| 门闸产物 | 至少一份 capture session + 固化 endpoints（可脱敏后进仓） |

### 7.2 下一阶段（非本次）

- Excel→JSON
- Agent `AliExpressFactory` 旁路妙手
- 多账号调度、类目属性全量适配

---

## 8. 里程碑

| 里程碑 | 内容 | 出口条件 |
|--------|------|----------|
| M1 | 脚手架：pyproject、CLI 骨架、conf、gitignore、git init | `ae-csp --help` 可用 |
| M2 | login + check + 账号 URL 覆盖 | 验收标准 1–2 |
| M3 | capture + filtered 完整 body + 文档模板 | 能产出 session |
| **M4 硬门闸** | **用户手点发品 + 固化 endpoints** | **capture-notes + endpoints 可 dry-run** |
| M5 | product_schema + publish（含 dry-run、RefreshCookie） | dry-run 通过 |
| M6 | 真实 publish 验收 | 验收标准 4–6 |

**规则**：未完成 M4，不得宣称 M6 / 验收 4 通过。

---

## 9. 依赖与协作

| 依赖 | 说明 |
|------|------|
| 有效全托管卖家账号 | 可登录 CSP |
| 本机有头浏览器环境 | Patchright Chromium |
| 用户时间（M4） | 完整手点发品一轮 |
| 设计开放问题 | 上传路径 / 发布语义 / CSRF 名 — 在 M4 抓包后关闭 |

---

## 10. 风险与对策（产品级）

| 风险 | 对策 |
|------|------|
| 回放被风控 | 保留有头 check/login；限速；不自动破解 |
| 接口改版 | 保留 capture；测试回归重抓 |
| 范围蔓延到 Commander | PRD 非目标锁定；二期另开任务 |
| 「假完成」 | M4 硬门闸写进验收 |

---

## 11. MVP 完成签署标准

同时满足：

1. M1–M6 出口条件均达成  
2. 测试用例冒烟集全部通过（见测试文档）  
3. 文档可交接：新开发者按 CLI.md 能复现 login→check→capture→publish  
4. 无妙手/店小秘依赖  

签署后，方可启动「Excel 适配 / Agent 接入」二期立项。
