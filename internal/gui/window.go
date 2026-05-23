package gui

import (
	"context"
	"fmt"
	"image/color"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/whooslizi/dns-fetching/internal/cache"
	"github.com/whooslizi/dns-fetching/internal/config"
	"github.com/whooslizi/dns-fetching/internal/doh"
	"github.com/whooslizi/dns-fetching/internal/proxy"
	"github.com/whooslizi/dns-fetching/internal/system"
)

type Window struct {
	app    fyne.App
	win    fyne.Window
	cfg    *config.Config
	client *doh.Client
	proxy  *proxy.Server
	cache  *cache.Cache

	statusDot   *StatusDot
	statusLabel *canvas.Text
	statsCard   *fyne.Container
	enableBtn   *widget.Button
	disableBtn  *widget.Button
	providerSel *widget.Select
	customEntry *widget.Entry
}

func NewWindow(app fyne.App, cfg *config.Config, client *doh.Client, proxy *proxy.Server, dnsCache *cache.Cache) *Window {
	return &Window{
		app:    app,
		cfg:    cfg,
		client: client,
		proxy:  proxy,
		cache:  dnsCache,
	}
}

func (w *Window) Build() fyne.Window {
	w.win = w.app.NewWindow("DNS Fetching")
	w.win.Resize(fyne.NewSize(420, 560))
	w.win.SetFixedSize(true)
	w.win.CenterOnScreen()
	w.win.SetContent(w.buildLayout())
	go w.statsLoop()
	return w.win
}

func (w *Window) buildLayout() *fyne.Container {
	title := canvas.NewText("🛡  DNS Fetching", color.NRGBA{R: 0, G: 200, B: 180, A: 255})
	title.TextSize = 22
	title.TextStyle = fyne.TextStyle{Bold: true}

	w.statusDot = NewStatusDot(w.proxy.IsRunning())
	w.statusLabel = canvas.NewText("Disconnected", ColorDisconnected)
	w.statusLabel.TextSize = 14
	if w.proxy.IsRunning() {
		w.statusLabel.Text = "Connected — " + w.cfg.Provider
		w.statusLabel.Color = ColorConnected
	}
	statusRow := container.NewHBox(w.statusDot, w.statusLabel)

	providerLabel := canvas.NewText("Provider", color.NRGBA{R: 150, G: 150, B: 170, A: 255})
	providerLabel.TextSize = 12

	// customEntry must be created BEFORE SetSelected, because the
	// onProviderChanged callback references it via Hide()/Show().
	w.customEntry = widget.NewEntry()
	w.customEntry.SetPlaceHolder("https://your-doh-server/dns-query")
	w.customEntry.SetText(w.cfg.CustomURL)
	w.customEntry.Hide()

	options := append(doh.ProviderNames(), "Custom")
	w.providerSel = widget.NewSelect(options, w.onProviderChanged)
	w.providerSel.SetSelected(w.cfg.Provider)

	providerBox := container.NewVBox(providerLabel, w.providerSel, w.customEntry)

	w.enableBtn = widget.NewButton("● Enable", w.onEnable)
	w.enableBtn.Importance = widget.HighImportance
	w.disableBtn = widget.NewButton("Disable", w.onDisable)
	w.disableBtn.Importance = widget.DangerImportance

	if w.proxy.IsRunning() {
		w.enableBtn.Disable()
	} else {
		w.disableBtn.Disable()
	}
	buttonRow := container.NewGridWithColumns(2, w.enableBtn, w.disableBtn)

	statsLabel := canvas.NewText("Statistics", color.NRGBA{R: 150, G: 150, B: 170, A: 255})
	statsLabel.TextSize = 12
	w.statsCard = NewStatsCard("0", "0%", "0ms")

	testBtn := widget.NewButton("Test DNS", w.onTestDNS)
	restoreBtn := widget.NewButton("Restore Defaults", w.onRestore)
	actionRow := container.NewGridWithColumns(2, testBtn, restoreBtn)

	autoStartCheck := widget.NewCheck("Start on login", w.onAutoStartToggle)
	autoStartCheck.Checked = w.cfg.AutoStart
	trayCheck := widget.NewCheck("Minimize to tray", w.onTrayToggle)
	trayCheck.Checked = w.cfg.MinimizeToTray

	sep := canvas.NewLine(color.NRGBA{R: 50, G: 50, B: 70, A: 255})

	return container.NewVBox(
		container.NewCenter(title),
		widget.NewSeparator(),
		layout.NewSpacer(),
		container.NewCenter(statusRow),
		layout.NewSpacer(),
		providerBox,
		layout.NewSpacer(),
		buttonRow,
		layout.NewSpacer(),
		sep,
		statsLabel,
		w.statsCard,
		layout.NewSpacer(),
		actionRow,
		widget.NewSeparator(),
		container.NewVBox(autoStartCheck, trayCheck),
	)
}

func (w *Window) onProviderChanged(name string) {
	if name == "Custom" {
		w.customEntry.Show()
		return
	}
	w.customEntry.Hide()
	w.cfg.Provider = name
	w.cfg.Save()

	if p, ok := doh.FindProvider(name); ok {
		w.client.SetProvider(p)
		w.client.ResetMetrics()
		w.cache.Flush()
	}
}

func (w *Window) onEnable() {
	var provider doh.Provider
	if w.providerSel.Selected == "Custom" {
		url := w.customEntry.Text
		if url == "" {
			w.showError("Please enter a custom DoH URL")
			return
		}
		provider = doh.NewCustomProvider("Custom", url)
		w.cfg.CustomURL = url
	} else {
		p, ok := doh.FindProvider(w.providerSel.Selected)
		if !ok {
			w.showError("Unknown provider")
			return
		}
		provider = p
	}
	w.client.SetProvider(provider)

	selfPath, _ := os.Executable()
	_, err := system.RunHelper(system.ActionEnable, selfPath)
	if err != nil {
		w.showError(fmt.Sprintf("Setup failed: %v", err))
		return
	}

	if err := w.proxy.Start(); err != nil {
		w.showError(fmt.Sprintf("Proxy failed: %v", err))
		return
	}

	w.cfg.Enabled = true
	w.cfg.Save()
	w.updateStatus(true)
}

func (w *Window) onDisable() {
	if w.proxy.IsRunning() {
		w.proxy.Stop()
	}
	system.RunHelper(system.ActionDisable, "")
	w.cfg.Enabled = false
	w.cfg.Save()
	w.updateStatus(false)
}

func (w *Window) onTestDNS() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := w.client.TestConnection(ctx); err != nil {
		w.showError(fmt.Sprintf("DNS test failed: %v", err))
		return
	}
	w.app.SendNotification(fyne.NewNotification("DNS Fetching", "✓ DNS is working correctly!"))
}

func (w *Window) onRestore() {
	if w.proxy.IsRunning() {
		w.proxy.Stop()
	}
	system.RunHelper(system.ActionRestore, "")

	w.cfg.Enabled = false
	w.cfg.Provider = "Cloudflare"
	w.cfg.CustomURL = ""
	w.cfg.Save()

	w.providerSel.SetSelected("Cloudflare")
	w.updateStatus(false)
	w.app.SendNotification(fyne.NewNotification("DNS Fetching", "Settings restored to defaults"))
}

func (w *Window) onAutoStartToggle(checked bool) {
	w.cfg.AutoStart = checked
	w.cfg.Save()
	if checked {
		system.EnableAutostart()
	} else {
		system.DisableAutostart()
	}
}

func (w *Window) onTrayToggle(checked bool) {
	w.cfg.MinimizeToTray = checked
	w.cfg.Save()
	SetupCloseToTray(w.win, checked)
}

func (w *Window) updateStatus(connected bool) {
	w.statusDot.SetConnected(connected)
	if connected {
		w.statusLabel.Text = "Connected — " + w.cfg.Provider
		w.statusLabel.Color = ColorConnected
		w.enableBtn.Disable()
		w.disableBtn.Enable()
	} else {
		w.statusLabel.Text = "Disconnected"
		w.statusLabel.Color = ColorDisconnected
		w.enableBtn.Enable()
		w.disableBtn.Disable()
	}
	w.statusLabel.Refresh()
}

func (w *Window) statsLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		cs := w.cache.GetStats()
		ds := w.client.GetMetrics()
		ps := w.proxy.GetStats()

		hitRate := "0%"
		total := cs.Hits + cs.Misses
		if total > 0 {
			hitRate = fmt.Sprintf("%.0f%%", float64(cs.Hits)/float64(total)*100)
		}

		w.statsCard.Objects = NewStatsCard(
			fmt.Sprintf("%d", ps.QueriesServed),
			hitRate,
			fmt.Sprintf("%.0fms", ds.AvgLatencyMs),
		).Objects
		w.statsCard.Refresh()
	}
}

func (w *Window) showError(msg string) {
	w.app.SendNotification(fyne.NewNotification("DNS Fetching — Error", msg))
}
