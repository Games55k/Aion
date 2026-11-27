# 开发、测试与发布指南

Aion 使用 [MyGo](https://github.com/egoist/mygo) 直接构建 macOS / Windows 原生窗口应用，不打包 Chromium / WebView，也不启动本地 HTTP 服务。

## 环境与常用命令

需要 Go 1.27.3 或更新版本。

| 命令 | 用途 |
|---|---|
| `go run .` | 在本地启动 Aion 原生桌面端 |
| `go run . -data-dir /tmp/aion-dev` | 使用独立测试数据目录启动（不影响正式数据） |
| `go run ./cmd/promo` | 基于 MyGo Render 插件在 GPU 上渲染 1080p 60fps 项目介绍视频（`aion-promo.mp4`，支持 `-preview` 窗口预览与 `-stills` 关键帧导出） |
| `go run . mcp [-data-dir <路径>]` | 启动本地 stdio MCP Server（桌面端在线时经本地 Socket 实时联动，离线时按次直连） |
| `go run . summary [-json]` | 通过本地 CLI 查看工作区总览（还支持 `project` / `task` / `job` / `trash` 子命令，可用 `go run . help` 查看） |
| `go tool mygo dev` | 开发模式运行 |
| `go tool mygo build` | 构建当前平台原生安装包（macOS 输出 `.app` 与 `.dmg`，Universal 可加 `-platform darwin/universal`；Windows 输出 `Aion.exe` 与 NSIS 安装包 `aion-setup-<版本>-amd64.exe`） |
| `go tool mygo build -platform windows/amd64 -skip-installer` | 在 macOS/Linux 上无 CGO 快速交叉编译 Windows `Aion.exe` 验证 |
| `go test ./...` | 运行入口、后端与无头 UI 交互测试 |
| `go test -race ./...` | 运行竞态检测测试 |
| `go vet ./...` | Go 静态分析检查 |
| `go tool mygo vet .` | MyGo 布局与状态规则静态检查 |
| `git config core.hooksPath .githooks` | 启用仓库自带的本地 Git Hooks（提交前检查 `gofmt`/`go vet`/`mygo vet`，推送前运行 `go test ./...`） |

### 本地代码审查与 CI 防线

日常直接向 `main` 提交（或提交 PR）时，可按以下多层检查把关：

1. **本地 Git Hooks（`.githooks/`）**：执行一次 `git config core.hooksPath .githooks` 启用；`.githooks/pre-commit` 在暂存 `.go` 文件时自动检查 `gofmt`、`go vet ./...` 与 `go tool mygo vet .`（纯文档改动直接跳过），`.githooks/pre-push` 在推送含代码变更的提交前运行 `go test ./...`。
2. **变更前 AI 审查契约（`REVIEW.md`）**：明确了 Aion 的高/中/低风险路径分级、真实故障红线、反噪音原则与最多 3 轮的收敛上限；改动中高风险目录时，在提交前使用独立上下文对照 `REVIEW.md` 审查 `git diff`（高风险目录补充安全审查）。
3. **持续集成（`.github/workflows/ci.yml`）**：每次推送到 `main` 或发起 PR 时，自动在 `macos-latest` 运行 `gofmt` 校验、`go vet`、`mygo vet`、全量 `go test ./...` 与核心并发包（`.`、`./backend/...`、`./cli/...`、`./ipc/...`、`./mcp/...`）的 `go test -race`，并在 `windows-latest` 验证全量构建、静态检查与核心测试。
4. **PR 模板与代码所有者**：`.github/PULL_REQUEST_TEMPLATE.md` 要求说明风险级别、验证项与回滚方式，`.github/CODEOWNERS` 标记高风险目录责任人。

### 测试时钟与时间戳约定

后端测试可通过 `backend.Options.Now` 注入项目、任务、求职和备份使用的时钟，默认使用 `time.Now`。项目和任务以 UTC 存储时间戳，同一次任务移动或排序使用同一时间更新所有位置发生变化的任务。

## 无头渲染与视觉快照

```bash
# 为指定数据目录生成无头渲染快照 PNG（1360×860）
go run . -data-dir /tmp/aion-snap -snapshot /tmp/aion-snapshot.png

# 渲染带示例数据的看板视觉快照 PNG（1440×900，用于视觉回归检查；发布前用于更新 assets/screenshot.png）
AION_SAVE_SNAPSHOT=/tmp/aion-snapshot.png go test -count=1 ./frontend -run TestVisualSnapshotRender

# 渲染设置页视觉快照 PNG（1440×1280，可与上一条同时设置）
AION_SAVE_SETTINGS_SNAPSHOT=/tmp/aion-settings.png go test -count=1 ./frontend -run TestVisualSnapshotRender

# 渲染跨模块聚焦总览与全局命令面板视觉快照 PNG（1440×900）
AION_SAVE_FOCUS_SNAPSHOT=/tmp/aion-focus.png \
AION_SAVE_PALETTE_SNAPSHOT=/tmp/aion-palette.png \
go test -count=1 ./frontend -run TestVisualSnapshotRender

# 渲染应用内更新弹窗视觉快照 PNG（1440×900）
AION_SAVE_UPDATE_MODAL_SNAPSHOT=/tmp/aion-update-modal.png go test -count=1 ./frontend -run TestVisualSnapshotRender

# 渲染求职投递列表、窄窗口表格与投递详情视觉快照 PNG（可与其它快照环境变量组合使用）
AION_SAVE_JOBS_SNAPSHOT=/tmp/aion-jobs.png \
AION_SAVE_JOBS_NARROW_SNAPSHOT=/tmp/aion-jobs-narrow.png \
AION_SAVE_JOB_DETAIL_SNAPSHOT=/tmp/aion-job-detail.png \
go test -count=1 ./frontend -run TestVisualSnapshotRender
```

`-snapshot` 的输出路径必须位于当前数据目录之外（包括经符号链接访问的路径）。快照 PNG 完整写入后才会原子替换已有的普通文件，避免误覆盖数据库、附件或备份。

## 真实窗口自动化冒烟（开发专用）

`automation_e2e.go` 仅在 `-tags e2e` 构建时引入 `github.com/egoist/mygo/plugins/automation`，默认构建与正式发布构建（`mygo_noinspector`）均不启动自动化服务。可使用独立临时目录运行真实窗口操作与截图：

```bash
# 在仓库目录安装与当前 go.mod 一致的 mygo-automation 客户端
go install github.com/egoist/mygo/plugins/automation/cmd/mygo-automation

# 在隔离临时目录构建并启动 e2e 实例
aion_e2e_dir=$(mktemp -d /tmp/aion-e2e.XXXXXX)
go build -tags e2e -o "$aion_e2e_dir/Aion" .
MYGO_AUTOMATION="$aion_e2e_dir/app.sock" \
  "$aion_e2e_dir/Aion" -data-dir "$aion_e2e_dir/data"

# 另一个终端中（将上面的输出路径赋给 aion_e2e_dir）连接、检查控件与截图
mygo-automation open --socket "$aion_e2e_dir/app.sock"
mygo-automation snapshot -i
mygo-automation screenshot "$aion_e2e_dir/window.png"
mygo-automation close
```

`mygo-automation close` 仅结束客户端会话，应用正常退出（如菜单 `Aion` → `退出Aion`）后会自动清理 socket，再清理创建的临时目录即可。

## 项目结构

```text
Aion/
├── main.go                                 # 原生窗口宿主入口、CLI/MCP 子命令分发、应用菜单与系统能力桥接
├── updater_controller.go                   # 宿主层静默定期检查更新、下载安装进度与重启调度控制器
├── automation_e2e.go                       # `-tags e2e` 构建专用的 MyGo Automation 插件入口
├── mygo.json                               # MyGo 应用配置、打包元数据与更新源配置
├── AGENTS.md                               # AI 开发、测试、文档同步与提交规范
├── REVIEW.md                               # 面向 AI/人工的代码审查契约（风险分级、故障红线、反噪音与收敛规则）
├── CHANGELOG.md                            # 版本更新说明（发版说明与客户端更新窗口共用）
├── .githooks/                              # 可选本地 Git Hooks（pre-commit 格式化与静态检查、pre-push 测试门禁）
├── .github/
│   ├── CODEOWNERS                          # 高风险目录与策略文件责任人配置
│   ├── PULL_REQUEST_TEMPLATE.md            # PR 模板（变更摘要、风险分级、测试清单与回滚说明）
│   ├── ISSUE_TEMPLATE/                     # GitHub Issue 表单模板与空白 Issue 禁用配置
│   └── workflows/                          # `ci.yml`（push main / PR 双平台 CI）与 `release.yml`（标签发版）
├── docs/                                   # 详细文档（交互行为、数据与备份、开发与发布）
├── skills/aion-workbench/SKILL.md          # 面向 Antigravity / Codex 的 Aion 工作流 Skill
├── ipc/                                    # 本地 IPC Socket 服务端、按次降级路由、局部更新处理与跨平台路径校验
├── mcp/                                    # 标准 stdio JSON-RPC 2.0 MCP Server（仅含安全读写与软删除工具集）
├── cli/                                    # 复用本地 IPC 路由的轻量 CLI（summary / project / task / job / trash）
├── cmd/promo/                              # 基于 MyGo Render 插件的 1080p 60fps 项目介绍视频生成器
├── frontend/                               # 原生界面层：工作台、顶栏导航、今天总览、命令面板、看板、求职、回收站、设置、异步保存与无头 UI 测试
├── backend/                                # 后端总入口、模块注册、全局回收站分发、设置与自动备份调度
│   ├── model/                              # 领域模型（项目/任务、求职投递与事件）、状态枚举与哨兵错误定义
│   ├── store/                              # SQLite 事务、Schema v3 迁移、跨模块回收站快照与读写一致性锁
│   ├── projects/                           # 项目、任务、三列移动/排序、跨项目搜索筛选与回收站服务
│   ├── jobapplications/                    # 求职投递记录、事件去重、筛选排序统计、补录与回收站服务
│   ├── attachments/                        # 本地受管文件暂存导入、导出、路径解析与孤儿文件 GC
│   └── backup/                             # 全量 ZIP 导出、Schema v3 完整性校验、安全备份与自动轮转
└── assets/                                 # 品牌图标源文件与无头渲染预览截图（screenshot.png）
```

## 发布工作流

GitHub Actions 工作流 `.github/workflows/release.yml` 在推送 `v*` 标签时先后构建 macOS Universal（`.dmg`）与 Windows amd64（`aion-setup-<版本>-amd64.exe`）安装包并上传至同一个草稿 Release：

- 两个平台作业在上传前都会先查询 GitHub Release；只有查询成功且该版本尚未创建或仍为草稿时，工作流才会继续。
- `release-macos` 完成版本与 `CHANGELOG.md` 校验、完整测试集、签名/公证及草稿上传后，`release-windows` 在 `windows-latest` 上执行构建、静态检查、核心测试，并通过 `go tool mygo build -platform windows/amd64 -upload` 自动打包 NSIS `.exe` 安装包、签名更新归档与合并后的 `SHA256SUMS`。
- Apple 公证优先使用完整的 API Key 三项凭据，否则使用已通过校验的 Apple ID 凭据；临时签名钥匙串从创建前即登记清理路径，后续导入或公证失败时也会清理。
