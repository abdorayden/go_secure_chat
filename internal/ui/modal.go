package ui

import (
	"strings"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

func layoutModal(gtx layout.Context, th *material.Theme, state *AppState, palette Palette) layout.Dimensions {
	fill(gtx, palette.ModalScrim)
	submittedText, submitted := consumeSubmit(gtx, &state.UI.Modal.ConfirmEditor)
	if submitted {
		state.UI.Modal.ConfirmEditor.SetText(submittedText)
		state.ConfirmModal()
	}

	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		if max := gtx.Dp(unit.Dp(520)); gtx.Constraints.Max.X > max {
			gtx.Constraints.Max.X = max
		}
		return card(gtx, palette.Surface, 22, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(22)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(
					gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						title := material.H5(th, state.UI.Modal.Title)
						title.Color = palette.TextPrimary
						return title.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						body := material.Body1(th, state.UI.Modal.Message)
						body.Color = palette.TextMuted
						return body.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if state.UI.Modal.Kind != "delete_account" {
							return layout.Dimensions{}
						}
						return textField(gtx, th, palette, &state.UI.Modal.ConfirmEditor, "Type DELETE", true)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal}.Layout(
							gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								btn := material.Button(th, &state.UI.Modal.CancelBtn, firstNonEmpty(state.UI.Modal.SecondaryLabel, "Cancel"))
								btn.Background = palette.SurfaceAlt
								btn.Color = palette.TextPrimary
								dims := btn.Layout(gtx)
								for state.UI.Modal.CancelBtn.Clicked(gtx) {
									state.CloseModal()
								}
								return dims
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								btn := material.Button(th, &state.UI.Modal.ConfirmBtn, firstNonEmpty(state.UI.Modal.PrimaryAction, "Close"))
								btn.Background = palette.Accent
								if state.UI.Modal.Kind == "delete_account" {
									btn.Background = palette.Danger
								}
								btn.Color = palette.TextOnAccent
								dims := btn.Layout(gtx)
								for state.UI.Modal.ConfirmBtn.Clicked(gtx) {
									state.ConfirmModal()
								}
								return dims
							}),
						)
					}),
				)
			})
		})
	})
}

func firstNonEmpty(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
