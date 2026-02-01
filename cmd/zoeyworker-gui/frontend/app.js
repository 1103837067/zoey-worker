/**
 * Zoey Worker GUI - 前端应用 (Wails v3)
 */

// ========== Wails v3 API ==========
// Wails v3 使用 wails.Call.ByName("ServiceName.MethodName", args...)
function waitForWails() {
  return new Promise((resolve) => {
    const checkWails = () => {
      if (window.wails && window.wails.Call && window.wails.Call.ByName) {
        return window.wails.Call.ByName
      }
      return null
    }
    
    const result = checkWails()
    if (result) {
      resolve(result)
    } else {
      const check = setInterval(() => {
        const result = checkWails()
        if (result) {
          clearInterval(check)
          resolve(result)
        }
      }, 100)
      setTimeout(() => {
        clearInterval(check)
        console.error('Wails.Call.ByName not found')
        resolve(null)
      }, 5000)
    }
  })
}

let callByName = null

async function callBackend(method, ...args) {
  if (!callByName) callByName = await waitForWails()
  if (!callByName) throw new Error('Wails not available')
  return callByName(method, ...args)
}

// 封装后端调用 - Wails v3 格式: 包名.类型名.方法名
const SERVICE = 'main.App'
const App = {
  LoadConfig: () => callBackend(`${SERVICE}.LoadConfig`),
  SaveConfig: (config) => callBackend(`${SERVICE}.SaveConfig`, config),
  Connect: (url, accessKey, secretKey) => callBackend(`${SERVICE}.Connect`, url, accessKey, secretKey),
  Disconnect: () => callBackend(`${SERVICE}.Disconnect`),
  GetStatus: () => callBackend(`${SERVICE}.GetStatus`),
  GetLogs: (count) => callBackend(`${SERVICE}.GetLogs`, count),
  GetSystemInfo: () => callBackend(`${SERVICE}.GetSystemInfo`),
  CheckPermissions: () => callBackend(`${SERVICE}.CheckPermissions`),
  OpenAccessibilitySettings: () => callBackend(`${SERVICE}.OpenAccessibilitySettings`),
  OpenScreenRecordingSettings: () => callBackend(`${SERVICE}.OpenScreenRecordingSettings`),
  GetOCRPluginStatus: () => callBackend(`${SERVICE}.GetOCRPluginStatus`),
  InstallOCRPlugin: () => callBackend(`${SERVICE}.InstallOCRPlugin`),
  UninstallOCRPlugin: () => callBackend(`${SERVICE}.UninstallOCRPlugin`),
  ShowWindow: () => callBackend(`${SERVICE}.ShowWindow`),
  HideWindow: () => callBackend(`${SERVICE}.HideWindow`),
  QuitApp: () => callBackend(`${SERVICE}.QuitApp`),
  GetDebugData: (lastVersion) => callBackend(`${SERVICE}.GetDebugData`, lastVersion),
}

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
  refreshLogsBtn: $('refreshLogsBtn'),
  emptyLogs: $('emptyLogs'),
  logList: $('logList'),
  systemInfo: $('systemInfo'),
  currentTime: $('currentTime'),
  // Header 连接信息
  headerConnectionInfo: $('headerConnectionInfo'),
  headerAgentName: $('headerAgentName'),
  headerAgentId: $('headerAgentId'),
  copyAgentIdBtn: $('copyAgentIdBtn'),
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
  config: null,
  reconnecting: false,
  reconnectTimer: null
}

// ========== 初始化 ==========
async function init() {
  lucide.createIcons()
  bindEvents()
  updateTime()
  setInterval(updateTime, 1000)

  // 加载配置
  try {
    const config = await App.LoadConfig()
    state.config = config
    
    if (config.server_url) els.serverUrl.value = config.server_url
    if (config.access_key) els.accessKey.value = config.access_key
    if (config.secret_key) els.secretKey.value = config.secret_key
    
    loadSettingsToUI(config)
    
    // 自动连接
    if (config.auto_connect && config.server_url && config.access_key && config.secret_key) {
      setTimeout(() => connect(), 500)
    }
  } catch (e) {
    console.error('加载配置失败:', e)
  }

  // 加载系统信息
  try {
    const info = await App.GetSystemInfo()
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
  
  // 复制 Agent ID
  if (els.copyAgentIdBtn) {
    els.copyAgentIdBtn.addEventListener('click', copyAgentId)
  }
}

// ========== 复制 Agent ID ==========
async function copyAgentId() {
  if (!state.agentId) return
  
  try {
    await navigator.clipboard.writeText(state.agentId)
    // 显示复制成功反馈
    const icon = els.copyAgentIdBtn.querySelector('i')
    if (icon) {
      icon.setAttribute('data-lucide', 'check')
      lucide.createIcons()
      setTimeout(() => {
        icon.setAttribute('data-lucide', 'copy')
        lucide.createIcons()
      }, 1500)
    }
  } catch (e) {
    console.error('复制失败:', e)
  }
}

// ========== Tab 切换 ==========
function switchTab(tabName) {
  document.querySelectorAll('.tab-btn').forEach(btn => {
    const isActive = btn.dataset.tab === tabName
    btn.classList.toggle('border-primary', isActive)
    btn.classList.toggle('text-foreground', isActive)
    btn.classList.toggle('border-transparent', !isActive)
    btn.classList.toggle('text-muted-foreground', !isActive)
  })

  document.querySelectorAll('.tab-content').forEach(content => {
    content.classList.toggle('hidden', content.id !== `tab-${tabName}`)
  })

  lucide.createIcons()

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
    const result = await App.Connect(serverUrl, accessKey, secretKey)

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
  cancelReconnect()
  
  try {
    await App.Disconnect()
  } catch (e) {
    console.error('断开连接失败:', e)
  }

  state.connected = false
  state.agentId = ''
  state.agentName = ''
  setConnecting(false)
  updateUI()
}

async function checkConnectionStatus() {
  try {
    const status = await App.GetStatus()
    const wasConnected = state.connected
    
    state.connected = status.connected
    state.agentId = status.agent_id || ''
    state.agentName = status.agent_name || ''
    
    if (wasConnected !== state.connected) {
      setConnecting(false)
      updateUI()
      
      if (wasConnected && !state.connected && state.config?.auto_reconnect && !state.reconnecting) {
        scheduleReconnect()
      }
    }
  } catch (e) {
    console.error('检查连接状态失败:', e)
  }
}

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
      <span class="w-2 h-2 bg-emerald-500 rounded-full animate-pulse"></span>
      <span class="text-emerald-600">已连接</span>
    `
  } else {
    els.statusIndicator.innerHTML = `
      <span class="w-2 h-2 bg-gray-400 rounded-full"></span>
      <span class="text-muted-foreground">未连接</span>
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

  // Header 连接信息（使用 flex 显示）
  if (state.connected && els.headerConnectionInfo) {
    els.headerConnectionInfo.classList.remove('hidden')
    els.headerConnectionInfo.classList.add('flex')
    els.headerAgentName.textContent = state.agentName || '-'
    els.headerAgentName.title = state.agentName || ''
    els.headerAgentId.textContent = state.agentId || '-'
    lucide.createIcons()
  } else if (els.headerConnectionInfo) {
    els.headerConnectionInfo.classList.add('hidden')
    els.headerConnectionInfo.classList.remove('flex')
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
    const logs = await App.GetLogs(100)

    if (!logs || logs.length === 0) {
      els.emptyLogs.classList.remove('hidden')
      els.logList.innerHTML = ''
      return
    }

    els.emptyLogs.classList.add('hidden')
    els.logList.innerHTML = logs
      .map(
        log => `
      <div class="px-2 py-1 hover:bg-muted/50 rounded log-${log.level.toLowerCase()}">
        <span class="text-muted-foreground">${log.timestamp}</span>
        <span class="mx-2 font-medium">[${log.level}]</span>
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
  els.settingAutoReconnect.checked = config.auto_reconnect !== false
  els.settingReconnectInterval.value = config.reconnect_interval || 5
  els.settingLogLevel.value = config.log_level || 'INFO'
  els.settingMinimizeToTray.checked = config.minimize_to_tray !== false
  els.settingStartMinimized.checked = config.start_minimized || false
}

async function saveSettings() {
  try {
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
    
    await App.SaveConfig(config)
    state.config = config
    
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

// ========== 权限管理 (macOS) ==========
async function checkPermissions() {
  try {
    const info = await App.GetSystemInfo()
    if (info.platform !== 'macOS') {
      $('permissionsSection').classList.add('hidden')
      return
    }
    
    const permissions = await App.CheckPermissions()
    updatePermissionsUI(permissions)
  } catch (e) {
    console.error('检查权限失败:', e)
  }
}

function updatePermissionsUI(permissions) {
  const accessibilityStatus = $('accessibilityStatus')
  const screenRecordingStatus = $('screenRecordingStatus')
  const openAccessibilityBtn = $('openAccessibilityBtn')
  const openScreenRecordingBtn = $('openScreenRecordingBtn')
  const permissionWarning = $('permissionWarning')
  
  if (permissions.accessibility) {
    accessibilityStatus.innerHTML = `
      <span class="w-2 h-2 bg-emerald-500 rounded-full"></span>
      <span class="text-sm text-emerald-600">已授权</span>
    `
    openAccessibilityBtn.classList.add('hidden')
  } else {
    accessibilityStatus.innerHTML = `
      <span class="w-2 h-2 bg-red-500 rounded-full"></span>
      <span class="text-sm text-red-500">未授权</span>
    `
    openAccessibilityBtn.classList.remove('hidden')
  }
  
  if (permissions.screen_recording) {
    screenRecordingStatus.innerHTML = `
      <span class="w-2 h-2 bg-emerald-500 rounded-full"></span>
      <span class="text-sm text-emerald-600">已授权</span>
    `
    openScreenRecordingBtn.classList.add('hidden')
  } else {
    screenRecordingStatus.innerHTML = `
      <span class="w-2 h-2 bg-red-500 rounded-full"></span>
      <span class="text-sm text-red-500">未授权</span>
    `
    openScreenRecordingBtn.classList.remove('hidden')
  }
  
  if (!permissions.all_granted) {
    permissionWarning.classList.remove('hidden')
  } else {
    permissionWarning.classList.add('hidden')
  }
  
  lucide.createIcons()
}

function bindPermissionEvents() {
  const openAccessibilityBtn = $('openAccessibilityBtn')
  const openScreenRecordingBtn = $('openScreenRecordingBtn')
  
  if (openAccessibilityBtn) {
    openAccessibilityBtn.addEventListener('click', async () => {
      try {
        await App.OpenAccessibilitySettings()
      } catch (e) {
        console.error('打开辅助功能设置失败:', e)
      }
    })
  }
  
  if (openScreenRecordingBtn) {
    openScreenRecordingBtn.addEventListener('click', async () => {
      try {
        await App.OpenScreenRecordingSettings()
      } catch (e) {
        console.error('打开屏幕录制设置失败:', e)
      }
    })
  }
}

// ========== 后台运行提示 ==========
function setupBackgroundEvents() {
  // 监听最小化到后台事件
  if (window.runtime && window.runtime.EventsOn) {
    window.runtime.EventsOn('minimized-to-background', () => {
      showBackgroundNotification()
    })
  }
}

function showBackgroundNotification() {
  // 创建提示元素
  const notification = document.createElement('div')
  notification.className = 'fixed bottom-4 right-4 bg-card border shadow-lg rounded-lg p-4 max-w-sm z-50 animate-slide-up'
  notification.innerHTML = `
    <div class="flex items-start gap-3">
      <div class="w-8 h-8 rounded-md flex items-center justify-center flex-shrink-0" style="background-color: #00FFAE;">
        <svg width="16" height="16" viewBox="0 0 1864 1864" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path d="M231.385 758.89V308H1691C1666.84 346.916 1636.87 393.79 1602.42 448.903C1559.75 517.159 1493.7 624.026 1493.7 624.026C1493.7 624.026 1415.85 746.813 1376.93 807.2C1339.35 867.587 1304.45 923.277 1272.24 974.271C1240.03 1023.92 1214.53 1064.85 1195.74 1097.06C1195.74 1098.4 1195.07 1099.74 1193.72 1101.08C1195.07 1101.08 1195.74 1101.08 1195.74 1101.08H1658.79V1556H173L702.488 762.916C702.488 761.574 702.488 760.903 702.488 760.903C703.83 759.561 704.501 758.89 704.501 758.89C704.501 758.89 703.83 758.89 702.488 758.89H231.385Z" fill="#FFCC00"/>
        </svg>
      </div>
      <div class="flex-1">
        <p class="text-sm font-medium">应用已在后台运行</p>
        <p class="text-xs text-muted-foreground mt-1">点击系统托盘图标可重新打开窗口</p>
      </div>
      <button onclick="this.parentElement.parentElement.remove()" class="text-muted-foreground hover:text-foreground">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
        </svg>
      </button>
    </div>
  `
  document.body.appendChild(notification)
  
  // 3 秒后自动移除
  setTimeout(() => {
    notification.remove()
  }, 3000)
}

// ========== 调试面板 ==========
const debugState = {
  history: [], // 匹配历史记录
  maxHistory: 20,
  lastVersion: 0, // 上次获取的数据版本
  pollInterval: null // 轮询定时器
}

// 设置调试数据轮询（类似日志的方式）
function setupDebugPolling() {
  // 每 500ms 轮询一次调试数据
  debugState.pollInterval = setInterval(async () => {
    try {
      const data = await App.GetDebugData(debugState.lastVersion)
      if (data && data.version > debugState.lastVersion) {
        console.log('收到新的调试数据:', data)
        debugState.lastVersion = data.version
        updateDebugPanel(data)
      }
    } catch (e) {
      // 忽略错误，继续轮询
    }
  }, 500)
  
  console.log('调试数据轮询已启动')
  
  // 绑定清空历史按钮
  const clearBtn = document.getElementById('clearDebugHistoryBtn')
  if (clearBtn) {
    clearBtn.onclick = () => {
      debugState.history = []
      updateDebugHistory()
    }
  }
}

// 停止调试轮询
function stopDebugPolling() {
  if (debugState.pollInterval) {
    clearInterval(debugState.pollInterval)
    debugState.pollInterval = null
  }
}

// 更新调试面板
function updateDebugPanel(data) {
  console.log('更新调试面板:', data)
  
  // 更新状态（根据 status 字段）
  const statusEl = document.getElementById('debugStatus')
  if (statusEl) {
    const statusMap = {
      'searching': { color: 'yellow', text: '搜索中...' },
      'found': { color: 'green', text: '匹配成功' },
      'not_found': { color: 'red', text: '未找到' },
      'error': { color: 'red', text: '错误' }
    }
    const status = statusMap[data.status] || statusMap['searching']
    statusEl.innerHTML = `<span class="w-2 h-2 bg-${status.color}-500 rounded-full animate-pulse"></span><span class="text-${status.color}-600">${status.text}</span>`
  }
  
  // 更新信息
  const taskIdEl = document.getElementById('debugTaskId')
  const actionTypeEl = document.getElementById('debugActionType')
  const matchResultEl = document.getElementById('debugMatchResult')
  const confidenceEl = document.getElementById('debugConfidence')
  const positionEl = document.getElementById('debugPosition')
  const durationEl = document.getElementById('debugDuration')
  
  if (taskIdEl) taskIdEl.textContent = data.task_id || '-'
  if (actionTypeEl) actionTypeEl.textContent = data.action_type || '-'
  if (matchResultEl) matchResultEl.textContent = data.status === 'searching' ? '搜索中' : (data.matched ? '成功' : '失败')
  if (confidenceEl) confidenceEl.textContent = data.confidence ? `${(data.confidence * 100).toFixed(1)}%` : '-'
  if (positionEl) positionEl.textContent = data.matched ? `(${data.x}, ${data.y})` : '-'
  if (durationEl) durationEl.textContent = data.duration_ms ? `${data.duration_ms}ms` : '-'
  
  // 更新目标图片（支持 data:image/png;base64,... 格式或纯 base64）
  const templateImg = document.getElementById('debugTemplateImg')
  const templateEmpty = document.getElementById('debugTemplateEmpty')
  if (data.template_base64 && templateImg && templateEmpty) {
    // 如果已经是 data: URL 格式，直接使用；否则添加前缀
    templateImg.src = data.template_base64.startsWith('data:') 
      ? data.template_base64 
      : `data:image/png;base64,${data.template_base64}`
    templateImg.classList.remove('hidden')
    templateEmpty.classList.add('hidden')
  } else if (templateImg && templateEmpty) {
    templateImg.classList.add('hidden')
    templateEmpty.classList.remove('hidden')
  }
  
  // 更新截图（支持 data:image/png;base64,... 格式或纯 base64）
  const screenImg = document.getElementById('debugScreenshotImg')
  const screenEmpty = document.getElementById('debugScreenshotEmpty')
  if (data.screen_base64 && screenImg && screenEmpty) {
    // 如果已经是 data: URL 格式，直接使用；否则添加前缀
    screenImg.src = data.screen_base64.startsWith('data:') 
      ? data.screen_base64 
      : `data:image/png;base64,${data.screen_base64}`
    screenImg.classList.remove('hidden')
    screenEmpty.classList.add('hidden')
  } else if (screenImg && screenEmpty) {
    screenImg.classList.add('hidden')
    screenEmpty.classList.remove('hidden')
  }
  
  // 显示错误信息
  if (data.error) {
    console.error('匹配错误:', data.error)
  }
  
  // 只在最终状态时添加到历史记录（不是 searching）
  if (data.status !== 'searching') {
    debugState.history.unshift({
      ...data,
      timestamp: new Date().toLocaleTimeString()
    })
    if (debugState.history.length > debugState.maxHistory) {
      debugState.history.pop()
    }
    updateDebugHistory()
  }
}

// 更新调试历史列表
function updateDebugHistory() {
  const listEl = document.getElementById('debugHistoryList')
  const emptyEl = document.getElementById('debugHistoryEmpty')
  
  if (debugState.history.length === 0) {
    emptyEl.classList.remove('hidden')
    listEl.innerHTML = ''
    listEl.appendChild(emptyEl)
    return
  }
  
  emptyEl.classList.add('hidden')
  listEl.innerHTML = debugState.history.map((item, idx) => `
    <div class="flex items-center justify-between px-3 py-2 hover:bg-muted/50 cursor-pointer border-b last:border-0 text-xs" onclick="selectDebugHistory(${idx})">
      <div class="flex items-center gap-2">
        <span class="w-2 h-2 rounded-full ${item.matched ? 'bg-green-500' : 'bg-red-500'}"></span>
        <span class="font-medium">${item.action_type || '未知'}</span>
      </div>
      <div class="flex items-center gap-3 text-muted-foreground">
        <span>${item.confidence ? (item.confidence * 100).toFixed(0) + '%' : '-'}</span>
        <span>${item.duration_ms || 0}ms</span>
        <span>${item.timestamp}</span>
      </div>
    </div>
  `).join('')
}

// 选择历史记录项
window.selectDebugHistory = function(idx) {
  const item = debugState.history[idx]
  if (item) {
    updateDebugPanel(item)
  }
}

// ========== 启动 ==========
document.addEventListener('DOMContentLoaded', () => {
  init()
  checkPermissions()
  bindPermissionEvents()
  setupBackgroundEvents()
  setupDebugPolling() // 使用轮询方式获取调试数据
})
