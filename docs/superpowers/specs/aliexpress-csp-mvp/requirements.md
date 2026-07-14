# 需求分析：速卖通全托管（CSP）独立上架 MVP

**日期**：2026-07-14  
**对齐设计**：[`../2026-07-14-aliexpress-csp-auto-upload-design.md`](../2026-07-14-aliexpress-csp-auto-upload-design.md)  
**目标仓**：`aliexpress-csp-auto-upload`（独立 git，不并入 Commander 子工程）

---

## 1. 背景与问题

当前 Commander 速卖通批量上架依赖 **妙手 ERP**（导入 Excel → OSS → `processImportCopyV2`），存在：

- 第三方平台绑定与稳定性不可控
- 字段映射受 ERP 模板限制
- 排查困难（问题可能出在妙手侧而非本系统）

业务要求改为：**不依赖妙手**，走官方 **全托管 CSP**（`csp.aliexpress.com`）链路。

对照 `social-auto-upload` 抖音上传经验，本需求采用：

- 浏览器只做 **登录 / 鉴权校验 / 人工过风控**
- 主路径为 **页面抓包 → Cookie/CSRF API 回放** 完成创建/发布

---

## 2. 必须验证的假设

| ID | 假设 | MVP 内如何验证 |
|----|------|----------------|
| H1 | CSP 网页登录态可持久化为 Playwright `storage_state` | `login` → 文件落盘 → `check` |
| H2 | 登录态可支撑后台业务 API 调用 | 优先用可选 `ae-csp api` 探活；**签署级**证明仍以 `publish` 经 `APIRequestContext` 成功建品为准 |
| H3 | 发品关键接口可通过人工操作抓包并固化 | `capture` → `filtered.json` 完整 body → `endpoints.py` |
| H4 | CSRF/业务头可从抓包结果复现 | 回放请求头策略写入 endpoints；失败可定位 |

若 H2/H3 任一长期无法成立，则本技术路线需重新评估（降级到 UI 填表或放弃直连接口），不在本需求内拍板替代方案。

---

## 3. 范围

### 3.1 必做（MVP）

1. **登录与态持久化**：有头浏览器；按账号 `cookies/csp_<account>.json`
2. **会话校验 `check`**：写死发品页门闸（见设计 §6）
3. **抓包 `capture`**：`requests.jsonl`（可截断）+ `filtered.json`（**完整 body**）
4. **人工硬门闸**：完整手点发品一轮 → 固化 `endpoints.py` + `capture-notes.md`
5. **发布 `publish`**：读 `product.json`，`APIRequestContext` 回放最小接口链；支持 `--dry-run`
6. **按账号 URL 覆盖**：`switchId` / `channelId` 等不得假定全局唯一

### 3.2 明确不做（MVP）

- Excel → JSON、接 Commander Web/Server/Agent
- UI 全自动填表发品
- 妙手 / 店小秘
- 自动破解滑块/短信/2FA
- 多账号并发展品、任务队列
- 经典海外卖家中心（`seller.aliexpress.com`）发品

### 3.3 二期（仅需求预留）

- Excel 适配对齐 Commander 模板
- 旁路/替换 Agent `AliExpressFactory` 妙手路径；Web 入口不变

---

## 4. 角色与用户故事

| 角色 | 诉求 |
|------|------|
| 操作者（开发/运营） | 本机 CLI 登录、抓包、用 JSON 发一品 |
| 工程侧 | 固化 endpoints，证明可扩展批量与接系统 |
| 测试 | 按用例验收登录、抓包、回放、失效与安全 |

**用户故事**

1. 作为操作者，我执行 `ae-csp login --account demo`，在有头浏览器完成 CSP 登录后得到可复用 cookie 文件。
2. 作为操作者，我执行 `check`，在 cookie 有效时得到明确通过，失效时得到「请先 login」类提示（退出码 2）。
3. 作为工程侧，我执行 `capture` 并手点完整发品，得到带完整 body 的 `filtered.json`，可据此填写 `endpoints.py`。
4. 作为工程侧，在 endpoints 固化后，我用 `publish --from products/demo.json` 在全托管后台看到新建商品（草稿或已提交）。
5. 作为工程侧，我可以用 `--dry-run` 只查看将发出的请求而不改后台数据。

---

## 5. 输入与产出

### 5.1 输入

- 账号配置：`conf` 默认 URL + 可选 `accounts/<account>.yaml`（`login_url` / `publish_url`）
- 商品：`product.json`（最少：`title`、`main_images≥1`、`skus≥1` 含 `price`/`stock`）

### 5.2 产出

- `cookies/csp_<account>.json`
- `captures/<session>/`（meta、jsonl、filtered、可选 har/critical）
- `docs/capture-notes.md`、`uploader/csp_uploader/endpoints.py`
- 后台可见商品（验收定义见 §6）

---

## 6. 验收标准（与设计 §11 一致）

1. `login` → 有效 `cookies/csp_demo.json`
2. `check` → 按发品页标准通过
3. `capture` → 相关成功请求 + **完整 body** + 接口清单文档化
4. **仅当 3 与里程碑「人工固化 endpoints」完成后**：`publish` → 后台可见新商品
5. cookie 失效提示明确，重新 login 可恢复
6. 全程零妙手 / 店小秘调用

未完成人工固化门闸时，**不得**宣称 publish 验收通过。

---

## 7. 约束与非功能需求

| 类别 | 要求 |
|------|------|
| 安全 | cookies/captures/真实 accounts yaml 不进 git |
| 人机 | 风控只人工；程序等待不自动破解 |
| 可观测 | 退出码 0/2/3/4；关键步骤日志可定位 |
| 限速 | publish 间隔可配（默认约 1–3s） |
| 技术栈 | Python 3.11+、Patchright、CLI 名 `ae-csp` |
| 回放通道 | MVP 指定 Playwright `APIRequestContext`（非整仓默认 httpx） |

---

## 8. 院线与入口（需求级）

- 院线：**全托管 CSP**
- 登录：`login.aliexpress.com`（`bizSegment=CSP`）→ 回跳 `csp.aliexpress.com`…`product_publish`
- `return_url` 中店铺相关参数按账号覆盖

---

## 9. 风险（需求视角）

| 风险 | 影响 | 对策 |
|------|------|------|
| 接口需 runtime 签名 | 纯 cookie 回放失败 | 已指定 BrowserContext 内 `APIRequestContext`；仍失败则扩大抓包面 |
| 发品接口频繁改版 | endpoints 失效 | 保留 capture；回归用例重抓 |
| 店铺参数写死导致登错店 | 发到错误店铺 | 账号级 URL 覆盖（硬性） |
| 人工门闸未完成就宣验收 | 虚假完成 | publish 验收显式依赖门闸 |

---

## 10. 成功定义（一句话）

在**不依赖妙手**的前提下，用独立 CLI 证明：CSP 登录态可复用，且经抓包固化后的 API 回放能在全托管后台创建可见商品。
