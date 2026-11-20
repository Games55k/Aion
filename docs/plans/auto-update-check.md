# 自动静默检查更新与侧边栏更新入口实现计划

- **状态**：已完成（Completed）
- **关联 Issue**：[#15 自动检查软件更新](https://github.com/Games55k/Aion/issues/15)

## 目标

为不希望开启“后台自动更新”、但希望及时获知新版本的用户提供无打扰的更新体验：

1. **后台静默定期检查**：启动后延迟检查一次，之后按用户设定周期定期检查，并在系统休眠唤醒后补查；全程不自动下载安装、不弹窗打扰，网络异常静默退避。
2. **侧边栏极简图标入口**：发现新版本（或下载/安装/待重启）时，仅在左侧边栏展示一个「圆圈包着向上箭头」的图标按钮（带悬停提示），无新版本或已跳过该版本时隐藏。
3. **应用内统一更新弹窗**：点击侧边栏图标、菜单栏「检查更新…」或设置页「检查更新」统一打开应用内更新弹窗，展示 Markdown 更新日志、跳过此版本、稍后提醒、下载进度与取消、以及安全重启。
4. **设置页偏好管理**：在「应用版本与更新」卡片新增「自动检查更新」开关（默认开启）与检查频率配置（每 6 小时 / 每 12 小时 / 每天），异步写入本地偏好并同步调度器。

## 架构设计

- **偏好存储（`backend/settings.go`）**：在 SQLite `settings` 表新增 `app.update_check` 偏好（`Enabled` 默认 `true`，`Interval` 默认 `24h`），保持 `backend` 完全离线。
- **宿主层调度器（`updater_controller.go`、`main.go`）**：
  - 使用 MyGo 底层 `mygo.Updater.Check`、`Update.Install`、`mygo.App.Relaunch`、`mygo.Power.OnResume` 替代会强制弹窗的 `plugins/updater/native`。
  - 本机运行态（`last_check`、`skipped_version`）原子写入数据目录下的 `.updater-state.json`（首次迁移兼容读取旧 `updater.json`），不写 SQLite，也不随备份恢复覆盖。
  - 状态变化通过 `win.Update` 推送到 `Workbench.SetUpdateCapability`，绝不持有 `Store` 锁。
- **前端界面（`frontend/`）**：
  - `icons.go`：新增 `iconCircleArrowUp()`。
  - `sidebar.go`：在侧边栏顶部品牌行右侧（折叠态居中位于 Logo 下方）渲染仅图标的更新按钮与悬停提示。
  - `update_modal.go`：基于 `dialogPresent` 与现有 Markdown 渲染器实现应用内更新弹窗。
  - `settings_view.go` / `settings_write.go` / `workbench.go`：补充自动检查更新开关、检查周期异步保存与恢复同步。

## 验证计划

- `backend` 单元测试：偏好默认值、持久化往返与非法周期校验。
- 宿主层 `updateController` 单元测试：启动检查、周期检查、休眠唤醒补查、失败退避、手动/后台单飞合并、跳过版本、下载进度与取消、关闭清理、旧 `updater.json` 迁移。
- 前端 `ui.Tester` 测试（`-count=5`）：侧边栏展开/折叠态仅图标按钮、点击弹窗流转、跳过版本隐藏、下载与待重启状态、设置页开关与频率异步保存（含 `Store` 锁持有期间不阻塞与备份恢复同步）。
- 视觉快照渲染检查，确认侧边栏图标按钮与更新弹窗样式。
- 运行 `go test ./...`、`go vet ./...`、`go tool mygo vet .`。
