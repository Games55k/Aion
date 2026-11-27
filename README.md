<p align="center">
  <img src="assets/icon/aion-icon.svg" width="64" height="64" alt="Aion logo" />
</p>

<h1 align="center">Aion</h1>

<p align="center">
  轻量、纯原生的 macOS / Windows 本地项目与求职投递管理工具。
</p>

<p align="center">
  <img src="https://img.shields.io/badge/macOS-000000?style=flat&logo=apple&logoColor=white" alt="macOS" />
  <img src="https://img.shields.io/badge/Windows-0078D4?style=flat&logo=windows&logoColor=white" alt="Windows" />
  <img src="https://img.shields.io/badge/Go-1.27.3+-4B5563?style=flat&logo=go&logoColor=00ADD8&labelColor=18181B" alt="Go 1.27.3+" />
  <a href="https://mygo.run"><img src="https://img.shields.io/badge/MyGo-Native_UI-2563EB?style=flat&labelColor=18181B" alt="MyGo Native UI" /></a>
</p>

<p align="center">
  <img src="assets/screenshot.png" alt="按状态分组卡片的 Aion 三列看板" />
</p>

Aion（读作 `EYE-on`，源自古希腊语 αἰών，意为“生命、时代与永恒”）帮你把日常项目、任务拆解、本地资料与求职投递记录收敛在同一处。所有数据 100% 存储在你的本地设备，无需注册账号，不依赖云服务，断网环境下依然拥有毫秒级响应。

## 功能

- **极简顶栏工作台与「今天」总览**：顶部悬浮导航胶囊切换**今天**（`⌘1`）、**项目**（`⌘2`）、**求职**（`⌘3`）与**设置**（`⌘,`），集成全局 **`⌘K` 命令面板**、回收站、问题反馈、深浅外观与 `+ 新建` 菜单，并支持按阶段无损隐藏模块
- **多项目与三列看板**：管理活跃与已归档项目，围绕「计划 / 进行中 / 已完成」支持拖拽跨列、列内排序、优先级快速切换，以及 `⌘F` 搜索、截止日期与多标签交集筛选
- **任务详情、Markdown 与子任务**：右侧详情弹窗支持实时自动保存、「编辑 / 预览 / 分栏」Markdown 编辑器与子任务拆分、勾选、重命名及排序
- **求职投递与流程时间线**：按「全部 / 进行中 / 已获 Offer / 已结束」跟踪投递与 Offer 转化率，自动抓取公司网站图标（无链接时降级为首字图标），支持阶段流转、备注、历史阶段补录与流程结束/重开
- **本地受管附件**：支持点击选择或拖拽导入文件与图片，提供缩略图与大图预览、默认应用打开、文件管理器定位与另存为导出
- **本地 AI / MCP 与 CLI 联动**：内置标准 stdio `mcp` 服务与轻量本地 CLI（`summary` / `project` / `task` / `job` / `trash`），支持桌面端在线实时同步（保护未保存草稿）与离线直连，并附带 [aion-workbench Skill](skills/aion-workbench/SKILL.md)
- **统一回收站与全量备份**：项目、任务与求职投递软删除至统一回收站；内置周期性自动备份轮转、可展开历史备份记录（含恢复前安全备份）与手动 `.zip` 导出/恢复

## 快速开始

### 直接使用（推荐）

前往 [GitHub Releases](https://github.com/Games55k/Aion/releases) 下载对应平台的最新版本安装包（均支持后台静默检查更新，发现新版本时在顶部栏右侧展示更新图标，点击可在应用内查看说明并决定是否安装）：

- **macOS（Universal）**：下载 `aion-<版本>.dmg`，将 `Aion.app` 拖入「应用程序」目录即可使用。
- **Windows（64 位 x64）**：下载 `aion-setup-<版本>-amd64.exe` 安装包直接运行安装（默认安装到当前用户的 `%LOCALAPPDATA%\Programs\Aion`，包含开始菜单/桌面快捷方式与应用内更新支持）。

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

默认数据目录位于系统应用配置目录下：

- **macOS**：`~/Library/Application Support/Aion/`
- **Windows**：`%AppData%\Aion\`

目录内包含 `aion.sqlite`（Schema v3）、`attachments/`（受管附件）与 `backups/`（自动备份与恢复前安全备份）。

在设置页（`⌘,` / `Ctrl+,`）中可配置周期性自动备份（默认每天一次、保留 7 份），或随时手动导出 / 恢复全量 `.zip` 备份。完整的目录规范、附件与并发机制、Schema 升级与回收站规则见 [数据存储、备份与回收站](docs/data-and-backup.md)。

## 开发与构建

Aion 基于 [MyGo](https://github.com/egoist/mygo) 构建 macOS 与 Windows 原生应用，不打包 Chromium / WebView。

```bash
go tool mygo dev                              # 开发模式运行
go tool mygo build                            # 构建当前平台原生安装包（macOS .dmg / Windows .exe）
go tool mygo build -platform windows/amd64    # 交叉构建 Windows amd64 .exe（在 Windows 上自动打包 NSIS 安装包）
go test ./...                                 # 运行入口、后端与无头 UI 测试
go vet ./...                                  # Go 静态检查
go tool mygo vet .                            # MyGo 布局与状态检查
```

视觉快照、真实窗口自动化冒烟、目录结构与发布流程见 [开发、测试与发布指南](docs/development.md)。

## 文档

- [交互与持久化行为说明](docs/behavior.md) — 自动保存、草稿恢复、后台写入与求职视图行为
- [数据存储、备份与回收站](docs/data-and-backup.md) — 数据目录、受管附件、备份限制、Schema v3 恢复与回收站机制
- [开发、测试与发布指南](docs/development.md) — 构建打包、无头快照、自动化冒烟、项目结构与发版工作流

---

基于 [MyGo](https://github.com/egoist/mygo) 原生 UI 构建。
