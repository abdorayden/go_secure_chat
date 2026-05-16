// Copyright (c) 2026 abdenour souane. All Rights Reserved.

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

	state := newAppState(&w)
	if err := state.Connect(&w, addr); err != nil {
		state.Session.ErrorText = err.Error()
		state.Session.StatusText = "Disconnected"
	}

	var ops op.Ops
	th := material.NewTheme()
	applyTheme(th, state.ActivePalette())

	for {
		switch ev := w.Event().(type) {
		case app.DestroyEvent:
			state.Close()
			return ev.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, ev)
			state.DrainEvents()
			applyTheme(th, state.ActivePalette())
			if state.Session.Joined {
				layoutChat(gtx, th, state)
			} else {
				layoutAuth(gtx, th, state, &w, addr)
			}
			ev.Frame(gtx.Ops)
		}
	}
}
