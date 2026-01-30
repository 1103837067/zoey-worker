package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/zoeyai/zoeyworker/pkg/auto"
	"github.com/zoeyai/zoeyworker/pkg/config"
	"github.com/zoeyai/zoeyworker/pkg/executor"
	"github.com/zoeyai/zoeyworker/pkg/grpc"
)

// 版本信息 (可通过 ldflags 注入)
var (
	Version   = "1.0.0"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// 命令行参数
	var (
		serverURL   = flag.String("server", "", "服务端地址 (例: localhost:50051)")
		accessKey   = flag.String("access-key", "", "访问密钥")
		secretKey   = flag.String("secret-key", "", "秘密密钥")
		saveConfig  = flag.Bool("save", false, "保存配置到本地")
		showVersion = flag.Bool("version", false, "显示版本信息")
		showHelp    = flag.Bool("help", false, "显示帮助信息")
	)

	flag.Parse()

	// 显示版本
	if *showVersion {
		printVersion()
		return
	}

	// 显示帮助
	if *showHelp {
		printHelp()
		return
	}

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("[WARN] 加载配置失败: %v\n", err)
	}

	// 命令行参数优先级高于配置文件
	if *serverURL != "" {
		cfg.ServerURL = *serverURL
	}
	if *accessKey != "" {
		cfg.AccessKey = *accessKey
	}
	if *secretKey != "" {
		cfg.SecretKey = *secretKey
	}

	// 验证必要参数
	if cfg.ServerURL == "" {
		fmt.Println("[ERROR] 缺少服务端地址，请使用 -server 参数指定")
		printHelp()
		os.Exit(1)
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		fmt.Println("[ERROR] 缺少认证信息，请使用 -access-key 和 -secret-key 参数")
		printHelp()
		os.Exit(1)
	}

	// 保存配置
	if *saveConfig {
		if err := config.Save(cfg); err != nil {
			fmt.Printf("[WARN] 保存配置失败: %v\n", err)
		} else {
			fmt.Printf("[INFO] 配置已保存到 %s\n", config.GetDefaultManager().GetConfigFile())
		}
	}

	// 打印启动信息
	fmt.Println("========================================")
	fmt.Printf("  Zoey Worker v%s\n", Version)
	fmt.Println("========================================")
	fmt.Printf("服务端: %s\n", cfg.ServerURL)
	fmt.Println()

	// macOS 权限检查
	if runtime.GOOS == "darwin" {
		checkMacOSPermissions()
	}

	// 创建 gRPC 客户端
	client := grpc.NewClient(nil)

	// 设置状态回调
	client.SetStatusCallback(func(status grpc.ClientStatus) {
		fmt.Printf("[STATUS] %s\n", status)
	})

	// 创建任务执行器
	exec := executor.NewExecutor(client)

	// 设置 executor 日志函数
	executor.SetLogFunc(func(level, message string) {
		client.Log(level, message)
	})

	// 设置任务回调
	client.SetTaskCallback(func(taskID, taskType, payloadJSON string) {
		go exec.Execute(taskID, taskType, payloadJSON)
	})

	// 设置执行器状态回调（用于心跳上报）
	client.SetExecutorStatusCallback(func() (string, string, string, int64, int) {
		return exec.GetStatus()
	})

	// 连接服务端
	fmt.Println("[INFO] 正在连接服务端...")
	if err := client.Connect(cfg.ServerURL, cfg.AccessKey, cfg.SecretKey); err != nil {
		fmt.Printf("[ERROR] 连接失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("[INFO] 连接成功，等待任务...")
	fmt.Println("[INFO] 按 Ctrl+C 退出")

	// 等待中断信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println()
	fmt.Println("[INFO] 正在断开连接...")
	client.Disconnect()
	fmt.Println("[INFO] 已退出")
}


// printVersion 打印版本信息
func printVersion() {
	fmt.Printf("Zoey Worker v%s\n", Version)
	fmt.Printf("Build Time: %s\n", BuildTime)
	fmt.Printf("Git Commit: %s\n", GitCommit)
}

// printHelp 打印帮助信息
func printHelp() {
	fmt.Println("Zoey Worker - UI 自动化测试工具")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  zoeyworker [选项]")
	fmt.Println()
	fmt.Println("选项:")
	fmt.Println("  -server string      服务端地址 (例: localhost:50051)")
	fmt.Println("  -access-key string  访问密钥")
	fmt.Println("  -secret-key string  秘密密钥")
	fmt.Println("  -save               保存配置到本地")
	fmt.Println("  -version            显示版本信息")
	fmt.Println("  -help               显示帮助信息")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  # 连接服务端")
	fmt.Println("  zoeyworker -server localhost:50051 -access-key KEY -secret-key SECRET")
	fmt.Println()
	fmt.Println("  # 连接并保存配置")
	fmt.Println("  zoeyworker -server localhost:50051 -access-key KEY -secret-key SECRET -save")
	fmt.Println()
	fmt.Println("  # 使用已保存的配置连接")
	fmt.Println("  zoeyworker")
	fmt.Println()
	fmt.Printf("配置文件位置: %s\n", config.GetDefaultManager().GetConfigFile())
}

// checkMacOSPermissions 检查 macOS 权限
func checkMacOSPermissions() {
	fmt.Println("[INFO] 正在检查 macOS 权限...")
	status := auto.CheckPermissions()
	
	fmt.Printf("[INFO] 辅助功能权限: %v\n", status.Accessibility)
	fmt.Printf("[INFO] 屏幕录制权限: %v\n", status.ScreenRecording)
	
	if status.AllGranted {
		fmt.Println("[INFO] ✓ 所有权限已授予")
		return
	}
	
	fmt.Println()
	fmt.Println("[WARN] ========== 缺少权限 ==========")
	
	if !status.Accessibility {
		fmt.Println("[WARN] ❌ 辅助功能: 未授权 (用于控制鼠标/键盘)")
	}
	
	if !status.ScreenRecording {
		fmt.Println("[WARN] ❌ 屏幕录制: 未授权 (用于截屏)")
	}
	
	fmt.Println()
	fmt.Println("[WARN] 请在 系统设置 > 隐私与安全性 中授权")
	fmt.Println("[WARN] 授权后需要重启应用")
	fmt.Println("[WARN] ==================================")
	fmt.Println()
}
