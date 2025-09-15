<template>
  <div class="historical-traffic">
    <div class="controls">
      <div class="time-range">
        <button 
          v-for="range in timeRanges" 
          :key="range.value"
          :class="['range-button', selectedRange === range.value && 'active']"
          @click="selectTimeRange(range.value)"
        >
          {{ range.label }}
        </button>
      </div>
      <div class="custom-range">
        <input type="datetime-local" v-model="customStart" class="datetime-input" />
        <span>至</span>
        <input type="datetime-local" v-model="customEnd" class="datetime-input" />
        <button @click="loadCustomRange" class="apply-button">应用</button>
      </div>
    </div>

    <div class="chart-container" ref="chartContainer"></div>

    <div class="statistics">
      <div class="stat-card">
        <h4>平均入站流量</h4>
        <div class="stat-value">{{ formatBps(stats.avgInBytes) }}</div>
      </div>
      <div class="stat-card">
        <h4>平均出站流量</h4>
        <div class="stat-value">{{ formatBps(stats.avgOutBytes) }}</div>
      </div>
      <div class="stat-card">
        <h4>峰值入站流量</h4>
        <div class="stat-value">{{ formatBps(stats.maxInBytes) }}</div>
      </div>
      <div class="stat-card">
        <h4>峰值出站流量</h4>
        <div class="stat-value">{{ formatBps(stats.maxOutBytes) }}</div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import * as echarts from 'echarts'

const chartContainer = ref(null)
let chart = null

const timeRanges = [
  { label: '最近1小时', value: '1h' },
  { label: '最近6小时', value: '6h' },
  { label: '最近24小时', value: '24h' },
  { label: '自定义', value: 'custom' }
]

const selectedRange = ref('1h')
const customStart = ref('')
const customEnd = ref('')

const stats = ref({
  avgInBytes: 0,
  avgOutBytes: 0,
  maxInBytes: 0,
  maxOutBytes: 0
})

const historicalData = ref([])

function selectTimeRange(range) {
  selectedRange.value = range
  if (range !== 'custom') {
    loadHistoricalData(range)
  }
}

async function loadHistoricalData(range) {
  let start, end
  const now = new Date()
  
  switch(range) {
    case '1h':
      start = new Date(now - 60 * 60 * 1000)
      break
    case '6h':
      start = new Date(now - 6 * 60 * 60 * 1000)
      break
    case '24h':
      start = new Date(now - 24 * 60 * 60 * 1000)
      break
  }
  
  end = now
  
  try {
    const url = `/api/traffic/history?start=${start.toISOString()}&end=${end.toISOString()}`
    console.log('Fetching historical data:', url)
    const response = await fetch(url)
    const data = await response.json()
    console.log('Historical data received:', data)
    historicalData.value = data
    updateChart(data)
    calculateStats(data)
  } catch (error) {
    console.error('Failed to load historical data:', error)
  }
}

function loadCustomRange() {
  if (customStart.value && customEnd.value) {
    const start = new Date(customStart.value)
    const end = new Date(customEnd.value)
    
    fetch(`/api/traffic/history?start=${start.toISOString()}&end=${end.toISOString()}`)
      .then(res => res.json())
      .then(data => {
        historicalData.value = data
        updateChart(data)
        calculateStats(data)
      })
  }
}

function updateChart(data) {
  if (!chart) {
    console.error('Chart not initialized')
    return
  }
  
  if (!data || data.length === 0) {
    console.log('No data to display')
    // 显示空图表
    chart.setOption({
      title: {
        text: '暂无数据',
        left: 'center',
        top: 'center',
        textStyle: {
          color: '#999'
        }
      },
      xAxis: { data: [] },
      series: [
        { name: '入站流量', data: [] },
        { name: '出站流量', data: [] }
      ]
    })
    return
  }
  
  const timestamps = data.map(d => new Date(d.timestamp).toLocaleTimeString())
  const inBytes = data.map(d => d.in_bytes * 8) // Convert to bps
  const outBytes = data.map(d => d.out_bytes * 8)
  
  console.log('Updating chart with', data.length, 'data points')
  
  chart.setOption({
    title: {
      text: ''  // 清除标题
    },
    tooltip: {
      trigger: 'axis',
      formatter: function(params) {
        if (!params || params.length === 0) return ''
        let result = params[0].axisValue + '<br/>'
        params.forEach(param => {
          result += param.marker + param.seriesName + ': ' + formatBps(param.value) + '<br/>'
        })
        return result
      }
    },
    legend: {
      data: ['入站流量', '出站流量']
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: timestamps,
      boundaryGap: false
    },
    yAxis: {
      type: 'value',
      axisLabel: {
        formatter: function(value) {
          return formatBps(value)
        }
      }
    },
    series: [
      {
        name: '入站流量',
        type: 'line',
        data: inBytes,
        smooth: true,
        areaStyle: {
          opacity: 0.3
        },
        itemStyle: {
          color: '#3498db'
        }
      },
      {
        name: '出站流量',
        type: 'line',
        data: outBytes,
        smooth: true,
        areaStyle: {
          opacity: 0.3
        },
        itemStyle: {
          color: '#2ecc71'
        }
      }
    ]
  })
}

function calculateStats(data) {
  if (data.length === 0) return
  
  let totalIn = 0, totalOut = 0
  let maxIn = 0, maxOut = 0
  
  data.forEach(d => {
    const inBps = d.in_bytes * 8
    const outBps = d.out_bytes * 8
    
    totalIn += inBps
    totalOut += outBps
    
    if (inBps > maxIn) maxIn = inBps
    if (outBps > maxOut) maxOut = outBps
  })
  
  stats.value = {
    avgInBytes: totalIn / data.length,
    avgOutBytes: totalOut / data.length,
    maxInBytes: maxIn,
    maxOutBytes: maxOut
  }
}

function formatBps(bytesPerSec) {
  if (!bytesPerSec) return '0 bps'
  const bps = bytesPerSec
  if (bps < 1024) return `${bps.toFixed(0)} bps`
  if (bps < 1024 * 1024) return `${(bps / 1024).toFixed(2)} Kbps`
  if (bps < 1024 * 1024 * 1024) return `${(bps / (1024 * 1024)).toFixed(2)} Mbps`
  return `${(bps / (1024 * 1024 * 1024)).toFixed(2)} Gbps`
}

onMounted(() => {
  chart = echarts.init(chartContainer.value)
  window.addEventListener('resize', () => {
    chart.resize()
  })
  
  // Load initial data
  loadHistoricalData('1h')
})

onUnmounted(() => {
  if (chart) {
    chart.dispose()
  }
})
</script>

<style scoped>
.historical-traffic {
  max-width: 1200px;
  margin: 0 auto;
}

.controls {
  background: white;
  padding: 1.5rem;
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  margin-bottom: 2rem;
}

.time-range {
  display: flex;
  gap: 1rem;
  margin-bottom: 1rem;
}

.range-button {
  padding: 0.5rem 1rem;
  border: 1px solid #ddd;
  background: white;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.3s ease;
}

.range-button:hover {
  border-color: #3498db;
}

.range-button.active {
  background: #3498db;
  color: white;
  border-color: #3498db;
}

.custom-range {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.datetime-input {
  padding: 0.5rem;
  border: 1px solid #ddd;
  border-radius: 4px;
}

.apply-button {
  padding: 0.5rem 1rem;
  background: #3498db;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

.apply-button:hover {
  background: #2980b9;
}

.chart-container {
  background: white;
  padding: 2rem;
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  height: 400px;
  margin-bottom: 2rem;
}

.statistics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 1.5rem;
}

.stat-card {
  background: white;
  padding: 1.5rem;
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.stat-card h4 {
  color: #666;
  font-size: 0.9rem;
  margin-bottom: 0.5rem;
}

.stat-value {
  font-size: 1.5rem;
  font-weight: bold;
  color: #2c3e50;
}
</style>