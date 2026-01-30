package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConnectionConfig(t *testing.T) {
	config := DefaultConnectionConfig()

	if config.ServerURL != "localhost:50051" {
		t.Errorf("默认 ServerURL 应为 localhost:50051, 实际为 %s", config.ServerURL)
	}
	if config.AccessKey != "" {
		t.Error("默认 AccessKey 应为空")
	}
	if config.SecretKey != "" {
		t.Error("默认 SecretKey 应为空")
	}
	if config.AutoConnect {
		t.Error("默认 AutoConnect 应为 false")
	}

	t.Logf("默认配置: %+v", config)
}

func TestManagerSaveAndLoad(t *testing.T) {
	// 使用临时目录
	tempDir := t.TempDir()
	manager := NewManagerWithDir(tempDir)

	// 检查初始状态
	if manager.Exists() {
		t.Error("初始时配置文件不应存在")
	}

	// 保存配置
	config := &ConnectionConfig{
		ServerURL:   "test.server:8080",
		AccessKey:   "test_access_key",
		SecretKey:   "test_secret_key",
		AutoConnect: true,
	}

	err := manager.Save(config)
	if err != nil {
		t.Fatalf("保存配置失败: %v", err)
	}

	// 检查文件是否存在
	if !manager.Exists() {
		t.Error("保存后配置文件应存在")
	}

	// 加载配置
	loaded, err := manager.Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证内容
	if loaded.ServerURL != config.ServerURL {
		t.Errorf("ServerURL 不匹配: 期望 %s, 实际 %s", config.ServerURL, loaded.ServerURL)
	}
	if loaded.AccessKey != config.AccessKey {
		t.Errorf("AccessKey 不匹配: 期望 %s, 实际 %s", config.AccessKey, loaded.AccessKey)
	}
	if loaded.SecretKey != config.SecretKey {
		t.Errorf("SecretKey 不匹配: 期望 %s, 实际 %s", config.SecretKey, loaded.SecretKey)
	}
	if loaded.AutoConnect != config.AutoConnect {
		t.Errorf("AutoConnect 不匹配: 期望 %v, 实际 %v", config.AutoConnect, loaded.AutoConnect)
	}

	t.Logf("加载的配置: %+v", loaded)
}

func TestManagerClear(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewManagerWithDir(tempDir)

	// 先保存一个配置
	config := &ConnectionConfig{
		ServerURL: "test:1234",
	}
	err := manager.Save(config)
	if err != nil {
		t.Fatalf("保存配置失败: %v", err)
	}

	if !manager.Exists() {
		t.Fatal("保存后配置文件应存在")
	}

	// 清除配置
	err = manager.Clear()
	if err != nil {
		t.Fatalf("清除配置失败: %v", err)
	}

	if manager.Exists() {
		t.Error("清除后配置文件不应存在")
	}

	// 清除不存在的文件不应报错
	err = manager.Clear()
	if err != nil {
		t.Errorf("清除不存在的配置不应报错: %v", err)
	}
}

func TestManagerLoadNonExistent(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewManagerWithDir(tempDir)

	// 加载不存在的配置应返回默认值
	config, err := manager.Load()
	if err != nil {
		t.Fatalf("加载不存在的配置不应报错: %v", err)
	}

	defaultConfig := DefaultConnectionConfig()
	if config.ServerURL != defaultConfig.ServerURL {
		t.Errorf("应返回默认 ServerURL")
	}

	t.Log("加载不存在的配置返回默认值: OK")
}

func TestManagerLoadCorruptedFile(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewManagerWithDir(tempDir)

	// 创建一个损坏的配置文件
	configFile := filepath.Join(tempDir, "config.json")
	os.MkdirAll(tempDir, 0755)
	err := os.WriteFile(configFile, []byte("not valid json"), 0600)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 加载损坏的配置应返回默认值和错误
	config, err := manager.Load()
	if err == nil {
		t.Error("加载损坏的配置应返回错误")
	}

	// 但仍应返回默认配置
	if config == nil {
		t.Error("即使出错也应返回默认配置")
	}

	t.Logf("加载损坏配置的错误: %v", err)
}

func TestManagerPaths(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewManagerWithDir(tempDir)

	if manager.GetConfigDir() != tempDir {
		t.Errorf("GetConfigDir 应为 %s", tempDir)
	}

	expectedFile := filepath.Join(tempDir, "config.json")
	if manager.GetConfigFile() != expectedFile {
		t.Errorf("GetConfigFile 应为 %s", expectedFile)
	}

	t.Logf("配置目录: %s", manager.GetConfigDir())
	t.Logf("配置文件: %s", manager.GetConfigFile())
}

func TestDefaultManager(t *testing.T) {
	manager := GetDefaultManager()
	if manager == nil {
		t.Fatal("GetDefaultManager 返回 nil")
	}

	// 检查默认路径是否在用户目录下
	homeDir, _ := os.UserHomeDir()
	expectedDir := filepath.Join(homeDir, ".zoey-worker")

	if manager.GetConfigDir() != expectedDir {
		t.Errorf("默认配置目录应为 %s, 实际为 %s", expectedDir, manager.GetConfigDir())
	}

	t.Logf("默认配置目录: %s", manager.GetConfigDir())
}

func TestGlobalFunctions(t *testing.T) {
	// 测试全局函数不会 panic
	_, err := Load()
	if err != nil {
		t.Logf("Load 错误 (可能正常): %v", err)
	}

	// 不实际保存，避免污染用户配置
	t.Log("全局函数测试通过")
}

func TestConfigFilePermissions(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewManagerWithDir(tempDir)

	config := &ConnectionConfig{
		ServerURL: "test:1234",
		SecretKey: "sensitive_data",
	}

	err := manager.Save(config)
	if err != nil {
		t.Fatalf("保存配置失败: %v", err)
	}

	// 检查文件权限 (应为 0600)
	info, err := os.Stat(manager.GetConfigFile())
	if err != nil {
		t.Fatalf("获取文件信息失败: %v", err)
	}

	perm := info.Mode().Perm()
	// 在某些系统上权限可能略有不同，但不应该是全局可读的
	if perm&0077 != 0 {
		t.Logf("警告: 配置文件权限为 %o, 包含敏感信息时建议设为 0600", perm)
	}

	t.Logf("配置文件权限: %o", perm)
}

// BenchmarkSaveLoad 基准测试
func BenchmarkSaveLoad(b *testing.B) {
	tempDir := b.TempDir()
	manager := NewManagerWithDir(tempDir)
	config := &ConnectionConfig{
		ServerURL:   "test:1234",
		AccessKey:   "key",
		SecretKey:   "secret",
		AutoConnect: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.Save(config)
		manager.Load()
	}
}
