package ui

import (
	"gioui.org/app"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

const appLabel = "Secure Chat"

func Run(addr string) error {
	var w app.Window
	w.Option(
		app.Title(appLabel),
		app.Size(unit.Dp(980), unit.Dp(660)),
		app.MinSize(unit.Dp(480), unit.Dp(400)),
	)

	state := newAppState()
	if err := state.Connect(&w, addr); err != nil {
		state.ErrorText = err.Error()
		state.StatusText = "Disconnected"
	}

	var ops op.Ops
	th := material.NewTheme()
	applyTheme(th)

	for {
		switch ev := w.Event().(type) {
		case app.DestroyEvent:
			state.Close()
			return ev.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, ev)
			state.DrainEvents()
			if state.Joined {
				layoutChat(gtx, th, state)
			} else {
				layoutAuth(gtx, th, state, &w, addr)
			}
			ev.Frame(gtx.Ops)
		}
	}
}
