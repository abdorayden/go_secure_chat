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

func textField(gtx layout.Context, th *material.Theme, palette Palette, editor *widget.Editor, hint string, fullWidth bool) layout.Dimensions {
	if fullWidth {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
	}
	return card(gtx, palette.InputBg, 14, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(11), Bottom: unit.Dp(11), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			w := material.Editor(th, editor, hint)
			w.Color = palette.TextPrimary
			w.HintColor = palette.TextMuted
			return w.Layout(gtx)
		})
	})
}

func card(gtx layout.Context, bg color.NRGBA, radius int, child layout.Widget) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := child(gtx)
	call := macro.Stop()
	r := image.Rectangle{Max: dims.Size}
	rr := clip.RRect{Rect: r, NE: radius, NW: radius, SE: radius, SW: radius}
	paint.FillShape(gtx.Ops, bg, rr.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

func divider(gtx layout.Context, c color.NRGBA) layout.Dimensions {
	height := gtx.Dp(unit.Dp(1))
	r := image.Rectangle{Max: image.Pt(gtx.Constraints.Max.X, height)}
	paint.FillShape(gtx.Ops, c, clip.Rect(r).Op())
	return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, height)}
}

func iconPillButton(gtx layout.Context, th *material.Theme, palette Palette, btn *widget.Clickable, label string) layout.Dimensions {
	b := material.Button(th, btn, label)
	b.Background = palette.SurfaceAlt
	b.Color = palette.TextPrimary
	b.CornerRadius = unit.Dp(12)
	return b.Layout(gtx)
}
