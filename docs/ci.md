# CI and Release Conventions

## Workflow Ownership

- `.github/workflows/quality-gates.yml`: pull request and main-branch quality checks.
- `.github/workflows/docker-image.yml`: release pipeline for Docker images and binary assets.

## Deterministic Package Install

- All CI and repo scripts must use `npm ci`.
- `npm install` is reserved for explicit dependency update workflows only.

## Step Size Rule

- If a workflow `run` block grows beyond ~10 lines, move logic into versioned repo scripts under `scripts/`.
- Keep workflow YAML focused on orchestration; keep business logic in scripts.

## Shared Baselines

- Node runtime baseline: `20`
- Go runtime baseline: `1.21`
- Coverage baselines are defined in `CONTRIBUTING.md` and enforced by `quality-gates.yml`.

