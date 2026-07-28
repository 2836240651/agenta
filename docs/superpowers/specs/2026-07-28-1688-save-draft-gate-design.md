# 1688 保存草稿门禁与安全填表（方案 A）

**日期**：2026-07-28  
**状态**：已实现（方案 A）  
**权威**：`docs/superpowers/specs/1688-listing-wizard-v2/README.md`  
**前置**：`2026-07-25-1688-listing-wizard-live-publish-eligibility-design.md`  
**范围**：场景 A · 自家 1688 商家后台 · 仅已验证 field_bindings

---

## 1. 决策

采用方案 A：打通「表单校验 → 保存草稿 → 真实发布」门禁闭环。

- Server 暴露 `POST .../form/save-draft`，同步调用 Agent `listing1688_seller_save_draft`，成功后回写 `draft_saved=true`。
- Agent 仅用已验证 bindings 填写 `title` + `images`，再点「保存草稿」并确认成功提示。
- 真实 `publish` 必须 `validation_status=ready` 且 `draft_saved=true`；Agent 必须收到 `draft_verified=true`，禁止旧「盲填标题即提交」路径。
- **本轮不做**：品牌/属性/物流等无稳定 binding 字段的自动填写；远程部署；企业店 live 发品验收。

---

## 2. 现状缺口（代码事实）

| 层 | 事实 |
|----|------|
| Agent | `Listing1688SellerSaveDraft` 已实现，但 dispatcher **未注册**该协议 |
| Agent | `Listing1688SellerPublish` 仍开工作台盲填标题/图后直接提交，忽略 `draft_verified` / `publish_form` |
| Server | `protocol.go` **无** `listing1688_seller_save_draft` 常量 |
| Server | 无 HTTP 入口调用 save-draft；`publish` 已检查 `ready + draft_saved`，但草稿永远无法被置为 true（除非手改库） |
| Bindings | 仅 `title` / `images` / `save_draft` / `submit` |

---

## 3. 主路径

```
运营在 Agent Chrome 打开目标类目 offer-new 页
→ category/inspect（可选，同步 required_fields）
→ form/validate → 草案 ready（draft_saved 强制 false）
→ form/save-draft
     Server: SyncSend listing1688_seller_save_draft
             { confirm_save_draft: true, category_id, publish_form: {title, images, ...} }
     Agent:  定位已打开类目页 → 按 bindings 填 title/images → 点保存草稿
             → 无校验错误 + 出现草稿成功提示 → 返回 draft_saved=true
     Server: 回写草案 draft_saved=true（失败保持 false，返回可诊断错误）
→ publish
     Server: 仅 ready + draft_saved 才入队；payload 带 draft_verified=true + publish_form
     Agent:  无 draft_verified → 立即失败
             复用已打开 offer-new 页 → 点最终发布
             → 非空 offer_id + platform_status published|auditing + success_hint
```

状态机对 Workflow 不变：`img_ready → publishing → published | failed`。  
草案侧新增可观察状态：`draft_saved` 布尔（表字段已有）。

---

## 4. Server 设计

### 4.1 协议与路由

- 常量：`ProtocolListing1688SellerSaveDraft = "listing1688_seller_save_draft"`
- 路由：`POST /api/v1/listing1688/workflows/:id/form/save-draft`
- 鉴权 / 归属：与现有 `form/validate`、`category/inspect` 一致

### 4.2 `form/save-draft` 前置

全部满足才调 Agent：

1. Workflow 归属当前用户，且 `img_ready`（与 publish 一致）
2. Agent 在线
3. 商家会话有效（`listing1688RequireSellerSession`）
4. 存在该 workflow 草案，且 `validation_status == ready`
5. 草案 `title` 非空；选定图片 ≥ 1

不满足则 Forbidden，错误码沿用 `Listing1688ErrWizardPublishFail` 或现有会话/Agent 离线码，message 可读。

### 4.3 SyncSend 载荷

```json
{
  "confirm_save_draft": true,
  "category_id": "<draft.category_id>",
  "publish_form": { "...草案 form_data..." },
  "title": "<from form>",
  "images": ["...selected result urls..."]
}
```

超时建议：60s（填图可能慢于 inspect 的 30s）。

### 4.4 成功 / 失败回写

- Agent `ok` 且 `draft_saved=true` → 更新草案 `draft_saved=true`；可选写入 `offer_id` / `platform_status` 若 Agent 返回（本轮可不强制）
- 否则保持或置 `draft_saved=false`，返回 Agent 错误原文（规范化）
- **`form/validate` Upsert 时必须显式 `draft_saved=false`**，避免重校验后仍带着旧草稿门禁

### 4.5 `publish`（收紧，不改入队结构）

保持现有：

- `listing1688PublishDraftCanSubmit(ready, draft_saved)` 门禁
- payload 含 `publish_form`、`draft_verified: true`

补充：若草案 `draft_saved` 为 false，错误文案继续提示「请先完成类目表单校验并保存草稿」。

---

## 5. Agent 设计

### 5.1 Dispatcher

注册 `listing1688_seller_save_draft` → `Listing1688SellerSaveDraft`（当前缺失，必须补）。

### 5.2 SaveDraft 行为（扩展现有函数）

前置：`confirm_save_draft == true`，否则 `DRAFT_SAVE_CONFIRM_REQUIRED`。

1. 在已打开的 `offer-new` 类目页上操作（`listing1688SellerFindOpenCategoryForm`）；找不到 → `CATEGORY_FORM_NOT_OPEN`
2. **仅**用 `listing1688SellerVerifiedBindings()`：
   - `title`：写入 `publish_form.title` 或顶层 `title`
   - `images`：上传 `publish_form.images` 或顶层 `images`（≥1）
3. 点击「保存草稿」
4. 读取页面：若有必填/不能为空类错误 → `DRAFT_VALIDATION_FAILED`
5. 若无「草稿保存成功 / 保存草稿成功 / 已保存到草稿箱」→ `DRAFT_SAVE_UNCONFIRMED`
6. 成功返回 `{ draft_saved: true, validation_errors: [], page_url }`

禁止：按输入顺序盲填未绑定字段；禁止点击最终发布。

### 5.3 Publish 行为（改造）

1. 解析 payload：若 `draft_verified != true` → 立即 `DRAFT_NOT_VERIFIED`，**不点击提交**
2. 定位已打开的 `offer-new` 类目页（同 SaveDraft）；找不到 → `CATEGORY_FORM_NOT_OPEN`（不再 `MustPage` 开工作台走旧盲填路径）
3. 点击最终发布（已验证 binding `submit` / `#submitFormButton`）
4. 成功判定不变（与 live-publish-eligibility 设计一致）：
   - `ok` + `submitted` + `title_filled` + `images_set>=1` + 非空 `offer_id` + `platform_status` ∈ {published, auditing} + `success_hint`

说明：本轮 Publish **不再重复填表**（假设 save-draft 已写入标题与图）；若页面上标题/图缺失，应失败而不是静默补填后强提（避免绕过草稿门禁）。可选：提交前只读检查标题非空与至少一张图，失败码 `DRAFT_CONTENT_MISSING`。

---

## 6. 错误码（约定）

| 码 | 含义 |
|----|------|
| `DRAFT_SAVE_CONFIRM_REQUIRED` | 未显式 confirm |
| `CATEGORY_FORM_NOT_OPEN` | 未打开目标类目 offer-new |
| `DRAFT_SAVE_UNAVAILABLE` | 无保存草稿按钮 |
| `DRAFT_VALIDATION_FAILED` | 保存后仍有必填错误 |
| `DRAFT_SAVE_UNCONFIRMED` | 无草稿成功提示 |
| `DRAFT_NOT_VERIFIED` | Publish 未带 draft_verified |
| `DRAFT_CONTENT_MISSING` | 提交前页面缺标题/图（可选） |

Server 对外可包装为现有 Wizard 错误格式。

---

## 7. 测试

### Server

- `form/validate` 后 `draft_saved` 为 false
- `save-draft`：无 ready 草案 / Agent 离线 / 会话无效 → Forbidden
- 成功路径 mock SyncAgent → `draft_saved=true`
- `publish`：`ready` 但 `draft_saved=false` → 拒绝

### Agent

- dispatcher 对 `listing1688_seller_save_draft` 可路由（编译期接口含该方法）
- SaveDraft：`confirm_save_draft=false` 失败
- 草稿成功文案识别单测（已有则保留）
- Publish：`draft_verified` 缺失/false → 失败且不调用 submit
- Publish：不再依赖「新开工作台填标题」作为成功前置

### 非目标本轮

- 真实企业店浏览器 E2E
- Web 前端按钮（可后续加「保存草稿」；本轮 API 可用 Postman/curl）

---

## 8. 文件清单（预期）

**Server**

- `internal/constants/protocol.go`
- `internal/router/register.go`
- `internal/services/listing1688_publish_profile_handler.go`（save-draft + validate 清 draft_saved）
- `internal/services/listing1688_publish.go`（如需）
- `internal/repos/listing_1688_publish_profile.go`（Upsert 显式 draft_saved）
- 对应 `*_test.go`

**Agent**

- `backend/internal/dispatcher/default.go`
- `backend/internal/factory/alibaba/listing1688_seller.go`
- `backend/internal/factory/alibaba/listing1688_seller_form_test.go`
- `backend/internal/constants/protocol.go`（已有常量则不动）

**文档**

- 本文件
- 实现后更新 `交接文档.md` 顶部停点 + `开发文档.md`

---

## 9. 非目标

- 不自动填写无 binding 的类目属性
- 不绕过企业店资格 / 验证码 / 平台风控
- 不部署远程服务器
- 不修改 CSP / Temu / v1 妙手路径

---

## 10. 验收标准

1. 无 `form/save-draft` 成功记录时，`publish` 恒 Forbidden。  
2. `save-draft` 成功后草案 `draft_saved=true`，`publish` 可入队且 Agent payload 含 `draft_verified=true`。  
3. Agent 在 `draft_verified=false` 时绝不点击最终发布。  
4. 单元测试覆盖上述门禁；静态编译通过。  
5. 重新 `form/validate` 后必须重新 `save-draft` 才能 publish。
