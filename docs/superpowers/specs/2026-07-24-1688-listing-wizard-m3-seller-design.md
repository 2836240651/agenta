# 1688 向导 M3 商家发品设计（场景 A）

**日期**：2026-07-24  
**状态**：用户确认 Approach A + Agent 开商家页记 Cookie  
**权威**：`docs/superpowers/specs/1688-listing-wizard-v2/README.md`  
**计划**：`docs/superpowers/plans/2026-07-24-1688-listing-wizard-m3.md`

## 1. 目标

Step4：发布到**自家 1688 商家后台**，全自动填表提交（Approach A）。禁止妙手/店小秘/`ProductIssue`。

## 2. 商家登录（对齐店小秘开页）

Agent 启动时已会打开 `work.1688.com` 等页。M3 Step4 再显式：

1. `listing1688_seller_open_login`：打开商家登录页 + 工作台（**保持打开，不关页**）
2. 用户在 Agent 内嵌浏览器完成商家登录
3. `confirm`：导出 Cookie → Server AES-GCM → `listing_1688_sessions.session_type=seller`
4. 商家探针通过后才允许 publish

采集 Cookie（`session_type=collect`）**不得**用于发品（C7 / TC-P-05）。

## 3. 发品

- `POST publish` → `system_tasks` + 协议 `listing1688_seller_publish`
- Agent Rod：发品入口 → 标题/图/SKU → 提交
- 选择器集中配置；失败 `PUBLISH_FAILED` 可重试

## 4. 非目标

多店切换、官方 ERP、完美对抗任意 DOM 改版（后续可热更选择器）。
