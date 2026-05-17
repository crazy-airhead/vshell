package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"vshell/internal/app"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	svc := app.New()

	wailsApp := application.New(application.Options{
		Name:        "vshell",
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

	wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "vshell",
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
