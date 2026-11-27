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
  <img src="assets/screenshot.png" alt="按状态分组卡片的 Aion 三列看板" />
</p>

Aion（读作 `EYE-on`，源自古希腊语 αἰών，意为“生命、时代与永恒”）帮你把日常项目、任务拆解、本地资料与求职投递记录收敛在同一处。所有数据 100% 存储在你的 Mac 本地，无需注册账号，不依赖云服务，断网环境下依然拥有毫秒级响应。

## 功能

- **极简顶栏工作台与聚焦总览**：顶部悬浮导航胶囊切换**今天**（`⌘1`，按「待推进 / 进行中 / 全部未完成 / 已完成」汇总任务，并展示进行中求职与本地 MCP 状态，支持行内快捷推进）、**项目**（`⌘2`）、**求职**（`⌘3`）与**设置**（`⌘,`）；右侧集成全局 **`⌘K` 命令面板**（跨活跃与已归档项目/任务/投递实时搜索与快捷动作）、**回收站**、**GitHub Issues 问题反馈**、**深浅外观切换**与统一 **`+ 新建` 菜单**，并支持按阶段无损隐藏任意模块
- **项目与三列看板**：管理多个活跃与归档项目，双行分层头部支持零位移悬停项目切换与编辑、直达「所有项目 / 已归档」视图；围绕「计划 / 进行中 / 已完成」支持拖拽跨列、列内排序、右键菜单、语义优先级快速切换，以及 `⌘F` 搜索、优先级、截止日期（已逾期 / 7 天内 / 无截止日）与多标签交集筛选
- **任务详情、Markdown 与子任务**：右侧抽屉编辑标题、状态、优先级、截止日期与标签，内置「编辑 / 预览 / 分栏」Markdown 编辑器及快捷工具栏，支持子任务拆分、勾选、重命名、排序，底部展示创建、更新与完成时间
- **求职投递、网站图标与时间线**：提供「全部投递 / 进行中 / 已获 Offer / 已结束」视角与 Offer 转化率指标；自动抓取投递链接网站图标（支持 PNG/JPEG/GIF/WebP/ICO 与 SPA 内嵌图标发现并缓存于系统缓存目录，无链接时降级为公司首字图标），在「今天」与「求职」双视图展示；支持 `⌘F` 搜索、多条件筛选分页、投递链接复制/打开、同名进行中提醒、流程结束/重开（含可选重启备注）、备注与历史阶段补录
- **本地受管附件**：支持点击选择或从 Finder 拖入文件/图片，提供图片缩略图与大图预览、默认应用打开、Finder 定位与另存为导出
- **本地 AI / MCP 与 CLI 联动**：内置 `mcp` 子命令（标准 stdio MCP Server，支持跨项目搜索任务、附件导入、项目/任务/投递读写与回收站恢复，一键复制到 **Antigravity** 与 **Codex**）及轻量本地 CLI（`summary` / `project` / `task` / `job` / `trash`，支持 `-json`），桌面端运行时通过本地 Unix Socket 实时刷新界面并保护未保存草稿；仓库附带 [aion-workbench Skill](skills/aion-workbench/SKILL.md) 供 AI 助手直接按工作流调用
- **回收站与全量备份**：删除项目、任务与求职投递默认进入统一回收站（支持按类型筛选、完整恢复与彻底清除）；内置自动备份轮转、可展开的**历史备份记录**（区分自动备份与安全备份，支持 Finder 定位与一键恢复确认）以及手动全量 `.zip` 导出/恢复

## 快速开始

### 直接使用（推荐）

前往 [GitHub Releases](https://github.com/Games55k/Aion/releases) 下载最新版本的 `.dmg` 安装包，将 `Aion.app` 拖入「应用程序」目录即可使用（支持后台静默检查更新，发现新版本时在顶部栏右侧展示更新图标，点击可在应用内查看说明并决定是否安装）。

### 从源码运行

需要 Go 1.27.3 或更新版本：

```bash
git clone https://github.com/Games55k/Aion.git
cd Aion
go run .
```

如需使用独立的测试数据目录启动（不影响正式数据）：

```bash
go run . -data-dir /tmp/aion-dev
```

## 数据与备份

默认数据目录位于 `~/Library/Application Support/Aion/`，包含 `aion.sqlite`（Schema v3）、`attachments/`（受管附件）与 `backups/`（自动备份与恢复前安全备份）。

在设置页（`⌘,`）中可配置周期性自动备份（默认每天一次、保留 7 份），或随时手动导出 / 恢复全量 `.zip` 备份。完整的目录规范、附件与并发机制、Schema 升级与回收站规则见 [数据存储、备份与回收站](docs/data-and-backup.md)。

## 开发与构建

Aion 基于 [MyGo](https://github.com/egoist/mygo) 构建 macOS 原生应用，不打包 Chromium / WebView。

```bash
go tool mygo dev        # 开发模式运行
go tool mygo build      # 构建 macOS 原生 .app 与 .dmg
go test ./...           # 运行入口、后端与无头 UI 测试
go vet ./...            # Go 静态检查
go tool mygo vet .      # MyGo 布局与状态检查
```

视觉快照、真实窗口自动化冒烟、目录结构与发布流程见 [开发、测试与发布指南](docs/development.md)。

## 文档

- [交互与持久化行为说明](docs/behavior.md) — 自动保存、草稿恢复、后台写入与求职视图行为
- [数据存储、备份与回收站](docs/data-and-backup.md) — 数据目录、受管附件、备份限制、Schema v3 恢复与回收站机制
- [开发、测试与发布指南](docs/development.md) — 构建打包、无头快照、自动化冒烟、项目结构与发版工作流

---

基于 [MyGo](https://github.com/egoist/mygo) 原生 UI 构建。
