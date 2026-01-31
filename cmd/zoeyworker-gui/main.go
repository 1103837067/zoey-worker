package main

import (
	"context"
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed frontend/*
var assets embed.FS

func main() {
	// 创建应用实例
	app := NewApp()

	// 初始化系统托盘（仅 Windows 支持）
	initSystray(app)

	// 创建 Wails 应用
	err := wails.Run(&options.App{
		Title:     "Zoey Worker",
		Width:     480,
		Height:    580,
		MinWidth:  400,
		MinHeight: 500,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1}, // 白色背景
		OnStartup:        app.startup,
		OnShutdown: func(ctx context.Context) {
			quitSystray()
			app.shutdown(ctx)
		},
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			// 关闭窗口时隐藏而不是退出
			cfg := app.LoadConfig()
			if cfg.MinimizeToTray {
				wailsRuntime.WindowHide(ctx)
				// 发送事件通知前端显示提示
				if !app.hasShownTrayNotification {
					app.hasShownTrayNotification = true
					wailsRuntime.EventsEmit(ctx, "minimized-to-background")
				}
				return true // 阻止关闭
			}
			return false // 允许关闭
		},
		Bind: []interface{}{
			app,
		},
		// macOS 配置
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
