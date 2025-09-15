package storage

import (
	"sync"
	"time"

	"github.com/raojinlin/traffic-sniff/internal/models"
)

type MemoryStorage struct {
	mu          sync.RWMutex
	connections map[string]*models.Connection
	interfaces  map[string]*models.InterfaceStats
	snapshots   []models.TrafficSnapshot
	maxSnapshots int
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		connections:  make(map[string]*models.Connection),
		interfaces:   make(map[string]*models.InterfaceStats),
		snapshots:    make([]models.TrafficSnapshot, 0),
		maxSnapshots: 3600, // Keep 1 hour of snapshots
	}
}

func (m *MemoryStorage) UpdateConnection(conn *models.Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := connectionKey(conn)
	if existing, ok := m.connections[key]; ok {
		// Average the bytes per second
		existing.BytesPerSec = (existing.BytesPerSec + conn.BytesPerSec) / 2
		existing.Bytes = conn.Bytes
		existing.Packets = conn.Packets
		existing.LastSeen = conn.LastSeen
	} else {
		m.connections[key] = conn
	}
}

func (m *MemoryStorage) UpdateInterface(stats *models.InterfaceStats) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.interfaces[stats.Interface] = stats
}

func (m *MemoryStorage) GetSnapshot() *models.TrafficSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a snapshot of current state
	snapshot := &models.TrafficSnapshot{
		Timestamp:   time.Now(),
		Connections: make([]*models.Connection, 0, len(m.connections)),
	}

	// Copy interface stats
	for _, iface := range m.interfaces {
		snapshot.Interface = &models.InterfaceStats{
			Interface:        iface.Interface,
			InBytes:          iface.InBytes,
			OutBytes:         iface.OutBytes,
			InBytesPerSec:    iface.InBytesPerSec,
			OutBytesPerSec:   iface.OutBytesPerSec,
			InPackets:        iface.InPackets,
			OutPackets:       iface.OutPackets,
			InPacketsPerSec:  iface.InPacketsPerSec,
			OutPacketsPerSec: iface.OutPacketsPerSec,
		}
		break // Only one interface for now
	}

	// Copy active connections
	cutoff := time.Now().Add(-5 * time.Second)
	for _, conn := range m.connections {
		if conn.LastSeen.After(cutoff) {
			connCopy := *conn
			snapshot.Connections = append(snapshot.Connections, &connCopy)
		}
	}

	return snapshot
}

func (m *MemoryStorage) GetFilteredConnections(filter *models.Filter) []*models.Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*models.Connection, 0)
	cutoff := time.Now().Add(-5 * time.Second)

	for _, conn := range m.connections {
		if conn.LastSeen.Before(cutoff) {
			continue
		}

		// Apply filters
		if filter.IP != "" && conn.SrcIP != filter.IP && conn.DstIP != filter.IP {
			continue
		}
		if filter.Port != 0 && conn.SrcPort != filter.Port && conn.DstPort != filter.Port {
			continue
		}
		if filter.Protocol != "" && conn.Protocol != filter.Protocol {
			continue
		}

		connCopy := *conn
		result = append(result, &connCopy)
	}

	return result
}

func (m *MemoryStorage) AddSnapshot(snapshot models.TrafficSnapshot) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.snapshots = append(m.snapshots, snapshot)
	
	// Remove old snapshots if we exceed the limit
	if len(m.snapshots) > m.maxSnapshots {
		m.snapshots = m.snapshots[len(m.snapshots)-m.maxSnapshots:]
	}
}

func (m *MemoryStorage) GetHistoricalData(start, end time.Time, interval time.Duration) []models.HistoricalData {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Group snapshots by interval
	result := make([]models.HistoricalData, 0)
	
	// This is a simplified implementation
	// In production, we'd aggregate data properly
	for _, snapshot := range m.snapshots {
		if snapshot.Timestamp.Before(start) || snapshot.Timestamp.After(end) {
			continue
		}
		
		if snapshot.Interface != nil {
			result = append(result, models.HistoricalData{
				Timestamp:  snapshot.Timestamp,
				InBytes:    snapshot.Interface.InBytesPerSec,
				OutBytes:   snapshot.Interface.OutBytesPerSec,
				InPackets:  snapshot.Interface.InPacketsPerSec,
				OutPackets: snapshot.Interface.OutPacketsPerSec,
			})
		}
	}

	return result
}

func connectionKey(conn *models.Connection) string {
	return conn.SrcIP + ":" + string(conn.SrcPort) + "-" + 
		   conn.DstIP + ":" + string(conn.DstPort) + "-" + conn.Protocol
}

// ClearConnections clears all connection data (used when switching interfaces)
func (m *MemoryStorage) ClearConnections() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.connections = make(map[string]*models.Connection)
	// Also clear interface stats for the old interface
	m.interfaces = make(map[string]*models.InterfaceStats)
}