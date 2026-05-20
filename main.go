package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"vshell/internal/app"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed internal/app/icons/save.png
var saveIcon []byte

//go:embed internal/app/icons/close.png
var closeIcon []byte

func main() {
	svc := app.New()

	wailsApp := application.New(application.Options{
		Name:        "vShell",
		Description: "SSH Client Management Tool",
		Services: []application.Service{
			application.NewService(svc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	svc.SetApp(wailsApp)

	// Set application menu: vShell (with Settings) + File + Edit + Window
	menu := application.NewMenu()

	appMenu := menu.AddSubmenu("vShell")
	appMenu.AddRole(application.About)
	appMenu.AddSeparator()
	appMenu.Add("Settings...").
		SetAccelerator("CommandOrControl+,").
		OnClick(func(ctx *application.Context) {
			wailsApp.Event.Emit("menu:settings", nil)
		})
	appMenu.AddSeparator()
	appMenu.AddRole(application.ServicesMenu)
	appMenu.AddSeparator()
	appMenu.AddRole(application.Hide)
	appMenu.AddRole(application.HideOthers)
	appMenu.AddRole(application.UnHide)
	appMenu.AddSeparator()
	appMenu.AddRole(application.Quit)

	fileMenu := menu.AddSubmenu("File")
	fileMenu.Add("Save").
		SetBitmap(saveIcon).
		SetAccelerator("CommandOrControl+S").
		OnClick(func(ctx *application.Context) {
			wailsApp.Event.Emit("menu:save", nil)
		})
	fileMenu.AddSeparator()
	fileMenu.Add("Close").
		SetBitmap(closeIcon).
		SetAccelerator("CommandOrControl+W").
		OnClick(func(ctx *application.Context) {
			wailsApp.Event.Emit("menu:close-tab", nil)
		})

	editMenu := menu.AddSubmenu("Edit")
	editMenu.AddRole(application.Undo)
	editMenu.AddRole(application.Redo)
	editMenu.AddSeparator()
	editMenu.AddRole(application.Cut)
	editMenu.AddRole(application.Copy)
	editMenu.AddRole(application.Paste)
	editMenu.AddRole(application.SelectAll)

	menu.AddRole(application.WindowMenu)
	wailsApp.Menu.Set(menu)

	wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "vShell",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 30,
			TitleBar:                application.MacTitleBarHidden,
		},
		BackgroundColour: application.NewRGB(30, 30, 30),
		Width:            1280,
		Height:           800,
		MinWidth:         960,
		MinHeight:        600,
		URL:              "/",
	})

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}
