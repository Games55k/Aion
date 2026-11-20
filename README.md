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

- **模块化工作台**：侧边栏支持拖拽调节宽度与折叠，顶部可在**项目**与**求职**工作区之间一键切换，各自保留最近页面、搜索与筛选上下文，并共享全局**回收站**与**设置 / 备份**
- **项目与三列看板**：管理多个活跃与归档项目，围绕“计划 / 进行中 / 已完成”支持拖拽跨列、列内排序、右键菜单、优先级快速切换，以及 `⌘K` 搜索、优先级与多标签交集筛选
- **任务详情、Markdown 与子任务**：居中弹窗编辑标题、状态、优先级、截止日期与标签，内置“编辑 / 预览 / 分栏” Markdown 编辑器及快捷工具栏，支持子任务拆分、勾选、重命名、排序与进度条
- **求职投递与时间线**：提供“全部投递 / 进行中 / 已获 Offer / 已结束”视角与统计卡片，支持 `⌘K` 搜索、多条件筛选分页、投递链接复制/打开、同名进行中提醒、流程结束/重开、备注与历史阶段补录
- **本地受管附件**：支持点击选择或从 Finder 拖入文件/图片，提供图片缩略图与大图预览、默认应用打开、Finder 定位与另存为导出
- **回收站与全量备份**：删除项目、任务与求职投递默认进入统一回收站（支持按类型筛选、完整恢复与彻底清除）；内置自动备份轮转与手动全量 `.zip` 导出/恢复（可通过 `⌘,` 打开设置）

## 快速开始

### 直接使用（推荐）

前往 [GitHub Releases](https://github.com/Games55k/Aion/releases) 下载最新版本的 `.dmg` 安装包，将 `Aion.app` 拖入「应用程序」目录即可使用（支持后台静默检查更新，发现新版本时在侧边栏展示更新图标，点击可在应用内查看说明并决定是否安装）。

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
