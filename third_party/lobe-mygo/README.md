<a name="readme-top"></a>

<div align="center">

![Lobe MyGo](./.github/banner.png)

<h1>Lobe MyGo</h1>

<h3>The UI Kit for Native Go Apps</h3>

Lobe UI components for [MyGo](https://github.com/egoist/mygo) native UI, drawn on the GPU in pure Go.

[Docs](https://mygo.lobehub.com) · [Catalog](./catalog) · [Changelog](./CHANGELOG.md) · [Report Bug][github-issues-link] · [Request Feature][github-issues-link]

<!-- SHIELD GROUP -->

[![][go-reference-shield]][go-reference-link]
[![][go-version-shield]][go-version-link]
[![][discord-shield]][discord-link]<br/>
[![][github-contributors-shield]][github-contributors-link]
[![][github-forks-shield]][github-forks-link]
[![][github-stars-shield]][github-stars-link]
[![][github-issues-shield]][github-issues-link]
[![][github-license-shield]][github-license-link]

</div>

<details>
<summary><kbd>Table of contents</kbd></summary>

#### TOC

- [✨ Why Lobe MyGo?](#-why-lobe-mygo)
- [🚀 Getting Started](#-getting-started)
- [📦 Packages](#-packages)
- [📖 Documentation](#-documentation)
- [💬 Community](#-community)
- [⌨️ Local Development](#️-local-development)
- [🤝 Contributing](#-contributing)
- [🩷 Sponsor](#-sponsor)
- [🔗 Links](#-links)

####

</details>

## ✨ Why Lobe MyGo?

Lobe MyGo brings the [Lobe UI](https://github.com/lobehub/lobe-ui) design language to desktop apps written with MyGo's native UI: no HTML, no JavaScript and no webview.

- **Lobe UI, in Go**: 70+ components (buttons, inputs, menus, modals, toasts, tabs and more) that look and move like their React counterparts.
- **Same tokens as Lobe UI**: colors, radii and font sizes are generated from `@lobehub/ui`, so a Go app and a web app share one look.
- **Light and dark built in**: `theme.Apply` follows the system appearance with no extra setup.
- **Pure Go, no cgo**: one small binary per platform, cross-compiled from any machine.
- **Motion included**: springs and shared-element transitions for indicators, overlays and toasts.
- **Lucide icons**: all 2,300+ Lucide icons as Go functions.

![](./.github/gallery.png)

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## 🚀 Getting Started

```bash
$ go get github.com/lobehub/lobe-mygo
```

Lobe MyGo needs Go 1.27 or later and builds on [MyGo](https://github.com/egoist/mygo).

```go
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

var count int

func view(c *ui.Context) {
	theme.Apply(c)
	ui.Column(c).Fill().Center().Gap(12).Background(theme.TokensOf(c).BgContainer).Children(func() {
		if action.Button(c, "Hello Lobe", action.ButtonProps{Variant: action.Primary}).Clicked() {
			count++
		}
		ui.Text(c, fmt.Sprintf("Clicked %d times", count))
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Hello Lobe", Width: 480, Height: 320, Content: ui.View(view)})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
```

Call `theme.Apply(c)` at the top of each view, then read design tokens with `theme.TokensOf(c)`.

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## 📦 Packages

| Package        | Components                                                                                                                              |
| -------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| `pkg/action`   | Button, ActionIcon, ActionIconGroup, CopyButton, Segmented                                                                              |
| `pkg/field`    | Input, TextArea, InputNumber, InputPassword, InputOTP, Select, AutoComplete, SearchBar, HotkeyInput, EditableText, DatePicker, Calendar, Upload |
| `pkg/choice`   | Checkbox, RadioGroup, Switch, Slider, Rate, ToggleGroup, ColorSwatches, ColorPicker, ImageSelect                                        |
| `pkg/display`  | Text, Tag, Avatar, GroupAvatar, AgentAvatar, Icon, Hotkey, Image, List, Accordion, Empty, Skeleton                                      |
| `pkg/feedback` | Alert, Badge, Progress, Spin, Result, Modal, Drawer, Toast, FloatingPanel, FloatingSheet                                                |
| `pkg/overlay`  | Tooltip, Popover, DropdownMenu, ContextMenu                                                                                             |
| `pkg/nav`      | Tabs, Pagination, Breadcrumb, Steps, Burger                                                                                             |
| `pkg/layout`   | Block, Grid, Divider, ScrollArea, DraggablePanel, Header, SideNav, DraggableSideNav, Layout, GuideCard, SortableList                    |
| `pkg/code`     | Highlighter, Snippet, QRCode                                                                                                            |
| `pkg/markdown` | Markdown, with streaming                                                                                                                |
| `pkg/theme`    | Light and dark tokens, sizes, shadows                                                                                                   |
| `pkg/motion`   | Springs, easings, transitions                                                                                                           |
| `pkg/lucide`   | Lucide icons                                                                                                                            |

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## 📖 Documentation

The docs site shows every component as a live MyGo demo, compiled to WebAssembly and drawn in the browser, next to an API table generated from its Go `Props` struct.

```bash
$ bun install
$ bun run dev
```

- **Docs site**: [mygo.lobehub.com](https://mygo.lobehub.com), starting with [Getting Started](https://mygo.lobehub.com/getting-started)
- **Component pages and demos**: [`catalog/`](./catalog)
- **Changelog**: [CHANGELOG.md](./CHANGELOG.md)

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## 💬 Community

Questions, ideas and showcases are all welcome:

- [Discord][discord-link]
- [GitHub Issues][github-issues-link]

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## ⌨️ Local Development

Clone it for local development (Go >= 1.27 and Bun):

```bash
$ git clone https://github.com/lobehub/lobe-mygo.git
$ cd lobe-mygo
$ go test ./...
$ go run ./catalog            # native catalog window
$ go run ./catalog button/basic
$ bun install && bun run dev  # docs site with WebAssembly demos
```

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## 🤝 Contributing

Contributions of all types are more than welcome, if you are interested in contributing code, feel free to check out our GitHub [Issues][github-issues-link] to get stuck in to show us what you’re made of.

[![][pr-welcome-shield]][pr-welcome-link]

[![][contributors-contrib]][contributors-link]

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## 🩷 Sponsor

Every bit counts and your one-time donation sparkles in our galaxy of support! You're a shooting star, making a swift and bright impact on our journey. Thank you for believing in us – your generosity guides us toward our mission, one brilliant flash at a time.

<a href="https://opencollective.com/lobehub" target="_blank">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://github.com/lobehub/.github/blob/main/static/sponsor-dark.png?raw=true">
    <img  src="https://github.com/lobehub/.github/blob/main/static/sponsor-light.png?raw=true">
  </picture>
</a>

<div align="right">

[![][back-to-top]](#readme-top)

</div>

## 🔗 Links

- **[🍭 Lobe UI](https://github.com/lobehub/lobe-ui)** - React component library for AI apps
- **[🐹 MyGo](https://github.com/egoist/mygo)** - Desktop apps in Go, with native UI or a web frontend
- **[🥨 Lobe Icons](https://github.com/lobehub/lobe-icons)** - AI / LLM brand logos and icons
- **[✒️ Lobe Editor](https://github.com/lobehub/lobe-editor)** - Lexical-based rich text editor for React, with Markdown, plugins and a chat composer
- **[😀 Fluent Emoji](https://github.com/lobehub/fluent-emoji)** - Fluent Emoji assets
- **[📖 Docs Kit](https://github.com/lobehub/docs-kit)** - Reusable documentation toolkit and lobedocs CLI for LobeHub component libraries

<div align="right">

[![][back-to-top]](#readme-top)

</div>

---

Copyright © 2026 [LobeHub][profile-link]. <br />
This project is [MIT](./LICENSE) licensed.

<!-- LINK GROUP -->

[back-to-top]: https://img.shields.io/badge/-BACK_TO_TOP-151515?style=flat-square
[contributors-contrib]: https://contrib.rocks/image?repo=lobehub/lobe-mygo
[contributors-link]: https://github.com/lobehub/lobe-mygo/graphs/contributors
[discord-link]: https://discord.gg/AYFPHvv2jT
[discord-shield]: https://img.shields.io/discord/1127171173982154893?color=5865F2&label=discord&labelColor=black&logo=discord&logoColor=white&style=flat-square
[github-contributors-link]: https://github.com/lobehub/lobe-mygo/graphs/contributors
[github-contributors-shield]: https://img.shields.io/github/contributors/lobehub/lobe-mygo?color=c4f042&labelColor=black&style=flat-square
[github-forks-link]: https://github.com/lobehub/lobe-mygo/network/members
[github-forks-shield]: https://img.shields.io/github/forks/lobehub/lobe-mygo?color=8ae8ff&labelColor=black&style=flat-square
[github-issues-link]: https://github.com/lobehub/lobe-mygo/issues
[github-issues-shield]: https://img.shields.io/github/issues/lobehub/lobe-mygo?color=ff80eb&labelColor=black&style=flat-square
[github-license-link]: https://github.com/lobehub/lobe-mygo/blob/main/LICENSE
[github-license-shield]: https://img.shields.io/github/license/lobehub/lobe-mygo?color=white&labelColor=black&style=flat-square
[github-stars-link]: https://github.com/lobehub/lobe-mygo/stargazers
[github-stars-shield]: https://img.shields.io/github/stars/lobehub/lobe-mygo?color=ffcb47&labelColor=black&style=flat-square
[go-reference-link]: https://pkg.go.dev/github.com/lobehub/lobe-mygo
[go-reference-shield]: https://img.shields.io/badge/go-reference-00ADD8?labelColor=black&logo=go&logoColor=white&style=flat-square
[go-version-link]: ./go.mod
[go-version-shield]: https://img.shields.io/github/go-mod/go-version/lobehub/lobe-mygo?color=00ADD8&labelColor=black&logo=go&logoColor=white&style=flat-square
[pr-welcome-link]: https://github.com/lobehub/lobe-mygo/pulls
[pr-welcome-shield]: https://img.shields.io/badge/🤯_pr_welcome-%E2%86%92-ffcb47?labelColor=black&style=for-the-badge
[profile-link]: https://github.com/lobehub
