package main

import (
	"embed"
	"log"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed frontend/*
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

//go:embed build/trayicon.png
var trayIcon []byte

var (
	mainApp    *application.App
	mainWindow *application.WebviewWindow
	appService *App
)

func main() {
	// 创建应用实例
	appService = NewApp()

	// 创建 Wails v3 应用
	mainApp = application.New(application.Options{
		Name:        "Zoey Worker",
		Description: "UI 自动化执行客户端",
		Services: []application.Service{
			application.NewService(appService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	// 创建主窗口
	mainWindow = mainApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Zoey Worker",
		Width:            480,
		Height:           580,
		MinWidth:         400,
		MinHeight:        500,
		BackgroundColour: application.NewRGB(255, 255, 255),
		URL:              "/frontend/index.html",
		Hidden:           false,
		Windows: application.WindowsWindow{
			HiddenOnTaskbar: false,
		},
	})

	// 监听窗口关闭事件，最小化到托盘而不是退出
	mainWindow.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
		// 阻止默认关闭行为
		e.Cancel()
		// 隐藏窗口到托盘
		mainWindow.Hide()
	})

	// 设置系统托盘
	setupSystemTray(mainApp, mainWindow, appService)

	// 运行应用
	err := mainApp.Run()
	if err != nil {
		log.Fatal(err)
	}
}

// setupSystemTray 设置系统托盘
func setupSystemTray(app *application.App, window *application.WebviewWindow, svc *App) {
	// 创建系统托盘
	tray := app.SystemTray.New()

	// 设置图标（macOS 使用 22x22 模板图标，Windows 使用完整图标）
	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(trayIcon)
	} else {
		tray.SetIcon(appIcon)
	}
	tray.SetTooltip("Zoey Worker - UI 自动化执行客户端")

	// 点击托盘图标显示/隐藏窗口
	tray.OnClick(func() {
		if window.IsVisible() {
			window.Hide()
		} else {
			window.Show()
			window.Focus()
		}
	})

	// 创建托盘菜单
	trayMenu := app.NewMenu()

	// 显示窗口
	trayMenu.Add("显示窗口").OnClick(func(ctx *application.Context) {
		window.Show()
		window.Focus()
	})

	trayMenu.AddSeparator()

	// 连接状态相关菜单
	connectItem := trayMenu.Add("连接服务器")
	disconnectItem := trayMenu.Add("断开连接")

	connectItem.OnClick(func(ctx *application.Context) {
		// 显示窗口让用户输入连接信息
		window.Show()
		window.Focus()
	})

	disconnectItem.OnClick(func(ctx *application.Context) {
		svc.Disconnect()
	})

	trayMenu.AddSeparator()

	// 退出
	trayMenu.Add("退出").OnClick(func(ctx *application.Context) {
		// 断开连接后退出
		svc.Disconnect()
		app.Quit()
	})

	tray.SetMenu(trayMenu)
}
