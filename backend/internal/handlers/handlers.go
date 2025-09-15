package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/gopacket/pcap"
	"github.com/gorilla/websocket"
	"github.com/raojinlin/traffic-sniff/internal/models"
	"github.com/raojinlin/traffic-sniff/internal/storage"
)

type Handler struct {
	storage         *storage.MemoryStorage
	historicalStore *storage.FileStorage
	captureManager  CaptureManager
	upgrader        websocket.Upgrader
}

type CaptureManager interface {
	SwitchInterface(iface string) error
	GetCurrentInterface() string
}

func NewHandler(storage *storage.MemoryStorage, historicalStore *storage.FileStorage, captureManager CaptureManager) *Handler {
	return &Handler{
		storage:         storage,
		historicalStore: historicalStore,
		captureManager:  captureManager,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
		},
	}
}

// RealtimeTraffic returns current interface statistics
func (h *Handler) RealtimeTraffic(w http.ResponseWriter, r *http.Request) {
	snapshot := h.storage.GetSnapshot()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshot.Interface)
}

// ConnectionList returns filtered list of active connections
func (h *Handler) ConnectionList(w http.ResponseWriter, r *http.Request) {
	// Parse filters from query params
	filter := &models.Filter{
		IP:       r.URL.Query().Get("ip"),
		Protocol: r.URL.Query().Get("protocol"),
	}
	
	if portStr := r.URL.Query().Get("port"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			filter.Port = uint16(port)
		}
	}

	connections := h.storage.GetFilteredConnections(filter)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(connections)
}

// HistoricalTraffic returns historical traffic data
func (h *Handler) HistoricalTraffic(w http.ResponseWriter, r *http.Request) {
	// Parse time range
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	
	var start, end time.Time
	var err error

	if startStr != "" {
		start, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			http.Error(w, "Invalid start time", http.StatusBadRequest)
			return
		}
		// Convert to local time
		start = start.Local()
		fmt.Printf("Start time: UTC=%s, Local=%s\n", startStr, start.Format(time.RFC3339))
	} else {
		start = time.Now().Add(-time.Hour)
	}

	if endStr != "" {
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			http.Error(w, "Invalid end time", http.StatusBadRequest)
			return
		}
		// Convert to local time
		end = end.Local()
		fmt.Printf("End time: UTC=%s, Local=%s\n", endStr, end.Format(time.RFC3339))
	} else {
		end = time.Now()
	}

	// Get historical data
	data, err := h.historicalStore.GetHistoricalData(start, end)
	if err != nil {
		http.Error(w, "Failed to retrieve historical data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// ListInterfaces returns available network interfaces
func (h *Handler) ListInterfaces(w http.ResponseWriter, r *http.Request) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		http.Error(w, "Failed to list interfaces", http.StatusInternalServerError)
		return
	}
	
	type InterfaceInfo struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Addresses   []string `json:"addresses"`
	}
	
	interfaces := make([]InterfaceInfo, 0)
	for _, dev := range devices {
		if dev.Name == "lo" || dev.Name == "lo0" {
			continue // Skip loopback
		}
		
		addresses := make([]string, 0)
		for _, addr := range dev.Addresses {
			addresses = append(addresses, addr.IP.String())
		}
		
		interfaces = append(interfaces, InterfaceInfo{
			Name:        dev.Name,
			Description: dev.Description,
			Addresses:   addresses,
		})
	}
	
	// Add current interface info
	type Response struct {
		Interfaces []InterfaceInfo `json:"interfaces"`
		Current    string          `json:"current"`
	}
	
	response := Response{
		Interfaces: interfaces,
		Current:    h.captureManager.GetCurrentInterface(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SwitchInterface handles interface switching
func (h *Handler) SwitchInterface(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIP(r)
	
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Interface string `json:"interface"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("[Interface] Client %s sent invalid request body: %v\n", clientIP, err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if req.Interface == "" {
		fmt.Printf("[Interface] Client %s sent empty interface name\n", clientIP)
		http.Error(w, "Interface name is required", http.StatusBadRequest)
		return
	}
	
	currentInterface := h.captureManager.GetCurrentInterface()
	fmt.Printf("[Interface] Client %s requesting switch from '%s' to '%s'\n", clientIP, currentInterface, req.Interface)
	
	// Switch to the new interface
	if err := h.captureManager.SwitchInterface(req.Interface); err != nil {
		fmt.Printf("[Interface] Failed to switch interface for client %s: %v\n", clientIP, err)
		http.Error(w, fmt.Sprintf("Failed to switch interface: %v", err), http.StatusInternalServerError)
		return
	}
	
	fmt.Printf("[Interface] Successfully switched to interface '%s' for client %s\n", req.Interface, clientIP)
	
	// Return success with the new interface
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"interface": req.Interface,
	})
}

// WebSocketHandler handles WebSocket connections for real-time updates
func (h *Handler) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIP(r)
	fmt.Printf("[WebSocket] Client %s attempting to connect\n", clientIP)
	
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("[WebSocket] Failed to upgrade connection from %s: %v\n", clientIP, err)
		return
	}
	defer func() {
		conn.Close()
		fmt.Printf("[WebSocket] Client %s disconnected\n", clientIP)
	}()

	fmt.Printf("[WebSocket] Client %s connected successfully\n", clientIP)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	
	// Save historical data every 30 seconds instead of every second
	historicalTicker := time.NewTicker(30 * time.Second)
	defer historicalTicker.Stop()
	
	// Save initial snapshot immediately
	initialSnapshot := h.storage.GetSnapshot()
	go h.historicalStore.SaveSnapshot(initialSnapshot)

	for {
		select {
		case <-ticker.C:
			snapshot := h.storage.GetSnapshot()
			
			// Send snapshot to client
			if err := conn.WriteJSON(snapshot); err != nil {
				fmt.Printf("[WebSocket] Error sending data to client %s: %v\n", clientIP, err)
				return
			}
		case <-historicalTicker.C:
			// Save to historical storage periodically
			snapshot := h.storage.GetSnapshot()
			go h.historicalStore.SaveSnapshot(snapshot)
		}
	}
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		return xff
	}
	
	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}
	
	// Fall back to RemoteAddr
	return r.RemoteAddr
}