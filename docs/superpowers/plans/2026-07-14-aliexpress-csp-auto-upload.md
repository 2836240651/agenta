# AliExpress CSP Auto-Upload MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build independent Python CLI `ae-csp` that logs into AliExpress CSP (全托管), captures publish APIs, and replays them via Playwright `APIRequestContext` to create a product from `product.json` — with no 妙手/店小秘 dependency.

**Architecture:** SAU-style layout: `ae_cli.py` dispatches to `uploader/csp_uploader/{auth,capture,publish,product_schema,endpoints}.py`. Cookies are Playwright `storage_state` files. Capture writes full bodies to `filtered.json`. Publish runs pluggable steps over `context.request`; RefreshCookie only on exit code 0.

**Tech Stack:** Python 3.11+, Patchright, PyYAML, pytest, Click or argparse (prefer argparse for fewer deps)

**Spec:** `docs/superpowers/specs/2026-07-14-aliexpress-csp-auto-upload-design.md`  
**Doc pack:** `docs/superpowers/specs/aliexpress-csp-mvp/`

## Global Constraints

- Project root: `D:\dev\workspace\aliexpress-csp-auto-upload` — **independent git repo** (not nested under Commander / Agent)
- CLI entry name after install: `ae-csp`
- Exit codes: `0` success, `2` auth, `3` business/validation, `4` network/unknown
- `login` / `capture`: headed only; `publish` may use `--headless`; optional `api` diagnostic
- Cookie path: `cookies/csp_<account>.json`
- Account URL resolve order: `accounts/<account>.yaml` > `conf.py` defaults
- Never call `erp.91miaoshou.com` or `dianxiaomi.com`
- `cookies/`, `captures/`, `accounts/*.yaml` (except `*.example.yaml`) gitignored
- RefreshCookie: write `storage_state` **only** when overall publish exit code is 0
- `filtered.json` must keep **full** request/response bodies; truncate only in `requests.jsonl`
- Do not implement Excel adapter or Commander Agent wiring in this plan
- PowerShell: use `;` not `&&`

---

## File structure (lock-in)

```text
aliexpress-csp-auto-upload/
├── pyproject.toml
├── README.md
├── conf.py
├── ae_cli.py
├── .gitignore
├── accounts/demo.example.yaml
├── products/demo.json
├── cookies/                    # gitignore + .gitkeep optional
├── captures/                   # gitignore
├── uploader/csp_uploader/
│   ├── __init__.py
│   ├── paths.py                # resolve cookie/account/capture paths
│   ├── account_config.py       # load yaml + conf defaults
│   ├── auth.py
│   ├── capture.py
│   ├── product_schema.py
│   ├── endpoints.py            # placeholder steps until M4
│   └── publish.py
├── utils/
│   ├── __init__.py
│   └── logging_util.py
├── tests/
│   ├── test_account_config.py
│   ├── test_paths.py
│   ├── test_product_schema.py
│   ├── test_capture_filter.py
│   ├── test_exit_codes.py
│   └── test_publish_dry_run.py
└── docs/                       # copy from workspace pack after Task 1
    ├── CLI.md
    ├── capture-notes.md        # template until M4 fill
    ├── requirements.md
    ├── prd-实施包.md
    ├── 技术分析文档.md
    └── 测试用例.md
```

---

### Task 1: Scaffold independent repo + package entry

**Files:**
- Create: `D:\dev\workspace\aliexpress-csp-auto-upload/pyproject.toml`
- Create: `D:\dev\workspace\aliexpress-csp-auto-upload/README.md`
- Create: `D:\dev\workspace\aliexpress-csp-auto-upload/.gitignore`
- Create: `D:\dev\workspace\aliexpress-csp-auto-upload/ae_cli.py`
- Create: `D:\dev\workspace\aliexpress-csp-auto-upload/conf.py`
- Create: `D:\dev\workspace\aliexpress-csp-auto-upload/uploader/csp_uploader/__init__.py`
- Create: `D:\dev\workspace\aliexpress-csp-auto-upload/utils/__init__.py`
- Create: `D:\dev\workspace\aliexpress-csp-auto-upload/docs/CLI.md` (stub)

**Interfaces:**
- Consumes: none
- Produces: installable package exposing console script `ae-csp` → `ae_cli:main`

- [ ] **Step 1: Create directory and git init**

```powershell
New-Item -ItemType Directory -Force -Path "D:\dev\workspace\aliexpress-csp-auto-upload" | Out-Null
Set-Location "D:\dev\workspace\aliexpress-csp-auto-upload"
git init
```

- [ ] **Step 2: Write `.gitignore`**

```gitignore
__pycache__/
*.py[cod]
.venv/
dist/
*.egg-info/
.pytest_cache/
cookies/
captures/
accounts/*.yaml
!accounts/*.example.yaml
.local/
```

- [ ] **Step 3: Write `pyproject.toml`**

```toml
[project]
name = "aliexpress-csp-auto-upload"
version = "0.1.0"
description = "AliExpress CSP (full-managed) listing CLI: login, capture, API replay"
readme = "README.md"
requires-python = ">=3.11"
dependencies = [
  "patchright>=1.58.0",
  "PyYAML>=6.0",
]

[project.optional-dependencies]
dev = ["pytest>=8.0"]

[project.scripts]
ae-csp = "ae_cli:main"

[build-system]
requires = ["setuptools>=68"]
build-backend = "setuptools.build_meta"

[tool.setuptools.packages.find]
include = ["uploader*", "utils*"]

[tool.setuptools]
py-modules = ["ae_cli", "conf"]
```

- [ ] **Step 4: Write minimal `ae_cli.py` and `conf.py`**

```python
# conf.py
from pathlib import Path

BASE_DIR = Path(__file__).resolve().parent

# IMPORTANT: shop-specific; override per account via accounts/<name>.yaml
DEFAULT_LOGIN_URL = (
    "https://login.aliexpress.com/user/seller/login?bizSegment=CSP"
    "&return_url=http%3A%2F%2Fcsp.aliexpress.com%2Fait%2Fcn_ams%2Fitem_choice%2Fproduct_publish"
)
DEFAULT_PUBLISH_URL = (
    "https://csp.aliexpress.com/ait/cn_ams/item_choice/product_publish"
)

LOGIN_MARKERS = ("登录", "请登录", "Sign in")
CAPTURE_HOST_ALLOW = ("csp.aliexpress.com", "aliexpress.com")
CAPTURE_PATH_KEYWORDS = (
    "product", "item", "sku", "upload", "publish", "draft", "choice", "ams",
)
PUBLISH_REQUEST_INTERVAL_SEC = 1.5
```

```python
# ae_cli.py
from __future__ import annotations
import argparse
import sys

EXIT_OK = 0
EXIT_AUTH = 2
EXIT_BIZ = 3
EXIT_OTHER = 4

def build_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(prog="ae-csp")
    sub = p.add_subparsers(dest="command", required=True)
    login = sub.add_parser("login")
    login.add_argument("--account", required=True)
    check = sub.add_parser("check")
    check.add_argument("--account", required=True)
    capt = sub.add_parser("capture")
    capt.add_argument("--account", required=True)
    capt.add_argument("--out", default=None)
    capt.add_argument("--har", action="store_true")
    pub = sub.add_parser("publish")
    pub.add_argument("--account", required=True)
    pub.add_argument("--from", dest="from_path", required=True)
    pub.add_argument("--dry-run", action="store_true")
    pub.add_argument("--headless", action="store_true")
    api = sub.add_parser("api")
    api.add_argument("--account", required=True)
    api.add_argument("--method", required=True)
    api.add_argument("--url", required=True)
    api.add_argument("--body-file", default=None)
    return p

def main(argv: list[str] | None = None) -> None:
    args = build_parser().parse_args(argv)
    print(f"command={args.command} (not implemented yet)", file=sys.stderr)
    raise SystemExit(EXIT_OTHER)

if __name__ == "__main__":
    main()
```

- [ ] **Step 5: Write README stub + create venv + install editable**

```powershell
Set-Location "D:\dev\workspace\aliexpress-csp-auto-upload"
python -m venv .venv
.\.venv\Scripts\python.exe -m pip install -e ".[dev]"
$env:PLAYWRIGHT_DOWNLOAD_HOST="https://npmmirror.com/mirrors/playwright"
.\.venv\Scripts\patchright.exe install chromium
.\.venv\Scripts\ae-csp.exe --help
```

Expected: help lists `login`/`check`/`capture`/`publish`/`api`.

- [ ] **Step 6: Copy doc pack into `docs/`**

```powershell
Copy-Item "D:\dev\workspace\docs\superpowers\specs\aliexpress-csp-mvp\*.md" `
  "D:\dev\workspace\aliexpress-csp-auto-upload\docs\" -Force
```

- [ ] **Step 7: Commit in the new repo**

```powershell
Set-Location "D:\dev\workspace\aliexpress-csp-auto-upload"
git add pyproject.toml README.md .gitignore ae_cli.py conf.py uploader utils docs
git commit -m "chore: scaffold ae-csp package and CLI skeleton"
```

---

### Task 2: Paths + account config resolution

**Files:**
- Create: `uploader/csp_uploader/paths.py`
- Create: `uploader/csp_uploader/account_config.py`
- Create: `accounts/demo.example.yaml`
- Create: `tests/test_paths.py`
- Create: `tests/test_account_config.py`

**Interfaces:**
- Consumes: `conf.BASE_DIR`, `DEFAULT_LOGIN_URL`, `DEFAULT_PUBLISH_URL`
- Produces:
  - `cookie_path(account: str) -> Path`
  - `account_yaml_path(account: str) -> Path`
  - `dataclass AccountConfig(login_url: str, publish_url: str)`
  - `load_account_config(account: str) -> AccountConfig`

- [ ] **Step 1: Write failing tests**

```python
# tests/test_paths.py
from uploader.csp_uploader.paths import cookie_path

def test_cookie_path_uses_csp_prefix(tmp_path, monkeypatch):
    import conf
    monkeypatch.setattr(conf, "BASE_DIR", tmp_path)
    assert cookie_path("demo").name == "csp_demo.json"
    assert cookie_path("demo").parent.name == "cookies"
```

```python
# tests/test_account_config.py
from pathlib import Path
import yaml
from uploader.csp_uploader.account_config import load_account_config

def test_defaults_when_no_yaml(tmp_path, monkeypatch):
    import conf
    monkeypatch.setattr(conf, "BASE_DIR", tmp_path)
    (tmp_path / "accounts").mkdir()
    cfg = load_account_config("demo")
    assert "login.aliexpress.com" in cfg.login_url
    assert "csp.aliexpress.com" in cfg.publish_url

def test_yaml_overrides_defaults(tmp_path, monkeypatch):
    import conf
    monkeypatch.setattr(conf, "BASE_DIR", tmp_path)
    acc = tmp_path / "accounts"
    acc.mkdir()
    (acc / "demo.yaml").write_text(
        yaml.dump({
            "login_url": "https://login.example/csp?shop=A",
            "publish_url": "https://csp.example/publish?channelId=1",
        }),
        encoding="utf-8",
    )
    cfg = load_account_config("demo")
    assert cfg.login_url.endswith("shop=A")
    assert "channelId=1" in cfg.publish_url
```

- [ ] **Step 2: Run tests — expect FAIL**

```powershell
Set-Location "D:\dev\workspace\aliexpress-csp-auto-upload"
.\.venv\Scripts\pytest.exe tests/test_paths.py tests/test_account_config.py -v
```

Expected: import/module errors.

- [ ] **Step 3: Implement**

```python
# uploader/csp_uploader/paths.py
from __future__ import annotations
from pathlib import Path
import conf

def cookie_path(account: str) -> Path:
    return conf.BASE_DIR / "cookies" / f"csp_{account}.json"

def account_yaml_path(account: str) -> Path:
    return conf.BASE_DIR / "accounts" / f"{account}.yaml"

def captures_root() -> Path:
    return conf.BASE_DIR / "captures"
```

```python
# uploader/csp_uploader/account_config.py
from __future__ import annotations
from dataclasses import dataclass
from pathlib import Path
import yaml
import conf
from uploader.csp_uploader.paths import account_yaml_path

@dataclass(frozen=True)
class AccountConfig:
    login_url: str
    publish_url: str

def load_account_config(account: str) -> AccountConfig:
    path = account_yaml_path(account)
    data: dict = {}
    if path.is_file():
        loaded = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
        if not isinstance(loaded, dict):
            raise ValueError(f"invalid account yaml: {path}")
        data = loaded
    return AccountConfig(
        login_url=str(data.get("login_url") or conf.DEFAULT_LOGIN_URL),
        publish_url=str(data.get("publish_url") or conf.DEFAULT_PUBLISH_URL),
    )
```

```yaml
# accounts/demo.example.yaml
login_url: "https://login.aliexpress.com/user/seller/login?bizSegment=CSP&return_url=http%3A%2F%2Fcsp.aliexpress.com%2Fait%2Fcn_ams%2Fitem_choice%2Fproduct_publish%3FswitchId%3DREPLACE%26from%3DSELF%26channelId%3DREPLACE"
publish_url: "https://csp.aliexpress.com/ait/cn_ams/item_choice/product_publish?switchId=REPLACE&from=SELF&channelId=REPLACE"
```

- [ ] **Step 4: Run tests — expect PASS**

```powershell
.\.venv\Scripts\pytest.exe tests/test_paths.py tests/test_account_config.py -v
```

- [ ] **Step 5: Commit**

```powershell
git add uploader/csp_uploader/paths.py uploader/csp_uploader/account_config.py accounts/demo.example.yaml tests
git commit -m "feat: add account URL resolution and cookie paths"
```

---

### Task 3: Product schema validation

**Files:**
- Create: `uploader/csp_uploader/product_schema.py`
- Create: `products/demo.json`
- Create: `tests/test_product_schema.py`

**Interfaces:**
- Consumes: JSON file path
- Produces: `load_product(path: Path) -> dict` raises `ProductValidationError` with message; CLI maps to exit 3

- [ ] **Step 1: Failing tests**

```python
# tests/test_product_schema.py
import json
from pathlib import Path
import pytest
from uploader.csp_uploader.product_schema import load_product, ProductValidationError

def test_rejects_missing_title(tmp_path: Path):
    p = tmp_path / "bad.json"
    p.write_text(json.dumps({
        "main_images": ["https://example.com/a.jpg"],
        "skus": [{"price": 1.0, "stock": 1}],
    }), encoding="utf-8")
    with pytest.raises(ProductValidationError):
        load_product(p)

def test_accepts_minimal(tmp_path: Path):
    p = tmp_path / "ok.json"
    p.write_text(json.dumps({
        "title": "Demo",
        "main_images": ["https://example.com/a.jpg"],
        "skus": [{"outer_sku_id": "S1", "price": 9.9, "stock": 10}],
        "extras": {},
    }), encoding="utf-8")
    data = load_product(p)
    assert data["title"] == "Demo"
```

- [ ] **Step 2: Run — expect FAIL**

```powershell
.\.venv\Scripts\pytest.exe tests/test_product_schema.py -v
```

- [ ] **Step 3: Implement**

```python
# uploader/csp_uploader/product_schema.py
from __future__ import annotations
import json
from pathlib import Path

class ProductValidationError(ValueError):
    pass

def load_product(path: Path) -> dict:
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as e:
        raise ProductValidationError(str(e)) from e
    if not isinstance(data, dict):
        raise ProductValidationError("product root must be object")
    title = str(data.get("title") or "").strip()
    if not title:
        raise ProductValidationError("title is required")
    images = data.get("main_images")
    if not isinstance(images, list) or len(images) < 1:
        raise ProductValidationError("main_images requires >= 1 entry")
    skus = data.get("skus")
    if not isinstance(skus, list) or len(skus) < 1:
        raise ProductValidationError("skus requires >= 1 entry")
    for i, sku in enumerate(skus):
        if not isinstance(sku, dict):
            raise ProductValidationError(f"skus[{i}] must be object")
        if "price" not in sku or "stock" not in sku:
            raise ProductValidationError(f"skus[{i}] requires price and stock")
    if "extras" not in data:
        data["extras"] = {}
    return data
```

```json
{
  "title": "AE CSP Demo Product",
  "currency": "CNY",
  "category_id": "",
  "main_images": ["https://ae01.alicdn.com/kf/placeholder.jpg"],
  "detail_images": [],
  "description": "mvp demo",
  "skus": [
    {
      "outer_sku_id": "DEMO-001",
      "price": 19.9,
      "stock": 100,
      "weight_kg": 0.2,
      "image": "",
      "attrs": { "Color": "Black" }
    }
  ],
  "extras": {}
}
```

- [ ] **Step 4: Run — expect PASS**

```powershell
.\.venv\Scripts\pytest.exe tests/test_product_schema.py -v
```

- [ ] **Step 5: Commit**

```powershell
git add uploader/csp_uploader/product_schema.py products/demo.json tests/test_product_schema.py
git commit -m "feat: validate product.json minimal schema"
```

---

### Task 4: Auth — login + check

**Files:**
- Create: `uploader/csp_uploader/auth.py`
- Create: `utils/logging_util.py`
- Modify: `ae_cli.py` (wire login/check)
- Create: `tests/test_auth_helpers.py` (pure helpers only; live login is manual)

**Interfaces:**
- Consumes: `AccountConfig`, `cookie_path`
- Produces:
  - `async def cookie_auth(account: str) -> bool`
  - `async def login_account(account: str, timeout_sec: int = 300) -> Path`
  - `def page_looks_logged_out(page_text: str) -> bool`
  - `def url_looks_like_csp_workspace(url: str) -> bool`

- [ ] **Step 1: Failing unit tests for gate helpers**

```python
# tests/test_auth_helpers.py
from uploader.csp_uploader.auth import page_looks_logged_out, url_looks_like_csp_workspace

def test_login_markers():
    assert page_looks_logged_out("请登录后继续")
    assert not page_looks_logged_out("商品发布 草稿箱")

def test_csp_url():
    assert url_looks_like_csp_workspace(
        "https://csp.aliexpress.com/ait/cn_ams/item_choice/product_publish?x=1"
    )
    assert not url_looks_like_csp_workspace("https://login.aliexpress.com/user/seller/login")
```

- [ ] **Step 2: Run — expect FAIL**

```powershell
.\.venv\Scripts\pytest.exe tests/test_auth_helpers.py -v
```

- [ ] **Step 3: Implement auth.py (minimal)**

```python
# uploader/csp_uploader/auth.py
from __future__ import annotations
import asyncio
from pathlib import Path
from patchright.async_api import async_playwright
import conf
from uploader.csp_uploader.account_config import load_account_config
from uploader.csp_uploader.paths import cookie_path

def page_looks_logged_out(text: str) -> bool:
    return any(m in text for m in conf.LOGIN_MARKERS)

def url_looks_like_csp_workspace(url: str) -> bool:
    if "csp.aliexpress.com" not in url:
        return False
    if "login.aliexpress.com" in url:
        return False
    return any(k in url for k in ("product_publish", "item_choice", "cn_ams", "/ait/"))

async def cookie_auth(account: str, headless: bool = False) -> bool:
    path = cookie_path(account)
    if not path.is_file():
        return False
    cfg = load_account_config(account)
    async with async_playwright() as p:
        browser = await p.chromium.launch(headless=headless, channel="chromium")
        context = await browser.new_context(storage_state=str(path))
        page = await context.new_page()
        await page.goto(cfg.publish_url, wait_until="domcontentloaded", timeout=120_000)
        await page.wait_for_timeout(2000)
        ok = url_looks_like_csp_workspace(page.url) and not page_looks_logged_out(
            await page.content()
        )
        await browser.close()
        return ok

async def login_account(account: str, timeout_sec: int = 300) -> Path:
    cfg = load_account_config(account)
    out = cookie_path(account)
    out.parent.mkdir(parents=True, exist_ok=True)
    async with async_playwright() as p:
        browser = await p.chromium.launch(headless=False, channel="chromium")
        context = await browser.new_context()
        page = await context.new_page()
        await page.goto(cfg.login_url, wait_until="domcontentloaded", timeout=120_000)
        deadline = asyncio.get_event_loop().time() + timeout_sec
        while asyncio.get_event_loop().time() < deadline:
            if url_looks_like_csp_workspace(page.url) and not page_looks_logged_out(
                await page.content()
            ):
                await context.storage_state(path=str(out))
                await browser.close()
                return out
            await page.wait_for_timeout(2000)
        await browser.close()
        raise TimeoutError("login timed out; complete login in the browser window")
```

Wire `ae_cli.py`:

```python
def main(argv: list[str] | None = None) -> None:
    args = build_parser().parse_args(argv)
    try:
        if args.command == "login":
            path = asyncio.run(login_account(args.account))
            print(f"saved {path}")
            raise SystemExit(EXIT_OK)
        if args.command == "check":
            ok = asyncio.run(cookie_auth(args.account, headless=False))
            if not ok:
                print("auth failed: run ae-csp login --account ...", file=sys.stderr)
                raise SystemExit(EXIT_AUTH)
            print("check ok")
            raise SystemExit(EXIT_OK)
        # other commands still EXIT_OTHER until later tasks
        ...
    except TimeoutError as e:
        print(str(e), file=sys.stderr)
        raise SystemExit(EXIT_AUTH)
```

- [ ] **Step 4: Unit tests PASS**

```powershell
.\.venv\Scripts\pytest.exe tests/test_auth_helpers.py -v
```

- [ ] **Step 5: Manual smoke (operator)**

```powershell
# copy accounts/demo.example.yaml → accounts/demo.yaml and paste real switchId/channelId
.\.venv\Scripts\ae-csp.exe login --account demo
.\.venv\Scripts\ae-csp.exe check --account demo
```

Expected: cookie file created; check exit 0.

- [ ] **Step 6: Commit**

```powershell
git add uploader/csp_uploader/auth.py utils ae_cli.py tests/test_auth_helpers.py
git commit -m "feat: implement CSP login and headed session check"
```

---

### Task 5: Capture filter + session writer

**Files:**
- Create: `uploader/csp_uploader/capture.py`
- Create: `tests/test_capture_filter.py`
- Modify: `ae_cli.py`

**Interfaces:**
- Consumes: `conf.CAPTURE_*`, account config, cookie
- Produces:
  - `def truncate_body(raw: str | None, limit: int = 2048) -> str`
  - `def is_candidate_request(url: str, resource_type: str) -> bool`
  - `async def run_capture(account: str, out_dir: Path | None, save_har: bool) -> Path`
  - Session files: `meta.json`, `requests.jsonl`, `filtered.json` (full bodies)

- [ ] **Step 1: Failing filter tests**

```python
# tests/test_capture_filter.py
from uploader.csp_uploader.capture import is_candidate_request, truncate_body

def test_truncate():
    assert len(truncate_body("x" * 5000, 100)) <= 120

def test_candidate_keeps_product_xhr():
    assert is_candidate_request(
        "https://csp.aliexpress.com/api/ams/item/product/save", "xhr"
    )

def test_drops_static():
    assert not is_candidate_request(
        "https://csp.aliexpress.com/static/app.js", "script"
    )
```

- [ ] **Step 2: Run — FAIL then implement filter helpers + capture loop**

Core behavior for `run_capture`:

1. Require existing cookie; else raise auth error (CLI → exit 2)
2. Headed browser + `storage_state`
3. Goto `publish_url`
4. On response: append truncated line to jsonl; if candidate, append **full** body entry to in-memory `filtered` list
5. Print: "完成手点发品后，在此终端按 Enter 结束抓包"
6. `input()` wait; on KeyboardInterrupt still flush
7. Write `meta.json`, `filtered.json`, optional har via `context.tracing` or playwright har if easy — minimum: skip real HAR API if costly; `--har` may write note "not yet" only if blocked — prefer implement `record_har_path` on `new_context(..., record_har_path=...)` when `--har`

```python
def is_candidate_request(url: str, resource_type: str) -> bool:
    rt = (resource_type or "").lower()
    if rt in {"image", "stylesheet", "font", "media", "script"}:
        return False
    host_ok = any(h in url for h in conf.CAPTURE_HOST_ALLOW)
    if not host_ok:
        return False
    return any(k in url.lower() for k in conf.CAPTURE_PATH_KEYWORDS)
```

- [ ] **Step 3: pytest filter PASS**

```powershell
.\.venv\Scripts\pytest.exe tests/test_capture_filter.py -v
```

- [ ] **Step 4: Wire CLI `capture` + commit**

```powershell
git add uploader/csp_uploader/capture.py tests/test_capture_filter.py ae_cli.py
git commit -m "feat: add CSP capture session with full filtered bodies"
```

- [ ] **Step 5: Manual M3 smoke**

```powershell
.\.venv\Scripts\ae-csp.exe capture --account demo
# hand-publish one item in the browser, then press Enter
```

Verify `captures/<ts>/filtered.json` has non-truncated JSON bodies.

---

### Task 6: Endpoints placeholder + publish orchestration (dry-run)

**Files:**
- Create: `uploader/csp_uploader/endpoints.py`
- Create: `uploader/csp_uploader/publish.py`
- Create: `tests/test_publish_dry_run.py`
- Create: `docs/capture-notes.md` (template)
- Modify: `ae_cli.py`

**Interfaces:**
- Consumes: `load_product`, `cookie_auth`, `load_account_config`
- Produces:
  - `ENDPOINT_STEPS: list[EndpointStep]` initially empty or stub
  - `@dataclass EndpointStep(name: str, method: str, url_template: str, build_body: Callable | None)`
  - `async def publish_product(account, product_path, dry_run: bool, headless: bool) -> int` returns exit code
  - Dry-run prints planned steps; does not POST; does not rewrite cookie

- [ ] **Step 1: Write dry-run test with stub steps**

```python
# tests/test_publish_dry_run.py
import json
from pathlib import Path
import pytest
from uploader.csp_uploader import endpoints, publish

@pytest.mark.asyncio
async def test_dry_run_does_not_require_real_http(tmp_path, monkeypatch):
    # Point cookie to a fake file presence check bypass by mocking cookie_auth
    async def ok_auth(account, headless=False):
        return True
    monkeypatch.setattr(publish, "cookie_auth", ok_auth)
    monkeypatch.setattr(endpoints, "ENDPOINT_STEPS", [])
    product = tmp_path / "p.json"
    product.write_text(json.dumps({
        "title": "T",
        "main_images": ["https://x/y.jpg"],
        "skus": [{"price": 1, "stock": 1}],
        "extras": {},
    }), encoding="utf-8")
    code = await publish.publish_product("demo", product, dry_run=True, headless=True)
    assert code == 0
```

Add `pytest-asyncio` to dev deps if needed, or use `asyncio.run` wrapper in test without pytest-asyncio:

```python
def test_dry_run(...):
    code = asyncio.run(publish.publish_product(...))
    assert code == 0
```

- [ ] **Step 2: Implement publish state machine skeleton**

```python
async def publish_product(account: str, product_path: Path, dry_run: bool, headless: bool) -> int:
    if not await cookie_auth(account, headless=False):
        return 2
    try:
        product = load_product(product_path)
    except ProductValidationError:
        return 3
    steps = list(ENDPOINT_STEPS)
    if dry_run:
        for s in steps:
            print(f"[dry-run] {s.name} {s.method} {s.url_template}")
        if not steps:
            print("[dry-run] ENDPOINT_STEPS empty — complete M4 capture + fill endpoints.py")
        return 0
    if not steps:
        print("ENDPOINT_STEPS empty; refuse non-dry-run publish", file=sys.stderr)
        return 3
    # BrowserContext + APIRequestContext loop (Task 7 fills real fetch)
    ...
```

- [ ] **Step 3: `endpoints.py` placeholder**

```python
from __future__ import annotations
from dataclasses import dataclass
from typing import Any, Callable

@dataclass
class EndpointStep:
    name: str
    method: str
    url_template: str
    # build_payload(product: dict, ctx: dict) -> Any | None
    build_payload: Callable[[dict, dict], Any] | None = None
    headers: dict[str, str] | None = None

# Filled after M4 manual capture. Keep empty until then.
ENDPOINT_STEPS: list[EndpointStep] = []
```

- [ ] **Step 4: Template `docs/capture-notes.md`**

```markdown
# Capture notes (fill after M4)

## Session id
## Ordered endpoints
1. name / method / url
2. required headers (CSRF names)
3. success response fields
## Publish semantics
- draft vs submit vs online:
## Upload
- OSS direct vs CSP proxy:
```

- [ ] **Step 5: pytest + commit**

```powershell
.\.venv\Scripts\pytest.exe tests/test_publish_dry_run.py tests/test_product_schema.py -v
git add uploader/csp_uploader/endpoints.py uploader/csp_uploader/publish.py docs/capture-notes.md tests/test_publish_dry_run.py ae_cli.py pyproject.toml
git commit -m "feat: add publish dry-run orchestration and empty endpoints gate"
```

---

### Task 7: Live publish via APIRequestContext + optional `api`

**Files:**
- Modify: `uploader/csp_uploader/publish.py`
- Create: `uploader/csp_uploader/http_api.py` (shared request helper)
- Modify: `ae_cli.py` for `api` command
- Create: `tests/test_exit_codes.py`

**Interfaces:**
- Produces:
  - `async def with_api_context(account, headless, fn)` context manager-ish helper
  - `async def raw_api(account, method, url, body, headless) -> tuple[int, str]`
  - Live publish: for each step `context.request.fetch`; merge `extras`; interval sleep; on success `storage_state`; on failure return 3/4 **without** writing cookie

- [ ] **Step 1: Implement `http_api.py`**

```python
async def open_request_context(account: str, headless: bool):
    path = cookie_path(account)
    pw = await async_playwright().start()
    browser = await pw.chromium.launch(headless=headless, channel="chromium")
    context = await browser.new_context(storage_state=str(path))
    return pw, browser, context

async def close_all(pw, browser, context, cookie_out: Path | None):
    if cookie_out is not None:
        await context.storage_state(path=str(cookie_out))
    await browser.close()
    await pw.stop()
```

- [ ] **Step 2: `api` CLI maps status → exit codes**

- HTTP 401/403 → exit 2  
- HTTP 5xx → exit 4  
- business failure if JSON has obvious fail flag → exit 3 (keep heuristic minimal)  
- else exit 0  

- [ ] **Step 3: Live publish loop**

Pseudo:

```python
pw, browser, context = await open_request_context(account, headless)
ctx_state: dict = {"product": product, "ids": {}}
try:
    for step in ENDPOINT_STEPS:
        await asyncio.sleep(conf.PUBLISH_REQUEST_INTERVAL_SEC)
        body = step.build_payload(product, ctx_state) if step.build_payload else None
        resp = await context.request.fetch(
            step.url_template,
            method=step.method,
            data=body if isinstance(body, str) else None,
            headers=step.headers or {},
            # if body is dict, use `data=json.dumps` + content-type json
        )
        text = await resp.text()
        if resp.status in (401, 403):
            return 2
        if resp.status >= 500:
            return 4
        # optional: parse ids into ctx_state["ids"]
    await context.storage_state(path=str(cookie_path(account)))
    return 0
finally:
    await browser.close()
    await pw.stop()
```

Important: on non-zero return paths, **skip** `storage_state` write.

- [ ] **Step 4: Unit test exit mapping without network**

```python
# tests/test_exit_codes.py
from uploader.csp_uploader.publish import map_http_to_exit

def test_map():
    assert map_http_to_exit(401) == 2
    assert map_http_to_exit(500) == 4
    assert map_http_to_exit(200) == 0
```

- [ ] **Step 5: Commit**

```powershell
git add uploader/csp_uploader/http_api.py uploader/csp_uploader/publish.py ae_cli.py tests/test_exit_codes.py
git commit -m "feat: APIRequestContext publish/api with cookie write only on success"
```

---

### Task 8: Hard gate M4 — human capture + fill endpoints (operator + engineer)

**Files:**
- Modify: `uploader/csp_uploader/endpoints.py` (real steps)
- Modify: `docs/capture-notes.md`
- Optionally add: `captures/.gitkeep` only (not real sessions)

**Interfaces:**
- Consumes: latest `captures/<session>/filtered.json`
- Produces: non-empty `ENDPOINT_STEPS` matching ordered create/upload/submit chain

- [ ] **Step 1: Operator runs capture of one full hand-publish**

```powershell
.\.venv\Scripts\ae-csp.exe capture --account demo --har
```

- [ ] **Step 2: Engineer extracts minimal chain**

From `filtered.json`, document in `capture-notes.md`:

1. upload image URL(s) if any  
2. create/save draft  
3. sku/price  
4. submit/publish  

- [ ] **Step 3: Implement `build_payload` closures in `endpoints.py`**

Map `product.json` fields + `extras` into each body. Keep headers CSRF names copied from filtered.

- [ ] **Step 4: Dry-run then live publish**

```powershell
.\.venv\Scripts\ae-csp.exe publish --account demo --from products/demo.json --dry-run
.\.venv\Scripts\ae-csp.exe publish --account demo --from products/demo.json
```

Expected: CSP backend shows new product (draft or submitted per notes).

- [ ] **Step 5: Commit endpoints + notes only (no cookies/captures secrets)**

```powershell
git add uploader/csp_uploader/endpoints.py docs/capture-notes.md
git commit -m "feat: lock CSP publish endpoints from capture session"
```

**Gate rule:** Do not mark MVP done until Step 4 live publish succeeds.

---

### Task 9: README + CLI.md polish + workspace plan/devlog sync

**Files:**
- Modify: `aliexpress-csp-auto-upload/README.md`
- Modify: `aliexpress-csp-auto-upload/docs/CLI.md`
- Modify: `D:\dev\workspace\开发文档.md` (entry that plan is ready / MVP scaffold status)
- This plan already at `docs/superpowers/plans/2026-07-14-aliexpress-csp-auto-upload.md`

- [ ] **Step 1: Document install, login URL override, exit codes, M4 gate**
- [ ] **Step 2: Run full unit suite**

```powershell
.\.venv\Scripts\pytest.exe -v
```

- [ ] **Step 3: Commit docs in project repo + commit plan in workspace `agenta` repo**

Workspace:

```powershell
Set-Location D:\dev\workspace
git add docs/superpowers/plans/2026-07-14-aliexpress-csp-auto-upload.md 开发文档.md
git commit -m "docs(aliexpress): add CSP auto-upload implementation plan"
git push origin HEAD
```

---

## Spec coverage checklist (self-review)

| Spec requirement | Task |
|------------------|------|
| Independent git repo + `ae-csp` | Task 1 |
| Account URL override | Task 2 |
| product.json schema | Task 3 |
| login / check headed gate | Task 4 |
| capture + full filtered body | Task 5 |
| dry-run + empty endpoints refuse live | Task 6 |
| APIRequestContext + RefreshCookie only on success | Task 7 |
| optional `api` | Task 7 |
| M4 human gate + real endpoints | Task 8 |
| docs / no 妙手 | Tasks 1, 8, 9 |
| Excel / Commander | **Out of scope** (explicit) |

## Placeholder scan

No TBD steps; M4 depends on live capture data — Task 8 steps say exactly what operator/engineer do.

## Type consistency

- `AccountConfig.login_url` / `publish_url` used by auth + capture + publish  
- `load_product` → `ProductValidationError` → exit 3  
- `ENDPOINT_STEPS: list[EndpointStep]` shared by dry-run and live  
- Exit constants `0/2/3/4` in `ae_cli.py` match publish return ints  

---

## Execution handoff

Plan saved to `docs/superpowers/plans/2026-07-14-aliexpress-csp-auto-upload.md`.

**Two execution options:**

1. **Subagent-Driven (recommended)** — fresh subagent per task, review between tasks  
2. **Inline Execution** — execute tasks in this session with executing-plans checkpoints  

Which approach?
