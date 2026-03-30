# Contributing to SkyLink

本文件定义项目的最小维护约定，确保本地与 CI 质量门一致。

## 开发约定

- 前端包管理器统一使用 `npm`。
- 变更优先遵循 SRP：页面/handler 保持编排职责，复杂逻辑下沉到 composable/service。
- 类型优先：避免 `any`，新增接口响应优先补充类型定义。

## 本地质量门（提交前）

### 后端

```bash
go test ./...
```

### 前端

```bash
cd web
npm ci
npm run type-check
npm run test
npm run test:coverage
npm run build
```

## 覆盖率基线

- Go 覆盖率阈值：`35%`
- Frontend statement 覆盖率阈值：`35%`

说明：当前阈值为第一阶段基线，后续按迭代逐步提高。

## 测试策略

- 工具层：保留纯函数单测（`web/src/utils/*.test.ts`）。
- 组件层：优先为高变更区域增加“事件透传 + 关键条件渲染”测试。
- Composable：覆盖关键状态流（加载成功、失败分支、关键副作用）。
- 后端：核心域（proxy/ddns/cloudflare/easytier）至少有包级 `_test.go`。

## CI 对齐

- 前端安装仅使用 `npm ci`，不得在 workflow 中混用 `npm install`。
- 质量门见 `.github/workflows/quality-gates.yml`，发布流程见 `.github/workflows/docker-image.yml`。
- CI 维护约定（职责边界、脚本抽取规则）见 `docs/ci.md`。
