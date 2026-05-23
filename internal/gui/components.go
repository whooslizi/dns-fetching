package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var (
	ColorConnected    = color.NRGBA{R: 0, G: 220, B: 130, A: 255}
	ColorDisconnected = color.NRGBA{R: 220, G: 60, B: 60, A: 255}
	ColorWarning      = color.NRGBA{R: 255, G: 180, B: 0, A: 255}
)

// StatusDot is a colored circle that shows connection state at a glance.
type StatusDot struct {
	widget.BaseWidget
	circle *canvas.Circle
}

func NewStatusDot(connected bool) *StatusDot {
	c := canvas.NewCircle(ColorDisconnected)
	if connected {
		c.FillColor = ColorConnected
	}
	c.Resize(fyne.NewSize(12, 12))

	s := &StatusDot{circle: c}
	s.ExtendBaseWidget(s)
	return s
}

func (s *StatusDot) SetConnected(connected bool) {
	if connected {
		s.circle.FillColor = ColorConnected
	} else {
		s.circle.FillColor = ColorDisconnected
	}
	s.circle.Refresh()
}

func (s *StatusDot) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(s.circle)
}

// StatsCard shows live DNS stats in a nice little box.
func NewStatsCard(queries, cacheHit, latency string) *fyne.Container {
	makeRow := func(label, value string) *fyne.Container {
		l := canvas.NewText(label, color.NRGBA{R: 150, G: 150, B: 170, A: 255})
		l.TextSize = 12
		v := canvas.NewText(value, color.NRGBA{R: 230, G: 230, B: 240, A: 255})
		v.TextSize = 12
		v.Alignment = fyne.TextAlignTrailing
		return container.NewBorder(nil, nil, l, v)
	}

	return container.NewVBox(
		makeRow("Queries", queries),
		makeRow("Cache hits", cacheHit),
		makeRow("Avg latency", latency),
	)
}
