package ui

import (
	"image/color"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func layoutAuth(gtx layout.Context, th *material.Theme, state *AppState, w *app.Window, addr string) layout.Dimensions {
	palette := state.ActivePalette()
	applyTheme(th, palette)
	fill(gtx, th.Palette.Bg)
	handleAuthModeToggle(gtx, state)
	submitted := consumeAuthSubmit(gtx, state)

	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		if gtx.Constraints.Max.X > gtx.Dp(unit.Dp(420)) {
			gtx.Constraints.Max.X = gtx.Dp(unit.Dp(420))
			gtx.Constraints.Min.X = gtx.Dp(unit.Dp(420))
		}

		return card(gtx, palette.Surface, 20, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(32)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				children := []layout.FlexChild{
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(
							gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								title := material.H4(th, appLabel)
								title.Color = palette.TextPrimary
								return title.Layout(gtx)
							}),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								label := "Dark"
								if state.UI.Theme == ThemeDark {
									label = "Light"
								}
								btn := material.Button(th, &state.Auth.ThemeToggleBtn, label)
								btn.Background = palette.SurfaceAlt
								btn.Color = palette.TextPrimary
								btn.CornerRadius = unit.Dp(10)
								dims := btn.Layout(gtx)
								for state.Auth.ThemeToggleBtn.Clicked(gtx) {
									state.ToggleTheme()
								}
								return dims
							}),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						sub := material.Body2(th, "End-to-end encrypted messaging")
						sub.Color = palette.TextMuted
						return sub.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(24)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return authTabs(gtx, th, state, palette)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
				}

				if state.Auth.SigningUp {
					children = append(children,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return textField(gtx, th, palette, &state.Auth.UsernameInput, "Username", true)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					)
				}

				children = append(children,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return textField(gtx, th, palette, &state.Auth.EmailInput, "Email", true)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return textField(gtx, th, palette, &state.Auth.PasswordInput, "Password", true)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return authStatus(gtx, th, state)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return submitAuthButton(gtx, th, state, submitted)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return reconnectButton(gtx, th, state, w, addr)
					}),
				)

				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
			})
		})
	})
}

func authTabs(gtx layout.Context, th *material.Theme, state *AppState, palette Palette) layout.Dimensions {
	tabBg := palette.SurfaceAlt
	return card(gtx, tabBg, 12, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(
				gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return authTabButton(gtx, th, &state.Auth.LoginTab, "Log In", !state.Auth.SigningUp, palette)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return authTabButton(gtx, th, &state.Auth.SignupTab, "Sign Up", state.Auth.SigningUp, palette)
				}),
			)
		})
	})
}

func authTabButton(gtx layout.Context, th *material.Theme, btn *widget.Clickable, label string, active bool, palette Palette) layout.Dimensions {
	w := material.Button(th, btn, label)
	w.CornerRadius = unit.Dp(8)
	if active {
		w.Background = palette.Surface
		w.Color = palette.Accent
	} else {
		w.Background = color.NRGBA{A: 0x00}
		w.Color = palette.TextMuted
	}
	return w.Layout(gtx)
}

func authStatus(gtx layout.Context, th *material.Theme, state *AppState) layout.Dimensions {
	palette := state.ActivePalette()
	text := state.Session.StatusText
	c := palette.TextMuted
	if state.Auth.Loading {
		text = "Working..."
		c = palette.Accent
	}
	if state.Session.ErrorText != "" {
		text = state.Session.ErrorText
		c = palette.Danger
	}
	lbl := material.Body2(th, text)
	lbl.Color = c
	return lbl.Layout(gtx)
}

func submitAuthButton(gtx layout.Context, th *material.Theme, state *AppState, submitted bool) layout.Dimensions {
	palette := state.ActivePalette()
	label := "Log In"
	if state.Auth.SigningUp {
		label = "Create Account"
	}
	btn := material.Button(th, &state.Auth.ConnectBtn, label)
	btn.Background = palette.Accent
	btn.Color = palette.TextOnAccent
	btn.CornerRadius = unit.Dp(10)
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	dims := btn.Layout(gtx)
	triggered := submitted
	for state.Auth.ConnectBtn.Clicked(gtx) {
		triggered = true
	}
	if triggered {
		submitAuthAction(state)
	}
	return dims
}

func reconnectButton(gtx layout.Context, th *material.Theme, state *AppState, w *app.Window, addr string) layout.Dimensions {
	palette := state.ActivePalette()
	btn := material.Button(th, &state.Auth.ReconnectBtn, "Reconnect to server")
	btn.Background = palette.SurfaceAlt
	btn.Color = palette.TextPrimary
	btn.CornerRadius = unit.Dp(10)
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	dims := btn.Layout(gtx)
	for state.Auth.ReconnectBtn.Clicked(gtx) {
		state.Session.ErrorText = ""
		state.Auth.Loading = false
		if err := state.Connect(w, addr); err != nil {
			state.Session.ErrorText = err.Error()
		}
	}
	return dims
}

func submitAuthAction(state *AppState) {
	if state.Auth.Loading {
		return
	}
	state.Auth.Loading = true
	state.Session.ErrorText = ""
	if err := state.SubmitAuth(); err != nil {
		state.Auth.Loading = false
		state.Session.ErrorText = err.Error()
	}
}

func consumeAuthSubmit(gtx layout.Context, state *AppState) bool {
	found := false
	if state.Auth.SigningUp && drainEditorForSubmit(gtx, &state.Auth.UsernameInput) {
		found = true
	}
	if drainEditorForSubmit(gtx, &state.Auth.EmailInput) {
		found = true
	}
	if drainEditorForSubmit(gtx, &state.Auth.PasswordInput) {
		found = true
	}
	return found
}

func drainEditorForSubmit(gtx layout.Context, editor *widget.Editor) bool {
	found := false
	for {
		ev, ok := editor.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.SubmitEvent); ok {
			found = true
		}
	}
	return found
}

func handleAuthModeToggle(gtx layout.Context, state *AppState) {
	for state.Auth.LoginTab.Clicked(gtx) {
		state.SwitchAuthMode(false)
	}
	for state.Auth.SignupTab.Clicked(gtx) {
		state.SwitchAuthMode(true)
	}
}
