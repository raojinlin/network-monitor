package models

import (
	"time"
)

// Connection represents a network connection
type Connection struct {
	SrcIP       string    `json:"src_ip"`
	SrcPort     uint16    `json:"src_port"`
	DstIP       string    `json:"dst_ip"`
	DstPort     uint16    `json:"dst_port"`
	Protocol    string    `json:"protocol"`
	Bytes       uint64    `json:"bytes"`
	BytesPerSec uint64    `json:"bytes_per_sec"`
	Packets     uint64    `json:"packets"`
	StartTime   time.Time `json:"start_time"`
	LastSeen    time.Time `json:"last_seen"`
}

// InterfaceStats represents network interface statistics
type InterfaceStats struct {
	Interface        string `json:"interface"`
	InBytes          uint64 `json:"in_bytes"`
	OutBytes         uint64 `json:"out_bytes"`
	InBytesPerSec    uint64 `json:"in_bytes_per_sec"`
	OutBytesPerSec   uint64 `json:"out_bytes_per_sec"`
	InPackets        uint64 `json:"in_packets"`
	OutPackets       uint64 `json:"out_packets"`
	InPacketsPerSec  uint64 `json:"in_packets_per_sec"`
	OutPacketsPerSec uint64 `json:"out_packets_per_sec"`
}

// TrafficSnapshot represents traffic data at a point in time
type TrafficSnapshot struct {
	Timestamp   time.Time       `json:"timestamp"`
	Interface   *InterfaceStats `json:"interface"`
	Connections []*Connection   `json:"connections"`
}

// HistoricalData represents aggregated historical traffic data
type HistoricalData struct {
	Timestamp  time.Time `json:"timestamp"`
	InBytes    uint64    `json:"in_bytes"`
	OutBytes   uint64    `json:"out_bytes"`
	InPackets  uint64    `json:"in_packets"`
	OutPackets uint64    `json:"out_packets"`
}

// Filter represents traffic filter criteria
type Filter struct {
	IP       string `json:"ip,omitempty"`
	Port     uint16 `json:"port,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}