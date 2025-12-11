package app

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// LoadAllYears loads all calendar years from the data directory
func LoadAllYears() error {
	store := &CalendarStore{
		Years:     make(map[int]*YearData),
		YearsList: []int{},
	}

	// Read all JSON files from data directory
	files, err := filepath.Glob(filepath.Join(DataPath, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to list data files: %w", err)
	}

	for _, file := range files {
		// Skip tmp files
		if strings.HasSuffix(file, TmpSuffix) {
			continue
		}

		yearData, err := loadYearFromFile(file)
		if err != nil {
			log.Printf("Warning: failed to load %s: %v", file, err)
			continue
		}

		store.Years[yearData.Year] = yearData
		store.YearsList = append(store.YearsList, yearData.Year)
		log.Printf("Loaded calendar: %d (%d districts)", yearData.Year, len(yearData.Districts))
	}

	// Sort years
	sort.Ints(store.YearsList)

	CalendarMutex.Lock()
	Store = store
	CalendarMutex.Unlock()

	if len(store.YearsList) == 0 {
		return fmt.Errorf("no calendar data found in %s", DataPath)
	}

	log.Printf("Loaded %d calendar year(s): %v", len(store.YearsList), store.YearsList)
	return nil
}

// loadYearFromFile loads a single year's data from a file
func loadYearFromFile(filename string) (*YearData, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var yearData YearData
	if err := json.Unmarshal(data, &yearData); err != nil {
		return nil, err
	}

	return &yearData, nil
}

// GetYear returns data for a specific year
func GetYear(year int) (*YearData, bool) {
	CalendarMutex.RLock()
	defer CalendarMutex.RUnlock()

	if Store == nil {
		return nil, false
	}

	data, ok := Store.Years[year]
	return data, ok
}

// GetAvailableYears returns sorted list of available years
func GetAvailableYears() []int {
	CalendarMutex.RLock()
	defer CalendarMutex.RUnlock()

	if Store == nil {
		return []int{}
	}

	return Store.YearsList
}

// GetCurrentYear returns the most relevant year (current or next)
func GetCurrentYear() int {
	years := GetAvailableYears()
	if len(years) == 0 {
		return time.Now().Year()
	}

	currentYear := time.Now().Year()

	// Return current year if available
	for _, y := range years {
		if y == currentYear {
			return y
		}
	}

	// Return next available year
	for _, y := range years {
		if y > currentYear {
			return y
		}
	}

	// Return latest year
	return years[len(years)-1]
}

// SaveYear saves a specific year's data
func SaveYear(year int) error {
	CalendarMutex.RLock()
	yearData, ok := Store.Years[year]
	CalendarMutex.RUnlock()

	if !ok {
		return fmt.Errorf("year %d not found", year)
	}

	filename := filepath.Join(DataPath, fmt.Sprintf("%d.json", year))
	return saveYearToFile(filename, yearData)
}

// saveYearToFile saves year data to a file with backup
func saveYearToFile(filename string, yearData *YearData) error {
	data, err := json.MarshalIndent(yearData, "", "  ")
	if err != nil {
		return err
	}

	// Create backup if file exists
	if _, err := os.Stat(filename); err == nil {
		backupDirPath := filepath.Join(filepath.Dir(filename), "..", BackupDir)
		if err := os.MkdirAll(backupDirPath, 0755); err != nil {
			log.Printf("Warning: failed to create backup dir: %v", err)
		} else {
			timestamp := time.Now().Unix()
			backupFile := filepath.Join(backupDirPath, fmt.Sprintf("%d_%d.json%s", timestamp, yearData.Year, BackupSuffix))
			if err := copyFile(filename, backupFile); err != nil {
				log.Printf("Warning: failed to create backup: %v", err)
			}
		}
	}

	// Write to temp file first
	tmpFile := filename + TmpSuffix
	if err := os.WriteFile(tmpFile, data, FilePermissions); err != nil {
		return err
	}

	// Rename temp file to actual file
	return os.Rename(tmpFile, filename)
}

// copyFile copies a file
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, FilePermissions)
}

// saveTmpYear saves year data to tmp file (auto-save for edit mode)
func saveTmpYear(year int) error {
	CalendarMutex.RLock()
	yearData, ok := Store.Years[year]
	CalendarMutex.RUnlock()

	if !ok {
		return fmt.Errorf("year %d not found", year)
	}

	data, err := json.MarshalIndent(yearData, "", "  ")
	if err != nil {
		return err
	}

	tmpFile := filepath.Join(DataPath, fmt.Sprintf("%d.json%s", year, TmpSuffix))
	return os.WriteFile(tmpFile, data, FilePermissions)
}

// LoadAllYearsWithTmpCheck loads all years, using tmp files if they exist
func LoadAllYearsWithTmpCheck() error {
	store := &CalendarStore{
		Years:     make(map[int]*YearData),
		YearsList: []int{},
	}

	// Read all JSON files from data directory
	files, err := filepath.Glob(filepath.Join(DataPath, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to list data files: %w", err)
	}

	for _, file := range files {
		// Skip tmp files in listing, we'll check for them separately
		if strings.HasSuffix(file, TmpSuffix) {
			continue
		}

		// Check if tmp file exists
		tmpFile := file + TmpSuffix
		loadFile := file
		if _, err := os.Stat(tmpFile); err == nil {
			log.Printf("⚠️  Found temporary file: %s (loading unsaved changes)", tmpFile)
			loadFile = tmpFile
		}

		yearData, err := loadYearFromFile(loadFile)
		if err != nil {
			log.Printf("Warning: failed to load %s: %v", loadFile, err)
			continue
		}

		store.Years[yearData.Year] = yearData
		store.YearsList = append(store.YearsList, yearData.Year)
	}

	sort.Ints(store.YearsList)

	CalendarMutex.Lock()
	Store = store
	CalendarMutex.Unlock()

	return nil
}

// CommitYear commits tmp changes for a specific year
func CommitYear(year int) error {
	CalendarMutex.Lock()
	defer CalendarMutex.Unlock()

	filename := filepath.Join(DataPath, fmt.Sprintf("%d.json", year))
	tmpFile := filename + TmpSuffix

	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		return fmt.Errorf("no temporary changes for year %d", year)
	}

	// Create backup
	backupDirPath := filepath.Join(DataPath, "..", BackupDir)
	if err := os.MkdirAll(backupDirPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	if _, err := os.Stat(filename); err == nil {
		timestamp := time.Now().Unix()
		backupFile := filepath.Join(backupDirPath, fmt.Sprintf("%d_%d.json%s", timestamp, year, BackupSuffix))
		if err := os.Rename(filename, backupFile); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		log.Printf("✅ Backup created: %s", backupFile)
	}

	if err := os.Rename(tmpFile, filename); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	log.Printf("✅ Changes committed for year %d", year)
	return nil
}

// CommitAllYears commits all tmp changes
func CommitAllYears() error {
	files, err := filepath.Glob(filepath.Join(DataPath, "*.json"+TmpSuffix))
	if err != nil {
		return err
	}

	for _, tmpFile := range files {
		// Extract year from filename
		base := filepath.Base(tmpFile)
		yearStr := strings.TrimSuffix(base, ".json"+TmpSuffix)
		year, err := strconv.Atoi(yearStr)
		if err != nil {
			continue
		}

		if err := CommitYear(year); err != nil {
			return err
		}
	}

	return nil
}

// RevertYear discards tmp changes for a specific year
func RevertYear(year int) error {
	tmpFile := filepath.Join(DataPath, fmt.Sprintf("%d.json%s", year, TmpSuffix))

	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		return fmt.Errorf("no temporary changes for year %d", year)
	}

	if err := os.Remove(tmpFile); err != nil {
		return fmt.Errorf("failed to remove tmp file: %w", err)
	}

	// Reload year from main file
	filename := filepath.Join(DataPath, fmt.Sprintf("%d.json", year))
	yearData, err := loadYearFromFile(filename)
	if err != nil {
		return fmt.Errorf("failed to reload year %d: %w", year, err)
	}

	CalendarMutex.Lock()
	Store.Years[year] = yearData
	CalendarMutex.Unlock()

	log.Printf("✅ Changes reverted for year %d", year)
	return nil
}

// RevertAllYears discards all tmp changes
func RevertAllYears() error {
	files, err := filepath.Glob(filepath.Join(DataPath, "*.json"+TmpSuffix))
	if err != nil {
		return err
	}

	for _, tmpFile := range files {
		base := filepath.Base(tmpFile)
		yearStr := strings.TrimSuffix(base, ".json"+TmpSuffix)
		year, err := strconv.Atoi(yearStr)
		if err != nil {
			continue
		}

		if err := RevertYear(year); err != nil {
			log.Printf("Warning: failed to revert year %d: %v", year, err)
		}
	}

	return nil
}

// HasTmpChanges checks if any temporary files exist
func HasTmpChanges() bool {
	files, err := filepath.Glob(filepath.Join(DataPath, "*.json"+TmpSuffix))
	if err != nil {
		return false
	}
	return len(files) > 0
}

// GetAllEvents returns all events across all years for a district
func GetAllEvents(district string) []Event {
	CalendarMutex.RLock()
	defer CalendarMutex.RUnlock()

	var allEvents []Event
	for _, yearData := range Store.Years {
		if dist, ok := yearData.Districts[district]; ok {
			allEvents = append(allEvents, dist.Events...)
		}
	}

	// Sort by date
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].Date < allEvents[j].Date
	})

	return allEvents
}
