package main

import (
	"log"
	"time"

	"fyne.io/fyne/v2/app"

	"github.com/whooslizi/dns-fetching/internal/cache"
	"github.com/whooslizi/dns-fetching/internal/config"
	"github.com/whooslizi/dns-fetching/internal/doh"
	"github.com/whooslizi/dns-fetching/internal/gui"
	"github.com/whooslizi/dns-fetching/internal/proxy"
	"github.com/whooslizi/dns-fetching/internal/system"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	// set up the DoH client with the saved provider
	provider, ok := doh.FindProvider(cfg.Provider)
	if !ok {
		if cfg.CustomURL != "" {
			provider = doh.NewCustomProvider(cfg.Provider, cfg.CustomURL)
		} else {
			provider = doh.BuiltinProviders[0] // default to Cloudflare
		}
	}

	// use Google as fallback if primary isn't Google, otherwise use Quad9
	var fallback *doh.Provider
	if provider.Name != "Google" {
		fb, _ := doh.FindProvider("Google")
		fallback = &fb
	} else {
		fb, _ := doh.FindProvider("Quad9")
		fallback = &fb
	}

	dohClient := doh.NewClient(provider, fallback)

	// cache with janitor
	dnsCache := cache.New(cfg.CacheMaxSize)
	stopJanitor := make(chan struct{})
	dnsCache.StartJanitor(60*time.Second, stopJanitor)
	defer close(stopJanitor)

	// proxy (not started yet — whooslizi clicks Enable)
	dnsProxy := proxy.New(cfg.ListenAddr, dohClient, dnsCache)

	// if it was enabled before reboot, try to start it again
	if cfg.Enabled {
		if err := dnsProxy.Start(); err != nil {
			log.Printf("auto-start failed (need capabilities?): %v", err)
			cfg.Enabled = false
		}
	}

	// GUI
	a := app.NewWithID("com.dnsfetching.app")
	a.Settings().SetTheme(gui.NewTheme())

	w := gui.NewWindow(a, cfg, dohClient, dnsProxy, dnsCache)
	win := w.Build()

	// system tray
	gui.SetupTray(a, win, func() {
		// toggle callback for tray menu
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
