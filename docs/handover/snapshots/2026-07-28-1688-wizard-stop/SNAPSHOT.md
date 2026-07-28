# 全量工作快照 — 2026-07-28 停点

**时间**：2026-07-28 ~17:22（UTC+8）  
**指令**：用户要求「停吧；保存当前工作全量快照」  
**范围**：1688 竞品向导本地联调 + 母版匹配（发布相似品）相关三端代码与文档  

## 停点结论

| 项 | 状态 |
|---|---|
| Workflow | **13** |
| Offer | `https://detail.1688.com/offer/688486165480.html`（用户提供；旧链 `735580842917` 已软拦截） |
| 主状态 | **`img_ready`** |
| 图位 | 5 槽均已 **`selected`**，含 `result_url`（aito/humanize CDN） |
| 采集 | 成功（标题约「千鲤莱子线双钩…」，SKU/价 3.35–3.45） |
| 母版匹配 | 未跑：`match_status` / candidates / similar page 仍空 |
| 下一步（未做） | Agent 在线 → 卖家登录 → templates/candidates → select → open-similar → form/save-draft |

## 关键运行注意

- Server：`localhost:34206`，启动需 `LISTING_1688_COOKIE_KEY`（Base64）  
- Web：Vite ~`5173`  
- Agent：`wails dev` / Vite `34115`；`agent.go` 当前为 **`ws` + `localhost:34206`**（本地联调，非线上）  
- 登录：`admin` / `admin123`  
- 多开 Agent 会导致 WebSocket `replaced by new connection`；Server 重启后需等 Agent 重连  

## 关键行为结论

1. `done` ≠ `img_ready`：需各图位 `images/select` 至 `selected` 后 `listing1688RecomputeImageReady` 才进 `img_ready`。  
2. 旧 offer 软拦截：`soft_block_notfound_home`（跳 `www.1688.com/?spm=...notfound`），非可见验证码壳。  
3. 母版匹配与「打开发布相似品」代码已落地，但 workflow 13 停在图优完成，尚未进入 Step4 匹配。  

## 归档内容

| 路径 | 说明 |
|---|---|
| `workspace-code-snapshot.zip` | 三端相关源码 + `AGENTS.md` / 开发文档 / `docs/superpowers`（**不含** `config.yaml`、Cookie、Browser、`.local`、运行日志） |
| 本文件 | 停点元数据 |

## Git 落点（同次提交）

- `commander-agent-t260220-main/commander-agent-t260220-main`：1688 listing / seller / protocol / 本地 WS  
- `commander-web-t260220-main/commander-web-t260220-main`：listing1688 API + wizard UI 及相关导航  
- `D:\dev\workspace`：本快照目录、开发文档条目、AGENTS/交接相关文档（Server 整树仍多为未跟踪；以 zip 为准）  

## 明确未纳入快照

- `etc/config/config.yaml` 与任何 Cookie / API Key  
- Agent `agent/logs/*`、浏览器用户数据、`.local/`  
- `public/assets` 大资源、`node_modules`、构建产物  
