package capture

import (
	"context"
	"fmt"
	"sync"
	"time"
	
	"github.com/raojinlin/traffic-sniff/internal/storage"
)

// Manager handles dynamic interface switching
type Manager struct {
	mu              sync.RWMutex
	currentCapture  *PacketCapture
	currentIface    string
	storage         Storage
	ctx             context.Context
	cancel          context.CancelFunc
	captureRunning  bool
}

// NewManager creates a new capture manager
func NewManager(storage Storage) *Manager {
	return &Manager{
		storage: storage,
	}
}

// Start begins capturing on the specified interface
func (m *Manager) Start(iface string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop current capture if running
	if m.captureRunning {
		if err := m.stopCapture(); err != nil {
			return fmt.Errorf("failed to stop current capture: %w", err)
		}
	}

	// Clear old connection data when switching interfaces
	if ms, ok := m.storage.(*storage.MemoryStorage); ok {
		fmt.Printf("[Capture] Clearing connection data for interface switch\n")
		ms.ClearConnections()
	}

	// Create new capture
	fmt.Printf("[Capture] Creating packet capture for interface '%s'\n", iface)
	capturer, err := NewPacketCapture(iface)
	if err != nil {
		return fmt.Errorf("failed to create packet capture: %w", err)
	}

	// Create new context for this capture session
	ctx, cancel := context.WithCancel(context.Background())

	m.currentCapture = capturer
	m.currentIface = iface
	m.ctx = ctx
	m.cancel = cancel
	m.captureRunning = true

	// Start capture in background
	go func() {
		fmt.Printf("[Capture] Starting packet capture on interface '%s'\n", iface)
		if err := capturer.Start(ctx, m.storage); err != nil {
			// Log error but don't crash
			fmt.Printf("[Capture] Capture error on interface '%s': %v\n", iface, err)
		}
		
		m.mu.Lock()
		m.captureRunning = false
		fmt.Printf("[Capture] Packet capture stopped on interface '%s'\n", iface)
		m.mu.Unlock()
	}()

	return nil
}

// SwitchInterface switches to a new interface
func (m *Manager) SwitchInterface(newIface string) error {
	return m.Start(newIface)
}

// GetCurrentInterface returns the currently monitored interface
func (m *Manager) GetCurrentInterface() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentIface
}

// Stop stops all capture
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	return m.stopCapture()
}

// stopCapture stops the current capture (must be called with lock held)
func (m *Manager) stopCapture() error {
	if !m.captureRunning {
		return nil
	}

	// Cancel context to stop capture goroutine
	if m.cancel != nil {
		m.cancel()
	}

	// Close the capture handle
	if m.currentCapture != nil {
		m.currentCapture.Close()
		m.currentCapture = nil
	}

	// Give the goroutine time to exit cleanly
	time.Sleep(100 * time.Millisecond)

	m.captureRunning = false
	return nil
}