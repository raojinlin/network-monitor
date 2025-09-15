package storage

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/raojinlin/traffic-sniff/internal/models"
)

type FileStorage struct {
	basePath string
	mu       sync.Mutex
}

func NewFileStorage(basePath string) *FileStorage {
	// Get absolute path
	absPath, err := filepath.Abs(basePath)
	if err != nil {
		absPath = basePath
	}
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(absPath, 0755); err != nil {
		fmt.Printf("Failed to create data directory %s: %v\n", absPath, err)
	} else {
		fmt.Printf("Using data directory: %s\n", absPath)
	}
	
	return &FileStorage{
		basePath: absPath,
	}
}

func (f *FileStorage) SaveSnapshot(snapshot *models.TrafficSnapshot) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Create hourly files using local time
	localTime := snapshot.Timestamp.Local()
	fileName := fmt.Sprintf("traffic_%s.json", localTime.Format("2006-01-02_15"))
	filePath := filepath.Join(f.basePath, fileName)

	// Read existing data
	var snapshots []models.TrafficSnapshot
	if data, err := ioutil.ReadFile(filePath); err == nil {
		json.Unmarshal(data, &snapshots)
	}

	// Append new snapshot
	snapshots = append(snapshots, *snapshot)

	// Write back
	data, err := json.Marshal(snapshots)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filePath, data, 0644)
	if err == nil {
		fmt.Printf("Saved snapshot to %s (total: %d snapshots)\n", filePath, len(snapshots))
	}
	return err
}

func (f *FileStorage) GetHistoricalData(start, end time.Time) ([]models.HistoricalData, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	result := make([]models.HistoricalData, 0)
	fmt.Printf("Getting historical data from %s to %s\n", start.Format(time.RFC3339), end.Format(time.RFC3339))

	// Also check current hour and next hour to handle edge cases
	startHour := start.Truncate(time.Hour)
	endHour := end.Add(time.Hour).Truncate(time.Hour)
	
	// Iterate through hourly files using local time
	for t := startHour; !t.After(endHour); t = t.Add(time.Hour) {
		localT := t.Local()
		fileName := fmt.Sprintf("traffic_%s.json", localT.Format("2006-01-02_15"))
		filePath := filepath.Join(f.basePath, fileName)
		fmt.Printf("Checking file: %s\n", filePath)

		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			fmt.Printf("  File not found: %v\n", err)
			continue // Skip missing files
		}

		var snapshots []models.TrafficSnapshot
		if err := json.Unmarshal(data, &snapshots); err != nil {
			fmt.Printf("  Failed to unmarshal: %v\n", err)
			continue
		}
		
		fmt.Printf("  Found %d snapshots in file\n", len(snapshots))

		// Convert snapshots to historical data
		for _, snapshot := range snapshots {
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
	}

	fmt.Printf("Returning %d historical data points\n", len(result))
	return result, nil
}