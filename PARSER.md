# Abfall-Kalender PDF Parser

Automatisches Parsen der Winterberg Abfall-Kalender PDFs.

## Installation

```bash
# Python venv erstellen
python3 -m venv venv
source venv/bin/activate

# Dependencies installieren
pip install -r requirements-parser.txt
```

## Verwendung

```bash
# Kalender parsen
python parse_calendar_pdf.py Abfallkalender_2026.pdf

# Mit Custom Output
python parse_calendar_pdf.py Abfallkalender_2026.pdf --output calendar_data.json --year 2026
```

## Wie es funktioniert

1. **PDF zu Bild**: Jede Seite wird mit 300 DPI gerendert
2. **Grid-Erkennung**: Hough Lines findet die Tabellenlinien automatisch
3. **Icon-Erkennung**: HSV-Farbfilter erkennen die verschiedenen Mülltonnen-Icons:
    - Grau (V=115-140): Restmüll
    - Grün (H=35-85): Biotonne
    - Blau (H=90-110): Papiertonne
    - Gelb (H=20-35): Gelber Sack
4. **Mapping**: Icon-Koordinaten werden auf das Grid gemappt → Monat/Tag
5. **JSON-Export**: Ausgabe im gleichen Format wie `calendar_data.json`

## Unterstützte Ortsteile

Der Parser erkennt automatisch alle 14 Ortsteile:

- Winterberg, Siedlinghausen, Züschen, Silbach, Niedersfeld
- Langewiese, Mollseifen, Neuastenberg, Hoheleye
- Grönebach, Hildfeld, Elkeringhausen, Altastenberg, Altenfeld

Kombinierte PDF-Seiten (z.B. "Neuastenberg, Mollseifen, Langewiese...") werden automatisch auf die einzelnen Ortsteile
aufgeteilt.

## Ausgabe-Format

```json
{
  "year": 2026,
  "districts": {
    "Winterberg": {
      "events": [
        {"date": "2026-01-07", "type": "restmuell", "description": "Restmüll"},
        {"date": "2026-01-13", "type": "gelber_sack", "description": "Gelber Sack"},
        ...
      ]
    },
    ...
  }
}
```
