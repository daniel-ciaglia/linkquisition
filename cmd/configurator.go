package main

import (
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/resources"
)

type Configurator struct {
	gtkApp          *gtk.Application
	browserService  linkquisition.BrowserService
	settingsService linkquisition.SettingsService
}

func NewConfigurator(
	gtkApp *gtk.Application,
	browserService linkquisition.BrowserService,
	settingsService linkquisition.SettingsService,
) *Configurator {
	return &Configurator{
		gtkApp:          gtkApp,
		browserService:  browserService,
		settingsService: settingsService,
	}
}

func (c *Configurator) Run() error {
	win := gtk.NewApplicationWindow(c.gtkApp)
	win.SetTitle("Linkquisition settings")
	win.SetResizable(false)
	win.SetDefaultSize(500, 400) //nolint:mnd

	notebook := gtk.NewNotebook()
	notebook.AppendPage(c.getGeneralTab(), gtk.NewLabel("General"))
	notebook.AppendPage(c.getAboutTab(), gtk.NewLabel("About"))
	win.SetChild(notebook)

	win.SetVisible(true)

	return nil
}

func (c *Configurator) getGeneralTab() gtk.Widgetter {
	vbox := gtk.NewBox(gtk.OrientationVertical, 6)

	// MAKE DEFAULT -LABEL
	makeDefaultLabel := gtk.NewLabel(
		"In order to Linkquisition to function as a browser-picker\n" +
			"it has to be set as the default browser:",
	)
	vbox.Append(makeDefaultLabel)

	setupMakeDefaultButton := func(button *gtk.Button, isDefault bool) {
		if isDefault {
			button.SetLabel("All good!")
			button.SetSensitive(false)
		} else {
			button.SetLabel("Make default")
			button.SetSensitive(true)
		}
	}

	// MAKE DEFAULT -BUTTON
	makeDefaultButton := gtk.NewButton()
	makeDefaultButton.SetLabel("checking")
	makeDefaultButton.SetSensitive(false)

	makeDefaultButton.ConnectClicked(func() {
		makeDefaultButton.SetSensitive(false)
		err := c.browserService.MakeUsTheDefaultBrowser()
		if err != nil {
			makeDefaultButton.SetLabel("Error making default!")
			makeDefaultButton.SetSensitive(true)
			fmt.Printf("error making Linkquisition the default browser: %v", err)
		} else {
			setupMakeDefaultButton(makeDefaultButton, true)
		}
	})

	setupMakeDefaultButton(makeDefaultButton, c.browserService.AreWeTheDefaultBrowser())
	vbox.Append(makeDefaultButton)

	// SCAN BROWSERS -BUTTON
	setupScanBrowsersButton := func(button *gtk.Button, alreadyScanned bool) {
		if alreadyScanned {
			button.SetLabel("Re-scan browsers")
		} else {
			button.SetLabel("Scan browsers")
		}
		button.SetSensitive(true)
	}

	scanBrowsersButton := gtk.NewButton()
	scanBrowsersButton.SetLabel("Scan now")

	scanBrowsersButton.ConnectClicked(func() {
		scanBrowsersButton.SetSensitive(false)
		err := c.settingsService.ScanBrowsers()
		if err != nil {
			scanBrowsersButton.SetLabel("Error scanning browsers!")
			scanBrowsersButton.SetSensitive(true)
			fmt.Printf("error scanning browsers: %v", err)
		} else {
			isConfigured, _ := c.settingsService.IsConfigured()
			setupScanBrowsersButton(scanBrowsersButton, isConfigured)
		}
	})

	// TODO show a spinner while scanning
	// TODO show a message when scanning is done
	// TODO show a message (instead of the button) if configuration is invalid (corrupted file etc)
	isConfigured, _ := c.settingsService.IsConfigured()
	setupScanBrowsersButton(scanBrowsersButton, isConfigured)

	descLabel := gtk.NewLabel(
		"The browsers should be scanned and stored in a configuration file for\n" +
			"faster startup and for enabling custom configuration.\n" +
			"\n" +
			"The scan should be safe to execute at any time: only newly detected\n" +
			"browsers are added and the ones no longer present in the system are\n" +
			"removed.\n\nAny existing rules, ordering or customization shouldn't be affected.",
	)
	vbox.Append(descLabel)
	vbox.Append(scanBrowsersButton)

	return vbox
}

func (c *Configurator) getAboutTab() gtk.Widgetter {
	vbox := gtk.NewBox(gtk.OrientationVertical, 6)

	loader := gdkpixbuf.NewPixbufLoader()
	if err := loader.Write(resources.LinkquisitionIconBytes); err == nil {
		if err := loader.Close(); err == nil {
			if pixbuf := loader.Pixbuf(); pixbuf != nil {
				img := gtk.NewImageFromPaintable(gdk.NewTextureForPixbuf(pixbuf))
				btn := gtk.NewButton()
				btn.SetChild(img)
				btn.ConnectClicked(func() {
					if err := c.browserService.OpenUrlWithDefaultBrowser("https://github.com/Strobotti/linkquisition"); err != nil {
						fmt.Printf("error opening url: %s", err.Error())
					}
				})
				headerBox := gtk.NewBox(gtk.OrientationHorizontal, 4)
				headerBox.Append(btn)
				headerBox.Append(gtk.NewLabel(fmt.Sprintf("Linkquisition %s", version)))
				vbox.Append(headerBox)
			}
		}
	}

	return vbox
}
