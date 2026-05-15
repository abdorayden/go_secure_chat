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
	fill(gtx, th.Palette.Bg)
	handleAuthModeToggle(gtx, state)
	submitted := consumeAuthSubmit(gtx, state)

	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// Constrain card width for readability on wide windows.
		if gtx.Constraints.Max.X > gtx.Dp(unit.Dp(420)) {
			gtx.Constraints.Max.X = gtx.Dp(unit.Dp(420))
			gtx.Constraints.Min.X = gtx.Dp(unit.Dp(420))
		}

		return card(gtx, color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(32)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				children := []layout.FlexChild{
					// App title
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						title := material.H4(th, appLabel)
						title.Color = color.NRGBA{R: 0x0D, G: 0x12, B: 0x24, A: 0xFF}
						return title.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						sub := material.Body2(th, "End-to-end encrypted messaging")
						sub.Color = color.NRGBA{R: 0x8A, G: 0x93, B: 0xA8, A: 0xFF}
						return sub.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(24)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return authTabs(gtx, th, state)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
				}

				if state.SigningUp {
					children = append(children,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return textField(gtx, th, &state.UsernameInput, "Username", true)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					)
				}

				children = append(children,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return textField(gtx, th, &state.EmailInput, "Email", true)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return textField(gtx, th, &state.PasswordInput, "Password", true)
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

func authTabs(gtx layout.Context, th *material.Theme, state *AppState) layout.Dimensions {
	// Tab bar background
	tabBg := color.NRGBA{R: 0xEE, G: 0xF1, B: 0xFA, A: 0xFF}
	return card(gtx, tabBg, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(
				gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return authTabButton(gtx, th, &state.LoginTab, "Log In", !state.SigningUp)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return authTabButton(gtx, th, &state.SignupTab, "Sign Up", state.SigningUp)
				}),
			)
		})
	})
}

func authTabButton(gtx layout.Context, th *material.Theme, btn *widget.Clickable, label string, active bool) layout.Dimensions {
	w := material.Button(th, btn, label)
	w.CornerRadius = unit.Dp(8)
	if active {
		w.Background = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
		w.Color = color.NRGBA{R: 0x1A, G: 0x5C, B: 0xF5, A: 0xFF}
	} else {
		w.Background = color.NRGBA{A: 0x00} // transparent
		w.Color = color.NRGBA{R: 0x8A, G: 0x93, B: 0xA8, A: 0xFF}
	}
	return w.Layout(gtx)
}

func authStatus(gtx layout.Context, th *material.Theme, state *AppState) layout.Dimensions {
	text := state.StatusText
	c := color.NRGBA{R: 0x55, G: 0x62, B: 0x74, A: 0xFF}
	if state.Loading {
		text = "Working…"
		c = color.NRGBA{R: 0x1A, G: 0x5C, B: 0xF5, A: 0xFF}
	}
	if state.ErrorText != "" {
		text = "⚠  " + state.ErrorText
		c = color.NRGBA{R: 0xC0, G: 0x20, B: 0x20, A: 0xFF}
	}
	lbl := material.Body2(th, text)
	lbl.Color = c
	return lbl.Layout(gtx)
}

func submitAuthButton(gtx layout.Context, th *material.Theme, state *AppState, submitted bool) layout.Dimensions {
	label := "Log In"
	if state.SigningUp {
		label = "Create Account"
	}
	btn := material.Button(th, &state.ConnectBtn, label)
	btn.Background = color.NRGBA{R: 0x1A, G: 0x5C, B: 0xF5, A: 0xFF}
	btn.CornerRadius = unit.Dp(10)
	// Make the button full-width.
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	dims := btn.Layout(gtx)
	triggered := submitted
	for state.ConnectBtn.Clicked(gtx) {
		triggered = true
	}
	if triggered {
		submitAuthAction(state)
	}
	return dims
}

func reconnectButton(gtx layout.Context, th *material.Theme, state *AppState, w *app.Window, addr string) layout.Dimensions {
	btn := material.Button(th, &state.ReconnectBtn, "Reconnect to server")
	btn.Background = color.NRGBA{R: 0xEE, G: 0xF1, B: 0xFA, A: 0xFF}
	btn.Color = color.NRGBA{R: 0x55, G: 0x62, B: 0x74, A: 0xFF}
	btn.CornerRadius = unit.Dp(10)
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	dims := btn.Layout(gtx)
	for state.ReconnectBtn.Clicked(gtx) {
		state.ErrorText = ""
		state.Loading = false
		if err := state.Connect(w, addr); err != nil {
			state.ErrorText = err.Error()
		}
	}
	return dims
}

func submitAuthAction(state *AppState) {
	if state.Loading {
		return
	}
	state.Loading = true
	state.ErrorText = ""
	if err := state.SubmitAuth(); err != nil {
		state.Loading = false
		state.ErrorText = err.Error()
	}
}

// consumeAuthSubmit drains ALL relevant editor events.
// BUG FIX: previous editorSubmitted returned false on the first non-submit
// event, leaving submit events queued and causing submit to be missed entirely
// when an editor had any change events before the submit event.
func consumeAuthSubmit(gtx layout.Context, state *AppState) bool {
	found := false
	if state.SigningUp {
		if drainEditorForSubmit(gtx, &state.UsernameInput) {
			found = true
		}
	}
	if drainEditorForSubmit(gtx, &state.EmailInput) {
		found = true
	}
	if drainEditorForSubmit(gtx, &state.PasswordInput) {
		found = true
	}
	return found
}

// drainEditorForSubmit consumes all pending events for an editor and returns
// true if any of them were a SubmitEvent.
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
	for state.LoginTab.Clicked(gtx) {
		state.SwitchAuthMode(false)
	}
	for state.SignupTab.Clicked(gtx) {
		state.SwitchAuthMode(true)
	}
}
