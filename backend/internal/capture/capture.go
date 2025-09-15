package capture

import (
	"context"
	"fmt"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/raojinlin/traffic-sniff/internal/models"
)

type Storage interface {
	UpdateConnection(conn *models.Connection)
	UpdateInterface(stats *models.InterfaceStats)
}

type PacketCapture struct {
	handle *pcap.Handle
	iface  string
}

func NewPacketCapture(iface string) (*PacketCapture, error) {
	// If no interface specified, get the first active one
	if iface == "" {
		devices, err := pcap.FindAllDevs()
		if err != nil {
			return nil, fmt.Errorf("failed to find devices: %w", err)
		}
		if len(devices) == 0 {
			return nil, fmt.Errorf("no network devices found")
		}
		// Find first non-loopback interface
		for _, dev := range devices {
			if len(dev.Addresses) > 0 && dev.Name != "lo" && dev.Name != "lo0" {
				iface = dev.Name
				break
			}
		}
	}

	handle, err := pcap.OpenLive(iface, 65536, true, pcap.BlockForever)
	if err != nil {
		return nil, fmt.Errorf("failed to open device %s: %w", iface, err)
	}

	return &PacketCapture{
		handle: handle,
		iface:  iface,
	}, nil
}

func (pc *PacketCapture) Start(ctx context.Context, storage Storage) error {
	packetSource := gopacket.NewPacketSource(pc.handle, pc.handle.LinkType())
	
	stats := &models.InterfaceStats{
		Interface: pc.iface,
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	connections := make(map[string]*models.Connection)

	for {
		select {
		case <-ctx.Done():
			return nil
		case packet := <-packetSource.Packets():
			pc.processPacket(packet, connections, stats)
		case <-ticker.C:
			// Update storage with current stats
			storage.UpdateInterface(stats)
			for _, conn := range connections {
				storage.UpdateConnection(conn)
			}
			// Reset per-second counters
			stats.InBytesPerSec = 0
			stats.OutBytesPerSec = 0
			stats.InPacketsPerSec = 0
			stats.OutPacketsPerSec = 0
		}
	}
}

func (pc *PacketCapture) processPacket(packet gopacket.Packet, connections map[string]*models.Connection, stats *models.InterfaceStats) {
	// Extract network layer
	networkLayer := packet.NetworkLayer()
	if networkLayer == nil {
		return
	}

	var srcIP, dstIP string
	var isIncoming bool
	packetLen := len(packet.Data())

	// Determine if packet is incoming or outgoing based on local interface IPs
	if ipLayer, ok := networkLayer.(*layers.IPv4); ok {
		srcIP = ipLayer.SrcIP.String()
		dstIP = ipLayer.DstIP.String()
		// Simple heuristic: if destination is local, it's incoming
		// In production, we'd check against actual interface IPs
		isIncoming = isLocalIP(dstIP)
	} else if ipLayer, ok := networkLayer.(*layers.IPv6); ok {
		srcIP = ipLayer.SrcIP.String()
		dstIP = ipLayer.DstIP.String()
		isIncoming = isLocalIP(dstIP)
	} else {
		return
	}

	// Update interface stats
	if isIncoming {
		stats.InBytes += uint64(packetLen)
		stats.InBytesPerSec += uint64(packetLen)
		stats.InPackets++
		stats.InPacketsPerSec++
	} else {
		stats.OutBytes += uint64(packetLen)
		stats.OutBytesPerSec += uint64(packetLen)
		stats.OutPackets++
		stats.OutPacketsPerSec++
	}

	// Extract transport layer
	transportLayer := packet.TransportLayer()
	if transportLayer == nil {
		return
	}

	var srcPort, dstPort uint16
	var protocol string

	switch transport := transportLayer.(type) {
	case *layers.TCP:
		srcPort = uint16(transport.SrcPort)
		dstPort = uint16(transport.DstPort)
		protocol = "TCP"
	case *layers.UDP:
		srcPort = uint16(transport.SrcPort)
		dstPort = uint16(transport.DstPort)
		protocol = "UDP"
	default:
		return
	}

	// Create connection key
	connKey := fmt.Sprintf("%s:%d-%s:%d-%s", srcIP, srcPort, dstIP, dstPort, protocol)
	
	// Update connection stats
	conn, exists := connections[connKey]
	if !exists {
		conn = &models.Connection{
			SrcIP:     srcIP,
			SrcPort:   srcPort,
			DstIP:     dstIP,
			DstPort:   dstPort,
			Protocol:  protocol,
			StartTime: time.Now(),
		}
		connections[connKey] = conn
	}

	conn.Bytes += uint64(packetLen)
	conn.BytesPerSec = uint64(packetLen) // Will be averaged in storage
	conn.Packets++
	conn.LastSeen = time.Now()
}

func (pc *PacketCapture) Close() {
	if pc.handle != nil {
		pc.handle.Close()
	}
}

// Simple check for local IPs - in production, we'd check actual interface IPs
func isLocalIP(ip string) bool {
	// Check for common private IP ranges
	return len(ip) > 0 && (ip[0:3] == "192" || ip[0:3] == "172" || ip[0:2] == "10" || ip == "127.0.0.1" || ip == "::1")
}