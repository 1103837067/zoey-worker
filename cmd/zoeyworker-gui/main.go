package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed frontend/*
var assets embed.FS

func main() {
	// 创建应用实例
	app := NewApp()

	// 创建 Wails 应用
	err := wails.Run(&options.App{
		Title:     "Zoey Worker",
		Width:     800,
		Height:    600,
		MinWidth:  600,
		MinHeight: 400,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 17, G: 24, B: 39, A: 1}, // gray-900
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		// 关闭窗口时隐藏而不是退出（托盘模式）
		HideWindowOnClose: true,
		Bind: []interface{}{
			app,
		},
		// macOS 配置 - 使用默认标题栏
		Mac: &mac.Options{
			TitleBar: mac.TitleBarDefault(),
			About: &mac.AboutInfo{
				Title:   "Zoey Worker",
				Message: "UI 自动化执行客户端 v1.0.0",
			},
		},
		// Windows 配置
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
