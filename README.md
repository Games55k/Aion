<p align="center">
  <img src="assets/icon/aion-icon.svg" width="64" height="64" alt="Aion logo" />
</p>

<h1 align="center">Aion</h1>

<p align="center">
  轻量、纯原生的 macOS 本地项目与求职投递管理工具。
</p>

<p align="center">
  <img src="https://img.shields.io/badge/macOS-000000?style=flat&logo=apple&logoColor=white" alt="macOS" />
  <img src="https://img.shields.io/badge/Go-1.27.3+-4B5563?style=flat&logo=go&logoColor=00ADD8&labelColor=18181B" alt="Go 1.27.3+" />
  <a href="https://mygo.run"><img src="https://img.shields.io/badge/MyGo-Native_UI-2563EB?style=flat&labelColor=18181B" alt="MyGo Native UI" /></a>
</p>

<p align="center">
  <img src="assets/screenshot.png" alt="按状态分组卡片的 Aion 三列看板与任务详情弹窗" />
</p>

Aion（读作 `EYE-on`，源自古希腊语 αἰών，意为“生命、时代与永恒”）帮你把日常项目、任务拆解、本地资料与求职投递记录收敛在同一处。所有数据 100% 存储在你的 Mac 本地，无需注册账号，不依赖任何云服务，断网环境下依然拥有毫秒级响应。

## 功能

- **模块化工作台**：侧边栏顶部（展开及折叠态）可在**项目**与**求职**两个独立工作区之间一键切换，各自保留最近页面、搜索与筛选上下文，并共享全局**回收站**与**设置 / 备份**
- **三列看板**：围绕“计划 / 进行中 / 已完成”管理项目进度，支持鼠标拖拽跨列流转、列内排序、右键菜单、点击卡片优先级标签快速切换，以及 `⌘K` 搜索、优先级筛选与多标签交集筛选
- **多项目工作区**：统一管理多个活跃项目与已归档项目，左侧导航栏支持拖拽调节宽度与折叠为紧凑图标栏
- **任务详情弹窗**：点击任务卡片弹出居中的任务详情编辑窗口（不挤压看板列布局），支持标题、状态、优先级、截止日期、分类标签编辑与自动保存状态反馈
- **求职投递记录**：提供“全部投递 / 进行中 / 已获 Offer / 已结束”四个固定视角与全局统计卡片，支持表格横向滚动与行内阶段/状态快速流转、公司/岗位 `⌘K` 搜索与多条件筛选分页、`http/https` 投递链接一键复制与系统浏览器打开，以及同名进行中投递非阻塞提醒
- **投递详情与时间线**：详情弹窗自动串行保存字段草稿，支持“结束流程（含 Offer 结果阶段确认）/ 重新开启”、追加时间线备注以及只写历史不覆盖当前进度的阶段事件补录
- **Markdown 编辑与预览**：内置“编辑 / 预览 / 分栏”三种模式切换与快捷语法工具栏，自动持久化原始 Markdown 文本
- **子任务清单**：支持子任务拆分、勾选、重命名、排序与完成度进度条（勾选全部子任务不会强制将父任务标为已完成）
- **本地受管附件**：支持点击选择或直接从 Finder 拖入文件/图片，提供图片缩略图与大图预览、系统默认应用打开、Finder 定位与另存为导出。另存为仅接受常规文件作为来源，并在数据锁外复制和校验内容，即使复制期间清理了原附件，也能完成已开始的导出
- **回收站与全量备份**：删除项目、任务与求职投递默认进入统一回收站（支持按类型筛选、带完整时间线恢复与彻底清除）；内置周期性自动备份轮转与手动全量 `.zip` 导出/恢复

任务与投递详情输入先保留在本地草稿中，后台按顺序自动保存；连续输入只保留最新的待保存版本。保存期间会显示“正在保存”，切换模块、切换项目/记录或关闭详情会在保存成功后执行最后一次请求。保存失败或输入无效时保留草稿；保存尚未完成时，请等待后再退出应用或恢复备份。投递阶段变更、时间线写入和详情保存按输入顺序执行，写入完成前会阻止退出和恢复备份。提交备注后可以继续输入下一条。页面、项目、投递列表与详情在后台加载，较早的加载结果不会覆盖新页面或新输入。

后台保存期间点击“放弃修改”后，可以继续编辑。若前一笔保存失败，编辑器会恢复受影响的字段，并保留你随后提交的修改和正在输入的文字，包括对子任务的改名和投递备注中的换行。调整子任务顺序时，后续保存优先保留你在放弃后改变的相对顺序，包括新增子任务的位置，并尽量保留数据库里原有的顺序。

看板移动、排序、修改优先级、创建记录、回收站恢复和保存设置在后台执行，等待数据库期间仍可继续输入或切换页面。记录写入完成后可以再次提交，等待期间输入的新内容会保留。连续保存同一项设置时，会保留最新的待保存值。请等待写入完成后再关闭窗口或恢复备份。

## 快速开始

### 直接使用（推荐）

前往 [GitHub Releases](https://github.com/Games55k/Aion/releases) 下载最新版本的 `.dmg` 安装包，将 `Aion.app` 拖入「应用程序」目录即可使用（支持在设置页检查并安装更新）。

### 从源码运行（开发者）

如需自行构建或二次开发，请准备 Go 1.27.3 或更新版本并克隆仓库：

```bash
git clone https://github.com/Games55k/Aion.git
cd Aion
go run .
```

如需使用独立的测试数据目录启动（不影响正式数据）：

```bash
go run . -data-dir /tmp/aion-dev
```

## 构建与打包

Aion 使用 `mygo` 直接构建 macOS 原生窗口应用，不打包 Chromium / WebView，也不启动本地 HTTP 服务。

```bash
# 开发模式运行
go tool mygo dev

# 构建 macOS 原生 .app 与 .dmg 安装包（默认输出到 build/darwin-arm64/；Universal 构建可用 -platform darwin/universal）
go tool mygo build

# 为指定数据目录生成无头渲染快照 PNG（1360×860）
go run . -data-dir /tmp/aion-snap -snapshot /tmp/aion-snapshot.png

# 渲染带示例数据的看板视觉快照 PNG（1440×900，用于视觉回归检查）
AION_SAVE_SNAPSHOT=/tmp/aion-snapshot.png go test -count=1 ./frontend -run TestVisualSnapshotRender

# 渲染设置页视觉快照 PNG（1440×1280，可与上一条同时设置）
AION_SAVE_SETTINGS_SNAPSHOT=/tmp/aion-settings.png go test -count=1 ./frontend -run TestVisualSnapshotRender

# 渲染求职投递列表、窄窗口表格与投递详情视觉快照 PNG（可与其它快照环境变量组合使用）
AION_SAVE_JOBS_SNAPSHOT=/tmp/aion-jobs.png \
AION_SAVE_JOBS_NARROW_SNAPSHOT=/tmp/aion-jobs-narrow.png \
AION_SAVE_JOB_DETAIL_SNAPSHOT=/tmp/aion-job-detail.png \
go test -count=1 ./frontend -run TestVisualSnapshotRender
```

`-snapshot` 的输出路径必须位于当前数据目录之外（包括经符号链接访问的路径）。快照 PNG 完整写入后才会原子替换已有的普通文件，避免误覆盖数据库、附件或备份。

## 数据与备份

默认数据目录位于 macOS 标准应用支持路径：

```text
~/Library/Application Support/Aion/
├── aion.sqlite     # SQLite 主数据库（Schema v3：项目、任务、子任务、求职投递与时间线、回收站、设置）
├── attachments/    # 受管本地附件存储
└── backups/        # 自动快照（auto-*.zip）与恢复前安全备份（safety-*.zip，按生成顺序保留最近 5 份，另保护本次恢复所用的源文件）
```

在左侧导航底部的**设置**（全局设置与数据管理，也可按 `⌘,` 打开）中可以管理数据生命周期：

- **自动备份**：应用启动及运行期间按周期（默认每天一次、保留最近 7 份）自动生成一致性快照并轮转清理旧备份
- **手动导出**：随时将当前 SQLite 快照、全部受管附件及 `manifest.json` 校验清单打包导出为 `.zip`
- **完整恢复**：仅接受使用当前数据库结构（Schema v3）的备份包。恢复前会校验清单、文件 SHA-256 摘要与数据库结构。已结束投递即使在结束后修正过阶段，也会同时保留修正后的阶段和原有结果。备份版本不匹配、文件损坏或校验失败时，恢复会中止。正式替换前会自动将当前数据打包为一份 `safety-*.zip` 安全备份以便回滚。若生成安全备份后恢复失败，设置页会显示错误原因和该备份的路径

首次启动新版时，本地数据库会自动升级到 Schema v3，并删除投递的跟进日期列。其他投递字段和时间线会保留。Schema v1、v2 的旧备份无法在新版中恢复，升级后可重新导出完整备份。

自动备份与手动导出只在生成 SQLite 快照、固定附件引用期间占用数据锁，压缩、校验和自动备份轮转在锁外完成。临时附件引用采用同一文件系统的硬链接，保证并发删除附件不会破坏正在生成的备份，完成或取消后会清理。恢复前的安全备份仍与数据替换保持互斥，避免遗漏恢复前最后的修改。

自动备份目录（含默认目录）不能是符号链接，也不能位于受管附件存储内。V1 备份上限为 10,000 个附件、单文件（含数据库快照）256 MiB、解压后总计 2 GiB（导出时预留 2 MiB 清单空间）、ZIP 文件 3 GiB；导入时只检查单附件大小，数据累计超过整包限制后需要另行处理。

回收站操作会区分条目已不存在和条目状态冲突。若单独移入回收站的任务所属项目也在回收站，可以一并恢复项目和任务，也可以单独彻底删除任务。

彻底删除或清空回收站时，数据库删除与附件文件清理分步完成。清空回收站时，项目、任务和投递记录在同一事务中删除，任一模块删除失败都会回滚。若文件清理失败，条目仍会从回收站移除并显示警告；未引用文件会在后续清理操作中重试删除。

## 开发

开发方案：[开发自动化与 UI 封装精简](docs/plans/automation-and-ui-simplification.md)（已完成开发用 Automation 接入与未用 UI 封装精简）。

后端测试可通过 `backend.Options.Now` 注入项目、任务、求职和备份使用的时钟，默认使用 `time.Now`。项目和任务以 UTC 存储时间戳，同一次任务移动或排序使用同一时间更新所有位置发生变化的任务。

| 命令 | 用途 |
|---|---|
| `go run .` | 在本地启动 Aion 原生桌面端 |
| `go test ./...` | 运行入口、后端与无头 UI 交互测试 |
| `go test -race ./...` | 运行竞态检测测试 |
| `go vet ./...` | Go 静态分析检查 |
| `go tool mygo vet .` | MyGo 布局与状态规则静态检查 |

### 真实窗口自动化冒烟（开发专用）

`automation_e2e.go` 仅在 `-tags e2e` 构建时引入 `github.com/egoist/mygo/plugins/automation`，默认构建与正式发布构建（`mygo_noinspector`）均不启动自动化服务。使用独立临时目录运行真实窗口操作与截图：

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

```text
Aion/
├── main.go                                 # 原生窗口宿主入口、应用菜单/更新插件与系统能力桥接
├── automation_e2e.go                       # e2e 构建标签专用的 MyGo Automation 插件入口
├── main_test.go                            # 快照输出数据保护与原子替换回归测试
├── mygo.json                               # MyGo 应用配置、打包元数据与更新源配置
├── CHANGELOG.md                            # 版本更新说明（发版说明与客户端更新窗口共用）
├── docs/plans/                             # 开发与重构方案及验收记录（状态以各文档为准）
├── .github/
│   ├── workflows/release.yml               # 标签触发的 macOS Universal 构建与草稿 Release 工作流
│   └── ISSUE_TEMPLATE/                     # GitHub Issue 表单模板与空白 Issue 禁用配置
├── frontend/                               # 原生界面层：工作台状态、模块切换侧栏、项目看板、求职投递、回收站、设置与附件预览
│   ├── ui_*.go / icons.go                  # 基于 MyGo 原生 UI 封装的设计令牌、控件、弹窗、Markdown 与 Lucide SVG 图标
│   ├── sidebar.go                          # 展开/折叠双形态模块切换器与各模块导航侧栏
│   ├── task_save.go                        # 任务草稿串行保存、合并输入与保存后的导航
│   ├── job_save.go                         # 求职投递详情草稿校验、串行保存与撤销恢复
│   ├── job_controller.go                   # 求职投递筛选、分页、流转、备注、补录与回收站动作控制
│   ├── job_applications_view.go            # 求职投递主视图、统计摘要、响应式表格与新增弹窗
│   ├── job_application_detail_view.go      # 求职投递详情弹窗、流程时间线、备注/补录表单与结束流程弹窗
│   ├── workbench_load.go                   # 后台加载与过期结果保护
│   ├── record_write.go / settings_write.go  # 后台记录操作与设置写入
│   ├── async_persistence_test.go           # 长持锁输入/导航、保存失败与并发回调回归测试
│   ├── async_record_write_test.go           # Store 持锁时的看板、表单、回收站和设置写入回归测试
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

基于 [MyGo](https://github.com/egoist/mygo) 原生 UI 构建。

标签构建会先查询 GitHub Release。只有查询成功且该版本尚未创建或仍为草稿时，工作流才会继续。

Apple 公证优先使用完整的 API Key 三项凭据，否则使用已通过校验的 Apple ID 凭据；临时签名钥匙串从创建前即登记清理路径，后续导入或公证失败时也会清理。
