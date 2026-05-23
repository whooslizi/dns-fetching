package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

func SetupTray(app fyne.App, win fyne.Window, onToggle func(), onQuit func()) {
	desk, ok := app.(desktop.App)
	if !ok {
		return
	}

	menu := fyne.NewMenu("DNS Fetching",
		fyne.NewMenuItem("Show", func() {
			win.Show()
			win.RequestFocus()
		}),
		fyne.NewMenuItem("Toggle DNS", onToggle),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Quit", onQuit),
	)
	desk.SetSystemTrayMenu(menu)
}

func SetupCloseToTray(win fyne.Window, minimizeToTray bool) {
	if !minimizeToTray {
		return
	}
	win.SetCloseIntercept(func() {
		win.Hide()
	})
}
