// Copyright (c) 2026 abdenour souane. All Rights Reserved.

package ui

import (
	"image/color"
	"strconv"
	"strings"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func layoutSidebar(gtx layout.Context, th *material.Theme, state *AppState, palette Palette) layout.Dimensions {
	width := gtx.Dp(unit.Dp(state.UI.SidebarWidth))
	gtx.Constraints.Min.X = width
	gtx.Constraints.Max.X = width
	fill(gtx, palette.SidebarBg)

	return layout.Inset{Left: unit.Dp(14), Right: unit.Dp(14), Top: unit.Dp(18), Bottom: unit.Dp(18)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(
			gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return sidebarBranding(gtx, th, palette) }),
			layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return sidebarSectionTitle(gtx, th, palette, "Chats") }),
			layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return sidebarRooms(gtx, th, state, palette) }),
			layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return sidebarSectionTitle(gtx, th, palette, "Settings") }),
			layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return sidebarSettings(gtx, th, state, palette) }),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return sidebarFooter(gtx, th, state, palette) }),
		)
	})
}

func sidebarBranding(gtx layout.Context, th *material.Theme, palette Palette) layout.Dimensions {
	return card(gtx, palette.SidebarCard, 18, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Left: unit.Dp(14), Right: unit.Dp(14), Top: unit.Dp(16), Bottom: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(
				gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return avatarBadge(gtx, th, palette.Accent, "L")
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(
						gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(th, unit.Sp(17), "Secure Chat")
							lbl.Color = palette.SidebarText
							return lbl.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							meta := material.Body2(th, "private desktop messenger")
							meta.Color = palette.SidebarMuted
							return meta.Layout(gtx)
						}),
					)
				}),
			)
		})
	})
}

func sidebarSectionTitle(gtx layout.Context, th *material.Theme, palette Palette, label string) layout.Dimensions {
	lbl := material.Body2(th, strings.ToUpper(label))
	lbl.Color = palette.SidebarMuted
	return lbl.Layout(gtx)
}

func sidebarRooms(gtx layout.Context, th *material.Theme, state *AppState, palette Palette) layout.Dimensions {
	children := make([]layout.FlexChild, 0, len(state.UI.Rooms)*2)
	for i := range state.UI.Rooms {
		room := &state.UI.Rooms[i]
		children = append(children,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return sidebarRoomItem(gtx, th, state, palette, room)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		)
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func sidebarRoomItem(gtx layout.Context, th *material.Theme, state *AppState, palette Palette, room *SidebarRoom) layout.Dimensions {
	bg := palette.SidebarCard
	textColor := palette.SidebarText
	if state.UI.ActiveRoom == room.Name {
		bg = palette.SidebarActive
	}
	dims := room.Clickable.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return card(gtx, bg, 16, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(12), Right: unit.Dp(12), Top: unit.Dp(12), Bottom: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(
					gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return avatarBadge(gtx, th, palette.AccentSoft, "#") }),
					layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Body1(th, room.Name)
						lbl.Color = textColor
						return lbl.Layout(gtx)
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if room.Badge <= 0 {
							return layout.Dimensions{}
						}
						return unreadBadge(gtx, th, palette, room.Badge)
					}),
				)
			})
		})
	})
	for room.Clickable.Clicked(gtx) {
		state.SetActiveRoom(room.Name)
	}
	return dims
}

func unreadBadge(gtx layout.Context, th *material.Theme, palette Palette, count int) layout.Dimensions {
	return card(gtx, palette.Accent, 10, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Left: unit.Dp(7), Right: unit.Dp(7), Top: unit.Dp(2), Bottom: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Caption(th, strconv.Itoa(count))
			lbl.Color = palette.TextOnAccent
			return lbl.Layout(gtx)
		})
	})
}

func sidebarSettings(gtx layout.Context, th *material.Theme, state *AppState, palette Palette) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(
		gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return sidebarAction(gtx, th, palette, &state.UI.SettingsBtn, "Settings", false)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			label := "Theme: Dark"
			if state.UI.Theme == ThemeLight {
				label = "Theme: Light"
			}
			return sidebarAction(gtx, th, palette, &state.UI.ThemeToggleBtn, label, false)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return sidebarAction(gtx, th, palette, &state.UI.NotificationsBtn, "Notifications", false)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return sidebarAction(gtx, th, palette, &state.UI.EncryptionInfoBtn, "Encryption Info", false)
		}),
	)
}

func sidebarFooter(gtx layout.Context, th *material.Theme, state *AppState, palette Palette) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(
		gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return sidebarAction(gtx, th, palette, &state.UI.LogoutBtn, "Logout", false)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return sidebarAction(gtx, th, palette, &state.UI.DeleteAccountBtn, "Delete Account", true)
		}),
	)
}

func sidebarAction(gtx layout.Context, th *material.Theme, palette Palette, btn *widget.Clickable, label string, danger bool) layout.Dimensions {
	bg := palette.SidebarCard
	textColor := palette.SidebarText
	if danger {
		bg = color.NRGBA{R: 0x3A, G: 0x18, B: 0x1D, A: 0xFF}
		textColor = palette.Danger
	}
	return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return card(gtx, bg, 16, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(12), Right: unit.Dp(12), Top: unit.Dp(12), Bottom: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body1(th, label)
				lbl.Color = textColor
				return lbl.Layout(gtx)
			})
		})
	})
}

func handleSidebarActions(gtx layout.Context, state *AppState) {
	for state.UI.ThemeToggleBtn.Clicked(gtx) {
		state.ToggleTheme()
	}
	for state.UI.SettingsBtn.Clicked(gtx) {
		state.OpenInfoModal("settings", "Settings", "Settings panel scaffolded. Persistent preferences can be connected here next.")
	}
	for state.UI.NotificationsBtn.Clicked(gtx) {
		state.OpenInfoModal("notifications", "Notifications", "Notification preferences UI placeholder.")
	}
	for state.UI.EncryptionInfoBtn.Clicked(gtx) {
		state.OpenInfoModal("encryption_info", "Encryption Info", "Messages are encrypted client-side and routed without plaintext decryption on the server.")
	}
	for state.UI.LogoutBtn.Clicked(gtx) {
		state.Logout()
	}
	for state.UI.DeleteAccountBtn.Clicked(gtx) {
		state.OpenDeleteAccountModal()
	}
	for state.UI.SearchBtn.Clicked(gtx) {
		state.OpenInfoModal("search", "Search", "Search UI placeholder.")
	}
	for state.UI.InfoBtn.Clicked(gtx) {
		state.OpenInfoModal("room_info", "Room Info", state.UI.ActiveRoom+" is currently a layout scaffold for future room navigation.")
	}
	for state.UI.AttachBtn.Clicked(gtx) {
		state.Session.ErrorText = "attachments are not implemented yet"
	}
}
