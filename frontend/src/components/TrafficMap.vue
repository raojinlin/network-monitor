<template>
  <div class="traffic-map-container">
    <div ref="mapContainer" class="map-view"></div>
    <div class="map-controls">
      <div class="location-stats">
        <h3>流量来源统计</h3>
        <div v-if="loading" class="loading-indicator">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="spin">
            <path d="M21 12a9 9 0 11-6.219-8.56"/>
          </svg>
          加载中...
        </div>
        <div v-else-if="Object.keys(countryStats).length === 0" class="no-stats">
          暂无数据
        </div>
        <div v-else class="stats-list">
          <div v-for="(count, country) in countryStats" :key="country" class="stat-item">
            <span class="country">{{ country }}</span>
            <span class="count">{{ count }} 个连接</span>
          </div>
        </div>
        <div v-if="apiCallCount >= maxApiCalls" class="api-limit-warning">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <line x1="12" y1="8" x2="12" y2="12"/>
            <line x1="12" y1="16" x2="12.01" y2="16"/>
          </svg>
          已达到API调用限制
        </div>
      </div>
    </div>
    <div class="map-legend">
      <div class="legend-item">
        <div class="legend-color incoming"></div>
        <span>入站连接</span>
      </div>
      <div class="legend-item">
        <div class="legend-color outgoing"></div>
        <span>出站连接</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch, computed } from 'vue'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'

const props = defineProps({
  connections: {
    type: Array,
    default: () => []
  }
})

const mapContainer = ref(null)
let map = null
let markersLayer = null
// Load cache from localStorage
const loadCacheFromStorage = () => {
  try {
    const cached = localStorage.getItem('ipLocationCache')
    if (cached) {
      const parsed = JSON.parse(cached)
      // Only keep cache entries from the last 7 days
      const weekAgo = Date.now() - (7 * 24 * 60 * 60 * 1000)
      const filtered = {}
      for (const [ip, data] of Object.entries(parsed)) {
        if (data.timestamp && data.timestamp > weekAgo) {
          filtered[ip] = data
        }
      }
      return filtered
    }
  } catch (e) {
    console.error('Failed to load IP cache from localStorage:', e)
  }
  return {}
}

const ipLocationCache = ref(loadCacheFromStorage())
const countryStats = ref({})
const loading = ref(false)
const apiCallCount = ref(0)
const maxApiCalls = 10 // Limit API calls per session to avoid rate limiting
const lastApiCall = ref(0)
const minApiCallInterval = 1000 // Minimum 1 second between API calls

// Default icon fix for Leaflet
delete L.Icon.Default.prototype._getIconUrl
L.Icon.Default.mergeOptions({
  iconRetinaUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-icon-2x.png',
  iconUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-icon.png',
  shadowUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-shadow.png'
})

onMounted(() => {
  initMap()
  loadConnectionsOnMap()
})

onUnmounted(() => {
  if (map) {
    map.remove()
  }
})

watch(() => props.connections, () => {
  loadConnectionsOnMap()
}, { deep: true })

function initMap() {
  // Initialize map centered on China
  map = L.map(mapContainer.value).setView([35.8617, 104.1954], 4)
  
  // Add tile layer (using OpenStreetMap)
  L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    attribution: '© OpenStreetMap contributors',
    maxZoom: 19
  }).addTo(map)
  
  // Create layer for markers
  markersLayer = L.layerGroup().addTo(map)
}

async function loadConnectionsOnMap() {
  if (!map || !markersLayer) return
  
  loading.value = true
  
  // Clear existing markers
  markersLayer.clearLayers()
  countryStats.value = {}
  
  // Get unique IPs
  const uniqueIPs = new Set()
  props.connections.forEach(conn => {
    // Only show public IPs (not local/private)
    if (!isPrivateIP(conn.src_ip)) {
      uniqueIPs.add(conn.src_ip)
    }
    if (!isPrivateIP(conn.dst_ip)) {
      uniqueIPs.add(conn.dst_ip)
    }
  })
  
  // Process IPs that are already cached first
  const cachedIPs = []
  const uncachedIPs = []
  
  Array.from(uniqueIPs).forEach(ip => {
    if (ipLocationCache.value[ip]) {
      cachedIPs.push(ip)
    } else {
      uncachedIPs.push(ip)
    }
  })
  
  // Show cached locations immediately
  cachedIPs.forEach(ip => {
    const location = ipLocationCache.value[ip]
    addMarkerToMap(ip, location)
    // Update country stats
    countryStats.value[location.country] = (countryStats.value[location.country] || 0) + 1
  })
  
  // Limit the number of new API calls
  const remainingApiCalls = Math.max(0, maxApiCalls - apiCallCount.value)
  const ipsToProcess = uncachedIPs.slice(0, remainingApiCalls)
  
  // Show a message if we've reached the API limit
  if (uncachedIPs.length > ipsToProcess.length) {
    console.warn(`API limit reached. Showing ${cachedIPs.length + ipsToProcess.length} of ${uniqueIPs.size} unique IPs`)
  }
  
  // Get location for each uncached IP with proper rate limiting
  for (let i = 0; i < ipsToProcess.length; i++) {
    const ip = ipsToProcess[i]
    
    // Ensure minimum interval between API calls
    const now = Date.now()
    const timeSinceLastCall = now - lastApiCall.value
    if (timeSinceLastCall < minApiCallInterval) {
      await new Promise(resolve => setTimeout(resolve, minApiCallInterval - timeSinceLastCall))
    }
    
    const location = await getIPLocation(ip)
    if (location) {
      addMarkerToMap(ip, location)
    }
  }
  
  loading.value = false
}

async function getIPLocation(ip) {
  // Check cache first
  if (ipLocationCache.value[ip]) {
    return ipLocationCache.value[ip]
  }
  
  // Check if we've exceeded API limit
  if (apiCallCount.value >= maxApiCalls) {
    console.warn('API call limit reached, skipping IP:', ip)
    return null
  }
  
  try {
    lastApiCall.value = Date.now()
    apiCallCount.value++
    
    // Using ipapi.co free service (limited to 1000 requests per day)
    const response = await fetch(`https://ipapi.co/${ip}/json/`)
    
    // Check for rate limiting
    if (response.status === 429) {
      console.error('Rate limit exceeded for IP geolocation API')
      // Reduce max API calls to prevent further rate limiting
      maxApiCalls = Math.min(maxApiCalls, apiCallCount.value - 1)
      return null
    }
    
    if (!response.ok) {
      console.error(`Failed to get location for IP ${ip}: ${response.status}`)
      return null
    }
    
    const data = await response.json()
    
    if (data.latitude && data.longitude) {
      const location = {
        lat: data.latitude,
        lon: data.longitude,
        city: data.city || 'Unknown',
        country: data.country_name || 'Unknown',
        countryCode: data.country_code || 'XX',
        timestamp: Date.now()
      }
      ipLocationCache.value[ip] = location
      
      // Save to localStorage
      try {
        localStorage.setItem('ipLocationCache', JSON.stringify(ipLocationCache.value))
      } catch (e) {
        console.error('Failed to save IP cache to localStorage:', e)
      }
      
      // Update country stats
      countryStats.value[location.country] = (countryStats.value[location.country] || 0) + 1
      
      return location
    }
  } catch (error) {
    console.error('Failed to get location for IP:', ip, error)
  }
  
  return null
}

function addMarkerToMap(ip, location) {
  // Count connections for this IP
  const connectionCount = props.connections.filter(conn => 
    conn.src_ip === ip || conn.dst_ip === ip
  ).length
  
  // Determine if this is primarily incoming or outgoing
  const incomingCount = props.connections.filter(conn => 
    conn.dst_ip === ip && isPrivateIP(conn.src_ip)
  ).length
  
  const isIncoming = incomingCount > 0
  
  // Create custom icon based on traffic direction
  const iconHtml = `
    <div class="custom-marker ${isIncoming ? 'incoming' : 'outgoing'}">
      <div class="marker-inner">
        <span>${connectionCount}</span>
      </div>
      <div class="marker-pulse"></div>
    </div>
  `
  
  const customIcon = L.divIcon({
    html: iconHtml,
    className: 'leaflet-custom-icon',
    iconSize: [40, 40],
    iconAnchor: [20, 20]
  })
  
  // Create marker
  const marker = L.marker([location.lat, location.lon], { icon: customIcon })
    .bindPopup(`
      <div class="map-popup">
        <h4>${location.city || 'Unknown'}, ${location.country}</h4>
        <p>IP: ${ip}</p>
        <p>连接数: ${connectionCount}</p>
      </div>
    `)
  
  // Add to layer
  markersLayer.addLayer(marker)
  
  // Add connection lines to local server
  if (isIncoming) {
    // Assuming local server is in current location (you can set this manually)
    navigator.geolocation.getCurrentPosition((position) => {
      const polyline = L.polyline([
        [location.lat, location.lon],
        [position.coords.latitude, position.coords.longitude]
      ], {
        color: '#3b82f6',
        weight: 2,
        opacity: 0.5,
        dashArray: '5, 10'
      })
      markersLayer.addLayer(polyline)
    })
  }
}

function isPrivateIP(ip) {
  // Check if IP is private/local
  return ip.startsWith('192.168.') || 
         ip.startsWith('10.') || 
         ip.startsWith('172.') ||
         ip === '127.0.0.1' ||
         ip === '::1'
}
</script>

<style scoped>
.traffic-map-container {
  position: relative;
  height: 600px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
  border: 1px solid #e5e7eb;
  overflow: hidden;
}

.map-view {
  height: 100%;
  width: 100%;
}

.map-controls {
  position: absolute;
  top: 1rem;
  right: 1rem;
  z-index: 1000;
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.15);
  padding: 1rem;
  max-width: 250px;
}

.location-stats h3 {
  font-size: 0.875rem;
  font-weight: 600;
  color: #374151;
  margin-bottom: 0.75rem;
}

.stats-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.stat-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.75rem;
}

.country {
  font-weight: 500;
  color: #1f2937;
}

.count {
  color: #6b7280;
}

.map-legend {
  position: absolute;
  bottom: 1rem;
  left: 1rem;
  z-index: 1000;
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.15);
  padding: 0.75rem 1rem;
  display: flex;
  gap: 1.5rem;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.75rem;
  color: #374151;
}

.legend-color {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.legend-color.incoming {
  background-color: #3b82f6;
}

.legend-color.outgoing {
  background-color: #10b981;
}

/* Custom marker styles */
:global(.leaflet-custom-icon) {
  background: transparent !important;
  border: none !important;
}

:global(.custom-marker) {
  position: relative;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
}

:global(.marker-inner) {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: bold;
  font-size: 12px;
  z-index: 2;
  position: relative;
}

:global(.custom-marker.incoming .marker-inner) {
  background-color: #3b82f6;
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.5);
}

:global(.custom-marker.outgoing .marker-inner) {
  background-color: #10b981;
  box-shadow: 0 2px 8px rgba(16, 185, 129, 0.5);
}

:global(.marker-pulse) {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 40px;
  height: 40px;
  border-radius: 50%;
  animation: pulse-animation 2s ease-out infinite;
}

:global(.custom-marker.incoming .marker-pulse) {
  background-color: #3b82f6;
}

:global(.custom-marker.outgoing .marker-pulse) {
  background-color: #10b981;
}

@keyframes pulse-animation {
  0% {
    transform: translate(-50%, -50%) scale(1);
    opacity: 0.7;
  }
  100% {
    transform: translate(-50%, -50%) scale(2);
    opacity: 0;
  }
}

:global(.map-popup) {
  padding: 0.5rem;
}

:global(.map-popup h4) {
  margin: 0 0 0.5rem 0;
  font-size: 0.875rem;
  font-weight: 600;
  color: #1f2937;
}

:global(.map-popup p) {
  margin: 0.25rem 0;
  font-size: 0.75rem;
  color: #6b7280;
}

.loading-indicator {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: #6b7280;
  font-size: 0.875rem;
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.no-stats {
  color: #9ca3af;
  font-size: 0.875rem;
  text-align: center;
  padding: 1rem 0;
}

.api-limit-warning {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: #dc2626;
  font-size: 0.75rem;
  margin-top: 0.75rem;
  padding-top: 0.75rem;
  border-top: 1px solid #fecaca;
}

/* Responsive */
@media (max-width: 768px) {
  .map-controls {
    top: auto;
    bottom: 4rem;
    right: 0.5rem;
    left: 0.5rem;
    max-width: none;
  }
}
</style>