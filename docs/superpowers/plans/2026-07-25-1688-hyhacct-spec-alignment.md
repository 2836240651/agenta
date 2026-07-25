# 1688 Hyhacct Spec Alignment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:executing-plans` to implement this documentation plan task-by-task.

**Goal:** Make `hyhacct` the sole current image-optimization upstream in the active 1688 Listing Wizard v2 specification while preserving historical smoke evidence.

**Architecture:** Documentation-only change. Align active v2 contracts, requirements, diagrams, and test cases with `hyhacct`; preserve `.superpowers/sdd/` and prior dated records. Archive the active development log before recording this decision because it exceeds 100 KB.

**Tech Stack:** Markdown, PowerShell, Git.

## Global Constraints

- Image flow: `Workflow images → Server → AiEditImage → hyhacct`.
- “Use original image” must not call `hyhacct`.
- Keep the ban on direct `produce_images` calls.
- Do not alter `.superpowers/sdd/`, prior dated history, runtime code, or runtime configuration.

### Task 1: Archive development history and record the decision

- [x] Verify `开发文档.md` exceeds 100 KB, move it to `docs/handover/开发文档-归档-20260725.md`, and create a new active log with archive index, five latest summaries, and the supplier-alignment record.

### Task 2: Align current 1688 v2 documentation

- [x] Replace active `grsai` references in the v2 specification package and active v2 design with `hyhacct`.
- [x] Update acceptance wording to test Server-mediated `hyhacct` generation and no upstream request for original-image reuse.

### Task 3: Verify and publish

- [x] Confirm no active v2 `grsai` reference remains; confirm `hyhacct`, `produce_images`, and original-image constraints are searchable.
- [x] Inspect only task files, commit `docs: align 1688 image workflow with hyhacct`, rebase, and push without force.
