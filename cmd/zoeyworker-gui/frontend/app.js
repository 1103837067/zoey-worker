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
  RequestPermissions: () => callBackend(`${SERVICE}.RequestPermissions`),
  OpenAccessibilitySettings: () => callBackend(`${SERVICE}.OpenAccessibilitySettings`),
  OpenScreenRecordingSettings: () => callBackend(`${SERVICE}.OpenScreenRecordingSettings`),
  ResetPermissions: () => callBackend(`${SERVICE}.ResetPermissions`),
  GetPythonInfo: () => callBackend(`${SERVICE}.GetPythonInfo`),
  RefreshPythonInfo: () => callBackend(`${SERVICE}.RefreshPythonInfo`),
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
  reconnectTimer: null,
  // 权限状态
  permissions: {
    accessibility: false,
    screen_recording: false,
    allGranted: false
  },
  isMacOS: false,
  permissionModalShown: false
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

// 检查权限并返回状态
async function checkPermissions(showModal = false) {
  try {
    const info = await App.GetSystemInfo()
    state.isMacOS = info.platform === 'macOS'
    
    if (!state.isMacOS) {
      $('permissionsSection')?.classList.add('hidden')
      state.permissions = { accessibility: true, screen_recording: true, allGranted: true }
      return state.permissions
    }
    
    const permissions = await App.CheckPermissions()
    state.permissions = {
      accessibility: permissions.accessibility,
      screen_recording: permissions.screen_recording,
      allGranted: permissions.accessibility && permissions.screen_recording
    }
    
    updatePermissionsUI(permissions)
    
    // 如果权限不完整且需要显示弹窗，则显示引导弹窗
    if (showModal && !state.permissions.allGranted && !state.permissionModalShown) {
      showPermissionModal()
    }
    
    return state.permissions
  } catch (e) {
    console.error('检查权限失败:', e)
    return { accessibility: false, screen_recording: false, allGranted: false }
  }
}

// 显示权限引导弹窗
function showPermissionModal() {
  state.permissionModalShown = true
  
  // 移除已存在的弹窗
  const existingModal = document.getElementById('permissionModal')
  if (existingModal) existingModal.remove()
  
  const modal = document.createElement('div')
  modal.id = 'permissionModal'
  modal.className = 'fixed inset-0 z-50 flex items-center justify-center bg-black/50'
  modal.innerHTML = `
    <div class="bg-card rounded-xl shadow-2xl max-w-md w-full mx-4 overflow-hidden animate-slide-up">
      <!-- 头部 -->
      <div class="px-6 py-5 border-b bg-gradient-to-r from-amber-50 to-orange-50">
        <div class="flex items-center gap-3">
          <div class="w-12 h-12 rounded-xl bg-amber-100 flex items-center justify-center">
            <i data-lucide="shield-alert" class="w-6 h-6 text-amber-600"></i>
          </div>
          <div>
            <h2 class="text-lg font-semibold text-foreground">需要系统权限</h2>
            <p class="text-sm text-muted-foreground">请授权以下权限以正常使用</p>
          </div>
        </div>
      </div>
      
      <!-- 权限列表 -->
      <div class="px-6 py-4 space-y-4">
        <!-- 辅助功能权限 -->
        <div class="flex items-start gap-4 p-4 rounded-lg border ${state.permissions.accessibility ? 'bg-emerald-50 border-emerald-200' : 'bg-red-50 border-red-200'}">
          <div class="w-10 h-10 rounded-lg ${state.permissions.accessibility ? 'bg-emerald-100' : 'bg-red-100'} flex items-center justify-center flex-shrink-0">
            <i data-lucide="${state.permissions.accessibility ? 'check-circle' : 'x-circle'}" class="w-5 h-5 ${state.permissions.accessibility ? 'text-emerald-600' : 'text-red-600'}"></i>
          </div>
          <div class="flex-1 min-w-0">
            <div class="flex items-center justify-between">
              <h3 class="font-medium ${state.permissions.accessibility ? 'text-emerald-900' : 'text-red-900'}">辅助功能</h3>
              <span class="text-xs px-2 py-0.5 rounded-full ${state.permissions.accessibility ? 'bg-emerald-200 text-emerald-700' : 'bg-red-200 text-red-700'}">
                ${state.permissions.accessibility ? '已授权' : '未授权'}
              </span>
            </div>
            <p class="text-sm ${state.permissions.accessibility ? 'text-emerald-700' : 'text-red-700'} mt-1">用于控制鼠标、键盘和窗口操作</p>
            ${!state.permissions.accessibility ? `
              <button onclick="openAccessibilityFromModal()" class="mt-2 text-sm font-medium text-red-700 hover:text-red-800 flex items-center gap-1">
                <i data-lucide="external-link" class="w-3.5 h-3.5"></i>
                打开设置
              </button>
            ` : ''}
          </div>
        </div>
        
        <!-- 屏幕录制权限 -->
        <div class="flex items-start gap-4 p-4 rounded-lg border ${state.permissions.screen_recording ? 'bg-emerald-50 border-emerald-200' : 'bg-red-50 border-red-200'}">
          <div class="w-10 h-10 rounded-lg ${state.permissions.screen_recording ? 'bg-emerald-100' : 'bg-red-100'} flex items-center justify-center flex-shrink-0">
            <i data-lucide="${state.permissions.screen_recording ? 'check-circle' : 'x-circle'}" class="w-5 h-5 ${state.permissions.screen_recording ? 'text-emerald-600' : 'text-red-600'}"></i>
          </div>
          <div class="flex-1 min-w-0">
            <div class="flex items-center justify-between">
              <h3 class="font-medium ${state.permissions.screen_recording ? 'text-emerald-900' : 'text-red-900'}">屏幕录制</h3>
              <span class="text-xs px-2 py-0.5 rounded-full ${state.permissions.screen_recording ? 'bg-emerald-200 text-emerald-700' : 'bg-red-200 text-red-700'}">
                ${state.permissions.screen_recording ? '已授权' : '未授权'}
              </span>
            </div>
            <p class="text-sm ${state.permissions.screen_recording ? 'text-emerald-700' : 'text-red-700'} mt-1">用于截图和图像识别匹配</p>
            ${!state.permissions.screen_recording ? `
              <button onclick="openScreenRecordingFromModal()" class="mt-2 text-sm font-medium text-red-700 hover:text-red-800 flex items-center gap-1">
                <i data-lucide="external-link" class="w-3.5 h-3.5"></i>
                打开设置
              </button>
            ` : ''}
          </div>
        </div>
        
        <!-- 提示信息 -->
        <div class="bg-blue-50 border border-blue-200 rounded-lg p-3">
          <p class="text-sm text-blue-800">
            <i data-lucide="info" class="w-4 h-4 inline mr-1.5"></i>
            <strong>提示：</strong>授权后需要<strong>重启应用</strong>才能生效。请在系统设置中勾选 "ZoeyWorker"。
          </p>
        </div>
      </div>
      
      <!-- 底部按钮 -->
      <div class="px-6 py-4 border-t bg-muted/30 flex gap-3">
        <button onclick="refreshPermissionsInModal()" class="flex-1 bg-secondary hover:bg-secondary/80 text-secondary-foreground font-medium py-2.5 px-4 rounded-lg transition-colors text-sm flex items-center justify-center gap-2">
          <i data-lucide="refresh-cw" class="w-4 h-4"></i>
          刷新状态
        </button>
        <button onclick="closePermissionModal()" class="flex-1 ${state.permissions.allGranted ? 'bg-primary hover:bg-primary/90 text-primary-foreground' : 'bg-muted hover:bg-muted/80 text-muted-foreground'} font-medium py-2.5 px-4 rounded-lg transition-colors text-sm">
          ${state.permissions.allGranted ? '开始使用' : '稍后设置'}
        </button>
      </div>
    </div>
  `
  
  document.body.appendChild(modal)
  lucide.createIcons()
}

// 从弹窗打开辅助功能设置
window.openAccessibilityFromModal = async function() {
  try {
    await App.OpenAccessibilitySettings()
  } catch (e) {
    console.error('打开辅助功能设置失败:', e)
  }
}

// 从弹窗打开屏幕录制设置
window.openScreenRecordingFromModal = async function() {
  try {
    await App.OpenScreenRecordingSettings()
  } catch (e) {
    console.error('打开屏幕录制设置失败:', e)
  }
}

// 在弹窗中刷新权限状态
window.refreshPermissionsInModal = async function() {
  const btn = document.querySelector('#permissionModal button:first-of-type')
  if (btn) {
    btn.disabled = true
    btn.innerHTML = '<i data-lucide="loader-2" class="w-4 h-4 animate-spin"></i> 检查中...'
    lucide.createIcons()
  }
  
  await checkPermissions(false)
  
  // 重新渲染弹窗
  closePermissionModal()
  if (!state.permissions.allGranted) {
    showPermissionModal()
  } else {
    // 权限已全部授权，显示成功提示
    showPermissionSuccessToast()
  }
}

// 关闭权限弹窗
window.closePermissionModal = function() {
  const modal = document.getElementById('permissionModal')
  if (modal) modal.remove()
}

// 显示权限授权成功提示
function showPermissionSuccessToast() {
  const toast = document.createElement('div')
  toast.className = 'fixed bottom-4 right-4 bg-emerald-500 text-white px-4 py-3 rounded-lg shadow-lg flex items-center gap-2 z-50 animate-slide-up'
  toast.innerHTML = `
    <i data-lucide="check-circle" class="w-5 h-5"></i>
    <span>所有权限已授权，可以正常使用了！</span>
  `
  document.body.appendChild(toast)
  lucide.createIcons()
  
  setTimeout(() => toast.remove(), 3000)
}

function updatePermissionsUI(permissions) {
  const accessibilityStatus = $('accessibilityStatus')
  const screenRecordingStatus = $('screenRecordingStatus')
  const openAccessibilityBtn = $('openAccessibilityBtn')
  const openScreenRecordingBtn = $('openScreenRecordingBtn')
  const permissionWarning = $('permissionWarning')
  const refreshPermissionsBtn = $('refreshPermissionsBtn')
  
  if (permissions.accessibility) {
    accessibilityStatus.innerHTML = `
      <span class="w-2 h-2 bg-emerald-500 rounded-full"></span>
      <span class="text-sm text-emerald-600">已授权</span>
    `
    openAccessibilityBtn?.classList.add('hidden')
  } else {
    accessibilityStatus.innerHTML = `
      <span class="w-2 h-2 bg-red-500 rounded-full animate-pulse"></span>
      <span class="text-sm text-red-500 font-medium">未授权</span>
    `
    openAccessibilityBtn?.classList.remove('hidden')
  }
  
  if (permissions.screen_recording) {
    screenRecordingStatus.innerHTML = `
      <span class="w-2 h-2 bg-emerald-500 rounded-full"></span>
      <span class="text-sm text-emerald-600">已授权</span>
    `
    openScreenRecordingBtn?.classList.add('hidden')
  } else {
    screenRecordingStatus.innerHTML = `
      <span class="w-2 h-2 bg-red-500 rounded-full animate-pulse"></span>
      <span class="text-sm text-red-500 font-medium">未授权</span>
    `
    openScreenRecordingBtn?.classList.remove('hidden')
  }
  
  const allGranted = permissions.accessibility && permissions.screen_recording
  if (!allGranted) {
    permissionWarning?.classList.remove('hidden')
  } else {
    permissionWarning?.classList.add('hidden')
  }
  
  lucide.createIcons()
}

function bindPermissionEvents() {
  const openAccessibilityBtn = $('openAccessibilityBtn')
  const openScreenRecordingBtn = $('openScreenRecordingBtn')
  const refreshPermissionsBtn = $('refreshPermissionsBtn')
  const resetPermissionsBtn = $('resetPermissionsBtn')
  
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
  
  if (refreshPermissionsBtn) {
    refreshPermissionsBtn.addEventListener('click', async () => {
      refreshPermissionsBtn.disabled = true
      refreshPermissionsBtn.innerHTML = '<i data-lucide="loader-2" class="w-4 h-4 animate-spin"></i>'
      lucide.createIcons()
      
      await checkPermissions(false)
      
      refreshPermissionsBtn.disabled = false
      refreshPermissionsBtn.innerHTML = '<i data-lucide="refresh-cw" class="w-4 h-4"></i>'
      lucide.createIcons()
    })
  }
  
  if (resetPermissionsBtn) {
    resetPermissionsBtn.addEventListener('click', resetPermissions)
  }
}

// 重置权限
async function resetPermissions() {
  const btn = $('resetPermissionsBtn')
  if (btn) {
    btn.disabled = true
    btn.innerHTML = '<i data-lucide="loader-2" class="w-4 h-4 animate-spin"></i> 重置中...'
    lucide.createIcons()
  }
  
  try {
    await App.ResetPermissions()
    
    // 显示提示
    showResetSuccessModal()
  } catch (e) {
    console.error('重置权限失败:', e)
    alert('重置权限失败: ' + e.message)
  } finally {
    if (btn) {
      btn.disabled = false
      btn.innerHTML = '<i data-lucide="rotate-ccw" class="w-4 h-4"></i> 重置权限'
      lucide.createIcons()
    }
  }
}

// 显示重置成功提示弹窗
function showResetSuccessModal() {
  const modal = document.createElement('div')
  modal.className = 'fixed inset-0 z-50 flex items-center justify-center bg-black/50'
  modal.innerHTML = `
    <div class="bg-card rounded-xl shadow-2xl max-w-sm w-full mx-4 overflow-hidden animate-slide-up">
      <div class="px-6 py-5 text-center">
        <div class="w-16 h-16 rounded-full bg-emerald-100 flex items-center justify-center mx-auto mb-4">
          <i data-lucide="check-circle" class="w-8 h-8 text-emerald-600"></i>
        </div>
        <h2 class="text-lg font-semibold text-foreground mb-2">权限已重置</h2>
        <p class="text-sm text-muted-foreground mb-4">
          权限已清除，请按以下步骤重新授权：
        </p>
        <ol class="text-sm text-left bg-muted/50 rounded-lg p-4 space-y-2 mb-4">
          <li class="flex items-start gap-2">
            <span class="w-5 h-5 rounded-full bg-primary text-primary-foreground text-xs flex items-center justify-center flex-shrink-0 mt-0.5">1</span>
            <span>重启应用（必须完全退出再打开）</span>
          </li>
          <li class="flex items-start gap-2">
            <span class="w-5 h-5 rounded-full bg-primary text-primary-foreground text-xs flex items-center justify-center flex-shrink-0 mt-0.5">2</span>
            <span>系统会弹出权限请求，点击"打开系统设置"</span>
          </li>
          <li class="flex items-start gap-2">
            <span class="w-5 h-5 rounded-full bg-primary text-primary-foreground text-xs flex items-center justify-center flex-shrink-0 mt-0.5">3</span>
            <span>在系统设置中勾选 "ZoeyWorker"</span>
          </li>
        </ol>
      </div>
      <div class="px-6 py-4 border-t bg-muted/30 flex gap-3">
        <button onclick="this.closest('.fixed').remove()" class="flex-1 bg-secondary hover:bg-secondary/80 text-secondary-foreground font-medium py-2.5 px-4 rounded-lg transition-colors text-sm">
          稍后重启
        </button>
        <button onclick="restartApp()" class="flex-1 bg-primary hover:bg-primary/90 text-primary-foreground font-medium py-2.5 px-4 rounded-lg transition-colors text-sm">
          立即重启
        </button>
      </div>
    </div>
  `
  document.body.appendChild(modal)
  lucide.createIcons()
}

// 重启应用
window.restartApp = async function() {
  try {
    await App.QuitApp()
  } catch (e) {
    console.error('退出应用失败:', e)
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

// ========== Python 环境检测 ==========

async function checkPythonInfo() {
  try {
    const info = await App.GetPythonInfo()
    updatePythonUI(info)
    return info
  } catch (e) {
    console.error('检查 Python 环境失败:', e)
    updatePythonUI({ available: false, version: '', path: '' })
    return null
  }
}

function updatePythonUI(info) {
  const statusEl = $('pythonStatus')
  const nameEl = $('pythonName')
  const pathEl = $('pythonPath')
  const notFoundEl = $('pythonNotFound')
  
  if (info && info.available) {
    if (statusEl) {
      statusEl.innerHTML = `
        <span class="w-2 h-2 bg-emerald-500 rounded-full"></span>
        <span class="text-sm text-emerald-600">v${info.version}</span>
      `
    }
    if (nameEl) nameEl.textContent = `Python ${info.version}`
    if (pathEl) pathEl.textContent = info.path
    notFoundEl?.classList.add('hidden')
  } else {
    if (statusEl) {
      statusEl.innerHTML = `
        <span class="w-2 h-2 bg-red-500 rounded-full"></span>
        <span class="text-sm text-red-600">未安装</span>
      `
    }
    if (nameEl) nameEl.textContent = 'Python 3'
    if (pathEl) pathEl.textContent = '未检测到 Python 3 环境'
    notFoundEl?.classList.remove('hidden')
  }
  
  lucide.createIcons()
}

function bindPythonEvents() {
  const refreshBtn = $('refreshPythonBtn')
  if (refreshBtn) {
    refreshBtn.addEventListener('click', async () => {
      refreshBtn.disabled = true
      refreshBtn.innerHTML = '<i data-lucide="loader-2" class="w-4 h-4 animate-spin"></i>'
      lucide.createIcons()
      
      try {
        const info = await App.RefreshPythonInfo()
        updatePythonUI(info)
      } catch (e) {
        console.error('刷新 Python 检测失败:', e)
      }
      
      refreshBtn.disabled = false
      refreshBtn.innerHTML = '<i data-lucide="refresh-cw" class="w-4 h-4"></i>'
      lucide.createIcons()
    })
  }
}

// ========== OCR 模块管理 ==========

// 检查 OCR 状态
async function checkOCRStatus() {
  try {
    const status = await App.GetOCRPluginStatus()
    updateOCRUI(status.installed)
    return status.installed
  } catch (e) {
    console.error('检查 OCR 状态失败:', e)
    return false
  }
}

// 更新 OCR UI
function updateOCRUI(installed) {
  const ocrStatus = $('ocrStatus')
  const installBtn = $('installOCRBtn')
  const progressDiv = $('ocrInstallProgress')
  
  if (installed) {
    ocrStatus.innerHTML = `
      <span class="w-2 h-2 bg-emerald-500 rounded-full"></span>
      <span class="text-sm text-emerald-600">已加载</span>
    `
    installBtn?.classList.add('hidden')
  } else {
    ocrStatus.innerHTML = `
      <span class="w-2 h-2 bg-amber-500 rounded-full animate-pulse"></span>
      <span class="text-sm text-amber-600">未安装</span>
    `
    installBtn?.classList.remove('hidden')
  }
  
  progressDiv?.classList.add('hidden')
  lucide.createIcons()
}

// 安装 OCR 模型
async function installOCR() {
  const installBtn = $('installOCRBtn')
  const progressDiv = $('ocrInstallProgress')
  const progressBar = $('ocrProgressBar')
  const progressText = $('ocrProgressText')
  
  // 显示进度条
  installBtn?.classList.add('hidden')
  progressDiv?.classList.remove('hidden')
  
  // 模拟进度（因为后端目前没有实时进度回调）
  let progress = 0
  const progressInterval = setInterval(() => {
    if (progress < 90) {
      progress += Math.random() * 10
      if (progress > 90) progress = 90
      progressBar.style.width = `${progress}%`
      progressText.textContent = `${Math.round(progress)}%`
    }
  }, 500)
  
  try {
    await App.InstallOCRPlugin()
    
    // 完成
    clearInterval(progressInterval)
    progressBar.style.width = '100%'
    progressText.textContent = '100%'
    
    setTimeout(async () => {
      await checkOCRStatus()
      showOCRSuccessToast()
    }, 500)
  } catch (e) {
    clearInterval(progressInterval)
    console.error('安装 OCR 失败:', e)
    
    progressDiv?.classList.add('hidden')
    installBtn?.classList.remove('hidden')
    
    alert('安装 OCR 模型失败: ' + e.message)
  }
}

// 显示 OCR 安装成功提示
function showOCRSuccessToast() {
  const toast = document.createElement('div')
  toast.className = 'fixed bottom-4 right-4 bg-emerald-500 text-white px-4 py-3 rounded-lg shadow-lg flex items-center gap-2 z-50 animate-slide-up'
  toast.innerHTML = `
    <i data-lucide="check-circle" class="w-5 h-5"></i>
    <span>OCR 模型安装成功！</span>
  `
  document.body.appendChild(toast)
  lucide.createIcons()
  
  setTimeout(() => toast.remove(), 3000)
}

// 绑定 OCR 事件
function bindOCREvents() {
  const refreshBtn = $('refreshOCRBtn')
  const installBtn = $('installOCRBtn')
  
  if (refreshBtn) {
    refreshBtn.addEventListener('click', async () => {
      refreshBtn.disabled = true
      refreshBtn.innerHTML = '<i data-lucide="loader-2" class="w-4 h-4 animate-spin"></i>'
      lucide.createIcons()
      
      await checkOCRStatus()
      
      refreshBtn.disabled = false
      refreshBtn.innerHTML = '<i data-lucide="refresh-cw" class="w-4 h-4"></i>'
      lucide.createIcons()
    })
  }
  
  if (installBtn) {
    installBtn.addEventListener('click', installOCR)
  }
}

// ========== 启动 ==========
document.addEventListener('DOMContentLoaded', async () => {
  init()
  bindPermissionEvents()
  bindPythonEvents()
  bindOCREvents()
  setupBackgroundEvents()
  setupDebugPolling() // 使用轮询方式获取调试数据
  
  // 检查权限并在需要时显示引导弹窗
  await checkPermissions(true)
  
  // 检查 Python 环境（使用启动时预热的缓存）
  await checkPythonInfo()
  
  // 检查 OCR 状态
  await checkOCRStatus()
})
