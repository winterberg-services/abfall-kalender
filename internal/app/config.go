package app

import (
	"os"
	"path/filepath"
	"sync"
)

// Constants
const (
	DataDir         = "data"
	BackupDir       = "backup"
	BackupSuffix    = ".backup"
	TmpSuffix       = ".tmp.json"
	FilePermissions = 0644

	// Error messages
	ErrEditModeDisabled     = "Edit mode disabled"
	ErrInvalidDateFormat    = "Invalid date format"
	ErrInvalidYear          = "Invalid year"
	ErrInvalidFormat        = "Invalid format"
	ErrInternalServer       = "Internal server error"
	ErrFailedToSave         = "Failed to save calendar"
	ErrFailedToGenerateJSON = "Failed to generate JSON"
	ErrYearNotFound         = "Year not found"

	// Mode strings
	ModeServe = "serve"
	ModeEdit  = "edit"

	// ICS constants
	ICSProductID = "-//Winterberg//Abfallkalender//DE"
	ICSTimezone  = "Europe/Berlin"
)

// Global variables
var (
	DataPath      string
	Store         *CalendarStore
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
	// Set data path from current directory
	if cwd, err := os.Getwd(); err == nil {
		DataPath = filepath.Join(cwd, DataDir)
	}
}
