# 1688 商家资格预检修复 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 1688 向导仅在真实企业卖家发品页可用时确认商家会话，消除隐藏登录组件导致的误判。

**Architecture:** Agent 探针直接访问卖家发品路由，并把 URL、可见页面文案和嵌入开店页作为资格证据。纯函数分类承担可测试的登录、个人账号开店和有效卖家页判断；运行时仅负责采集页面状态和返回协议错误。

**Tech Stack:** Go、Rod、现有 Agent/Server 同步协议。

## Global Constraints

- 仅使用自家 1688 商家后台，不调用妙手、店小秘。
- 不输出 Cookie、Token 或账号密码；Cookie 仅经现有 Server AES-GCM 链路保存。
- 本轮不点击最终发布按钮，也不修改远程服务器。

---

### Task 1: 添加商家资格分类测试

**Files:**
- Create: `backend/internal/factory/alibaba/listing1688_seller_test.go`
- Modify: `backend/internal/factory/alibaba/listing1688_seller.go`

- [ ] **Step 1:** 写入有效卖家页、隐藏登录文案、个人账号开店页与登录页 fixture 测试。
- [ ] **Step 2:** 运行 `go test ./backend/internal/factory/alibaba -run Listing1688Seller`，确认当前实现对隐藏登录文案错误失败。
- [ ] **Step 3:** 提取并实现纯函数资格分类，返回可区分的会话无效与账号无资格错误。
- [ ] **Step 4:** 重跑同一测试，确认全部通过。

### Task 2: 接入真实发品路由探针

**Files:**
- Modify: `backend/internal/factory/alibaba/listing1688_seller.go`
- Modify: `backend/internal/factory/alibaba/listing1688_seller_selectors.go`

- [ ] **Step 1:** 将探针目标改为卖家 `orderPost` 路由，并检查最终 URL/iframe 的 `cxt.1688.com` 开店引导。
- [ ] **Step 2:** 仅在页面具有卖家发品导航或类目入口时返回 `seller_ok`。
- [ ] **Step 3:** 在本机 Agent 重启后调用 Workflow `#12` 的 `seller/confirm`，确认 Server 仅报告加密保存结果。

### Task 3: 记录与验证

**Files:**
- Modify: `开发文档.md`

- [ ] **Step 1:** 运行 Agent 包定向测试与 `go vet`（若仓库现有配置允许）。
- [ ] **Step 2:** 在开发文档顶部记录根因、修复和验证结果，不记录 Cookie。
- [ ] **Step 3:** 只暂存本轮文件，检查 staged diff 后提交并推送；若本机 Git 身份或 TLS 阻断，原样报告。
