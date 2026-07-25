# 1688 竞品向导上架 Wizard v2 — 设计说明

**日期**：2026-07-23（全文重写）  
**文档包**：[`./1688-listing-wizard-v2/`](./1688-listing-wizard-v2/README.md)  
**主战场**：`D:\dev\workspace`  
**采集源**：`D:\reverselab\1688-`

---

## 1. 目标

在 Commander 交付 4 步向导：

```
采集登录 → 竞品 CDP 采集 → 交互图优(hyhacct) → 自家 1688 商家后台发品
```

入口：`/1688-auto-upload/wizard`（推荐）；保留 v1 `/listing` 一键快捷。

---

## 2. 为何不用妙手发品

| 能力 | 妙手 | 本向导 |
|------|------|--------|
| 采 1688 货源 → 铺淘宝/抖店/跨境等 | ✅ | 不做（场景 B，本期不做） |
| 发到**自家 1688 商家店** | ❌ | **要做（场景 A）** |

发品通道：**Agent 浏览器自动化 1688 商家发品页**（MVP）；备选再评估官方服务市场商家版 ERP。

竞品采集：**不依赖 ERP**，移植 `D:\reverselab\1688-`。

---

## 3. 方案

| 方案 | 结论 |
|------|------|
| Commander 四步向导 + 移植 1688- 解析 + 商家后台发品 | **采用** |
| 妙手 / 店小秘发到 1688 商家店 | **否** |
| 只扩 v1 一键链 | 否（保留为快捷，不做逐张图优） |
| 独立新仓 | 否 |

---

## 4. 流程与职责

```mermaid
flowchart LR
  S1[Step1 采集登录] --> S2[Step2 采集预览]
  S2 --> S3[Step3 图优]
  S3 --> S4[Step4 商家发品]
```

| Step | 做什么 | 谁做 |
|------|--------|------|
| 1 | 开浏览器登录 1688，Cookie 加密+探针 | Agent + Server |
| 2 | 链接 → CDP HTML → 解析 window.context → 预览 | Agent（解析自 1688-） |
| 3 | Slot prompt / 重生成 / 选定 / 沿用原图 | Web + Server（`AiEditImage` → hyhacct） |
| 4 | 商家后台登录校验 → 自动填表发品 | Agent |

状态机见 [README](./1688-listing-wizard-v2/README.md)。

---

## 5. 模块边界

| 模块 | 策略 |
|------|------|
| Web `/wizard` | 新建四步 UI |
| Server Workflow | 新建编排；不塞进 `RunProductIssue` |
| Agent 采集 | 扩展协议 + 移植 1688- sidecar |
| Agent 发品 | **新建**商家发品自动化；**不用** `AlibabaFactory.ProductIssue` |
| hyhacct | Server 经 `AiEditImage` 调用；向导不直调 `produce_images` |
| v1 一键 | 保留隔离 |

---

## 6. 非目标（MVP）

- 自动过码、批量链接、Prompt 模板  
- 场景 B（铺其它平台）  
- 妙手/店小秘发 1688 商家店  
- 从零重写采集解析  
- 改坏 Temu / AE / Ozon / v1  

---

## 7. 锁定决策

| ID | 决策 |
|----|------|
| D1 | 场景 A：自家 1688 商家后台 |
| D2 | 采集移植 `D:\reverselab\1688-` |
| D3 | 生图仅经 Server → `AiEditImage` → hyhacct |
| D4 | 发品禁止妙手/店小秘 |
| D5 | 采集会话与商家会话分开校验 |
| D6 | 主战场 `D:\dev\workspace` |

细则与冲突权威：[README](./1688-listing-wizard-v2/README.md)。
