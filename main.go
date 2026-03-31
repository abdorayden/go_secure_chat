package main

import (
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

func main() {
	go func() {
		var w app.Window
		w.Option(
			app.Title("Simple Window"),
			app.Size(unit.Dp(480), unit.Dp(320)),
		)
		if err := run(&w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func run(w *app.Window) error {
	var ops op.Ops
	th := material.NewTheme()
	for {
		e := w.Event()
		switch ev := e.(type) {
		case app.DestroyEvent:
			return ev.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, ev)
			layout.Center.Layout(gtx, material.H5(th, "Hello, Gio!").Layout)
			ev.Frame(gtx.Ops)
		}
	}
}
