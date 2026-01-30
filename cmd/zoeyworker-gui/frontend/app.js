/**
 * Zoey Worker GUI - 前端应用
 */

// ========== DOM 元素 ==========
const $ = id => document.getElementById(id)

const els = {
  statusIndicator: $('statusIndicator'),
  connectForm: $('connectForm'),
  serverUrl: $('serverUrl'),
  accessKey: $('accessKey'),
  secretKey: $('secretKey'),
  connectBtn: $('connectBtn'),
  disconnectBtn: $('disconnectBtn'),
  errorMessage: $('errorMessage'),
  connectionInfo: $('connectionInfo'),
  agentIdDisplay: $('agentIdDisplay'),
  agentNameDisplay: $('agentNameDisplay'),
  refreshLogsBtn: $('refreshLogsBtn'),
  emptyLogs: $('emptyLogs'),
  logList: $('logList'),
  systemInfo: $('systemInfo'),
  currentTime: $('currentTime'),
  // 设置
  settingAutoConnect: $('settingAutoConnect'),
  settingAutoReconnect: $('settingAutoReconnect'),
  settingReconnectInterval: $('settingReconnectInterval'),
  settingLogLevel: $('settingLogLevel'),
  settingMinimizeToTray: $('settingMinimizeToTray'),
  settingStartMinimized: $('settingStartMinimized'),
  settingsSaved: $('settingsSaved')
}

// ========== 状态 ==========
let state = {
  connected: false,
  agentId: '',
  agentName: '',
  config: null, // 当前配置
  reconnecting: false, // 是否正在重连
  reconnectTimer: null // 重连定时器
}

// ========== 初始化 ==========
async function init() {
  lucide.createIcons()
  bindEvents()
  updateTime()
  setInterval(updateTime, 1000)

  // 加载配置
  try {
    const config = await window.go.main.App.LoadConfig()
    state.config = config
    
    // 连接信息
    if (config.server_url) els.serverUrl.value = config.server_url
    if (config.access_key) els.accessKey.value = config.access_key
    if (config.secret_key) els.secretKey.value = config.secret_key
    
    // 设置选项
    loadSettingsToUI(config)
    
    // 自动连接
    if (config.auto_connect && config.server_url && config.access_key && config.secret_key) {
      setTimeout(() => connect(), 500) // 延迟连接，等待 UI 就绪
    }
  } catch (e) {
    console.error('加载配置失败:', e)
  }

  // 加载系统信息
  try {
    const info = await window.go.main.App.GetSystemInfo()
    els.systemInfo.textContent = `${info.platform} | ${info.hostname}`
  } catch (e) {
    console.error('获取系统信息失败:', e)
  }

  // 定时刷新日志
  setInterval(refreshLogs, 3000)

  // 定时检查连接状态
  setInterval(checkConnectionStatus, 2000)
}

// ========== 事件绑定 ==========
function bindEvents() {
  // Tab 切换
  document.querySelectorAll('.tab-btn').forEach(btn => {
    btn.addEventListener('click', () => switchTab(btn.dataset.tab))
  })

  // 连接表单
  els.connectForm.addEventListener('submit', async e => {
    e.preventDefault()
    await connect()
  })

  // 断开连接
  els.disconnectBtn.addEventListener('click', disconnect)

  // 刷新日志
  els.refreshLogsBtn.addEventListener('click', refreshLogs)
  
  // 设置变更 - 自动保存
  const settingInputs = [
    els.settingAutoConnect,
    els.settingAutoReconnect,
    els.settingReconnectInterval,
    els.settingLogLevel,
    els.settingMinimizeToTray,
    els.settingStartMinimized
  ]
  settingInputs.forEach(input => {
    if (input) {
      input.addEventListener('change', saveSettings)
    }
  })
}

// ========== Tab 切换 ==========
function switchTab(tabName) {
  document.querySelectorAll('.tab-btn').forEach(btn => {
    const isActive = btn.dataset.tab === tabName
    btn.classList.toggle('active', isActive)
    btn.classList.toggle('bg-gray-900', isActive)
    btn.classList.toggle('text-white', isActive)
    btn.classList.toggle('text-gray-400', !isActive)
  })

  document.querySelectorAll('.tab-content').forEach(content => {
    content.classList.toggle('hidden', content.id !== `tab-${tabName}`)
  })

  lucide.createIcons()

  // 切换到日志时刷新
  if (tabName === 'logs') {
    refreshLogs()
  }
}

// ========== 连接管理 ==========
async function connect() {
  const serverUrl = els.serverUrl.value.trim()
  const accessKey = els.accessKey.value.trim()
  const secretKey = els.secretKey.value.trim()

  if (!serverUrl || !accessKey || !secretKey) {
    showError('请填写完整的连接信息')
    return
  }

  hideError()
  setConnecting(true)

  try {
    const result = await window.go.main.App.Connect(serverUrl, accessKey, secretKey)

    if (result.success) {
      state.connected = true
      state.agentId = result.agent_id
      state.agentName = result.agent_name
      updateUI()
    } else {
      showError(result.message || '连接失败')
      setConnecting(false)
    }
  } catch (e) {
    showError(e.message || '连接错误')
    setConnecting(false)
  }
}

async function disconnect() {
  // 手动断开时取消自动重连
  cancelReconnect()
  
  try {
    await window.go.main.App.Disconnect()
  } catch (e) {
    console.error('断开连接失败:', e)
  }

  state.connected = false
  state.agentId = ''
  state.agentName = ''
  setConnecting(false)
  updateUI()
}

// 检查连接状态
async function checkConnectionStatus() {
  try {
    const status = await window.go.main.App.GetStatus()
    const wasConnected = state.connected
    
    state.connected = status.connected
    state.agentId = status.agent_id || ''
    state.agentName = status.agent_name || ''
    
    // 状态变化时更新 UI
    if (wasConnected !== state.connected) {
      setConnecting(false)
      updateUI()
      
      // 从已连接变为断开，且启用了自动重连
      if (wasConnected && !state.connected && state.config?.auto_reconnect && !state.reconnecting) {
        scheduleReconnect()
      }
    }
  } catch (e) {
    console.error('检查连接状态失败:', e)
  }
}

// 安排自动重连
function scheduleReconnect() {
  if (state.reconnectTimer) {
    clearTimeout(state.reconnectTimer)
  }
  
  const interval = (state.config?.reconnect_interval || 5) * 1000
  state.reconnecting = true
  
  console.log(`将在 ${interval / 1000} 秒后自动重连...`)
  
  state.reconnectTimer = setTimeout(async () => {
    if (!state.connected && state.config?.auto_reconnect) {
      console.log('正在自动重连...')
      await connect()
    }
    state.reconnecting = false
  }, interval)
}

// 取消自动重连
function cancelReconnect() {
  if (state.reconnectTimer) {
    clearTimeout(state.reconnectTimer)
    state.reconnectTimer = null
  }
  state.reconnecting = false
}

// ========== UI 更新 ==========
function updateUI() {
  // 状态指示器
  if (state.connected) {
    els.statusIndicator.innerHTML = `
      <span class="w-2 h-2 bg-green-400 rounded-full animate-pulse"></span>
      <span class="text-green-400">已连接</span>
    `
  } else {
    els.statusIndicator.innerHTML = `
      <span class="w-2 h-2 bg-gray-400 rounded-full"></span>
      <span class="text-gray-400">未连接</span>
    `
  }

  // 按钮状态
  if (state.connected) {
    els.connectBtn.classList.add('hidden')
    els.disconnectBtn.classList.remove('hidden')
    els.serverUrl.disabled = true
    els.accessKey.disabled = true
    els.secretKey.disabled = true
  } else {
    els.connectBtn.classList.remove('hidden')
    els.disconnectBtn.classList.add('hidden')
    els.serverUrl.disabled = false
    els.accessKey.disabled = false
    els.secretKey.disabled = false
  }

  // 连接信息
  if (state.connected) {
    els.connectionInfo.classList.remove('hidden')
    els.agentIdDisplay.textContent = state.agentId
    els.agentNameDisplay.textContent = state.agentName
  } else {
    els.connectionInfo.classList.add('hidden')
  }
}

function setConnecting(loading) {
  els.connectBtn.disabled = loading
  els.connectBtn.textContent = loading ? '连接中...' : '连接'
}

function showError(msg) {
  els.errorMessage.textContent = msg
  els.errorMessage.classList.remove('hidden')
}

function hideError() {
  els.errorMessage.classList.add('hidden')
}

// ========== 日志 ==========
async function refreshLogs() {
  try {
    const logs = await window.go.main.App.GetLogs(100)

    if (!logs || logs.length === 0) {
      els.emptyLogs.classList.remove('hidden')
      els.logList.innerHTML = ''
      return
    }

    els.emptyLogs.classList.add('hidden')
    els.logList.innerHTML = logs
      .map(
        log => `
      <div class="px-2 py-1 hover:bg-gray-700/30 rounded log-${log.level.toLowerCase()}">
        <span class="text-gray-500">${log.timestamp}</span>
        <span class="mx-2">[${log.level}]</span>
        <span>${log.message}</span>
      </div>
    `
      )
      .join('')
  } catch (e) {
    console.error('获取日志失败:', e)
  }
}

// ========== 设置管理 ==========
function loadSettingsToUI(config) {
  if (!config) return
  
  els.settingAutoConnect.checked = config.auto_connect || false
  els.settingAutoReconnect.checked = config.auto_reconnect !== false // 默认 true
  els.settingReconnectInterval.value = config.reconnect_interval || 5
  els.settingLogLevel.value = config.log_level || 'INFO'
  els.settingMinimizeToTray.checked = config.minimize_to_tray !== false // 默认 true
  els.settingStartMinimized.checked = config.start_minimized || false
}

async function saveSettings() {
  try {
    // 获取当前配置并更新设置
    const config = {
      server_url: els.serverUrl.value.trim() || 'localhost:50051',
      access_key: els.accessKey.value.trim(),
      secret_key: els.secretKey.value.trim(),
      auto_connect: els.settingAutoConnect.checked,
      auto_reconnect: els.settingAutoReconnect.checked,
      reconnect_interval: parseInt(els.settingReconnectInterval.value) || 5,
      log_level: els.settingLogLevel.value,
      minimize_to_tray: els.settingMinimizeToTray.checked,
      start_minimized: els.settingStartMinimized.checked
    }
    
    await window.go.main.App.SaveConfig(config)
    state.config = config
    
    // 显示保存成功提示
    showSettingsSaved()
  } catch (e) {
    console.error('保存设置失败:', e)
  }
}

function showSettingsSaved() {
  els.settingsSaved.classList.remove('hidden')
  lucide.createIcons()
  setTimeout(() => {
    els.settingsSaved.classList.add('hidden')
  }, 2000)
}

// ========== 工具函数 ==========
function updateTime() {
  els.currentTime.textContent = new Date().toLocaleTimeString('zh-CN')
}

// ========== 启动 ==========
document.addEventListener('DOMContentLoaded', init)
