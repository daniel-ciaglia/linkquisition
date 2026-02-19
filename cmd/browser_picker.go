package main

import (
	"context"
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/strobotti/linkquisition"
)

const (
	windowDefaultWidth = 600
	spacingSmall       = 4
	spacingMedium      = 6
	spacingLarge       = 8
)

type BrowserPicker struct {
	gtkApp          *gtk.Application
	browserService  linkquisition.BrowserService
	browsers        []linkquisition.Browser
	settingsService linkquisition.SettingsService
}

func NewBrowserPicker(
	gtkApp *gtk.Application,
	browserService linkquisition.BrowserService,
	browsers []linkquisition.Browser,
	settingsService linkquisition.SettingsService,
) *BrowserPicker {
	return &BrowserPicker{
		gtkApp:          gtkApp,
		browserService:  browserService,
		browsers:        browsers,
		settingsService: settingsService,
	}
}

func (picker *BrowserPicker) Run(_ context.Context, urlToOpen string) {
	var remember bool
	// TODO give user the option to choose between site and domain (and later on regex, too)
	rememberMatchType := linkquisition.BrowserMatchTypeSite

	win := gtk.NewApplicationWindow(picker.gtkApp)
	win.SetTitle("Linkquisition")
	win.SetResizable(false)
	win.SetDefaultSize(windowDefaultWidth, -1)

	vbox := gtk.NewBox(gtk.OrientationVertical, spacingMedium)

	var buttons []*gtk.Button

	for i := range picker.browsers {
		btn := picker.makeBrowserButton(picker.browsers[i], urlToOpen, &remember, &rememberMatchType)
		buttons = append(buttons, btn)
		vbox.Append(btn)
	}

	// URL display row
	urlEntry := gtk.NewEntry()
	urlEntry.SetText(urlToOpen)
	urlEntry.SetEditable(false)
	urlEntry.SetCanFocus(false)

	urlRow := gtk.NewBox(gtk.OrientationHorizontal, spacingSmall)
	urlRow.Append(gtk.NewLabel("Open:"))
	urlRow.Append(urlEntry)
	vbox.Append(urlRow)

	// Remember checkbox
	uto := linkquisition.NewURL(urlToOpen)
	site, _ := uto.GetSite()
	check := gtk.NewCheckButtonWithLabel("Remember this choice with " + site)
	check.ConnectToggled(func() {
		remember = check.Active()
	})
	vbox.Append(check)

	if !picker.settingsService.GetSettings().Ui.HideKeyboardGuideLabel {
		vbox.Append(gtk.NewLabel("Press 'ENTER' to pick first, 'ESC' to quit, 'ctrl+c' to copy URL to clipboard"))
	}

	win.SetChild(vbox)

	// Keyboard shortcuts: ESC and Enter and number keys
	keyCtrl := gtk.NewEventControllerKey()
	keyCtrl.ConnectKeyPressed(func(keyval, _ uint, _ gdk.ModifierType) bool {
		switch keyval {
		case gdk.KEY_Escape:
			picker.gtkApp.Quit()
			return true
		case gdk.KEY_Return:
			if len(buttons) > 0 {
				buttons[0].Activate()
			}
			return true
		}
		for i, btn := range buttons {
			if keyval == gdk.KEY_1+uint(i) {
				btn.Activate()
				return true
			}
		}
		return false
	})
	win.AddController(keyCtrl)

	// Ctrl+C: copy URL to clipboard
	shortcutCtrl := gtk.NewShortcutController()
	shortcutCtrl.AddShortcut(gtk.NewShortcut(
		gtk.NewKeyvalTrigger(gdk.KEY_c, gdk.ControlMask),
		gtk.NewCallbackAction(func(_ gtk.Widgetter, _ *glib.Variant) bool {
			fmt.Println("Copying URL to clipboard: " + urlToOpen)
			display := gdk.DisplayGetDefault()
			clipboard := display.Clipboard()
			clipboard.SetText(urlToOpen)
			picker.gtkApp.Quit()
			return true
		}),
	))
	win.AddController(shortcutCtrl)

	win.SetVisible(true)
}

func (picker *BrowserPicker) makeBrowserButton(
	browser linkquisition.Browser,
	urlToOpen string,
	remember *bool,
	rememberMatchType *string,
) *gtk.Button {
	btn := gtk.NewButton()
	box := gtk.NewBox(gtk.OrientationHorizontal, spacingLarge)

	iconBytes, err := picker.browserService.GetIconForBrowser(browser)
	if err != nil {
		fmt.Println(err)
	} else {
		loader := gdkpixbuf.NewPixbufLoader()
		if err := loader.Write(iconBytes); err == nil {
			if err := loader.Close(); err == nil {
				if pixbuf := loader.Pixbuf(); pixbuf != nil {
					img := gtk.NewImageFromPaintable(gdk.NewTextureForPixbuf(pixbuf))
					img.SetIconSize(gtk.IconSizeNormal)
					box.Append(img)
				}
			}
		}
	}

	label := gtk.NewLabel(browser.Name)
	box.Append(label)
	btn.SetChild(box)

	btn.ConnectClicked(func() {
		fmt.Printf("Opening URL with browser: %s; remember the choice: %v\n", browser.Name, *remember)

		settings := picker.settingsService.GetSettings()

		if *remember {
			uto := linkquisition.NewURL(urlToOpen)
			matchValue, _ := uto.GetDomain()

			if *rememberMatchType == linkquisition.BrowserMatchTypeSite {
				matchValue, _ = uto.GetSite()
			}

			settings.AddRuleToBrowser(&browser, *rememberMatchType, matchValue)
			if writeErr := picker.settingsService.WriteSettings(settings); writeErr != nil {
				fmt.Printf("Failed to write settings: %v\n", writeErr)
			}
		}

		go func() {
			_ = picker.browserService.OpenUrlWithBrowser(urlToOpen, &browser)
		}()
		picker.gtkApp.Quit()
	})

	return btn
}
