package app

import (
	"os"
	"path/filepath"
	"sync"
)

// Constants
const (
	DefaultCalendarFile = "calendar_data.json"
	BackupDir           = "backup"
	BackupSuffix        = ".backup"
	TmpSuffix           = ".tmp.json"
	FilePermissions     = 0644

	// Error messages
	ErrEditModeDisabled     = "Edit mode disabled"
	ErrInvalidDateFormat    = "Invalid date format"
	ErrInvalidYear          = "Invalid year"
	ErrInvalidFormat        = "Invalid format"
	ErrInternalServer       = "Internal server error"
	ErrFailedToSave         = "Failed to save calendar"
	ErrFailedToGenerateJSON = "Failed to generate JSON"

	// Metadata keys
	MetadataCreatedAt = "created_at"
	MetadataSource    = "manual"

	// Mode strings
	ModeServe = "serve"
	ModeEdit  = "edit"

	// ICS constants
	ICSProductID = "-//Winterberg//Abfallkalender//DE"
	ICSTimezone  = "Europe/Berlin"
)

// Global variables
var (
	CalendarFile  = DefaultCalendarFile
	Calendar      *CalendarData
	CalendarMutex sync.RWMutex
	EditMode      bool

	// Embedded files (set by main)
	StaticFiles interface{}
	IndexHTML   []byte
	EditHTML    []byte
)

// Districts list
var Districts = []string{
	"Winterberg",
	"Siedlinghausen",
	"Züschen",
	"Silbach",
	"Niedersfeld",
	"Langewiese",
	"Mollseifen",
	"Neuastenberg",
	"Hoheleye",
	"Grönebach",
	"Hildfeld",
	"Elkeringhausen",
	"Altastenberg",
	"Altenfeld",
}

// WasteTypes maps waste type keys to their German display names
var WasteTypes = map[string]string{
	"restmuell":   "Restmüll",
	"biotonne":    "Biotonne",
	"papiertonne": "Papiertonne",
	"gelber_sack": "Gelber Sack",
	"sondermuell": "Sondermüll",
	"altkleider":  "Altkleider",
}

func init() {
	// Get calendar file path from current directory or user home
	if cwd, err := os.Getwd(); err == nil {
		CalendarFile = filepath.Join(cwd, "calendar_data.json")
	}
}
