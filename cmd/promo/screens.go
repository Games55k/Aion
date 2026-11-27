package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"time"

	brand "aion/assets/icon"
	"aion/backend"
	"aion/backend/jobapplications"
	"aion/backend/model"
	"aion/backend/projects"
	"aion/frontend"

	"github.com/egoist/mygo/plugins/render"
	"github.com/egoist/mygo/ui"
)

type demoAssets struct {
	LogoSVG      *ui.SVG
	FocusLight   *ui.Bitmap
	BoardLight   *ui.Bitmap
	BoardDark    *ui.Bitmap
	TaskDetail   *ui.Bitmap
	JobsLight    *ui.Bitmap
	JobDetail    *ui.Bitmap
	PaletteDark  *ui.Bitmap
	SettingsDark *ui.Bitmap
}

func captureDemoAssets() (*demoAssets, error) {
	tmpRoot, err := os.MkdirTemp("", "aion-promo-seed-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpRoot)

	dataDir := filepath.Join(tmpRoot, "data")
	sampleImgPath := filepath.Join(tmpRoot, "aion-architecture-preview.png")
	if err := writeDemoAttachmentPNG(sampleImgPath); err != nil {
		return nil, fmt.Errorf("write demo attachment png: %w", err)
	}

	now := time.Date(2026, 11, 27, 10, 30, 0, 0, time.FixedZone("CST", 8*3600))
	ctx := context.Background()

	app, err := backend.Open(ctx, backend.Options{
		DataDir:          dataDir,
		DisableScheduler: true,
		Now:              func() time.Time { return now },
	})
	if err != nil {
		return nil, fmt.Errorf("open backend: %w", err)
	}
	defer app.Close()

	// Seed rich projects & tasks.
	p1, err := app.Projects.CreateProject(ctx, projects.ProjectInput{
		Name:        "Aion 1.0 桌面端发布",
		Icon:        "rocket",
		Description: "纯原生、本地优先的 macOS / Windows 个人数字化工作台",
	})
	if err != nil {
		return nil, err
	}
	p2, err := app.Projects.CreateProject(ctx, projects.ProjectInput{
		Name:        "AI Agent 工作流实验",
		Icon:        "code",
		Description: "基于本地 MCP Server 与 CLI 的无云端智能协同",
	})
	if err != nil {
		return nil, err
	}
	p3, err := app.Projects.CreateProject(ctx, projects.ProjectInput{
		Name:        "个人知识体系归档",
		Icon:        "book",
		Description: "季度复盘、读书笔记与架构沉淀",
	})
	if err != nil {
		return nil, err
	}
	_, _ = app.Projects.CreateTask(ctx, p3.ID, projects.TaskInput{
		Title:    "整理 SQLite WAL 事务与本地优先设计笔记",
		Status:   model.Done,
		Priority: model.Medium,
		Tags:     []string{"架构", "复盘"},
	})
	_, _ = app.Projects.ArchiveProject(ctx, p3.ID, true)

	// Main project tasks across Planned / InProgress / Done.
	_, _ = app.Projects.CreateTask(ctx, p1.ID, projects.TaskInput{
		Title:       "使用 MyGo Render 制作 60fps 项目介绍视频",
		Description: "通过 GPU 原生绘制与 AVFoundation 硬件编码生成产品宣传短片。",
		Status:      model.Planned,
		Priority:    model.High,
		Tags:        []string{"MyGo", "Render", "Video"},
		DueDate:     "2026-11-29",
		Subtasks: []model.Subtask{
			{Title: "设计 8 幕分镜与双外观切换动效", Done: true},
			{Title: "合成 48kHz 立体声配乐与 UI 交互音效", Done: false},
		},
	})
	_, _ = app.Projects.CreateTask(ctx, p1.ID, projects.TaskInput{
		Title:    "完善 Windows NSIS 安装包签名与增量更新",
		Status:   model.Planned,
		Priority: model.Medium,
		Tags:     []string{"Windows", "Release"},
		DueDate:  "2026-12-03",
	})

	heroTask, err := app.Projects.CreateTask(ctx, p1.ID, projects.TaskInput{
		Title: "实现三列看板拖拽排序与截止日多维筛选",
		Description: "## 核心交付标准\n" +
			"- 跨列拖拽与列内排序保持统一 UTC 时间戳\n" +
			"- 支持 `⌘F` 关键词、优先级、截止日与多标签交集筛选",
		Status:   model.InProgress,
		Priority: model.Urgent,
		Tags:     []string{"Kanban", "MyGo", "V1"},
		DueDate:  "2026-11-28",
		Subtasks: []model.Subtask{
			{Title: "完成 Planned / InProgress / Done 三列拖拽动画", Done: true},
			{Title: "接入截止日筛选（已逾期 / 7 天内 / 无截止日）", Done: true},
			{Title: "执行无头 UI 交互回归测试 (-count=5)", Done: false},
		},
	})
	if err != nil {
		return nil, err
	}
	_, _ = app.Projects.CreateTask(ctx, p1.ID, projects.TaskInput{
		Title:    "优化本地 Unix Socket 与 MCP Server 实时同步",
		Status:   model.InProgress,
		Priority: model.High,
		Tags:     []string{"MCP", "AI", "IPC"},
		DueDate:  "2026-11-30",
		Subtasks: []model.Subtask{
			{Title: "桌面端在线走 Socket 实时刷新并保护未保存草稿", Done: true},
			{Title: "离线模式自动降级按次直连 SQLite", Done: true},
		},
	})

	_, _ = app.Projects.CreateTask(ctx, p1.ID, projects.TaskInput{
		Title:    "完成 SQLite v3 事务存储与全量 ZIP 自动备份",
		Status:   model.Done,
		Priority: model.High,
		Tags:     []string{"SQLite", "Backup"},
	})
	_, _ = app.Projects.CreateTask(ctx, p1.ID, projects.TaskInput{
		Title:    "集成 Aion 品牌矢量图标与深浅外观无缝切换",
		Status:   model.Done,
		Priority: model.Medium,
		Tags:     []string{"Design", "macOS"},
	})

	// Secondary project tasks (visible in Today focus view).
	_, _ = app.Projects.CreateTask(ctx, p2.ID, projects.TaskInput{
		Title:    "编写 aion-workbench Skill 与一键配置复制",
		Status:   model.InProgress,
		Priority: model.High,
		Tags:     []string{"Antigravity", "Codex"},
		DueDate:  "2026-11-28",
	})
	_, _ = app.Projects.CreateTask(ctx, p2.ID, projects.TaskInput{
		Title:    "验证本地 CLI summary / task / job JSON 输出",
		Status:   model.Done,
		Priority: model.Medium,
		Tags:     []string{"CLI"},
	})

	// Seed iconic BAT + Magnificent Seven job applications with real deterministic brand logos.
	_, _ = app.JobApplications.Create(ctx, jobapplications.CreateInput{
		CompanyName:    "Google",
		PositionName:   "Senior Software Engineer · Infrastructure",
		AppliedDate:    "2026-11-04",
		Stage:          model.JobStageOffer,
		ApplicationURL: "https://careers.google.com/jobs/results/senior-swe-infra",
		Channel:        "内部推荐",
		Notes:          "四轮 Onsite 涵盖分布式一致性、系统设计与 Go 性能优化，HC 审批通过，已收到正式 Offer。",
	})
	heroJob, err := app.JobApplications.Create(ctx, jobapplications.CreateInput{
		CompanyName:    "字节跳动",
		PositionName:   "基础架构研发工程师（Go / 分布式存储）",
		AppliedDate:    "2026-11-12",
		Stage:          model.JobStageInterview2,
		ApplicationURL: "https://jobs.bytedance.com/experienced/position/infra-go",
		Channel:        "官网直投",
		Notes:          "一面深入讨论了 Go 并发调度、SQLite WAL 事务与本地优先架构；二面将聚焦高吞吐存储引擎设计。",
	})
	if err != nil {
		return nil, err
	}
	_, _ = app.JobApplications.BackfillStageEvent(ctx, jobapplications.BackfillStageInput{
		ApplicationID: heroJob.ID,
		Stage:         model.JobStageInterview1,
		OccurredDate:  "2026-11-16",
		Note:          "补录一面复盘：手写无锁 LRU 缓存与批处理管线，面试官反馈非常积极。",
	})
	_, _ = app.JobApplications.AddNoteEvent(ctx, jobapplications.AddNoteInput{
		ApplicationID: heroJob.ID,
		Note:          "HR 已确认二面时间：本周五 14:30，提前整理 Aion 原生 GPU 渲染基准测试数据。",
	})
	_, _ = app.JobApplications.Create(ctx, jobapplications.CreateInput{
		CompanyName:    "Apple",
		PositionName:   "macOS Core Graphics & Native UI Engineer",
		AppliedDate:    "2026-11-15",
		Stage:          model.JobStageInterview2,
		ApplicationURL: "https://jobs.apple.com/zh-cn/details/macos-native-ui",
		Channel:        "官网直投",
		Notes:          "技术面重点考察 Metal 渲染管线、Retina 字体排版与系统级交互体验打磨。",
	})
	_, _ = app.JobApplications.Create(ctx, jobapplications.CreateInput{
		CompanyName:    "Microsoft",
		PositionName:   "Principal Systems Engineer · Azure Core",
		AppliedDate:    "2026-11-09",
		Stage:          model.JobStageHR,
		ApplicationURL: "https://careers.microsoft.com/us/en/job/azure-core-systems",
		Channel:        "内部推荐",
	})
	_, _ = app.JobApplications.Create(ctx, jobapplications.CreateInput{
		CompanyName:    "腾讯",
		PositionName:   "微信后台高并发系统架构师",
		AppliedDate:    "2026-11-20",
		Stage:          model.JobStageInterview1,
		ApplicationURL: "https://careers.tencent.com/jobdesc.html?postId=wechat-backend",
		Channel:        "技术社区",
	})
	closedJob, _ := app.JobApplications.Create(ctx, jobapplications.CreateInput{
		CompanyName:    "阿里巴巴",
		PositionName:   "阿里云云原生基础引擎专家",
		AppliedDate:    "2026-10-28",
		Stage:          model.JobStageInterview2,
		ApplicationURL: "https://talent.alibaba.com/off-campus/position-detail?id=cloud-engine",
		Channel:        "猎头推荐",
	})
	_, _ = app.JobApplications.Close(ctx, jobapplications.CloseInput{
		ID:      closedJob.ID,
		Outcome: model.JobOutcomeWithdrawn,
		Reason:  "综合技术方向与已有 Offer 规划后主动结束流程",
	})

	// Create one automatic backup so Settings displays live backup history.
	_, _ = app.Backups.RunDue(ctx)

	wb := frontend.New(app, frontend.Host{
		OpenURL:     func(string) error { return nil },
		Now:         func() time.Time { return now },
		LoadFavicon: demoFaviconLoader,
		UpdateInfo: frontend.UpdateCapability{
			Version:          "0.2.0",
			AvailableVersion: "0.2.1",
			Status:           frontend.UpdateAvailable,
			LastCheck:        now.Add(-15 * time.Minute),
			CanSelfUpdate:    true,
		},
		MCPInfo: frontend.MCPCapability{
			Ready:          true,
			ExecutablePath: "/Applications/Aion.app/Contents/MacOS/Aion",
		},
	})
	wb.ReloadAll()
	wb.SelectProject(p1.ID)

	// Attach our crisp generated architecture image to heroTask.
	wb.SelectTask(heroTask.ID)
	wb.ImportAttachmentsToSelectedTask([]string{sampleImgPath})
	wb.WaitForTasks()
	wb.ClearSelectedTask()

	renderScreen := func(wDip, hDip int, dark bool, settleFrames int) (*ui.Bitmap, error) {
		if dark {
			wb.ApplyThemeSetting(frontend.ThemeDark)
		} else {
			wb.ApplyThemeSetting(frontend.ThemeLight)
		}
		wb.WaitForTasks()
		comp := render.Composition{
			Width:  wDip * 2,
			Height: hDip * 2,
			FPS:    30,
			Frames: max(settleFrames+1, 2),
			Scale:  2,
			Dark:   dark,
			Locale: "zh-Hans",
			View: func(c *ui.Context, _ render.Frame) {
				wb.View(c)
			},
		}
		img, err := render.Still(comp, settleFrames, render.Options{})
		if err != nil {
			return nil, err
		}
		return ui.NewBitmap(img), nil
	}

	out := &demoAssets{
		LogoSVG: ui.MustParseSVG([]byte(brand.SVG)),
	}

	// 1. Focus Today View (Light)
	wb.NavigateTo(frontend.NavFocus)
	if out.FocusLight, err = renderScreen(1280, 800, false, 2); err != nil {
		return nil, fmt.Errorf("render focus light: %w", err)
	}

	// 2. Three-Column Kanban Board (Light & Dark)
	wb.SelectProject(p1.ID)
	wb.NavigateTo(frontend.NavProjectBoard)
	if out.BoardLight, err = renderScreen(1280, 800, false, 2); err != nil {
		return nil, fmt.Errorf("render board light: %w", err)
	}
	if out.BoardDark, err = renderScreen(1280, 800, true, 2); err != nil {
		return nil, fmt.Errorf("render board dark: %w", err)
	}

	// 3. Task Detail Drawer with Markdown, Subtasks & Managed Attachment (Light)
	wb.SelectTask(heroTask.ID)
	if out.TaskDetail, err = renderScreen(1280, 860, false, 10); err != nil {
		return nil, fmt.Errorf("render task detail: %w", err)
	}
	wb.ClearSelectedTask()

	// 4. Job Applications Tracker (Light) — with Google, Apple, Microsoft, ByteDance, Tencent, Alibaba logos!
	wb.SwitchModule("job_applications")
	if out.JobsLight, err = renderScreen(1280, 800, false, 2); err != nil {
		return nil, fmt.Errorf("render jobs light: %w", err)
	}

	// 5. Job Detail & Full Timeline Modal (Light)
	wb.SelectJobApplication(heroJob.ID)
	if out.JobDetail, err = renderScreen(1280, 800, false, 10); err != nil {
		return nil, fmt.Errorf("render job detail: %w", err)
	}
	wb.CloseJobDetail()

	// 6. Command Palette ⌘K Overlay (Dark)
	wb.SelectProject(p1.ID)
	wb.NavigateTo(frontend.NavProjectBoard)
	wb.OpenCommandPalette()
	if out.PaletteDark, err = renderScreen(1280, 800, true, 16); err != nil {
		return nil, fmt.Errorf("render command palette dark: %w", err)
	}
	wb.CloseCommandPalette()

	// 7. Settings View (Backup + Local MCP Config, Dark)
	wb.NavigateTo(frontend.NavSettings)
	if out.SettingsDark, err = renderScreen(960, 946, true, 2); err != nil {
		return nil, fmt.Errorf("render settings dark: %w", err)
	}

	return out, nil
}

// writeDemoAttachmentPNG draws a clean botanical-charcoal architectural card preview
// so the task detail drawer displays a realistic local managed image thumbnail.
func writeDemoAttachmentPNG(path string) error {
	const w, h = 480, 300
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		fy := float64(y) / float64(h)
		for x := range w {
			fx := float64(x) / float64(w)
			dx := fx - 0.72
			dy := fy - 0.35
			d2 := dx*dx + dy*dy
			glow := 0.0
			if d2 < 0.45 {
				t := 1 - d2/0.45
				glow = t * t
			}
			r := uint8(min(255, int(24+glow*72+fx*14)))
			g := uint8(min(255, int(28+glow*148+fy*18)))
			b := uint8(min(255, int(34+glow*52+(1-fx)*28)))
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	fillRect := func(x0, y0, x1, y1 int, col color.RGBA) {
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				img.SetRGBA(x, y, col)
			}
		}
	}
	fillRect(34, 38, 178, 54, color.RGBA{R: 180, G: 228, B: 81, A: 255})
	fillRect(34, 72, 290, 82, color.RGBA{R: 235, G: 240, B: 245, A: 220})
	fillRect(34, 114, 162, 252, color.RGBA{R: 38, G: 44, B: 52, A: 255})
	fillRect(182, 114, 310, 252, color.RGBA{R: 38, G: 44, B: 52, A: 255})
	fillRect(330, 114, 446, 252, color.RGBA{R: 38, G: 44, B: 52, A: 255})
	fillRect(46, 134, 146, 170, color.RGBA{R: 0, G: 113, B: 227, A: 230})
	fillRect(194, 134, 298, 182, color.RGBA{R: 173, G: 218, B: 112, A: 230})
	fillRect(342, 134, 434, 168, color.RGBA{R: 120, G: 135, B: 155, A: 210})

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
