package ui

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func textField(gtx layout.Context, th *material.Theme, editor *widget.Editor, hint string, fullWidth bool) layout.Dimensions {
	if fullWidth {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
	}
	return card(gtx, color.NRGBA{R: 0xF0, G: 0xF3, B: 0xFA, A: 0xFF}, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{
			Top:    unit.Dp(10),
			Bottom: unit.Dp(10),
			Left:   unit.Dp(14),
			Right:  unit.Dp(14),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			w := material.Editor(th, editor, hint)
			w.Color = th.Palette.Fg
			w.HintColor = color.NRGBA{R: 0x8A, G: 0x93, B: 0xA8, A: 0xFF}
			return w.Layout(gtx)
		})
	})
}

func card(gtx layout.Context, bg color.NRGBA, child layout.Widget) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := child(gtx)
	call := macro.Stop()
	r := image.Rectangle{Max: dims.Size}
	rr := clip.RRect{Rect: r, NE: 12, NW: 12, SE: 12, SW: 12}
	paint.FillShape(gtx.Ops, bg, rr.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

func applyTheme(th *material.Theme) {
	th.Palette = material.Palette{
		Bg:         color.NRGBA{R: 0xF0, G: 0xF3, B: 0xFA, A: 0xFF},
		Fg:         color.NRGBA{R: 0x10, G: 0x14, B: 0x20, A: 0xFF},
		ContrastBg: color.NRGBA{R: 0x1A, G: 0x5C, B: 0xF5, A: 0xFF},
		ContrastFg: color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF},
	}
}

func fill(gtx layout.Context, c color.NRGBA) {
	r := image.Rectangle{Max: gtx.Constraints.Max}
	paint.FillShape(gtx.Ops, c, clip.Rect(r).Op())
}

// divider draws a 1dp horizontal line.
func divider(gtx layout.Context, c color.NRGBA) layout.Dimensions {
	height := gtx.Dp(unit.Dp(1))
	r := image.Rectangle{Max: image.Pt(gtx.Constraints.Max.X, height)}
	paint.FillShape(gtx.Ops, c, clip.Rect(r).Op())
	return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, height)}
}
