# 开发、测试与发布指南

Aion 使用 [MyGo](https://github.com/egoist/mygo) 直接构建 macOS 原生窗口应用，不打包 Chromium / WebView，也不启动本地 HTTP 服务。

## 环境与常用命令

需要 Go 1.27.3 或更新版本。

| 命令 | 用途 |
|---|---|
| `go run .` | 在本地启动 Aion 原生桌面端 |
| `go run . -data-dir /tmp/aion-dev` | 使用独立测试数据目录启动（不影响正式数据） |
| `go run . mcp [-data-dir <路径>]` | 启动本地 stdio MCP Server（桌面端在线时经 Unix Socket 实时联动，离线时按次直连） |
| `go tool mygo dev` | 开发模式运行 |
| `go tool mygo build` | 构建 macOS 原生 `.app` 与 `.dmg`（默认输出到 `build/darwin-arm64/`；Universal 构建可加 `-platform darwin/universal`） |
| `go test ./...` | 运行入口、后端与无头 UI 交互测试 |
| `go test -race ./...` | 运行竞态检测测试 |
| `go vet ./...` | Go 静态分析检查 |
| `go tool mygo vet .` | MyGo 布局与状态规则静态检查 |

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

# 渲染跨模块聚焦总览视觉快照 PNG（1440×900）
AION_SAVE_FOCUS_SNAPSHOT=/tmp/aion-focus.png go test -count=1 ./frontend -run TestVisualSnapshotRender

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
├── main.go                                 # 原生窗口宿主入口、`mcp` 子命令分发、应用菜单与系统能力桥接
├── updater_controller.go                   # 宿主层静默定期检查更新、下载安装进度与重启调度控制器
├── updater_controller_test.go              # 静默检查、跳过版本、单飞合并、进度取消与旧状态迁移测试
├── automation_e2e.go                       # e2e 构建标签专用的 MyGo Automation 插件入口
├── main_test.go                            # 快照输出数据保护、原子替换与 `mcp` 子命令往返回归测试
├── ipc/                                    # 本地 Unix Socket IPC 服务端、按次降级路由、局部更新处理与安全路径校验
├── mcp/                                    # 标准 stdio JSON-RPC 2.0 MCP Server 与仅含安全读写/软删的工具集定义
├── mygo.json                               # MyGo 应用配置、打包元数据与更新源配置
├── CHANGELOG.md                            # 版本更新说明（发版说明与客户端更新窗口共用）
├── docs/                                   # 详细文档与实现计划（交互行为、数据与备份、开发发布、plans/）
├── .github/
│   ├── workflows/release.yml               # 标签触发的 macOS Universal 构建与草稿 Release 工作流
│   └── ISSUE_TEMPLATE/                     # GitHub Issue 表单模板与空白 Issue 禁用配置
├── frontend/                               # 原生界面层：工作台状态、模块切换侧栏、项目看板、求职投递、回收站、设置与附件预览
│   ├── ui_*.go / icons.go                  # 基于 MyGo 原生 UI 封装的设计令牌、控件、弹窗、Markdown 与 Lucide SVG 图标
│   ├── sidebar.go                          # 展开/折叠双形态模块切换器、更新图标入口与各模块导航侧栏
│   ├── update_modal.go                     # 侧边栏更新图标按钮与应用内软件更新弹窗
│   ├── task_save.go                        # 任务草稿串行保存、合并输入与保存后的导航
│   ├── job_save.go                         # 求职投递详情草稿校验、串行保存与撤销恢复
│   ├── job_controller.go                   # 求职投递筛选、分页、流转、备注、补录与回收站动作控制
│   ├── job_applications_view.go            # 求职投递主视图、统计摘要、响应式表格与新增弹窗
│   ├── job_application_detail_view.go      # 求职投递详情弹窗、流程时间线、备注/补录表单与结束流程弹窗
│   ├── workbench_load.go                   # 后台加载与过期结果保护
│   ├── focus_view.go / modules_visibility.go # 跨模块聚焦总览视图与工作区模块显隐偏好控制
│   ├── external_sync.go / mcp_config.go    # 外部 IPC/MCP 变更实时刷新（草稿保护）与 Antigravity/Codex 配置格式化
│   ├── record_write.go / settings_write.go # 后台记录操作与设置写入
│   ├── async_persistence_test.go           # 长持锁输入/导航、保存失败与并发回调回归测试
│   ├── async_record_write_test.go          # Store 持锁时的看板、表单、回收站和设置写入回归测试
│   ├── external_sync_test.go               # 外部变更实时刷新、未保存草稿保护与设置页 MCP 配置复制测试
│   ├── focus_modules_test.go               # 工作区模块显隐持久化与跨模块聚焦总览跳转测试
│   ├── workbench_test.go                   # 基于 ui.Tester 的端到端无头 UI 交互测试
│   └── attachment_preview_test.go          # 图片缩略图解码、方向校正与有界缓存测试
├── backend/                                # 后端总入口、模块注册、全局回收站分发、设置与自动备份调度生命周期
│   ├── integration_test.go                 # 跨目录备份迁移、跨模块回收站分发与彻底删除 GC 集成测试
│   ├── model/                              # 领域模型（项目/任务、求职投递与事件）、状态枚举与哨兵错误定义
│   ├── store/                              # SQLite 事务、本地 Schema 迁移、跨模块回收站快照查询与读写一致性锁
│   ├── projects/                           # 项目、任务、三列移动/排序、搜索筛选与回收站服务
│   ├── jobapplications/                    # 求职投递记录、事件去重、筛选排序统计、补录与回收站服务
│   ├── attachments/                        # 本地受管文件导入、导出、路径解析与孤儿文件 GC
│   └── backup/                             # 全量 ZIP 导出、v1/v2 完整性校验、安全备份与自动轮转
└── assets/
    ├── icon/                               # Aion 品牌图标源文件（SVG）、打包用 PNG 与 Go 嵌入封装（icon.go）
    └── screenshot.png                      # 无头渲染的界面预览截图（1440×900）
```

## 发布工作流

GitHub Actions 工作流 `.github/workflows/release.yml` 在推送 `v*` 标签时构建 macOS Universal 包并上传至草稿 Release：

- 标签构建会先查询 GitHub Release；只有查询成功且该版本尚未创建或仍为草稿时，工作流才会继续。
- Apple 公证优先使用完整的 API Key 三项凭据，否则使用已通过校验的 Apple ID 凭据；临时签名钥匙串从创建前即登记清理路径，后续导入或公证失败时也会清理。
