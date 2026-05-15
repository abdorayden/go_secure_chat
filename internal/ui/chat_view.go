package ui

import (
	"image"
	"image/color"
	"strings"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func layoutChat(gtx layout.Context, th *material.Theme, state *AppState) layout.Dimensions {
	fill(gtx, th.Palette.Bg)

	// Consume submit events before layout so they aren't lost.
	submittedText, submitted := consumeSubmit(gtx, &state.Input)

	return layout.Flex{Axis: layout.Vertical}.Layout(
		gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return headerBar(gtx, th, state)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return divider(gtx, color.NRGBA{R: 0xD8, G: 0xDF, B: 0xF0, A: 0xFF})
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return messageList(gtx, th, state)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return divider(gtx, color.NRGBA{R: 0xD8, G: 0xDF, B: 0xF0, A: 0xFF})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return inputBar(gtx, th, state, submitted, submittedText)
		}),
	)
}

func inputBar(gtx layout.Context, th *material.Theme, state *AppState, submitted bool, submittedText string) layout.Dimensions {
	bg := color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	fill(gtx, bg)

	return layout.Inset{
		Left:   unit.Dp(16),
		Right:  unit.Dp(16),
		Top:    unit.Dp(12),
		Bottom: unit.Dp(12),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(
			gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return textField(gtx, th, &state.Input, "Write a message…", false)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				btn := material.Button(th, &state.SendBtn, "Send")
				btn.Background = th.Palette.ContrastBg
				btn.CornerRadius = unit.Dp(10)
				dims := btn.Layout(gtx)

				// BUG FIX: determine send intent independently of submitted flag.
				// submitted carries text from SubmitEvent; button click uses current
				// editor text captured before layout clears it.
				shouldSend := submitted
				pendingText := submittedText
				for state.SendBtn.Clicked(gtx) {
					if !shouldSend {
						pendingText = strings.TrimSpace(state.Input.Text())
						shouldSend = true
					}
				}
				if shouldSend && pendingText != "" {
					if err := state.SendMessage(pendingText); err != nil {
						state.ErrorText = err.Error()
					}
				}
				return dims
			}),
		)
	})
}

// consumeSubmit drains ALL editor events and returns the text of the first
// SubmitEvent found.  Previous implementation returned early on any non-submit
// event, silently dropping it and causing subsequent submit events to be missed.
func consumeSubmit(gtx layout.Context, editor *widget.Editor) (string, bool) {
	text := ""
	found := false
	for {
		ev, ok := editor.Update(gtx)
		if !ok {
			break
		}
		if submit, ok := ev.(widget.SubmitEvent); ok && !found {
			text = submit.Text
			found = true
		}
	}
	return text, found
}

func headerBar(gtx layout.Context, th *material.Theme, state *AppState) layout.Dimensions {
	headerBg := color.NRGBA{R: 0x0D, G: 0x12, B: 0x24, A: 0xFF}
	height := gtx.Dp(unit.Dp(58))
	gtx.Constraints.Min.Y = height
	gtx.Constraints.Max.Y = height
	fill(gtx, headerBg)

	return layout.Inset{Left: unit.Dp(20), Right: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(
			gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				// Accent dot
				size := gtx.Dp(unit.Dp(8))
				dotColor := color.NRGBA{R: 0x1A, G: 0xF0, B: 0x9A, A: 0xFF}
				if !state.Connected {
					dotColor = color.NRGBA{R: 0xFF, G: 0x5A, B: 0x5A, A: 0xFF}
				}
				r := image.Rectangle{Max: image.Pt(size, size)}
				rr := clip.RRect{Rect: r, NE: size / 2, NW: size / 2, SE: size / 2, SW: size / 2}
				paint.FillShape(gtx.Ops, dotColor, rr.Op(gtx.Ops))
				return layout.Dimensions{Size: image.Pt(size, size)}
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, unit.Sp(17), state.Username+" · "+appLabel)
				lbl.Color = color.NRGBA{R: 0xF0, G: 0xF4, B: 0xFF, A: 0xFF}
				return lbl.Layout(gtx)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Dimensions{}
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				statusColor := color.NRGBA{R: 0x1A, G: 0xF0, B: 0x9A, A: 0xCC}
				if !state.Connected {
					statusColor = color.NRGBA{R: 0xFF, G: 0x6B, B: 0x6B, A: 0xCC}
				}
				lbl := material.Body2(th, state.StatusText)
				lbl.Color = statusColor
				return lbl.Layout(gtx)
			}),
		)
	})
}

func messageList(gtx layout.Context, th *material.Theme, state *AppState) layout.Dimensions {
	// Keep list scrolled to the bottom when new messages arrive.
	state.List.ScrollToEnd = true

	return layout.Inset{
		Left:   unit.Dp(16),
		Right:  unit.Dp(16),
		Top:    unit.Dp(14),
		Bottom: unit.Dp(14),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		maxWidth := gtx.Constraints.Max.X * 7 / 10
		return state.List.Layout(gtx, len(state.Messages), func(gtx layout.Context, i int) layout.Dimensions {
			msg := state.Messages[i]
			return layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return messageRow(gtx, th, msg, maxWidth)
			})
		})
	})
}

func messageRow(gtx layout.Context, th *material.Theme, msg Message, maxWidth int) layout.Dimensions {
	bubble := func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = maxWidth
		var bg, fg color.NRGBA
		if msg.FromMe {
			bg = color.NRGBA{R: 0x1A, G: 0x5C, B: 0xF5, A: 0xFF}
			fg = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
		} else {
			bg = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
			fg = color.NRGBA{R: 0x10, G: 0x14, B: 0x20, A: 0xFF}
		}
		return messageBubble(gtx, th, msg, bg, fg)
	}
	if msg.FromMe {
		return layout.Flex{Axis: layout.Horizontal}.Layout(
			gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
			layout.Rigid(bubble),
		)
	}
	return layout.Flex{Axis: layout.Horizontal}.Layout(
		gtx,
		layout.Rigid(bubble),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
	)
}

func messageBubble(gtx layout.Context, th *material.Theme, msg Message, bg, fg color.NRGBA) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{
		Top:    unit.Dp(10),
		Bottom: unit.Dp(10),
		Left:   unit.Dp(14),
		Right:  unit.Dp(14),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(
			gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body1(th, msg.Text)
				lbl.Color = fg
				return lbl.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				metaColor := fg
				metaColor.A = 0xAA
				meta := material.Caption(th, msg.Timestamp.Format("15:04")+"  "+msg.Status)
				meta.Color = metaColor
				return meta.Layout(gtx)
			}),
		)
	})
	call := macro.Stop()
	r := image.Rectangle{Max: dims.Size}
	// Asymmetric corners: flat corner on the sender's side (chat bubble tail effect).
	var rr clip.RRect
	if msg.FromMe {
		rr = clip.RRect{Rect: r, NE: 4, NW: 14, SE: 14, SW: 14}
	} else {
		rr = clip.RRect{Rect: r, NE: 14, NW: 4, SE: 14, SW: 14}
	}
	paint.FillShape(gtx.Ops, bg, rr.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}
