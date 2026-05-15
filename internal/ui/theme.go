package ui

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget/material"
)

type Palette struct {
	AppBg          color.NRGBA
	SidebarBg      color.NRGBA
	SidebarCard    color.NRGBA
	SidebarActive  color.NRGBA
	SidebarText    color.NRGBA
	SidebarMuted   color.NRGBA
	MainBg         color.NRGBA
	Surface        color.NRGBA
	SurfaceAlt     color.NRGBA
	InputBg        color.NRGBA
	Border         color.NRGBA
	Accent         color.NRGBA
	AccentSoft     color.NRGBA
	TextPrimary    color.NRGBA
	TextMuted      color.NRGBA
	TextOnAccent   color.NRGBA
	IncomingBubble color.NRGBA
	OutgoingBubble color.NRGBA
	OutgoingText   color.NRGBA
	Success        color.NRGBA
	Danger         color.NRGBA
	ModalScrim     color.NRGBA
}

var DarkPalette = Palette{
	AppBg:          color.NRGBA{R: 0x09, G: 0x0F, B: 0x1D, A: 0xFF},
	SidebarBg:      color.NRGBA{R: 0x0B, G: 0x13, B: 0x26, A: 0xFF},
	SidebarCard:    color.NRGBA{R: 0x13, G: 0x1E, B: 0x36, A: 0xFF},
	SidebarActive:  color.NRGBA{R: 0x1B, G: 0x4E, B: 0xD8, A: 0xFF},
	SidebarText:    color.NRGBA{R: 0xF0, G: 0xF5, B: 0xFF, A: 0xFF},
	SidebarMuted:   color.NRGBA{R: 0x93, G: 0xA0, B: 0xBA, A: 0xFF},
	MainBg:         color.NRGBA{R: 0x0E, G: 0x15, B: 0x29, A: 0xFF},
	Surface:        color.NRGBA{R: 0x11, G: 0x1B, B: 0x31, A: 0xFF},
	SurfaceAlt:     color.NRGBA{R: 0x17, G: 0x22, B: 0x3D, A: 0xFF},
	InputBg:        color.NRGBA{R: 0x14, G: 0x20, B: 0x38, A: 0xFF},
	Border:         color.NRGBA{R: 0x1E, G: 0x2B, B: 0x49, A: 0xFF},
	Accent:         color.NRGBA{R: 0x2F, G: 0x6F, B: 0xFF, A: 0xFF},
	AccentSoft:     color.NRGBA{R: 0x1E, G: 0x38, B: 0x6E, A: 0xFF},
	TextPrimary:    color.NRGBA{R: 0xF2, G: 0xF6, B: 0xFF, A: 0xFF},
	TextMuted:      color.NRGBA{R: 0x9D, G: 0xA9, B: 0xC1, A: 0xFF},
	TextOnAccent:   color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF},
	IncomingBubble: color.NRGBA{R: 0x16, G: 0x22, B: 0x3A, A: 0xFF},
	OutgoingBubble: color.NRGBA{R: 0x2F, G: 0x6F, B: 0xFF, A: 0xFF},
	OutgoingText:   color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF},
	Success:        color.NRGBA{R: 0x2A, G: 0xD0, B: 0x93, A: 0xFF},
	Danger:         color.NRGBA{R: 0xE5, G: 0x51, B: 0x51, A: 0xFF},
	ModalScrim:     color.NRGBA{R: 0x03, G: 0x06, B: 0x10, A: 0xC8},
}

var LightPalette = Palette{
	AppBg:          color.NRGBA{R: 0xEB, G: 0xEF, B: 0xF7, A: 0xFF},
	SidebarBg:      color.NRGBA{R: 0x12, G: 0x1B, B: 0x31, A: 0xFF},
	SidebarCard:    color.NRGBA{R: 0x1A, G: 0x24, B: 0x40, A: 0xFF},
	SidebarActive:  color.NRGBA{R: 0x2F, G: 0x6F, B: 0xFF, A: 0xFF},
	SidebarText:    color.NRGBA{R: 0xF6, G: 0xF9, B: 0xFF, A: 0xFF},
	SidebarMuted:   color.NRGBA{R: 0xB4, G: 0xBF, B: 0xD5, A: 0xFF},
	MainBg:         color.NRGBA{R: 0xF4, G: 0xF7, B: 0xFC, A: 0xFF},
	Surface:        color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF},
	SurfaceAlt:     color.NRGBA{R: 0xF0, G: 0xF4, B: 0xFB, A: 0xFF},
	InputBg:        color.NRGBA{R: 0xF4, G: 0xF6, B: 0xFB, A: 0xFF},
	Border:         color.NRGBA{R: 0xD7, G: 0xDE, B: 0xEC, A: 0xFF},
	Accent:         color.NRGBA{R: 0x2F, G: 0x6F, B: 0xFF, A: 0xFF},
	AccentSoft:     color.NRGBA{R: 0xD9, G: 0xE7, B: 0xFF, A: 0xFF},
	TextPrimary:    color.NRGBA{R: 0x10, G: 0x16, B: 0x25, A: 0xFF},
	TextMuted:      color.NRGBA{R: 0x6C, G: 0x77, B: 0x8E, A: 0xFF},
	TextOnAccent:   color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF},
	IncomingBubble: color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF},
	OutgoingBubble: color.NRGBA{R: 0x2F, G: 0x6F, B: 0xFF, A: 0xFF},
	OutgoingText:   color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF},
	Success:        color.NRGBA{R: 0x1F, G: 0xB9, B: 0x7A, A: 0xFF},
	Danger:         color.NRGBA{R: 0xD4, G: 0x46, B: 0x46, A: 0xFF},
	ModalScrim:     color.NRGBA{R: 0x08, G: 0x10, B: 0x22, A: 0x88},
}

func applyTheme(th *material.Theme, palette Palette) {
	th.Palette = material.Palette{
		Bg:         palette.MainBg,
		Fg:         palette.TextPrimary,
		ContrastBg: palette.Accent,
		ContrastFg: palette.TextOnAccent,
	}
}

func fill(gtx layout.Context, c color.NRGBA) {
	r := image.Rectangle{Max: gtx.Constraints.Max}
	paint.FillShape(gtx.Ops, c, clip.Rect(r).Op())
}
