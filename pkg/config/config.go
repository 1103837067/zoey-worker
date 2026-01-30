package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ConnectionConfig 连接配置
type ConnectionConfig struct {
	ServerURL   string `json:"server_url"`
	AccessKey   string `json:"access_key"`
	SecretKey   string `json:"secret_key"`
	AutoConnect bool   `json:"auto_connect"`
}

// DefaultConnectionConfig 默认连接配置
func DefaultConnectionConfig() *ConnectionConfig {
	return &ConnectionConfig{
		ServerURL:   "localhost:50051",
		AccessKey:   "",
		SecretKey:   "",
		AutoConnect: false,
	}
}

// Manager 配置管理器
type Manager struct {
	configDir  string
	configFile string
	mu         sync.RWMutex
}

// NewManager 创建配置管理器
func NewManager() *Manager {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	configDir := filepath.Join(homeDir, ".zoey-worker")
	return &Manager{
		configDir:  configDir,
		configFile: filepath.Join(configDir, "config.json"),
	}
}

// NewManagerWithDir 使用指定目录创建配置管理器
func NewManagerWithDir(configDir string) *Manager {
	return &Manager{
		configDir:  configDir,
		configFile: filepath.Join(configDir, "config.json"),
	}
}

// ensureDir 确保配置目录存在
func (m *Manager) ensureDir() error {
	return os.MkdirAll(m.configDir, 0755)
}

// Load 加载配置
func (m *Manager) Load() (*ConnectionConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, err := os.Stat(m.configFile); os.IsNotExist(err) {
		return DefaultConnectionConfig(), nil
	}

	data, err := os.ReadFile(m.configFile)
	if err != nil {
		return DefaultConnectionConfig(), fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config ConnectionConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return DefaultConnectionConfig(), fmt.Errorf("解析配置文件失败: %w", err)
	}

	return &config, nil
}

// Save 保存配置
func (m *Manager) Save(config *ConnectionConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.ensureDir(); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(m.configFile, data, 0600); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// Clear 清除配置
func (m *Manager) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, err := os.Stat(m.configFile); os.IsNotExist(err) {
		return nil
	}

	return os.Remove(m.configFile)
}

// GetConfigDir 获取配置目录
func (m *Manager) GetConfigDir() string {
	return m.configDir
}

// GetConfigFile 获取配置文件路径
func (m *Manager) GetConfigFile() string {
	return m.configFile
}

// Exists 检查配置文件是否存在
func (m *Manager) Exists() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, err := os.Stat(m.configFile)
	return err == nil
}

// 全局配置管理器
var defaultManager = NewManager()

// GetDefaultManager 获取默认配置管理器
func GetDefaultManager() *Manager {
	return defaultManager
}

// Load 使用默认管理器加载配置
func Load() (*ConnectionConfig, error) {
	return defaultManager.Load()
}

// Save 使用默认管理器保存配置
func Save(config *ConnectionConfig) error {
	return defaultManager.Save(config)
}

// Clear 使用默认管理器清除配置
func Clear() error {
	return defaultManager.Clear()
}
