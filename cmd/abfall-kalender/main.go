package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/klabast/wb-services/abfall-kalender/internal/app"
	"github.com/klabast/wb-services/abfall-kalender/internal/commands"
)

//go:embed static/*
var staticFiles embed.FS

//go:embed static/index.html
var indexHTML []byte

//go:embed static/edit.html
var editHTML []byte

func main() {
	// Check for subcommands
	if len(os.Args) > 1 && os.Args[1] == "hash-password" {
		commands.HashPassword(os.Args[2:])
		return
	}

	// Parse flags
	port := flag.Int("port", 8080, "Port to listen on")
	flag.BoolVar(&app.EditMode, "edit", false, "Enable edit mode (default is serve mode)")
	flag.Parse()

	// Make embedded files available to app package
	app.StaticFiles = staticFiles
	app.IndexHTML = indexHTML
	app.EditHTML = editHTML

	// Load and validate auth credentials (if edit mode)
	if app.EditMode {
		if err := app.LoadAuthCredentials(); err != nil {
			log.Fatalf("Failed to load auth credentials: %v", err)
		}
	}

	// Load all calendar years (with tmp check in edit mode)
	var loadErr error
	if app.EditMode {
		loadErr = app.LoadAllYearsWithTmpCheck()
	} else {
		loadErr = app.LoadAllYears()
	}

	if loadErr != nil {
		log.Fatalf("Failed to load calendar data: %v", loadErr)
	}

	// Setup routes
	http.HandleFunc("/", app.ServeIndex)
	http.HandleFunc("/api/config", app.GetConfig)
	http.HandleFunc("/api/calendar/", app.HandleDistrictCalendar)
	http.HandleFunc("/api/calendar", app.HandleCalendar)
	http.HandleFunc("/api/download", app.HandleDownload)
	http.HandleFunc("/api/subscribe/", app.HandleSubscribe)

	// Edit mode routes (protected with Basic Auth)
	if app.EditMode {
		http.HandleFunc("/edit", app.RequireAuth(app.ServeEdit))
		http.HandleFunc("/api/events/add", app.RequireAuth(app.AddEvent))
		http.HandleFunc("/api/events/delete", app.RequireAuth(app.DeleteEvent))
		http.HandleFunc("/api/events/move", app.RequireAuth(app.MoveEvent))
		http.HandleFunc("/api/calendar/commit", app.RequireAuth(app.HandleCalendarCommit))
		http.HandleFunc("/api/calendar/revert", app.RequireAuth(app.HandleCalendarRevert))
		http.HandleFunc("/api/calendar/status", app.RequireAuth(app.HandleCalendarStatus))
	}

	// Serve static files
	http.Handle("/static/", http.FileServer(http.FS(staticFiles)))

	mode := app.ModeServe
	if app.EditMode {
		mode = app.ModeEdit
	}

	log.Printf("Starting Abfallkalender in %s mode on http://localhost:%d", mode, *port)
	log.Printf("Data directory: %s", app.DataPath)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil); err != nil {
		log.Fatal(err)
	}
}
