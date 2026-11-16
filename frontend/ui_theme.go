package frontend

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

type uiSize int

const (
	sizeMiddle uiSize = iota
	sizeSmall
	sizeLarge
)

type uiElevation int

const (
	boxShadow uiElevation = iota
	boxShadowSecondary
	boxShadowTertiary
)

type shadowLayer struct{ y, blur, spread, light, dark float32 }

var shadowLayers = map[uiElevation][]shadowLayer{
	boxShadow: {
		{40, 80, 0, 0.06, 0.09}, {20, 40, 0, 0.05, 0.08}, {10, 20, 0, 0.04, 0.06},
		{5, 10, 0, 0.03, 0.05}, {2, 4, 0, 0.02, 0.03},
	},
	boxShadowSecondary: {
		{17.5, 23.4, 0, 0.04, 0.06}, {9.4, 12.5, 0, 0.03, 0.05}, {5.25, 7, 0, 0.02, 0.03},
		{2.8, 3.7, -2, 0.01, 0.02}, {1.2, 1.5, 0, 0.01, 0.02},
	},
	boxShadowTertiary: {{1, 2, 0, 0.04, 0.06}, {2, 4, 0, 0.02, 0.03}},
}

type uiTokens struct {
	Dark                   bool
	Blue                   ui.Color
	BlueActive             ui.Color
	BlueFillQuaternary     ui.Color
	BlueFillTertiary       ui.Color
	BgBase                 ui.Color
	BgBlur                 ui.Color
	BgContainer            ui.Color
	BgContainerDisabled    ui.Color
	BgContainerSecondary   ui.Color
	BgElevated             ui.Color
	BgLayout               ui.Color
	BgMask                 ui.Color
	BgSolid                ui.Color
	BgSolidActive          ui.Color
	BgSolidHover           ui.Color
	BgSpotlight            ui.Color
	BgTextActive           ui.Color
	BgTextHover            ui.Color
	Border                 ui.Color
	BorderBg               ui.Color
	BorderDisabled         ui.Color
	BorderSecondary        ui.Color
	Error                  ui.Color
	ErrorActive            ui.Color
	ErrorAffix             ui.Color
	ErrorBg                ui.Color
	ErrorBgActive          ui.Color
	ErrorBgFilledHover     ui.Color
	ErrorBgHover           ui.Color
	ErrorBorder            ui.Color
	ErrorBorderHover       ui.Color
	ErrorFill              ui.Color
	ErrorFillQuaternary    ui.Color
	ErrorFillSecondary     ui.Color
	ErrorFillTertiary      ui.Color
	ErrorHover             ui.Color
	ErrorOutline           ui.Color
	ErrorText              ui.Color
	ErrorTextActive        ui.Color
	ErrorTextHover         ui.Color
	Fill                   ui.Color
	FillAlter              ui.Color
	FillContent            ui.Color
	FillContentHover       ui.Color
	FillQuaternary         ui.Color
	FillSecondary          ui.Color
	FillTertiary           ui.Color
	Highlight              ui.Color
	Icon                   ui.Color
	IconHover              ui.Color
	Info                   ui.Color
	InfoActive             ui.Color
	InfoBg                 ui.Color
	InfoBgHover            ui.Color
	InfoBorder             ui.Color
	InfoBorderHover        ui.Color
	InfoFill               ui.Color
	InfoFillQuaternary     ui.Color
	InfoFillSecondary      ui.Color
	InfoFillTertiary       ui.Color
	InfoHover              ui.Color
	InfoText               ui.Color
	InfoTextActive         ui.Color
	InfoTextHover          ui.Color
	Link                   ui.Color
	LinkActive             ui.Color
	LinkHover              ui.Color
	Primary                ui.Color
	PrimaryActive          ui.Color
	PrimaryBg              ui.Color
	PrimaryBgHover         ui.Color
	PrimaryBorder          ui.Color
	PrimaryBorderHover     ui.Color
	PrimaryFill            ui.Color
	PrimaryFillQuaternary  ui.Color
	PrimaryFillSecondary   ui.Color
	PrimaryFillTertiary    ui.Color
	PrimaryHover           ui.Color
	PrimaryText            ui.Color
	PrimaryTextActive      ui.Color
	PrimaryTextHover       ui.Color
	Shadow                 ui.Color
	Split                  ui.Color
	Success                ui.Color
	SuccessActive          ui.Color
	SuccessBg              ui.Color
	SuccessBgHover         ui.Color
	SuccessBorder          ui.Color
	SuccessBorderHover     ui.Color
	SuccessFill            ui.Color
	SuccessFillQuaternary  ui.Color
	SuccessFillSecondary   ui.Color
	SuccessFillTertiary    ui.Color
	SuccessHover           ui.Color
	SuccessText            ui.Color
	SuccessTextActive      ui.Color
	SuccessTextHover       ui.Color
	Text                   ui.Color
	TextBase               ui.Color
	TextDescription        ui.Color
	TextDisabled           ui.Color
	TextHeading            ui.Color
	TextLabel              ui.Color
	TextLightSolid         ui.Color
	TextPlaceholder        ui.Color
	TextQuaternary         ui.Color
	TextSecondary          ui.Color
	TextTertiary           ui.Color
	Warning                ui.Color
	WarningActive          ui.Color
	WarningAffix           ui.Color
	WarningBg              ui.Color
	WarningBgHover         ui.Color
	WarningBorder          ui.Color
	WarningBorderHover     ui.Color
	WarningFill            ui.Color
	WarningFillQuaternary  ui.Color
	WarningFillSecondary   ui.Color
	WarningFillTertiary    ui.Color
	WarningHover           ui.Color
	WarningOutline         ui.Color
	WarningText            ui.Color
	WarningTextActive      ui.Color
	WarningTextHover       ui.Color
	White                  ui.Color
	Cyan                   ui.Color
	CyanActive             ui.Color
	CyanFillQuaternary     ui.Color
	CyanFillTertiary       ui.Color
	Geekblue               ui.Color
	Geekblue8              ui.Color
	Geekblue9              ui.Color
	GeekblueActive         ui.Color
	GeekblueFillQuaternary ui.Color
	GeekblueFillTertiary   ui.Color
	Gold                   ui.Color
	GoldActive             ui.Color
	GoldFillQuaternary     ui.Color
	GoldFillTertiary       ui.Color
	Gray                   ui.Color
	Gray10                 ui.Color
	Gray11                 ui.Color
	Gray7                  ui.Color
	Gray9                  ui.Color
	GrayActive             ui.Color
	GrayFillQuaternary     ui.Color
	GrayFillTertiary       ui.Color
	Green                  ui.Color
	GreenActive            ui.Color
	GreenFillQuaternary    ui.Color
	GreenFillTertiary      ui.Color
	Lime                   ui.Color
	LimeActive             ui.Color
	LimeFillQuaternary     ui.Color
	LimeFillTertiary       ui.Color
	Magenta                ui.Color
	Magenta8               ui.Color
	Magenta9               ui.Color
	MagentaActive          ui.Color
	MagentaFillQuaternary  ui.Color
	MagentaFillTertiary    ui.Color
	Orange                 ui.Color
	Orange9                ui.Color
	OrangeActive           ui.Color
	OrangeFillQuaternary   ui.Color
	OrangeFillTertiary     ui.Color
	Purple                 ui.Color
	Purple8                ui.Color
	Purple9                ui.Color
	PurpleActive           ui.Color
	PurpleFillQuaternary   ui.Color
	PurpleFillTertiary     ui.Color
	Red                    ui.Color
	RedActive              ui.Color
	RedFillQuaternary      ui.Color
	RedFillTertiary        ui.Color
	Volcano                ui.Color
	Volcano8               ui.Color
	Volcano9               ui.Color
	VolcanoActive          ui.Color
	VolcanoFillQuaternary  ui.Color
	VolcanoFillTertiary    ui.Color
	Yellow                 ui.Color
	Yellow9                ui.Color
	YellowActive           ui.Color
	YellowFillQuaternary   ui.Color
	YellowFillTertiary     ui.Color
	BorderRadius           float32
	BorderRadiusLG         float32
	BorderRadiusSM         float32
	BorderRadiusXS         float32
	ControlHeight          float32
	ControlHeightLG        float32
	ControlHeightSM        float32
	ControlHeightXS        float32
	FontSize               float32
	FontSizeHeading1       float32
	FontSizeHeading2       float32
	FontSizeHeading3       float32
	FontSizeHeading4       float32
	FontSizeHeading5       float32
	FontSizeLG             float32
	FontSizeSM             float32
	FontSizeXL             float32
	FontWeightStrong       float32
	LineWidth              float32
}

var lightTokens = uiTokens{
	Dark:                   false,
	Blue:                   ui.RGB(87, 171, 249),
	BlueActive:             ui.RGB(13, 120, 206),
	BlueFillQuaternary:     ui.RGBA(105, 105, 255, 0.02),
	BlueFillTertiary:       ui.RGBA(38, 138, 255, 0.06),
	BgBase:                 ui.RGB(255, 255, 255),
	BgBlur:                 ui.RGBA(0, 0, 0, 0),
	BgContainer:            ui.RGB(255, 255, 255),
	BgContainerDisabled:    ui.RGBA(0, 0, 0, 0.03),
	BgContainerSecondary:   ui.RGB(251, 251, 251),
	BgElevated:             ui.RGB(255, 255, 255),
	BgLayout:               ui.RGB(248, 248, 248),
	BgMask:                 ui.RGBA(0, 0, 0, 0.44),
	BgSolid:                ui.RGB(0, 0, 0),
	BgSolidActive:          ui.RGBA(0, 0, 0, 0.95),
	BgSolidHover:           ui.RGBA(0, 0, 0, 0.75),
	BgSpotlight:            ui.RGB(221, 221, 221),
	BgTextActive:           ui.RGBA(0, 0, 0, 0.12),
	BgTextHover:            ui.RGBA(0, 0, 0, 0.06),
	Border:                 ui.RGB(227, 227, 227),
	BorderBg:               ui.RGB(255, 255, 255),
	BorderDisabled:         ui.RGB(217, 217, 217),
	BorderSecondary:        ui.RGB(238, 238, 238),
	Error:                  ui.RGB(243, 88, 55),
	ErrorActive:            ui.RGB(206, 70, 30),
	ErrorAffix:             ui.RGB(243, 88, 55),
	ErrorBg:                ui.RGB(254, 248, 247),
	ErrorBgActive:          ui.RGB(255, 204, 199),
	ErrorBgFilledHover:     ui.RGB(255, 223, 220),
	ErrorBgHover:           ui.RGB(255, 240, 238),
	ErrorBorder:            ui.RGB(255, 222, 216),
	ErrorBorderHover:       ui.RGB(255, 231, 228),
	ErrorFill:              ui.RGBA(255, 49, 11, 0.16),
	ErrorFillQuaternary:    ui.RGBA(230, 80, 55, 0.04),
	ErrorFillSecondary:     ui.RGBA(255, 37, 10, 0.11),
	ErrorFillTertiary:      ui.RGBA(255, 41, 12, 0.07),
	ErrorHover:             ui.RGB(255, 119, 92),
	ErrorOutline:           ui.RGBA(230, 80, 55, 0.04),
	ErrorText:              ui.RGB(206, 70, 30),
	ErrorTextActive:        ui.RGB(206, 70, 30),
	ErrorTextHover:         ui.RGB(255, 119, 92),
	Fill:                   ui.RGBA(0, 0, 0, 0.12),
	FillAlter:              ui.RGBA(0, 0, 0, 0.015),
	FillContent:            ui.RGBA(0, 0, 0, 0.06),
	FillContentHover:       ui.RGBA(0, 0, 0, 0.12),
	FillQuaternary:         ui.RGBA(0, 0, 0, 0.015),
	FillSecondary:          ui.RGBA(0, 0, 0, 0.06),
	FillTertiary:           ui.RGBA(0, 0, 0, 0.03),
	Highlight:              ui.RGB(243, 88, 55),
	Icon:                   ui.RGB(138, 138, 138),
	IconHover:              ui.RGB(31, 31, 31),
	Info:                   ui.RGB(0, 118, 227),
	InfoActive:             ui.RGB(0, 96, 204),
	InfoBg:                 ui.RGB(248, 249, 255),
	InfoBgHover:            ui.RGB(237, 244, 255),
	InfoBorder:             ui.RGB(215, 230, 255),
	InfoBorderHover:        ui.RGB(227, 237, 255),
	InfoFill:               ui.RGBA(5, 99, 255, 0.16),
	InfoFillQuaternary:     ui.RGBA(22, 55, 255, 0.03),
	InfoFillSecondary:      ui.RGBA(0, 91, 255, 0.11),
	InfoFillTertiary:       ui.RGBA(30, 117, 255, 0.08),
	InfoHover:              ui.RGB(44, 145, 255),
	InfoText:               ui.RGB(0, 96, 204),
	InfoTextActive:         ui.RGB(0, 96, 204),
	InfoTextHover:          ui.RGB(44, 145, 255),
	Link:                   ui.RGB(0, 96, 204),
	LinkActive:             ui.RGB(0, 96, 204),
	LinkHover:              ui.RGB(44, 145, 255),
	Primary:                ui.RGB(34, 34, 34),
	PrimaryActive:          ui.RGB(17, 17, 17),
	PrimaryBg:              ui.RGB(245, 245, 245),
	PrimaryBgHover:         ui.RGB(238, 238, 238),
	PrimaryBorder:          ui.RGB(170, 170, 170),
	PrimaryBorderHover:     ui.RGB(204, 204, 204),
	PrimaryFill:            ui.RGBA(0, 0, 0, 0.36),
	PrimaryFillQuaternary:  ui.RGBA(0, 0, 0, 0.08),
	PrimaryFillSecondary:   ui.RGBA(0, 0, 0, 0.22),
	PrimaryFillTertiary:    ui.RGBA(0, 0, 0, 0.16),
	PrimaryHover:           ui.RGB(51, 51, 51),
	PrimaryText:            ui.RGB(34, 34, 34),
	PrimaryTextActive:      ui.RGB(17, 17, 17),
	PrimaryTextHover:       ui.RGB(51, 51, 51),
	Shadow:                 ui.RGB(0, 0, 0),
	Split:                  ui.RGBA(12, 12, 12, 0.07),
	Success:                ui.RGB(0, 160, 63),
	SuccessActive:          ui.RGB(0, 134, 78),
	SuccessBg:              ui.RGB(248, 251, 241),
	SuccessBgHover:         ui.RGB(238, 246, 230),
	SuccessBorder:          ui.RGB(212, 236, 204),
	SuccessBorderHover:     ui.RGB(225, 241, 217),
	SuccessFill:            ui.RGBA(40, 160, 0, 0.2),
	SuccessFillQuaternary:  ui.RGBA(138, 188, 22, 0.06),
	SuccessFillSecondary:   ui.RGBA(55, 162, 2, 0.15),
	SuccessFillTertiary:    ui.RGBA(85, 165, 5, 0.1),
	SuccessHover:           ui.RGB(48, 181, 86),
	SuccessText:            ui.RGB(0, 134, 78),
	SuccessTextActive:      ui.RGB(0, 134, 78),
	SuccessTextHover:       ui.RGB(48, 181, 86),
	Text:                   ui.RGB(31, 31, 31),
	TextBase:               ui.RGB(0, 0, 0),
	TextDescription:        ui.RGB(138, 138, 138),
	TextDisabled:           ui.RGB(176, 176, 176),
	TextHeading:            ui.RGB(31, 31, 31),
	TextLabel:              ui.RGB(102, 102, 102),
	TextLightSolid:         ui.RGB(255, 255, 255),
	TextPlaceholder:        ui.RGB(176, 176, 176),
	TextQuaternary:         ui.RGB(176, 176, 176),
	TextSecondary:          ui.RGB(102, 102, 102),
	TextTertiary:           ui.RGB(138, 138, 138),
	Warning:                ui.RGB(254, 180, 0),
	WarningActive:          ui.RGB(215, 143, 0),
	WarningAffix:           ui.RGB(254, 180, 0),
	WarningBg:              ui.RGB(255, 248, 243),
	WarningBgHover:         ui.RGB(255, 242, 226),
	WarningBorder:          ui.RGB(255, 225, 179),
	WarningBorderHover:     ui.RGB(255, 233, 204),
	WarningFill:            ui.RGBA(255, 155, 2, 0.3),
	WarningFillQuaternary:  ui.RGBA(255, 115, 15, 0.05),
	WarningFillSecondary:   ui.RGBA(255, 145, 0, 0.2),
	WarningFillTertiary:    ui.RGBA(255, 147, 13, 0.12),
	WarningHover:           ui.RGB(255, 190, 43),
	WarningOutline:         ui.RGBA(255, 115, 15, 0.05),
	WarningText:            ui.RGB(215, 143, 0),
	WarningTextActive:      ui.RGB(215, 143, 0),
	WarningTextHover:       ui.RGB(255, 190, 43),
	White:                  ui.RGB(255, 255, 255),
	Cyan:                   ui.RGB(149, 243, 217),
	CyanActive:             ui.RGB(47, 162, 138),
	CyanFillQuaternary:     ui.RGBA(55, 255, 122, 0.03),
	CyanFillTertiary:       ui.RGBA(26, 255, 155, 0.07),
	Geekblue:               ui.RGB(0, 114, 245),
	Geekblue8:              ui.RGB(76, 144, 255),
	Geekblue9:              ui.RGB(0, 114, 245),
	GeekblueActive:         ui.RGB(0, 90, 224),
	GeekblueFillQuaternary: ui.RGBA(105, 55, 255, 0.02),
	GeekblueFillTertiary:   ui.RGBA(22, 88, 255, 0.06),
	Gold:                   ui.RGB(238, 158, 11),
	GoldActive:             ui.RGB(213, 147, 0),
	GoldFillQuaternary:     ui.RGBA(255, 145, 55, 0.04),
	GoldFillTertiary:       ui.RGBA(255, 155, 55, 0.08),
	Gray:                   ui.RGB(136, 136, 136),
	Gray10:                 ui.RGB(102, 102, 102),
	Gray11:                 ui.RGB(51, 51, 51),
	Gray7:                  ui.RGB(170, 170, 170),
	Gray9:                  ui.RGB(136, 136, 136),
	GrayActive:             ui.RGB(102, 102, 102),
	GrayFillQuaternary:     ui.RGBA(0, 0, 0, 0.03),
	GrayFillTertiary:       ui.RGBA(0, 0, 0, 0.06),
	Green:                  ui.RGB(55, 157, 74),
	GreenActive:            ui.RGB(0, 121, 68),
	GreenFillQuaternary:    ui.RGBA(117, 230, 5, 0.06),
	GreenFillTertiary:      ui.RGBA(84, 205, 12, 0.1),
	Lime:                   ui.RGB(196, 240, 66),
	LimeActive:             ui.RGB(118, 157, 0),
	LimeFillQuaternary:     ui.RGBA(242, 255, 5, 0.08),
	LimeFillTertiary:       ui.RGBA(218, 255, 11, 0.16),
	Magenta:                ui.RGB(227, 75, 169),
	Magenta8:               ui.RGB(239, 103, 186),
	Magenta9:               ui.RGB(227, 75, 169),
	MagentaActive:          ui.RGB(163, 36, 102),
	MagentaFillQuaternary:  ui.RGBA(255, 55, 105, 0.04),
	MagentaFillTertiary:    ui.RGBA(255, 22, 133, 0.09),
	Orange:                 ui.RGB(248, 140, 19),
	Orange9:                ui.RGB(248, 140, 19),
	OrangeActive:           ui.RGB(209, 128, 0),
	OrangeFillQuaternary:   ui.RGBA(255, 105, 255, 0.03),
	OrangeFillTertiary:     ui.RGBA(255, 80, 30, 0.08),
	Purple:                 ui.RGB(189, 84, 198),
	Purple8:                ui.RGB(203, 110, 210),
	Purple9:                ui.RGB(189, 84, 198),
	PurpleActive:           ui.RGB(137, 43, 138),
	PurpleFillQuaternary:   ui.RGBA(255, 30, 155, 0.04),
	PurpleFillTertiary:     ui.RGBA(255, 15, 235, 0.08),
	Red:                    ui.RGB(244, 65, 108),
	RedActive:              ui.RGB(214, 34, 93),
	RedFillQuaternary:      ui.RGBA(255, 55, 255, 0.02),
	RedFillTertiary:        ui.RGBA(255, 15, 15, 0.05),
	Volcano:                ui.RGB(236, 94, 65),
	Volcano8:               ui.RGB(251, 116, 90),
	Volcano9:               ui.RGB(236, 94, 65),
	VolcanoActive:          ui.RGB(165, 55, 22),
	VolcanoFillQuaternary:  ui.RGBA(255, 55, 30, 0.04),
	VolcanoFillTertiary:    ui.RGBA(255, 44, 11, 0.06),
	Yellow:                 ui.RGB(255, 239, 92),
	Yellow9:                ui.RGB(255, 239, 92),
	YellowActive:           ui.RGB(171, 152, 0),
	YellowFillQuaternary:   ui.RGBA(255, 155, 255, 0.01),
	YellowFillTertiary:     ui.RGBA(255, 105, 255, 0.02),
	BorderRadius:           8,
	BorderRadiusLG:         12,
	BorderRadiusSM:         6,
	BorderRadiusXS:         4,
	ControlHeight:          36,
	ControlHeightLG:        45,
	ControlHeightSM:        27,
	ControlHeightXS:        18,
	FontSize:               14,
	FontSizeHeading1:       38,
	FontSizeHeading2:       30,
	FontSizeHeading3:       24,
	FontSizeHeading4:       20,
	FontSizeHeading5:       16,
	FontSizeLG:             16,
	FontSizeSM:             12,
	FontSizeXL:             20,
	FontWeightStrong:       600,
	LineWidth:              1,
}

var darkTokens = uiTokens{
	Dark:                   true,
	Blue:                   ui.RGB(96, 177, 255),
	BlueActive:             ui.RGB(45, 138, 224),
	BlueFillQuaternary:     ui.RGBA(0, 44, 233, 0.09),
	BlueFillTertiary:       ui.RGBA(0, 88, 246, 0.26),
	BgBase:                 ui.RGB(0, 0, 0),
	BgBlur:                 ui.RGBA(255, 255, 255, 0.04),
	BgContainer:            ui.RGB(13, 13, 13),
	BgContainerDisabled:    ui.RGBA(255, 255, 255, 0.06),
	BgContainerSecondary:   ui.RGB(6, 6, 6),
	BgElevated:             ui.RGB(26, 26, 26),
	BgLayout:               ui.RGB(0, 0, 0),
	BgMask:                 ui.RGBA(0, 0, 0, 0.44),
	BgSolid:                ui.RGBA(255, 255, 255, 0.95),
	BgSolidActive:          ui.RGBA(255, 255, 255, 0.9),
	BgSolidHover:           ui.RGB(255, 255, 255),
	BgSpotlight:            ui.RGB(45, 45, 45),
	BgTextActive:           ui.RGBA(255, 255, 255, 0.16),
	BgTextHover:            ui.RGBA(255, 255, 255, 0.1),
	Border:                 ui.RGB(32, 32, 32),
	BorderBg:               ui.RGB(13, 13, 13),
	BorderDisabled:         ui.RGB(66, 66, 66),
	BorderSecondary:        ui.RGB(26, 26, 26),
	Error:                  ui.RGB(233, 83, 80),
	ErrorActive:            ui.RGB(190, 55, 56),
	ErrorAffix:             ui.RGB(233, 83, 80),
	ErrorBg:                ui.RGB(52, 11, 12),
	ErrorBgActive:          ui.RGB(91, 37, 38),
	ErrorBgFilledHover:     ui.RGB(68, 30, 31),
	ErrorBgHover:           ui.RGB(75, 20, 26),
	ErrorBorder:            ui.RGB(138, 51, 52),
	ErrorBorderHover:       ui.RGB(170, 63, 62),
	ErrorFill:              ui.RGBA(250, 78, 90, 0.4),
	ErrorFillQuaternary:    ui.RGBA(250, 100, 90, 0.1),
	ErrorFillSecondary:     ui.RGBA(250, 67, 87, 0.3),
	ErrorFillTertiary:      ui.RGBA(248, 52, 57, 0.21),
	ErrorHover:             ui.RGB(255, 168, 160),
	ErrorOutline:           ui.RGBA(242, 1, 7, 0.17),
	ErrorText:              ui.RGB(233, 83, 80),
	ErrorTextActive:        ui.RGB(190, 55, 56),
	ErrorTextHover:         ui.RGB(255, 168, 160),
	Fill:                   ui.RGBA(255, 255, 255, 0.16),
	FillAlter:              ui.RGBA(255, 255, 255, 0.02),
	FillContent:            ui.RGBA(255, 255, 255, 0.1),
	FillContentHover:       ui.RGBA(255, 255, 255, 0.16),
	FillQuaternary:         ui.RGBA(255, 255, 255, 0.02),
	FillSecondary:          ui.RGBA(255, 255, 255, 0.1),
	FillTertiary:           ui.RGBA(255, 255, 255, 0.06),
	Highlight:              ui.RGB(233, 83, 80),
	Icon:                   ui.RGB(143, 143, 143),
	IconHover:              ui.RGB(230, 230, 230),
	Info:                   ui.RGB(76, 179, 255),
	InfoActive:             ui.RGB(0, 145, 235),
	InfoBg:                 ui.RGB(0, 26, 53),
	InfoBgHover:            ui.RGB(0, 41, 73),
	InfoBorder:             ui.RGB(0, 83, 137),
	InfoBorderHover:        ui.RGB(0, 103, 169),
	InfoFill:               ui.RGBA(0, 150, 255, 0.38),
	InfoFillQuaternary:     ui.RGBA(60, 150, 240, 0.1),
	InfoFillSecondary:      ui.RGBA(0, 141, 252, 0.29),
	InfoFillTertiary:       ui.RGBA(0, 124, 252, 0.21),
	InfoHover:              ui.RGB(156, 218, 255),
	InfoText:               ui.RGB(76, 179, 255),
	InfoTextActive:         ui.RGB(0, 145, 235),
	InfoTextHover:          ui.RGB(156, 218, 255),
	Link:                   ui.RGB(76, 179, 255),
	LinkActive:             ui.RGB(0, 145, 235),
	LinkHover:              ui.RGB(156, 218, 255),
	Primary:                ui.RGB(238, 238, 238),
	PrimaryActive:          ui.RGB(204, 204, 204),
	PrimaryBg:              ui.RGB(17, 17, 17),
	PrimaryBgHover:         ui.RGB(51, 51, 51),
	PrimaryBorder:          ui.RGB(102, 102, 102),
	PrimaryBorderHover:     ui.RGB(136, 136, 136),
	PrimaryFill:            ui.RGBA(255, 255, 255, 0.22),
	PrimaryFillQuaternary:  ui.RGBA(255, 255, 255, 0.02),
	PrimaryFillSecondary:   ui.RGBA(255, 255, 255, 0.16),
	PrimaryFillTertiary:    ui.RGBA(255, 255, 255, 0.08),
	PrimaryHover:           ui.RGB(255, 255, 255),
	PrimaryText:            ui.RGB(238, 238, 238),
	PrimaryTextActive:      ui.RGB(204, 204, 204),
	PrimaryTextHover:       ui.RGB(255, 255, 255),
	Shadow:                 ui.RGB(0, 0, 0),
	Split:                  ui.RGBA(230, 230, 230, 0.06),
	Success:                ui.RGB(128, 201, 51),
	SuccessActive:          ui.RGB(103, 167, 10),
	SuccessBg:              ui.RGB(6, 31, 0),
	SuccessBgHover:         ui.RGB(16, 46, 0),
	SuccessBorder:          ui.RGB(49, 90, 9),
	SuccessBorderHover:     ui.RGB(62, 111, 6),
	SuccessFill:            ui.RGBA(112, 252, 0, 0.25),
	SuccessFillQuaternary:  ui.RGBA(143, 229, 57, 0.07),
	SuccessFillSecondary:   ui.RGBA(84, 242, 0, 0.19),
	SuccessFillTertiary:    ui.RGBA(46, 238, 0, 0.13),
	SuccessHover:           ui.RGB(186, 228, 140),
	SuccessText:            ui.RGB(128, 201, 51),
	SuccessTextActive:      ui.RGB(103, 167, 10),
	SuccessTextHover:       ui.RGB(186, 228, 140),
	Text:                   ui.RGB(230, 230, 230),
	TextBase:               ui.RGB(255, 255, 255),
	TextDescription:        ui.RGB(143, 143, 143),
	TextDisabled:           ui.RGB(107, 107, 107),
	TextHeading:            ui.RGB(230, 230, 230),
	TextLabel:              ui.RGB(180, 180, 180),
	TextLightSolid:         ui.RGB(255, 255, 255),
	TextPlaceholder:        ui.RGB(107, 107, 107),
	TextQuaternary:         ui.RGB(107, 107, 107),
	TextSecondary:          ui.RGB(180, 180, 180),
	TextTertiary:           ui.RGB(143, 143, 143),
	Warning:                ui.RGB(255, 190, 69),
	WarningActive:          ui.RGB(227, 154, 0),
	WarningAffix:           ui.RGB(255, 190, 69),
	WarningBg:              ui.RGB(37, 23, 0),
	WarningBgHover:         ui.RGB(60, 34, 0),
	WarningBorder:          ui.RGB(114, 70, 0),
	WarningBorderHover:     ui.RGB(141, 87, 0),
	WarningFill:            ui.RGBA(253, 147, 0, 0.32),
	WarningFillQuaternary:  ui.RGBA(233, 144, 22, 0.09),
	WarningFillSecondary:   ui.RGBA(250, 142, 0, 0.24),
	WarningFillTertiary:    ui.RGBA(247, 153, 0, 0.15),
	WarningHover:           ui.RGB(255, 221, 156),
	WarningOutline:         ui.RGBA(38, 23, 0, 0.97),
	WarningText:            ui.RGB(255, 190, 69),
	WarningTextActive:      ui.RGB(227, 154, 0),
	WarningTextHover:       ui.RGB(255, 221, 156),
	White:                  ui.RGB(255, 255, 255),
	Cyan:                   ui.RGB(149, 243, 217),
	CyanActive:             ui.RGB(85, 188, 164),
	CyanFillQuaternary:     ui.RGBA(0, 250, 150, 0.02),
	CyanFillTertiary:       ui.RGBA(0, 243, 200, 0.14),
	Geekblue:               ui.RGB(0, 114, 245),
	Geekblue8:              ui.RGB(0, 108, 243),
	Geekblue9:              ui.RGB(0, 114, 245),
	GeekblueActive:         ui.RGB(0, 101, 237),
	GeekblueFillQuaternary: ui.RGBA(0, 18, 245, 0.11),
	GeekblueFillTertiary:   ui.RGBA(0, 43, 253, 0.3),
	Gold:                   ui.RGB(255, 178, 36),
	GoldActive:             ui.RGB(234, 161, 0),
	GoldFillQuaternary:     ui.RGBA(233, 100, 0, 0.03),
	GoldFillTertiary:       ui.RGBA(247, 159, 0, 0.17),
	Gray:                   ui.RGB(119, 119, 119),
	Gray10:                 ui.RGB(170, 170, 170),
	Gray11:                 ui.RGB(221, 221, 221),
	Gray7:                  ui.RGB(102, 102, 102),
	Gray9:                  ui.RGB(119, 119, 119),
	GrayActive:             ui.RGB(102, 102, 102),
	GrayFillQuaternary:     ui.RGBA(255, 255, 255, 0.02),
	GrayFillTertiary:       ui.RGBA(255, 255, 255, 0.06),
	Green:                  ui.RGB(98, 196, 115),
	GreenActive:            ui.RGB(27, 141, 77),
	GreenFillQuaternary:    ui.RGBA(0, 250, 150, 0.02),
	GreenFillTertiary:      ui.RGBA(0, 242, 150, 0.12),
	Lime:                   ui.RGB(196, 240, 66),
	LimeActive:             ui.RGB(143, 184, 27),
	LimeFillQuaternary:     ui.RGBA(100, 200, 0, 0.02),
	LimeFillTertiary:       ui.RGBA(154, 254, 0, 0.13),
	Magenta:                ui.RGB(227, 75, 169),
	Magenta8:               ui.RGB(206, 62, 145),
	Magenta9:               ui.RGB(227, 75, 169),
	MagentaActive:          ui.RGB(184, 49, 123),
	MagentaFillQuaternary:  ui.RGBA(229, 0, 29, 0.07),
	MagentaFillTertiary:    ui.RGBA(252, 0, 81, 0.21),
	Orange:                 ui.RGB(255, 153, 39),
	Orange9:                ui.RGB(255, 153, 39),
	OrangeActive:           ui.RGB(232, 139, 0),
	OrangeFillQuaternary:   ui.RGBA(233, 100, 0, 0.03),
	OrangeFillTertiary:     ui.RGBA(241, 147, 0, 0.17),
	Purple:                 ui.RGB(189, 84, 198),
	Purple8:                ui.RGB(171, 70, 178),
	Purple9:                ui.RGB(189, 84, 198),
	PurpleActive:           ui.RGB(154, 57, 158),
	PurpleFillQuaternary:   ui.RGBA(217, 0, 183, 0.06),
	PurpleFillTertiary:     ui.RGBA(242, 0, 221, 0.19),
	Red:                    ui.RGB(244, 65, 108),
	RedActive:              ui.RGB(228, 49, 101),
	RedFillQuaternary:      ui.RGBA(250, 0, 50, 0.06),
	RedFillTertiary:        ui.RGBA(255, 0, 95, 0.22),
	Volcano:                ui.RGB(236, 94, 65),
	Volcano8:               ui.RGB(212, 81, 50),
	Volcano9:               ui.RGB(236, 94, 65),
	VolcanoActive:          ui.RGB(188, 68, 36),
	VolcanoFillQuaternary:  ui.RGBA(240, 20, 0, 0.05),
	VolcanoFillTertiary:    ui.RGBA(247, 53, 0, 0.19),
	Yellow:                 ui.RGB(255, 239, 92),
	Yellow9:                ui.RGB(255, 239, 92),
	YellowActive:           ui.RGB(199, 180, 38),
	YellowFillQuaternary:   ui.RGBA(250, 200, 0, 0.02),
	YellowFillTertiary:     ui.RGBA(247, 193, 0, 0.15),
	BorderRadius:           8,
	BorderRadiusLG:         12,
	BorderRadiusSM:         6,
	BorderRadiusXS:         4,
	ControlHeight:          36,
	ControlHeightLG:        45,
	ControlHeightSM:        27,
	ControlHeightXS:        18,
	FontSize:               14,
	FontSizeHeading1:       38,
	FontSizeHeading2:       30,
	FontSizeHeading3:       24,
	FontSizeHeading4:       20,
	FontSizeHeading5:       16,
	FontSizeLG:             16,
	FontSizeSM:             12,
	FontSizeXL:             20,
	FontWeightStrong:       600,
	LineWidth:              1,
}

func (t *uiTokens) Color(name string) (ui.Color, bool) {
	switch name {
	case "Blue":
		return t.Blue, true
	case "BlueActive":
		return t.BlueActive, true
	case "BlueFillQuaternary":
		return t.BlueFillQuaternary, true
	case "BlueFillTertiary":
		return t.BlueFillTertiary, true
	case "Cyan":
		return t.Cyan, true
	case "CyanActive":
		return t.CyanActive, true
	case "CyanFillQuaternary":
		return t.CyanFillQuaternary, true
	case "CyanFillTertiary":
		return t.CyanFillTertiary, true
	case "Geekblue":
		return t.Geekblue, true
	case "GeekblueActive":
		return t.GeekblueActive, true
	case "GeekblueFillQuaternary":
		return t.GeekblueFillQuaternary, true
	case "GeekblueFillTertiary":
		return t.GeekblueFillTertiary, true
	case "Gold":
		return t.Gold, true
	case "GoldActive":
		return t.GoldActive, true
	case "GoldFillQuaternary":
		return t.GoldFillQuaternary, true
	case "GoldFillTertiary":
		return t.GoldFillTertiary, true
	case "Gray":
		return t.Gray, true
	case "GrayActive":
		return t.GrayActive, true
	case "GrayFillQuaternary":
		return t.GrayFillQuaternary, true
	case "GrayFillTertiary":
		return t.GrayFillTertiary, true
	case "Green":
		return t.Green, true
	case "GreenActive":
		return t.GreenActive, true
	case "GreenFillQuaternary":
		return t.GreenFillQuaternary, true
	case "GreenFillTertiary":
		return t.GreenFillTertiary, true
	case "Lime":
		return t.Lime, true
	case "LimeActive":
		return t.LimeActive, true
	case "LimeFillQuaternary":
		return t.LimeFillQuaternary, true
	case "LimeFillTertiary":
		return t.LimeFillTertiary, true
	case "Magenta":
		return t.Magenta, true
	case "MagentaActive":
		return t.MagentaActive, true
	case "MagentaFillQuaternary":
		return t.MagentaFillQuaternary, true
	case "MagentaFillTertiary":
		return t.MagentaFillTertiary, true
	case "Orange":
		return t.Orange, true
	case "OrangeActive":
		return t.OrangeActive, true
	case "OrangeFillQuaternary":
		return t.OrangeFillQuaternary, true
	case "OrangeFillTertiary":
		return t.OrangeFillTertiary, true
	case "Purple":
		return t.Purple, true
	case "PurpleActive":
		return t.PurpleActive, true
	case "PurpleFillQuaternary":
		return t.PurpleFillQuaternary, true
	case "PurpleFillTertiary":
		return t.PurpleFillTertiary, true
	case "Red":
		return t.Red, true
	case "RedActive":
		return t.RedActive, true
	case "RedFillQuaternary":
		return t.RedFillQuaternary, true
	case "RedFillTertiary":
		return t.RedFillTertiary, true
	case "Volcano":
		return t.Volcano, true
	case "VolcanoActive":
		return t.VolcanoActive, true
	case "VolcanoFillQuaternary":
		return t.VolcanoFillQuaternary, true
	case "VolcanoFillTertiary":
		return t.VolcanoFillTertiary, true
	case "Yellow":
		return t.Yellow, true
	case "YellowActive":
		return t.YellowActive, true
	case "YellowFillQuaternary":
		return t.YellowFillQuaternary, true
	case "YellowFillTertiary":
		return t.YellowFillTertiary, true
	}
	return ui.Color{}, false
}

func lightPalette() *uiTokens {
	t := lightTokens
	return &t
}

func darkPalette() *uiTokens {
	t := darkTokens
	return &t
}

func mixColor(c, o ui.Color, t float32) ui.Color {
	t = max(0, min(t, 1))
	ca, oa := float64(c.A)/255, float64(o.A)/255
	a := ca*(1-float64(t)) + oa*float64(t)
	if a == 0 {
		return ui.Transparent
	}
	ch := func(x, y uint8) uint8 {
		v := (float64(x)*ca*(1-float64(t)) + float64(y)*oa*float64(t)) / a
		return uint8(math.Round(v))
	}
	return ui.Color{R: ch(c.R, o.R), G: ch(c.G, o.G), B: ch(c.B, o.B), A: uint8(math.Round(a * 255))}
}

func (t *uiTokens) FillBorder() ui.Color {
	if t.Dark {
		return t.FillSecondary
	}
	return t.Fill
}

func (t *uiTokens) FillBorderStrong() ui.Color {
	if t.Dark {
		return t.Fill
	}
	return mixColor(t.Fill, t.Text, 0.15)
}

func (t *uiTokens) Theme() *ui.Theme {
	return &ui.Theme{
		Dark:            t.Dark,
		Background:      t.BgLayout,
		Surface:         t.FillTertiary,
		SurfaceHover:    t.FillSecondary,
		SurfacePressed:  t.Fill,
		Border:          t.FillBorder(),
		Text:            t.Text,
		TextMuted:       t.TextSecondary,
		Accent:          t.Primary,
		AccentHover:     t.PrimaryHover,
		AccentPressed:   t.PrimaryActive,
		AccentText:      t.BgLayout,
		Danger:          t.Error,
		Warning:         t.Warning,
		Success:         t.Success,
		Selection:       t.Yellow9,
		Focus:           t.Info.Alpha(0.9),
		FocusRingWidth:  2,
		FocusRingOffset: 2,
		Scrollbar:       t.TextQuaternary,
		ScrollbarWidth:  6,
		Radius:          t.BorderRadius,
		Spacing:         4,
		FontSize:        t.FontSize,
	}
}

func (t *uiTokens) Elevate(e ui.Element, level uiElevation) ui.Element {
	for _, l := range shadowLayers[level] {
		a := l.light
		if t.Dark {
			a = l.dark
		}
		e.Shadow(0, l.y, l.blur, l.spread, ui.RGBA(0, 0, 0, a))
	}
	if t.Dark {
		return e.Shadow(0, 0, 0, 1, ui.RGBA(255, 255, 255, 0.18))
	}
	return e.Shadow(0, 0, 0, 1, ui.RGBA(0, 0, 0, 0.05))
}

var (
	uiLightTheme = lightTokens.Theme()
	uiDarkTheme  = darkTokens.Theme()
)

func applyUITheme(c *ui.Context) {
	if c.Theme().Dark {
		c.SetTheme(uiDarkTheme)
	} else {
		c.SetTheme(uiLightTheme)
	}
}

func tokensOf(c *ui.Context) *uiTokens {
	if c.Theme().Dark {
		return darkPalette()
	}
	return lightPalette()
}

// Motion curves, springs, and element transitions shared across native UI primitives.
var (
	softEase   = ui.CubicBezier(0.32, 0.72, 0, 1)
	exitEase   = ui.CubicBezier(0.4, 0, 1, 1)
	easeOut    = ui.CubicBezier(0.215, 0.61, 0.355, 1)
	cssEase    = ui.CubicBezier(0.25, 0.1, 0.25, 1)
	cssEaseIn  = ui.CubicBezier(0.42, 0, 1, 1)
	toastEase  = ui.CubicBezier(0.22, 1, 0.36, 1)
	switchThumbAnim = ui.Animation{Spring: ui.SpringOptions{Stiffness: 360, Damping: 24, Mass: 1}}
)

type motionSide int

const (
	sideBottom motionSide = iota
	sideTop
	sideLeft
	sideRight
)

func popupTransition() ui.ElementTransition {
	return ui.ElementTransition{
		Duration:     150 * time.Millisecond,
		Ease:         easeOut,
		ExitDuration: 75 * time.Millisecond,
		ExitEase:     cssEaseIn,
		Enter:        &ui.Motion{Scale: 0.96, Y: -6, Flip: true},
		Exit:         &ui.Motion{Scale: 0.96, Y: -6, Flip: true},
	}
}

func tooltipTransition() ui.ElementTransition {
	enter := bouncySpring(280 * time.Millisecond)
	return ui.ElementTransition{
		Animation:    &enter,
		ExitDuration: 100 * time.Millisecond,
		ExitEase:     exitEase,
		Enter:        &ui.Motion{Scale: 0.97, Y: -3, Flip: true},
		Exit:         &ui.Motion{Scale: 0.97, Y: -3, Flip: true},
	}
}

func modalTransition() ui.ElementTransition {
	return ui.ElementTransition{
		Duration:     220 * time.Millisecond,
		Ease:         softEase,
		ExitDuration: 120 * time.Millisecond,
		ExitEase:     exitEase,
		Enter:        &ui.Motion{Scale: 0.97},
		Exit:         &ui.Motion{Scale: 0.98},
	}
}

func backdropTransition() ui.ElementTransition {
	return ui.ElementTransition{
		Duration: 180 * time.Millisecond,
		Ease:     softEase,
		Enter:    &ui.Motion{},
		Exit:     &ui.Motion{},
	}
}

func toastTransition(from motionSide) ui.ElementTransition {
	offset := float32(150)
	if from == sideTop {
		offset = -150
	}
	return ui.ElementTransition{
		Duration: 400 * time.Millisecond,
		Ease:     toastEase,
		Enter:    &ui.Motion{YPercent: offset},
		Exit:     &ui.Motion{YPercent: offset},
	}
}

func collapseTransition() ui.ElementTransition {
	return ui.ElementTransition{
		Duration: 200 * time.Millisecond,
		Ease:     easeOut,
		Enter:    &ui.Motion{Collapse: true, Y: 6},
		Exit:     &ui.Motion{Collapse: true, Y: -6},
	}
}

func towardSide(tr ui.ElementTransition, side motionSide) ui.ElementTransition {
	turn := func(m *ui.Motion) *ui.Motion {
		if m == nil {
			return nil
		}
		t := *m
		switch side {
		case sideTop:
			t.Y = -m.Y
		case sideRight:
			t.X, t.Y = m.Y, 0
		case sideLeft:
			t.X, t.Y = -m.Y, 0
		}
		return &t
	}
	tr.Enter, tr.Exit = turn(tr.Enter), turn(tr.Exit)
	return tr
}

func bouncySpring(settle time.Duration) ui.Animation { return springAnim(0.7, 0.003, settle) }

func springAnim(zeta, threshold float64, settle time.Duration) ui.Animation {
	omega := unitSettle(zeta, threshold) / settle.Seconds()
	return ui.Animation{Spring: ui.SpringOptions{
		Stiffness: omega * omega,
		Damping:   2 * zeta * omega,
		Mass:      1,
	}}
}

func unitSettle(zeta, threshold float64) float64 {
	const step, horizon = 0.0005, 60.0
	last := 0.0
	for t := 0.0; t < horizon; t += step {
		if math.Abs(stepResponse(zeta, t)-1) > threshold {
			last = t
		}
	}
	return last
}

func stepResponse(zeta, t float64) float64 {
	if zeta >= 1 {
		return 1 - math.Exp(-t)*(1+t)
	}
	wd := math.Sqrt(1 - zeta*zeta)
	return 1 - math.Exp(-zeta*t)*(math.Cos(wd*t)+zeta/wd*math.Sin(wd*t))
}

func boolToFloat(b bool) float32 {
	if b {
		return 1
	}
	return 0
}
