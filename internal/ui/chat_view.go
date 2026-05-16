// Copyright (c) 2026 abdenour souane. All Rights Reserved.

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
	palette := state.ActivePalette()
	applyTheme(th, palette)
	fill(gtx, palette.AppBg)

	submittedText, submitted := consumeSubmit(gtx, &state.Chat.Input)

	dims := layout.Flex{Axis: layout.Horizontal}.Layout(
		gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layoutSidebar(gtx, th, state, palette)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layoutMainChat(gtx, th, state, palette, submitted, submittedText)
		}),
	)

	handleSidebarActions(gtx, state)

	if state.UI.Modal.Visible {
		layoutModal(gtx, th, state, palette)
	}
	return dims
}

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

func layoutMainChat(gtx layout.Context, th *material.Theme, state *AppState, palette Palette, submitted bool, submittedText string) layout.Dimensions {
	return layout.Inset{Left: unit.Dp(16), Top: unit.Dp(16), Right: unit.Dp(16), Bottom: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return card(gtx, palette.MainBg, 28, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(
				gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return chatHeader(gtx, th, state, palette) }),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return messageCanvas(gtx, th, state, palette) }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return composer(gtx, th, state, palette, submitted, submittedText)
				}),
			)
		})
	})
}

func chatHeader(gtx layout.Context, th *material.Theme, state *AppState, palette Palette) layout.Dimensions {
	return layout.Inset{Left: unit.Dp(22), Right: unit.Dp(22), Top: unit.Dp(18), Bottom: unit.Dp(18)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(
			gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return avatarBadge(gtx, th, palette.Accent, "G") }),
			layout.Rigid(layout.Spacer{Width: unit.Dp(14)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(
					gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, unit.Sp(20), state.UI.ActiveRoom)
						lbl.Color = palette.TextPrimary
						return lbl.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						meta := material.Body2(th, state.UI.EstimatedMemberText+"  |  encrypted transport active")
						meta.Color = palette.TextMuted
						return meta.Layout(gtx)
					}),
				)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(
					gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return iconPillButton(gtx, th, palette, &state.UI.SearchBtn, "Search")
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return iconPillButton(gtx, th, palette, &state.UI.InfoBtn, "Info")
					}),
				)
			}),
		)
	})
}

func messageCanvas(gtx layout.Context, th *material.Theme, state *AppState, palette Palette) layout.Dimensions {
	return layout.Inset{Left: unit.Dp(18), Right: unit.Dp(18), Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return card(gtx, palette.Surface, 24, func(gtx layout.Context) layout.Dimensions {
			state.Chat.List.ScrollToEnd = true
			return layout.Inset{Left: unit.Dp(18), Right: unit.Dp(18), Top: unit.Dp(18), Bottom: unit.Dp(18)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				maxWidth := gtx.Constraints.Max.X * 68 / 100
				return state.Chat.List.Layout(gtx, len(state.Chat.Messages), func(gtx layout.Context, i int) layout.Dimensions {
					msg := state.Chat.Messages[i]
					return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return messageRow(gtx, th, palette, msg, maxWidth)
					})
				})
			})
		})
	})
}

func composer(gtx layout.Context, th *material.Theme, state *AppState, palette Palette, submitted bool, submittedText string) layout.Dimensions {
	return layout.Inset{Left: unit.Dp(18), Right: unit.Dp(18), Top: unit.Dp(6), Bottom: unit.Dp(18)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return card(gtx, palette.SurfaceAlt, 22, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(14), Right: unit.Dp(14), Top: unit.Dp(12), Bottom: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(
					gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return iconPillButton(gtx, th, palette, &state.UI.AttachBtn, "Attach")
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return textField(gtx, th, palette, &state.Chat.Input, "Write a secure message", false)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(th, &state.Chat.SendBtn, "Send")
						btn.Background = palette.Accent
						btn.Color = palette.TextOnAccent
						btn.CornerRadius = unit.Dp(16)
						dims := btn.Layout(gtx)
						shouldSend := submitted
						pendingText := submittedText
						for state.Chat.SendBtn.Clicked(gtx) {
							if !shouldSend {
								pendingText = strings.TrimSpace(state.Chat.Input.Text())
								shouldSend = true
							}
						}
						if shouldSend && pendingText != "" {
							if err := state.SendMessage(pendingText); err != nil {
								state.Session.ErrorText = err.Error()
							}
						}
						return dims
					}),
				)
			})
		})
	})
}

func messageRow(gtx layout.Context, th *material.Theme, palette Palette, msg Message, maxWidth int) layout.Dimensions {
	bubble := func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = maxWidth
		bg := palette.IncomingBubble
		fg := palette.TextPrimary
		if msg.FromMe {
			bg = palette.OutgoingBubble
			fg = palette.OutgoingText
		}
		return chatBubble(gtx, th, msg, bg, fg)
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

func chatBubble(gtx layout.Context, th *material.Theme, msg Message, bg, fg color.NRGBA) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(
			gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				sender := material.Body2(th, msg.Sender)
				sender.Color = fg
				sender.Font.Weight = 600
				return sender.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body1(th, msg.Body)
				lbl.Color = fg
				return lbl.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				metaColor := fg
				metaColor.A = 0xB8
				meta := material.Caption(th, msg.Timestamp.Format("15:04")+"  "+msg.Status)
				meta.Color = metaColor
				return meta.Layout(gtx)
			}),
		)
	})
	call := macro.Stop()
	r := image.Rectangle{Max: dims.Size}
	var rr clip.RRect
	if msg.FromMe {
		rr = clip.RRect{Rect: r, NE: 6, NW: 18, SE: 18, SW: 18}
	} else {
		rr = clip.RRect{Rect: r, NE: 18, NW: 6, SE: 18, SW: 18}
	}
	paint.FillShape(gtx.Ops, bg, rr.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

func avatarBadge(gtx layout.Context, th *material.Theme, bg color.NRGBA, label string) layout.Dimensions {
	size := gtx.Dp(unit.Dp(38))
	return card(gtx, bg, size/2, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = image.Pt(size, size)
		gtx.Constraints.Max = image.Pt(size, size)
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Body1(th, label)
			lbl.Color = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
			return lbl.Layout(gtx)
		})
	})
}
