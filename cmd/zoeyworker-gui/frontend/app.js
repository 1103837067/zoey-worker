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
  currentTime: $('currentTime')
}

// ========== 状态 ==========
let state = {
  connected: false,
  agentId: '',
  agentName: ''
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
    if (config.server_url) els.serverUrl.value = config.server_url
    if (config.access_key) els.accessKey.value = config.access_key
    if (config.secret_key) els.secretKey.value = config.secret_key
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
  try {
    await window.go.main.App.Disconnect()
  } catch (e) {
    console.error('断开连接失败:', e)
  }

  state.connected = false
  state.agentId = ''
  state.agentName = ''
  updateUI()
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

// ========== 工具函数 ==========
function updateTime() {
  els.currentTime.textContent = new Date().toLocaleTimeString('zh-CN')
}

// ========== 启动 ==========
document.addEventListener('DOMContentLoaded', init)
