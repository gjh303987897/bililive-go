# AI 开发指南

本文档为在此项目中工作的 AI 助手（如 GitHub Copilot、Gemini、Claude、Codex、Antigravity 等）提供指导。

> **注意**：`AGENTS.md` 是 AI 指示的唯一源文件。任何 AI 指示变更都必须先修改 `AGENTS.md`，禁止直接修改下游同步文件。修改后请运行 `go run ./build.go sync-agents` 同步到其他位置：
> - `.github/copilot-instructions.md` (GitHub Copilot)
> - `.agent/rules/gemini-guide.md` (Gemini CLI)
> - `.gemini/GEMINI.md` (Antigravity)

## 语言要求

永远使用中文进行交流，包括代码注释和 AI 生成的 markdown 文本。

## 核心规则

1. **编译验证**：修改代码后**必须**验证编译通过，绝不能跳过
   - **仅修改后端 Go 代码**：运行 `go run ./build.go dev`
   - **修改了前端代码**：先运行 `npm --prefix src/webapp run build`，再运行 `go run ./build.go dev`（`tsc --noEmit` 不够，react-scripts 的 ESLint 规则更严格，如 `no-unused-vars` 会导致编译失败）
   - **前后端都改了**：使用与“修改了前端代码”相同的命令
2. **不要使用** `go build ./...`，必须使用 `build.go` 构建入口
3. **包管理器统一使用 npm**：不要新增或更新 Yarn 锁文件
4. **提交前检查**：确保 `npm --prefix src/webapp run build`、`go run ./build.go lint`、`go run ./build.go test` 全部通过
5. **禁止擅自提交**：不要主动执行 `git commit`、`git push` 等 git 操作，除非用户明确要求
6. **Git 提交信息使用英文**：`git commit` 的标题和正文必须使用英文

## 详细指南（Skills）

更多详细信息请查阅 `.agent/skills/` 目录下的相关指南：

| Skill | 说明 |
|-------|------|
| `build` | 编译命令、build tags、代码检查 |
| `config-modification` | 配置修改同步、层级配置系统 |
| `test-e2e` | Playwright E2E 测试 |
| `version-switching` | 不停机版本切换设计规范（Docker/独立运行） |

## 快速参考

```bash
# 安装依赖
npm ci
npm --prefix src/webapp ci

# 编译后端
go run ./build.go dev

# 编译前端 + 后端
npm --prefix src/webapp run build
go run ./build.go dev

# 代码检查
go run ./build.go lint

# 单元测试
go run ./build.go test

# E2E 测试
npx playwright test

# 同步 AI 指示文件
go run ./build.go sync-agents
```

