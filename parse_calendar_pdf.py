#!/usr/bin/env python3
"""
Abfall-Kalender PDF Parser

Parst die Winterberg Abfall-Kalender PDFs automatisch und extrahiert
alle Abholtermine als JSON.

Verwendung:
    python parse_calendar_pdf.py <pdf_datei> [--output <json_datei>] [--year <jahr>]

Beispiel:
    python parse_calendar_pdf.py Abfallkalender_2026.pdf --year 2026
    # Speichert nach data/2026.json

    python parse_calendar_pdf.py Abfallkalender_2025.pdf --year 2025 --output custom.json
    # Speichert nach custom.json
"""

import argparse
import json
import os
import re
import sys

import cv2
import fitz  # PyMuPDF
import numpy as np

# Icon-Farben in HSV
ICON_COLORS = {
    "restmuell": {"lower": np.array([0, 0, 115]), "upper": np.array([180, 50, 140])},
    "biotonne": {"lower": np.array([35, 100, 100]), "upper": np.array([85, 255, 255])},
    "papiertonne": {"lower": np.array([90, 200, 200]), "upper": np.array([110, 255, 255])},
    "gelber_sack": {"lower": np.array([20, 100, 100]), "upper": np.array([35, 255, 255])},
}

WASTE_DESCRIPTIONS = {
    "restmuell": "Restmüll",
    "biotonne": "Biotonne",
    "papiertonne": "Papiertonne",
    "gelber_sack": "Gelber Sack",
}

# Offizielle Ortsnamen
OFFICIAL_DISTRICTS = [
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
]

# Mapping für kombinierte PDF-Seiten
DISTRICT_EXPANSION = {
    "Neuastenberg": ["Langewiese", "Mollseifen", "Neuastenberg", "Hoheleye"],
}


def find_grid_lines(img):
    """Findet alle horizontalen und vertikalen Tabellenlinien."""
    gray = cv2.cvtColor(img, cv2.COLOR_BGR2GRAY)
    edges = cv2.Canny(gray, 30, 100)
    lines = cv2.HoughLinesP(
        edges, 1, np.pi / 180, threshold=100, minLineLength=500, maxLineGap=10
    )

    h_lines = []
    v_lines = []

    if lines is not None:
        for line in lines:
            x1, y1, x2, y2 = line[0]
            if abs(y2 - y1) < 10 and abs(x2 - x1) > 500:
                h_lines.append((y1 + y2) // 2)
            if abs(x2 - x1) < 10 and abs(y2 - y1) > 500:
                v_lines.append((x1 + x2) // 2)

    def cluster(values, threshold=10):
        if not values:
            return []
        values = sorted(values)
        clusters = [[values[0]]]
        for v in values[1:]:
            if v - clusters[-1][-1] < threshold:
                clusters[-1].append(v)
            else:
                clusters.append([v])
        return sorted([int(np.mean(c)) for c in clusters])

    return cluster(h_lines), cluster(v_lines)


def find_icons(img, color_config, min_area=100, max_area=15000):
    """Findet alle Icons einer bestimmten Farbe."""
    hsv = cv2.cvtColor(img, cv2.COLOR_BGR2HSV)
    mask = cv2.inRange(hsv, color_config["lower"], color_config["upper"])

    kernel = np.ones((3, 3), np.uint8)
    mask = cv2.morphologyEx(mask, cv2.MORPH_OPEN, kernel)
    mask = cv2.morphologyEx(mask, cv2.MORPH_CLOSE, kernel)

    contours, _ = cv2.findContours(mask, cv2.RETR_EXTERNAL, cv2.CHAIN_APPROX_SIMPLE)

    icons = []
    for cnt in contours:
        area = cv2.contourArea(cnt)
        if min_area < area < max_area:
            x, y, w, h = cv2.boundingRect(cnt)
            aspect = max(w, h) / min(w, h) if min(w, h) > 0 else 999

            # Icons sind kompakt, Hintergrund-Streifen sind lang
            if aspect < 3:
                M = cv2.moments(cnt)
                if M["m00"] > 0:
                    cx = int(M["m10"] / M["m00"])
                    cy = int(M["m01"] / M["m00"])
                    icons.append((cx, cy))
    return icons


def map_to_cell(x, y, h_lines, v_lines):
    """Mappt Pixel-Koordinaten auf Monat und Tag."""
    # Spalte (Monat)
    col = 0
    for i in range(len(v_lines) - 1):
        if v_lines[i] <= x < v_lines[i + 1]:
            col = i
            break
    else:
        if x >= v_lines[-1]:
            col = len(v_lines) - 2

    # Zeile (Tag)
    row = 0
    for i in range(len(h_lines) - 1):
        if h_lines[i] <= y < h_lines[i + 1]:
            row = i
            break
    else:
        if y >= h_lines[-1]:
            row = len(h_lines) - 2

    month = col + 1
    day = row  # row 0 = header, row 1 = tag 1

    return month, day


def render_pdf_page(pdf_path, page_num, dpi=300):
    """Rendert eine PDF-Seite als Bild."""
    doc = fitz.open(pdf_path)
    page = doc[page_num]
    mat = fitz.Matrix(dpi / 72, dpi / 72)
    pix = page.get_pixmap(matrix=mat)
    img = np.frombuffer(pix.samples, dtype=np.uint8).reshape(pix.height, pix.width, pix.n)
    if pix.n == 4:
        img = cv2.cvtColor(img, cv2.COLOR_RGBA2BGR)
    else:
        img = cv2.cvtColor(img, cv2.COLOR_RGB2BGR)
    doc.close()
    return img


def extract_district_name(pdf_path, page_num):
    """Extrahiert den Ortsnamen aus dem PDF-Text."""
    doc = fitz.open(pdf_path)
    page = doc[page_num]
    text = page.get_text()
    doc.close()

    match = re.search(r"Abfall-Kalender\s+\d{4}\s+(.+?)(?:\n|$)", text)
    if match:
        return match.group(1).strip()
    return f"Seite_{page_num}"


def parse_calendar_page(img):
    """Parst eine einzelne Kalenderseite."""
    h_lines, v_lines = find_grid_lines(img)

    if len(h_lines) < 30 or len(v_lines) < 10:
        return None

    result = {}

    for waste_type, color_config in ICON_COLORS.items():
        icons = find_icons(img, color_config)
        dates = []

        for x, y in icons:
            month, day = map_to_cell(x, y, h_lines, v_lines)
            if 1 <= month <= 12 and 1 <= day <= 31:
                dates.append((month, day))

        dates = sorted(set(dates))
        result[waste_type] = [f"{m}.{d}" for m, d in dates]

    return result


def parse_pdf(pdf_path, year=2026):
    """Parst das komplette PDF."""
    doc = fitz.open(pdf_path)
    num_pages = len(doc)
    doc.close()

    result = {
        "year": year,
        "districts": {},
        "metadata": {
            "generated": "auto-parsed",
            "source": pdf_path.split("/")[-1],
        },  # Note: metadata values must be strings for Go app compatibility
    }

    for page_num in range(num_pages):
        print(f"Verarbeite Seite {page_num}...", end=" ", flush=True)

        district = extract_district_name(pdf_path, page_num)
        img = render_pdf_page(pdf_path, page_num)
        page_result = parse_calendar_page(img)

        if page_result is None:
            print("übersprungen (kein Kalender)")
            continue

        # Events erstellen
        events = []
        for waste_type, dates in page_result.items():
            for date_str in dates:
                month, day = map(int, date_str.split("."))
                events.append({
                    "date": f"{year}-{month:02d}-{day:02d}",
                    "type": waste_type,
                    "description": WASTE_DESCRIPTIONS.get(waste_type, waste_type),
                })

        events.sort(key=lambda e: e["date"])

        # Prüfe ob Ortsname expandiert werden muss
        expanded = None
        for key, districts in DISTRICT_EXPANSION.items():
            if key in district:
                expanded = districts
                break

        if expanded:
            for exp_district in expanded:
                result["districts"][exp_district] = {"events": [e.copy() for e in events]}
            print(f"{len(expanded)} Ortsteile, je {len(events)} Termine")
        else:
            result["districts"][district] = {"events": events}
            print(f"{district}: {len(events)} Termine")

    return result


def main():
    parser = argparse.ArgumentParser(
        description="Parst Winterberg Abfall-Kalender PDFs"
    )
    parser.add_argument("pdf", help="PDF-Datei zum Parsen")
    parser.add_argument(
        "--output", "-o", help="Ausgabe JSON-Datei (Standard: data/{year}.json)"
    )
    parser.add_argument("--year", "-y", type=int, default=2026, help="Kalenderjahr")

    args = parser.parse_args()

    # Default output to data/{year}.json
    output_file = args.output
    if output_file is None:
        output_file = f"data/{args.year}.json"

    print(f"Parse {args.pdf}...")
    print()

    result = parse_pdf(args.pdf, args.year)

    # Ensure output directory exists
    os.makedirs(os.path.dirname(output_file) or ".", exist_ok=True)

    with open(output_file, "w", encoding="utf-8") as f:
        json.dump(result, f, indent=2, ensure_ascii=False)

    print()
    print(f"Gespeichert: {output_file}")
    print(f"Ortsteile: {len(result['districts'])}")
    print(f"Gesamt-Termine: {sum(len(d['events']) for d in result['districts'].values())}")


if __name__ == "__main__":
    main()
