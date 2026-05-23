package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/whooslizi/dns-fetching/internal/cache"
	"github.com/whooslizi/dns-fetching/internal/config"
	"github.com/whooslizi/dns-fetching/internal/doh"
	"github.com/whooslizi/dns-fetching/internal/proxy"
	"github.com/whooslizi/dns-fetching/internal/system"
)

var version = "dev"

func main() {
	headless := flag.Bool("headless", false, "run without GUI (CLI/server mode)")
	showVersion := flag.Bool("version", false, "print version and exit")
	provider := flag.String("provider", "", "DNS provider name (e.g. Cloudflare, Google, Quad9)")
	enable := flag.Bool("enable", false, "auto-enable DNS proxy on start (headless mode)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("dns-fetching %s\n", version)
		os.Exit(0)
	}

	// auto-detect: if no DISPLAY and no WAYLAND_DISPLAY, force headless
	if !*headless && os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		log.Println("no display detected, switching to headless mode")
		*headless = true
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	// override provider from CLI flag if given
	if *provider != "" {
		cfg.Provider = *provider
	}

	// set up the DoH client with the saved provider
	dohProvider, ok := doh.FindProvider(cfg.Provider)
	if !ok {
		if cfg.CustomURL != "" {
			dohProvider = doh.NewCustomProvider(cfg.Provider, cfg.CustomURL)
		} else {
			dohProvider = doh.BuiltinProviders[0] // default to Cloudflare
		}
	}

	// use Google as fallback if primary isn't Google, otherwise use Quad9
	var fallback *doh.Provider
	if dohProvider.Name != "Google" {
		fb, _ := doh.FindProvider("Google")
		fallback = &fb
	} else {
		fb, _ := doh.FindProvider("Quad9")
		fallback = &fb
	}

	dohClient := doh.NewClient(dohProvider, fallback)

	// cache with janitor
	dnsCache := cache.New(cfg.CacheMaxSize)
	stopJanitor := make(chan struct{})
	dnsCache.StartJanitor(60*time.Second, stopJanitor)
	defer close(stopJanitor)

	dnsProxy := proxy.New(cfg.ListenAddr, dohClient, dnsCache)

	if *headless {
		runHeadless(cfg, dohClient, dnsProxy, *enable)
		return
	}

	runGUI(cfg, dohClient, dnsProxy, dnsCache)
}

func runHeadless(cfg *config.Config, dohClient *doh.Client, dnsProxy *proxy.Server, autoEnable bool) {
	log.Printf("dns-fetching %s — headless mode", version)
	log.Printf("provider: %s | listen: %s", cfg.Provider, cfg.ListenAddr)

	if autoEnable || cfg.Enabled {
		selfPath, _ := os.Executable()
		if _, err := system.RunHelper(system.ActionEnable, selfPath); err != nil {
			log.Printf("warning: system setup failed (may need sudo): %v", err)
		}

		if err := dnsProxy.Start(); err != nil {
			log.Fatalf("proxy start failed: %v", err)
		}
		cfg.Enabled = true
		cfg.Save()
		log.Println("DNS proxy started")
	} else {
		log.Println("proxy not enabled — use --enable flag or set enabled: true in config")
		log.Printf("config: ~/.config/dns-fetching/config.yaml")
	}

	// wait for SIGINT/SIGTERM
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("shutting down...")
	if dnsProxy.IsRunning() {
		dnsProxy.Stop()
		system.RunHelper(system.ActionDisable, "")
	}
}

func runGUI(cfg *config.Config, dohClient *doh.Client, dnsProxy *proxy.Server, dnsCache *cache.Cache) {
	guiMain(cfg, dohClient, dnsProxy, dnsCache)
}
