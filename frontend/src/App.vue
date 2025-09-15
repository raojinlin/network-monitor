<template>
  <div id="app">
    <header class="app-header">
      <div class="header-brand">
        <svg class="logo" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/>
        </svg>
        <h1>Network Monitor</h1>
      </div>
      <div class="header-controls">
        <div class="interface-selector">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="2" y="7" width="20" height="10" rx="2" ry="2"/>
            <circle cx="8.5" cy="12" r="1.5"/>
            <circle cx="15.5" cy="12" r="1.5"/>
          </svg>
          <select v-model="selectedInterface" @change="onInterfaceChange" class="interface-select">
            <option value="">选择网络接口...</option>
            <option v-for="iface in interfaces" :key="iface.name" :value="iface.name">
              {{ iface.name }} {{ iface.description ? `- ${iface.description}` : '' }}
            </option>
          </select>
        </div>
        <div class="status" :title="connected ? '实时数据连接正常' : '等待连接...'">
          <span :class="['status-indicator', connected ? 'connected' : 'disconnected']">
            <span class="pulse"></span>
          </span>
          <span class="status-text">{{ connected ? '已连接' : '未连接' }}</span>
        </div>
      </div>
    </header>

    <nav class="app-nav">
      <div class="nav-wrapper">
        <button 
          :class="['nav-button', activeTab === 'realtime' && 'active']"
          @click="activeTab = 'realtime'"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
          </svg>
          <span>实时流量</span>
        </button>
        <button 
          :class="['nav-button', activeTab === 'history' && 'active']"
          @click="activeTab = 'history'"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <polyline points="12 6 12 12 16 14"/>
          </svg>
          <span>历史数据</span>
        </button>
      </div>
    </nav>

    <main class="app-main">
      <RealtimeTraffic v-if="activeTab === 'realtime'" :connected="connected" />
      <HistoricalTraffic v-else />
    </main>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import RealtimeTraffic from './components/RealtimeTraffic.vue'
import HistoricalTraffic from './components/HistoricalTraffic.vue'

const activeTab = ref('realtime')
const connected = ref(false)
const interfaces = ref([])
const selectedInterface = ref('')
let ws = null

onMounted(() => {
  connectWebSocket()
  fetchInterfaces()
})

onUnmounted(() => {
  if (ws) {
    ws.close()
  }
})

function connectWebSocket() {
  // In development, use the proxy configured in vite.config.js
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  
  // Use the current host (which will use Vite's proxy in development)
  ws = new WebSocket(`${protocol}//${host}/ws`)
  
  console.log(`Attempting WebSocket connection to: ${protocol}//${host}/ws`)
  
  ws.onopen = () => {
    connected.value = true
    console.log('WebSocket connected')
  }
  
  ws.onclose = () => {
    connected.value = false
    console.log('WebSocket disconnected')
    // Reconnect after 3 seconds
    setTimeout(connectWebSocket, 3000)
  }
  
  ws.onerror = (error) => {
    console.error('WebSocket error:', error)
  }
  
  ws.onmessage = (event) => {
    const data = JSON.parse(event.data)
    // Emit data to child components via event bus or state management
    window.dispatchEvent(new CustomEvent('traffic-update', { detail: data }))
  }
}

async function fetchInterfaces() {
  try {
    const response = await fetch('/api/interfaces')
    if (response.ok) {
      const data = await response.json()
      interfaces.value = data.interfaces || []
      // Set current interface
      if (data.current) {
        selectedInterface.value = data.current
      } else if (interfaces.value.length > 0 && !selectedInterface.value) {
        selectedInterface.value = interfaces.value[0].name
      }
    }
  } catch (error) {
    console.error('Failed to fetch interfaces:', error)
  }
}

async function onInterfaceChange() {
  if (!selectedInterface.value) return
  
  try {
    const response = await fetch('/api/interfaces/switch', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ interface: selectedInterface.value })
    })
    
    if (response.ok) {
      const result = await response.json()
      console.log('Successfully switched to interface:', result.interface)
      // Optionally show a success message
      // alert(`成功切换到接口: ${result.interface}`)
    } else {
      const error = await response.text()
      alert(`切换接口失败: ${error}`)
      // Revert to previous interface
      fetchInterfaces()
    }
  } catch (error) {
    console.error('Failed to switch interface:', error)
    alert('切换接口时发生错误')
    // Revert to previous interface
    fetchInterfaces()
  }
}
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
  background-color: #f8fafc;
  color: #1a202c;
  line-height: 1.5;
}

#app {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background: linear-gradient(to bottom, #f8fafc 0%, #e2e8f0 100%);
}

.app-header {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 1rem 2rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
}

.header-brand {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.logo {
  animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.app-header h1 {
  font-size: 1.5rem;
  font-weight: 600;
  letter-spacing: -0.025em;
}

.header-controls {
  display: flex;
  align-items: center;
  gap: 2rem;
}

.interface-selector {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background: rgba(255, 255, 255, 0.1);
  padding: 0.25rem 0.5rem 0.25rem 0.75rem;
  border-radius: 9999px;
  backdrop-filter: blur(10px);
}

.interface-select {
  padding: 0.5rem 0.75rem;
  border: none;
  background: transparent;
  color: white;
  font-size: 0.875rem;
  cursor: pointer;
  outline: none;
  min-width: 150px;
}

.interface-select option {
  background-color: #4c1d95;
  color: white;
}

.status {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  font-size: 0.875rem;
  background: rgba(255, 255, 255, 0.1);
  padding: 0.5rem 1rem;
  border-radius: 9999px;
  backdrop-filter: blur(10px);
}

.status-indicator {
  position: relative;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background-color: #ef4444;
}

.status-indicator.connected {
  background-color: #10b981;
}

.pulse {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  border-radius: 50%;
  background: inherit;
  opacity: 0.75;
  animation: ping 1s cubic-bezier(0, 0, 0.2, 1) infinite;
}

@keyframes ping {
  75%, 100% {
    transform: scale(2);
    opacity: 0;
  }
}

.app-nav {
  background-color: white;
  border-bottom: 1px solid #e5e7eb;
  box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.1), 0 1px 2px 0 rgba(0, 0, 0, 0.06);
}

.nav-wrapper {
  max-width: 1280px;
  margin: 0 auto;
  padding: 0 2rem;
  display: flex;
  gap: 0.5rem;
}

.nav-button {
  background: none;
  border: none;
  padding: 1rem 1.5rem;
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 500;
  color: #6b7280;
  border-bottom: 3px solid transparent;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.nav-button:hover {
  color: #4b5563;
  background-color: #f9fafb;
}

.nav-button.active {
  color: #7c3aed;
  border-bottom-color: #7c3aed;
}

.nav-button svg {
  transition: transform 0.2s ease;
}

.nav-button:hover svg {
  transform: translateY(-1px);
}

.app-main {
  flex: 1;
  padding: 2rem;
  overflow-y: auto;
  max-width: 1280px;
  margin: 0 auto;
  width: 100%;
}

/* Responsive design */
@media (max-width: 768px) {
  .app-header {
    flex-direction: column;
    gap: 1rem;
    padding: 1rem;
  }
  
  .header-controls {
    flex-direction: column;
    width: 100%;
    gap: 1rem;
  }
  
  .interface-selector {
    width: 100%;
    justify-content: center;
  }
  
  .nav-wrapper {
    padding: 0 1rem;
  }
  
  .nav-button {
    flex: 1;
    justify-content: center;
    padding: 0.75rem 1rem;
  }
  
  .nav-button span {
    display: none;
  }
  
  .app-main {
    padding: 1rem;
  }
}
</style>
