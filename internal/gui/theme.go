package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type dnsFetchingTheme struct{}

var _ fyne.Theme = (*dnsFetchingTheme)(nil)

func NewTheme() fyne.Theme {
	return &dnsFetchingTheme{}
}

func (t *dnsFetchingTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 18, G: 18, B: 30, A: 255}
	case theme.ColorNameButton:
		return color.NRGBA{R: 35, G: 35, B: 55, A: 255}
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 0, G: 200, B: 180, A: 255} // teal accent
	case theme.ColorNameForeground:
		return color.NRGBA{R: 230, G: 230, B: 240, A: 255}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 28, G: 28, B: 45, A: 255}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 120, G: 120, B: 140, A: 255}
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 80, G: 80, B: 100, A: 255}
	case theme.ColorNameHeaderBackground:
		return color.NRGBA{R: 25, G: 25, B: 40, A: 255}
	case theme.ColorNameHover:
		return color.NRGBA{R: 45, G: 45, B: 70, A: 255}
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (t *dnsFetchingTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *dnsFetchingTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *dnsFetchingTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 8
	case theme.SizeNameInlineIcon:
		return 24
	case theme.SizeNameText:
		return 14
	case theme.SizeNameHeadingText:
		return 22
	case theme.SizeNameSubHeadingText:
		return 16
	}
	return theme.DefaultTheme().Size(name)
}
