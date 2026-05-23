package main

import (
	"fyne.io/fyne/v2/app"

	"github.com/whooslizi/dns-fetching/internal/cache"
	"github.com/whooslizi/dns-fetching/internal/config"
	"github.com/whooslizi/dns-fetching/internal/doh"
	"github.com/whooslizi/dns-fetching/internal/gui"
	"github.com/whooslizi/dns-fetching/internal/proxy"
	"github.com/whooslizi/dns-fetching/internal/system"
)

func guiMain(cfg *config.Config, dohClient *doh.Client, dnsProxy *proxy.Server, dnsCache *cache.Cache) {
	if cfg.Enabled {
		if err := dnsProxy.Start(); err != nil {
			cfg.Enabled = false
		}
	}

	a := app.NewWithID("com.dnsfetching.app")
	a.Settings().SetTheme(gui.NewTheme())

	w := gui.NewWindow(a, cfg, dohClient, dnsProxy, dnsCache)
	win := w.Build()

	gui.SetupTray(a, win, func() {
		if dnsProxy.IsRunning() {
			dnsProxy.Stop()
			system.RunHelper(system.ActionDisable, "")
		} else {
			system.RunHelper(system.ActionEnable, "")
			dnsProxy.Start()
		}
	}, func() {
		if dnsProxy.IsRunning() {
			dnsProxy.Stop()
		}
		a.Quit()
	})

	gui.SetupCloseToTray(win, cfg.MinimizeToTray)

	win.ShowAndRun()
}
