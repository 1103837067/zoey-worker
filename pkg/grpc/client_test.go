package grpc

import (
	"encoding/json"
	"testing"
)

func TestGetSystemInfo(t *testing.T) {
	info := GetSystemInfo()

	t.Logf("系统信息:")
	t.Logf("  Hostname: %s", info.Hostname)
	t.Logf("  Platform: %s", info.Platform)
	t.Logf("  OSVersion: %s", info.OSVersion)
	t.Logf("  AgentVersion: %s", info.AgentVersion)
	t.Logf("  IPAddress: %s", info.IPAddress)

	if info.Hostname == "" {
		t.Error("Hostname 不应为空")
	}
	if info.Platform == "" {
		t.Error("Platform 不应为空")
	}
	if info.AgentVersion != Version {
		t.Errorf("AgentVersion 应为 %s, 实际为 %s", Version, info.AgentVersion)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.HeartbeatInterval != 5 {
		t.Errorf("HeartbeatInterval 应为 5, 实际为 %d", config.HeartbeatInterval)
	}
	if config.MaxHeartbeatFailures != 3 {
		t.Errorf("MaxHeartbeatFailures 应为 3, 实际为 %d", config.MaxHeartbeatFailures)
	}
	if len(config.ReconnectDelays) == 0 {
		t.Error("ReconnectDelays 不应为空")
	}

	t.Logf("默认配置: %+v", config)
}

func TestNewClient(t *testing.T) {
	client := NewClient(nil)

	if client == nil {
		t.Fatal("NewClient 返回 nil")
	}
	if client.config == nil {
		t.Error("client.config 不应为 nil")
	}
	if client.IsConnected() {
		t.Error("新建的客户端不应处于连接状态")
	}

	status, agentID, agentName := client.GetStatus()
	if status != StatusDisconnected {
		t.Errorf("新建客户端状态应为 disconnected, 实际为 %s", status)
	}
	if agentID != "" || agentName != "" {
		t.Error("新建客户端 agentID 和 agentName 应为空")
	}

	t.Log("NewClient 测试通过")
}

func TestClientCallbacks(t *testing.T) {
	client := NewClient(nil)

	// 测试状态回调
	var receivedStatus ClientStatus
	client.SetStatusCallback(func(status ClientStatus) {
		receivedStatus = status
	})

	// 测试任务回调
	var receivedTaskID string
	client.SetTaskCallback(func(taskID, taskType, payload string) {
		receivedTaskID = taskID
	})

	// 触发状态回调
	client.setStatus(StatusConnecting)
	if receivedStatus != StatusConnecting {
		t.Errorf("状态回调未正确触发: 期望 %s, 实际 %s", StatusConnecting, receivedStatus)
	}

	t.Log("回调设置测试通过")
	_ = receivedTaskID // 任务回调需要连接后才能测试
}

func TestClientLogs(t *testing.T) {
	client := NewClient(nil)

	// 记录一些日志
	client.log("INFO", "Test message 1")
	client.log("WARN", "Test message 2")
	client.log("ERROR", "Test message 3")

	logs := client.GetLogs(10)
	if len(logs) != 3 {
		t.Errorf("日志数量应为 3, 实际为 %d", len(logs))
	}

	// 验证日志内容
	if logs[0].Level != "INFO" || logs[0].Message != "Test message 1" {
		t.Error("第一条日志内容不正确")
	}

	t.Logf("获取到 %d 条日志", len(logs))
}

func TestDataHandler_GetApplications(t *testing.T) {
	result := HandleDataRequest(RequestTypeGetApplications, "{}")

	if result.RequestType != RequestTypeGetApplications {
		t.Errorf("RequestType 应为 %s", RequestTypeGetApplications)
	}

	t.Logf("GetApplications 结果: success=%v, message=%s", result.Success, result.Message)

	if result.Success {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(result.PayloadJSON), &data); err != nil {
			t.Errorf("解析 PayloadJSON 失败: %v", err)
		}

		apps, ok := data["applications"].([]interface{})
		if !ok {
			t.Error("返回数据应包含 applications 数组")
		} else {
			t.Logf("获取到 %d 个应用程序", len(apps))
			// 打印前3个
			for i, app := range apps {
				if i >= 3 {
					break
				}
				t.Logf("  %v", app)
			}
		}
	}
}

func TestDataHandler_GetWindows(t *testing.T) {
	result := HandleDataRequest(RequestTypeGetWindows, "{}")

	if result.RequestType != RequestTypeGetWindows {
		t.Errorf("RequestType 应为 %s", RequestTypeGetWindows)
	}

	t.Logf("GetWindows 结果: success=%v, message=%s", result.Success, result.Message)

	if result.Success {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(result.PayloadJSON), &data); err != nil {
			t.Errorf("解析 PayloadJSON 失败: %v", err)
		}

		windows, ok := data["windows"].([]interface{})
		if !ok {
			t.Error("返回数据应包含 windows 数组")
		} else {
			t.Logf("获取到 %d 个窗口", len(windows))
			// 打印前3个
			for i, win := range windows {
				if i >= 3 {
					break
				}
				t.Logf("  %v", win)
			}
		}
	}
}

func TestDataHandler_GetWindowsWithFilter(t *testing.T) {
	payload := `{"process_name": "Cursor"}`
	result := HandleDataRequest(RequestTypeGetWindows, payload)

	t.Logf("GetWindows(filter=Cursor) 结果: success=%v", result.Success)

	if result.Success {
		var data map[string]interface{}
		json.Unmarshal([]byte(result.PayloadJSON), &data)
		windows, _ := data["windows"].([]interface{})
		t.Logf("找到 %d 个 Cursor 窗口", len(windows))
	}
}

func TestDataHandler_GetElements(t *testing.T) {
	result := HandleDataRequest(RequestTypeGetElements, `{"window_handle": 1234}`)

	if result.RequestType != RequestTypeGetElements {
		t.Errorf("RequestType 应为 %s", RequestTypeGetElements)
	}

	// Go 版本不支持，应返回失败
	if result.Success {
		t.Error("GetElements 在 Go 版本中应返回失败")
	}

	t.Logf("GetElements 结果: success=%v, message=%s", result.Success, result.Message)
}

func TestDataHandler_UnknownType(t *testing.T) {
	result := HandleDataRequest("UNKNOWN_TYPE", "{}")

	if result.Success {
		t.Error("未知请求类型应返回失败")
	}
	if result.Message == "" {
		t.Error("失败时应有错误消息")
	}

	t.Logf("UnknownType 结果: message=%s", result.Message)
}

func TestClientStatus(t *testing.T) {
	statuses := []ClientStatus{
		StatusDisconnected,
		StatusConnecting,
		StatusConnected,
		StatusReconnecting,
	}

	for _, s := range statuses {
		if s == "" {
			t.Error("ClientStatus 不应为空字符串")
		}
	}

	t.Logf("所有状态: %v", statuses)
}

// TestConnectWithoutServer 测试无服务端时的连接行为
func TestConnectWithoutServer(t *testing.T) {
	client := NewClient(nil)

	// 尝试连接不存在的服务器
	err := client.Connect("localhost:59999", "test_key", "test_secret")

	if err == nil {
		t.Error("连接不存在的服务器应返回错误")
		client.Disconnect()
	} else {
		t.Logf("预期的连接错误: %v", err)
	}

	if client.IsConnected() {
		t.Error("连接失败后不应处于连接状态")
	}
}

// BenchmarkGetSystemInfo 基准测试
func BenchmarkGetSystemInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetSystemInfo()
	}
}

// BenchmarkDataHandler_GetApplications 基准测试
func BenchmarkDataHandler_GetApplications(b *testing.B) {
	for i := 0; i < b.N; i++ {
		HandleDataRequest(RequestTypeGetApplications, "{}")
	}
}
