# 1688 Listing Wizard M1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver M1 collect-login + CDP peer collect preview for the 1688 wizard (Step1→`session_valid`, Step2→`collected` with title/images/SKU).

**Architecture:** Web calls `/api/v1/listing1688/...`; Server orchestrates state + encrypts cookies (`LISTING_1688_COOKIE_KEY`); Agent (`platform=1688`) handles SyncSend protocols via existing Rod browser; HTML parse logic transplanted from `D:\reverselab\1688-\fetch_1688_offer.py` into Agent Go package `offer1688` (same field rules, no from-scratch rewrite).

**Tech Stack:** Go/Gin/GORM, Agent Rod CDP, Vue3 wizard UI.

## Global Constraints

- Main battlefield: `D:\dev\workspace`
- Collect source: transplant `D:\reverselab\1688-\fetch_1688_offer.py` (+ login URL/probe from `alibaba_1688_login.py`)
- Scene A only; no Miaoshou/店小秘 for wizard publish
- No silent Onebound fallback
- Frontend never shows cookie plaintext
- Probe: `GET https://sycm.1688.com/ms/common/information.json` and `code==0`
- Success collect (C4): title + ≥1 image + SKU
- v1 `/listing` untouched

## Approach (locked)

| Layer | Choice |
|-------|--------|
| Agent browser | Existing Rod profile (open login page for user) |
| Parser | Go port of Python extractors in `offer1688` |
| Probe | Agent SyncSend with cookies (domain-accurate) |
| Cookie at rest | Server AES-GCM; Agent only returns plaintext over SyncSend once |

## Protocols (platform=`1688`)

| Protocol | Send | Receive |
|----------|------|---------|
| `listing1688_open_login` | — | `{login_url}` |
| `listing1688_export_cookie` | — | `{cookie}` (plaintext once) |
| `listing1688_probe` | `{cookie?}` optional use browser session | `{ok, code, message}` |
| `listing1688_fetch_offer` | `{url, cookie?}` | `{title, images[], skus[], price_min, price_max, offer_id, captcha?}` |

## File Map

| Path | Role |
|------|------|
| Server `internal/utils/listing1688_cookie_crypto.go` | Encrypt/decrypt |
| Server `internal/services/listing1688_session.go` | open-login / confirm / validate |
| Server `internal/services/listing1688_collect.go` | collect / retry |
| Server `internal/constants/protocol.go` | protocol consts |
| Server `internal/repos/listing_1688_session.go` | session repo |
| Agent `constants/protocol.go` + `dispatcher` | route 4 protocols |
| Agent `factory/alibaba/listing1688_*.go` | Rod open/export/probe/fetch |
| Agent `internal/offer1688/parse.go` | transplanted parsers |
| Web wizard + `listing1688.js` | Step1/2 real actions |

---

### Task 1: Cookie crypto + session repo + Session APIs

- [x] AES-GCM util + tests (missing key → clear error)
- [x] Session repo Create/Update/GetByWorkflow
- [x] `open-login` → SyncSend → `session_pending`
- [x] `confirm` → export cookie → encrypt → probe → `session_valid` or fail
- [x] `validate` → probe with stored cookie

### Task 2: Agent four protocols + offer1688 parse

- [x] Port parse helpers + unit test on sample HTML fixture from 1688- or minimal fixture
- [x] Open login / export cookie / probe / fetch offer on Alibaba factory
- [x] Dispatcher cases for new protocols

### Task 3: Collect APIs + Web Step1/2

- [x] `collect` / `retry` → FetchOffer → C4 check → save snapshot → `collected`
- [x] Wizard UI: open login, confirm, paste URL, analyze, preview
- [x] Gate Step2 until `session_valid`; show captcha retry UX
- [x] 开发文档 entry

---

## Self-Review

1. Spec M1: login+probe+CDP collect+preview — covered
2. No Miaoshou on wizard path
3. Parser transplanted not rewritten from zero
