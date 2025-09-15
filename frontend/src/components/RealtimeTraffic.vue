<template>
  <div class="realtime-traffic">
    <div class="traffic-overview">
      <div class="traffic-card incoming">
        <div class="card-header">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 2v20M17 7l-5-5-5 5"/>
          </svg>
          <h3>入站流量</h3>
        </div>
        <div class="traffic-value">{{ formatBps(interfaceStats.in_bytes_per_sec) }}</div>
        <div class="traffic-packets">
          <span class="packets-value">{{ interfaceStats.in_packets_per_sec }}</span>
          <span class="packets-unit">pps</span>
        </div>
        <div class="traffic-total">
          总计: {{ formatBytes(interfaceStats.in_bytes) }}
        </div>
      </div>
      <div class="traffic-card outgoing">
        <div class="card-header">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 2v20M7 17l5 5 5-5"/>
          </svg>
          <h3>出站流量</h3>
        </div>
        <div class="traffic-value">{{ formatBps(interfaceStats.out_bytes_per_sec) }}</div>
        <div class="traffic-packets">
          <span class="packets-value">{{ interfaceStats.out_packets_per_sec }}</span>
          <span class="packets-unit">pps</span>
        </div>
        <div class="traffic-total">
          总计: {{ formatBytes(interfaceStats.out_bytes) }}
        </div>
      </div>
    </div>

    <div class="filter-section">
      <div class="filter-group">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M22 3H2l8 9.46V19l4 2v-8.54L22 3z"/>
        </svg>
        <span class="filter-label">过滤条件:</span>
      </div>
      <div class="filter-controls">
        <input 
          v-model="filters.ip" 
          placeholder="IP地址"
          class="filter-input"
          @input="applyFilters"
        />
        <input 
          v-model="filters.port" 
          type="number"
          placeholder="端口"
          class="filter-input"
          @input="applyFilters"
        />
        <select v-model="filters.protocol" class="filter-select" @change="applyFilters">
          <option value="">所有协议</option>
          <option value="TCP">TCP</option>
          <option value="UDP">UDP</option>
        </select>
        <button @click="clearFilters" class="clear-button" v-if="hasActiveFilters">
          清除
        </button>
      </div>
    </div>

    <div class="connections-section">
      <div class="section-header">
        <div class="header-left">
          <h3>活动连接</h3>
          <span class="connection-count">{{ filteredConnections.length }} 个连接</span>
        </div>
        <div class="view-toggle">
          <button 
            :class="['toggle-btn', viewMode === 'table' && 'active']"
            @click="viewMode = 'table'"
            title="表格视图"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
              <line x1="3" y1="9" x2="21" y2="9"/>
              <line x1="3" y1="15" x2="21" y2="15"/>
              <line x1="9" y1="3" x2="9" y2="21"/>
            </svg>
          </button>
          <button 
            :class="['toggle-btn', viewMode === 'map' && 'active']"
            @click="viewMode = 'map'"
            title="地图视图"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/>
              <line x1="2" y1="12" x2="22" y2="12"/>
              <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
            </svg>
          </button>
        </div>
      </div>
      <div v-if="viewMode === 'table'" class="connections-table">
        <table>
          <thead>
            <tr>
              <th><span class="th-content">源地址</span></th>
              <th><span class="th-content">目标地址</span></th>
              <th><span class="th-content">协议</span></th>
              <th><span class="th-content">实时速率</span></th>
              <th><span class="th-content">总流量</span></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="conn in filteredConnections" :key="getConnectionKey(conn)" class="connection-row">
              <td>
                <div class="address-cell">
                  <span class="ip">{{ conn.src_ip }}</span>
                  <span class="port">:{{ conn.src_port }}</span>
                </div>
              </td>
              <td>
                <div class="address-cell">
                  <span class="ip">{{ conn.dst_ip }}</span>
                  <span class="port">:{{ conn.dst_port }}</span>
                </div>
              </td>
              <td>
                <span :class="['protocol-badge', conn.protocol.toLowerCase()]">
                  {{ conn.protocol }}
                </span>
              </td>
              <td>
                <div class="rate-cell">
                  <span class="rate-value">{{ formatBps(conn.bytes_per_sec) }}</span>
                  <div class="rate-bar">
                    <div class="rate-fill" :style="{ width: getRateWidth(conn.bytes_per_sec) + '%' }"></div>
                  </div>
                </div>
              </td>
              <td>{{ formatBytes(conn.bytes) }}</td>
            </tr>
            <tr v-if="filteredConnections.length === 0">
              <td colspan="5" class="no-data">
                <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
                  <circle cx="12" cy="12" r="10"/>
                  <path d="M8 15s1.5-2 4-2 4 2 4 2"/>
                  <line x1="9" y1="9" x2="9.01" y2="9"/>
                  <line x1="15" y1="9" x2="15.01" y2="9"/>
                </svg>
                <p>暂无连接数据</p>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-else-if="viewMode === 'map'" class="map-view-container">
        <TrafficMap :connections="filteredConnections" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import TrafficMap from './TrafficMap.vue'

const props = defineProps({
  connected: Boolean
})

const interfaceStats = ref({
  in_bytes: 0,
  out_bytes: 0,
  in_bytes_per_sec: 0,
  out_bytes_per_sec: 0,
  in_packets_per_sec: 0,
  out_packets_per_sec: 0
})

const connections = ref([])
const filters = ref({
  ip: '',
  port: '',
  protocol: ''
})

const maxBytesPerSec = ref(1000000) // 1 Mbps for scale
const viewMode = ref('table') // 'table' or 'map'

const hasActiveFilters = computed(() => {
  return filters.value.ip || filters.value.port || filters.value.protocol
})

const filteredConnections = computed(() => {
  return connections.value.filter(conn => {
    if (filters.value.ip && !conn.src_ip.includes(filters.value.ip) && !conn.dst_ip.includes(filters.value.ip)) {
      return false
    }
    if (filters.value.port && conn.src_port != filters.value.port && conn.dst_port != filters.value.port) {
      return false
    }
    if (filters.value.protocol && conn.protocol !== filters.value.protocol) {
      return false
    }
    return true
  })
})

function handleTrafficUpdate(event) {
  const data = event.detail
  if (data.interface) {
    interfaceStats.value = data.interface
  }
  if (data.connections) {
    connections.value = data.connections
  }
}

function applyFilters() {
  // Filters are reactive, no need to manually update
}

function clearFilters() {
  filters.value.ip = ''
  filters.value.port = ''
  filters.value.protocol = ''
}

function getRateWidth(bytesPerSec) {
  if (!bytesPerSec) return 0
  const percent = (bytesPerSec / maxBytesPerSec.value) * 100
  // Update max if needed
  if (bytesPerSec > maxBytesPerSec.value) {
    maxBytesPerSec.value = bytesPerSec * 1.2
  }
  return Math.min(percent, 100)
}

function getConnectionKey(conn) {
  return `${conn.src_ip}:${conn.src_port}-${conn.dst_ip}:${conn.dst_port}-${conn.protocol}`
}

function formatBps(bytesPerSec) {
  if (!bytesPerSec) return '0 bps'
  const bps = bytesPerSec * 8
  if (bps < 1024) return `${bps} bps`
  if (bps < 1024 * 1024) return `${(bps / 1024).toFixed(2)} Kbps`
  if (bps < 1024 * 1024 * 1024) return `${(bps / (1024 * 1024)).toFixed(2)} Mbps`
  return `${(bps / (1024 * 1024 * 1024)).toFixed(2)} Gbps`
}

function formatBytes(bytes) {
  if (!bytes) return '0 B'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
}

onMounted(() => {
  window.addEventListener('traffic-update', handleTrafficUpdate)
})

onUnmounted(() => {
  window.removeEventListener('traffic-update', handleTrafficUpdate)
})
</script>

<style scoped>
.realtime-traffic {
  animation: fadeIn 0.3s ease-in;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

.traffic-overview {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 1.5rem;
  margin-bottom: 2rem;
}

.traffic-card {
  background: white;
  padding: 1.5rem;
  border-radius: 12px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
  border: 1px solid #e5e7eb;
  transition: all 0.3s ease;
}

.traffic-card:hover {
  box-shadow: 0 4px 6px -1px rgba(0,0,0,0.1), 0 2px 4px -1px rgba(0,0,0,0.06);
  transform: translateY(-2px);
}

.traffic-card.incoming {
  border-top: 3px solid #3b82f6;
}

.traffic-card.outgoing {
  border-top: 3px solid #10b981;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.card-header svg {
  color: #6b7280;
}

.traffic-card.incoming .card-header svg {
  color: #3b82f6;
}

.traffic-card.outgoing .card-header svg {
  color: #10b981;
}

.traffic-card h3 {
  color: #374151;
  font-size: 0.875rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.traffic-value {
  font-size: 2.25rem;
  font-weight: 700;
  color: #1f2937;
  margin-bottom: 0.5rem;
  line-height: 1;
}

.traffic-packets {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.25rem;
  color: #6b7280;
  font-size: 0.875rem;
  margin-bottom: 0.75rem;
}

.packets-value {
  font-weight: 600;
  color: #4b5563;
}

.traffic-total {
  font-size: 0.75rem;
  color: #9ca3af;
  padding-top: 0.75rem;
  border-top: 1px solid #f3f4f6;
}

.filter-section {
  background: white;
  padding: 1rem 1.5rem;
  border-radius: 12px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
  border: 1px solid #e5e7eb;
  margin-bottom: 2rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1.5rem;
  flex-wrap: wrap;
}

.filter-group {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: #6b7280;
}

.filter-label {
  font-size: 0.875rem;
  font-weight: 500;
}

.filter-controls {
  display: flex;
  gap: 0.75rem;
  flex: 1;
  align-items: center;
  flex-wrap: wrap;
}

.filter-input,
.filter-select {
  min-width: 140px;
  padding: 0.5rem 0.75rem;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  font-size: 0.875rem;
  background-color: #f9fafb;
  transition: all 0.2s ease;
}

.filter-input:focus,
.filter-select:focus {
  outline: none;
  border-color: #7c3aed;
  background-color: white;
  box-shadow: 0 0 0 3px rgba(124, 58, 237, 0.1);
}

.clear-button {
  padding: 0.5rem 1rem;
  background-color: #ef4444;
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.clear-button:hover {
  background-color: #dc2626;
}

.connections-section {
  background: white;
  border-radius: 12px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
  border: 1px solid #e5e7eb;
  overflow: hidden;
}

.section-header {
  padding: 1rem 1.5rem;
  border-bottom: 1px solid #e5e7eb;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.view-toggle {
  display: flex;
  gap: 0.25rem;
  background-color: #f3f4f6;
  padding: 0.25rem;
  border-radius: 8px;
}

.toggle-btn {
  padding: 0.5rem;
  background: transparent;
  border: none;
  border-radius: 6px;
  color: #6b7280;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;
}

.toggle-btn:hover {
  color: #4b5563;
  background-color: #e5e7eb;
}

.toggle-btn.active {
  background-color: white;
  color: #7c3aed;
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
}

.section-header h3 {
  font-size: 1rem;
  font-weight: 600;
  color: #1f2937;
}

.connection-count {
  font-size: 0.875rem;
  color: #6b7280;
  background-color: #f3f4f6;
  padding: 0.25rem 0.75rem;
  border-radius: 9999px;
}

.connections-table {
  overflow-x: auto;
}

table {
  width: 100%;
  border-collapse: collapse;
}

thead {
  background-color: #f9fafb;
}

th {
  padding: 0.75rem 1.5rem;
  text-align: left;
  font-size: 0.75rem;
  font-weight: 600;
  color: #6b7280;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  border-bottom: 1px solid #e5e7eb;
}

.th-content {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

td {
  padding: 0.75rem 1.5rem;
  border-bottom: 1px solid #f3f4f6;
  font-size: 0.875rem;
  color: #374151;
}

.connection-row {
  transition: background-color 0.2s ease;
}

.connection-row:hover {
  background-color: #f9fafb;
}

.address-cell {
  display: flex;
  align-items: baseline;
  gap: 0;
}

.address-cell .ip {
  font-weight: 500;
  color: #1f2937;
}

.address-cell .port {
  color: #9ca3af;
  font-size: 0.8rem;
}

.protocol-badge {
  display: inline-block;
  padding: 0.25rem 0.625rem;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.protocol-badge.tcp {
  background-color: #dbeafe;
  color: #1e40af;
}

.protocol-badge.udp {
  background-color: #d1fae5;
  color: #065f46;
}

.rate-cell {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.rate-value {
  font-weight: 500;
  color: #1f2937;
}

.rate-bar {
  height: 4px;
  background-color: #f3f4f6;
  border-radius: 2px;
  overflow: hidden;
  width: 100px;
}

.rate-fill {
  height: 100%;
  background: linear-gradient(to right, #3b82f6, #7c3aed);
  border-radius: 2px;
  transition: width 0.3s ease;
}

.no-data {
  text-align: center;
  color: #9ca3af;
  padding: 3rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
}

.no-data svg {
  color: #e5e7eb;
}

.no-data p {
  font-size: 0.875rem;
}

/* Responsive */
@media (max-width: 768px) {
  .filter-section {
    flex-direction: column;
    align-items: stretch;
  }
  
  .filter-controls {
    justify-content: stretch;
  }
  
  .filter-input,
  .filter-select {
    flex: 1;
  }
  
  .connections-table {
    font-size: 0.8rem;
  }
  
  th, td {
    padding: 0.5rem;
  }
  
  .rate-bar {
    width: 60px;
  }
}

.map-view-container {
  min-height: 600px;
}
</style>