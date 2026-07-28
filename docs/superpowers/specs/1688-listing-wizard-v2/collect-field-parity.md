# Collect field parity（TC-C-05）

**日期**：2026-07-24  
**权威解析源**：`D:\reverselab\1688-\fetch_1688_offer.py`（`_parse_product_from_html` / `_skus_from_context`）  
**Wizard 落点**：Agent `backend/internal/offer1688` → Server `offer_snapshot` → M2 `main_images` / M3 `skus`

> 注：`D:\reverselab\1688-\exports\` 当前无实网样例 JSON。本对照以 Python 成功返回字段契约 + Agent 合成 fixture（`offer1688/testdata/`）为准。

---

## Canonical 字段（向导锁定）

| Canonical 名 | 类型 / 要求 | 1688- 源字段 | M2 / M3 消费 |
|--------------|-------------|--------------|--------------|
| `title` | 非空 string | `title` | M2 槽位标题；M3 发品标题回退 |
| `main_images` | `string[]`，≥1 绝对 URL | `main_images` | **M2 唯一读图键**（`listing1688ParseSnapshotMainImages`） |
| `skus` | 对象数组，≥1 | `skus` | **M3** `listing1688SnapshotSkus` → Agent 填价/规格 |
| `price_min` / `price_max` | `number` 可空；有价则写出 | `price_range.min` / `.max`（扁平化） | 采集预览；发品以 SKU `price` 为准 |
| `source_item_id` | 非空 offerId | `source_item_id` | 可追溯；API preview 别名 `offer_id` |
| `url` | 源详情 URL | `url` / `source_item_url` | 可追溯；workflow `offer_url` |

**非 canonical（禁止当主路径）**

| 名 | 说明 |
|----|------|
| `images` | 仅 1688- meta fallback 内部名；**snapshot 根键不用**；M2 不认 |
| `offer_id` | 仅 collect HTTP preview 别名；**snapshot 存 `source_item_id`** |
| `price_range` | 1688- 嵌套结构；Wizard **扁平为** `price_min`/`price_max` |

---

## SKU 元素字段

| Canonical | 1688- | 说明 |
|-----------|-------|------|
| `sku_key` | `sku_key` | skuMap 键（HTML unescape） |
| `spec_attrs` | `spec_attrs` | 规格文案 |
| `name` | （无；向导别名） | **= `spec_attrs`**，兼容 M3 seller 读 `name` |
| `price` | `price` | `discountPrice` 优先，否则 `price` |
| `stock` | `stock` | `canBookCount` / `stock` |
| `source_sku_id` | `source_sku_id` | 来自 `skuId`；M3 优先作货号 |
| `spec_id` | `spec_id` | 来自 `specId` |

M3 Agent 读价键：`price` → `price_min` → `consignPrice` → `salePrice`  
M3 Agent 读规格/货号：`source_sku_id` → `sku_id`/`skuId` → `spec_attrs` → `sku_key` → `spec_id` → `name` …

---

## 有意不移植（YAGNI / MVP）

| 1688- 字段 | 原因 |
|------------|------|
| `description_html` / `description_url` | 发品自动化未读详情 HTML |
| `sku_props` | M3 未用 |
| `source` / `source_platform` / `from_cache` | 元数据，非 M2/M3 |
| `images[]`（SKU 内） | 源常空数组；图优走 `main_images` |

---

## Fixture / 单测

| 路径 | 用途 |
|------|------|
| `commander-agent-.../backend/internal/offer1688/testdata/init_data_offer.html` | `__INIT_DATA` 合成页 → 解析出 canonical 快照 |
| `.../testdata/sample_1688_direct_shape.json` | 1688- 成功返回形状样例（权威文档） |
| `TestParseProductFromHTML_InitDataCanonicalFields` | 锁定 title / main_images / skus / source_item_id / price_* |
| `TestListing1688SnapshotCanonicalFields_M2M3`（Server） | M2/M3 消费路径 |
| `TestListing1688FirstSKUCode_*`（Agent seller） | 读 `source_sku_id` / `spec_attrs` |

---

## 验收（TC-C-05）

- [x] 与 1688- 字段契约对照（Python + 合成样例；实网 exports 空）
- [x] Wizard snapshot 使用上表 canonical 名
- [x] 单测锁定 M2/M3 读键，防止漂移
