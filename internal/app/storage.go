package app

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// LoadCalendar loads the calendar data from the file
func LoadCalendar() error {
	file, err := os.Open(CalendarFile)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Error closing calendar file: %v", err)
		}
	}()

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	var newCalendar CalendarData
	if err := json.Unmarshal(data, &newCalendar); err != nil {
		return err
	}

	CalendarMutex.Lock()
	Calendar = &newCalendar
	CalendarMutex.Unlock()

	return nil
}

// SaveCalendar saves the calendar data to the file with backup
func SaveCalendar() error {
	CalendarMutex.RLock()
	defer CalendarMutex.RUnlock()
	return saveCalendarLocked()
}

// saveCalendarLocked saves calendar without locking (caller must hold lock)
func saveCalendarLocked() error {
	data, err := json.MarshalIndent(Calendar, "", "  ")
	if err != nil {
		return err
	}

	// Create backup
	if _, err := os.Stat(CalendarFile); err == nil {
		backupFile := CalendarFile + BackupSuffix
		if err := os.Rename(CalendarFile, backupFile); err != nil {
			log.Printf("Warning: failed to create backup: %v", err)
		}
	}

	// Write to temp file first
	tmpFile := CalendarFile + TmpSuffix
	if err := os.WriteFile(tmpFile, data, FilePermissions); err != nil {
		return err
	}

	// Rename temp file to actual file
	return os.Rename(tmpFile, CalendarFile)
}

// saveTmpCalendar saves the current calendar to tmp file (auto-save for edit mode)
func saveTmpCalendar() error {
	data, err := json.MarshalIndent(Calendar, "", "  ")
	if err != nil {
		return err
	}

	tmpFile := CalendarFile + TmpSuffix
	return os.WriteFile(tmpFile, data, FilePermissions)
}

// LoadCalendarWithTmpCheck loads calendar from tmp if exists, otherwise from main file
func LoadCalendarWithTmpCheck() error {
	tmpFile := CalendarFile + TmpSuffix

	// Check if tmp file exists
	if _, err := os.Stat(tmpFile); err == nil {
		log.Printf("⚠️  Found temporary calendar file: %s (loading unsaved changes)", tmpFile)
		return loadCalendarFromFile(tmpFile)
	}

	// Load from main file
	return LoadCalendar()
}

// loadCalendarFromFile loads calendar from a specific file
func loadCalendarFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Error closing file: %v", err)
		}
	}()

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	var newCalendar CalendarData
	if err := json.Unmarshal(data, &newCalendar); err != nil {
		return err
	}

	CalendarMutex.Lock()
	Calendar = &newCalendar
	CalendarMutex.Unlock()

	return nil
}

// CommitCalendar commits tmp changes: creates backup and makes tmp the new main
func CommitCalendar() error {
	CalendarMutex.Lock()
	defer CalendarMutex.Unlock()

	tmpFile := CalendarFile + TmpSuffix

	// Check if tmp file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		return fmt.Errorf("no temporary changes to commit")
	}

	// Ensure backup directory exists
	backupDirPath := filepath.Join(filepath.Dir(CalendarFile), BackupDir)
	if err := os.MkdirAll(backupDirPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create backup of current calendar (if exists)
	if _, err := os.Stat(CalendarFile); err == nil {
		timestamp := time.Now().Unix()
		backupFile := filepath.Join(backupDirPath, fmt.Sprintf("%d_calendar_data.json%s", timestamp, BackupSuffix))
		if err := os.Rename(CalendarFile, backupFile); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		log.Printf("✅ Backup created: %s", backupFile)
	}

	// Make tmp file the new main file
	if err := os.Rename(tmpFile, CalendarFile); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	log.Printf("✅ Changes committed to %s", CalendarFile)
	return nil
}

// RevertCalendar discards tmp changes and reloads from main file
func RevertCalendar() error {
	tmpFile := CalendarFile + TmpSuffix

	// Check if tmp file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		return fmt.Errorf("no temporary changes to revert")
	}

	// Delete tmp file
	if err := os.Remove(tmpFile); err != nil {
		return fmt.Errorf("failed to remove tmp file: %w", err)
	}

	// Reload from main file
	if err := LoadCalendar(); err != nil {
		return fmt.Errorf("failed to reload calendar: %w", err)
	}

	log.Printf("✅ Changes reverted, reloaded from %s", CalendarFile)
	return nil
}

// HasTmpCalendar checks if a temporary calendar file exists
func HasTmpCalendar() bool {
	tmpFile := CalendarFile + TmpSuffix
	_, err := os.Stat(tmpFile)
	return err == nil
}
