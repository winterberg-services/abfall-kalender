// API Client for Abfallkalender

class API {
    static async getConfig() {
        const response = await fetch('/api/config');
        return await response.json();
    }

    static async getCalendar(year = null) {
        const url = year ? `/api/calendar?year=${year}` : '/api/calendar';
        const response = await fetch(url);
        return await response.json();
    }

    static async getDistrictCalendar(district, year = null) {
        const url = year ? `/api/calendar/${district}?year=${year}` : `/api/calendar/${district}`;
        const response = await fetch(url);
        return await response.json();
    }

    static async addEvent(district, date, wasteType) {
        const response = await fetch('/api/events/add', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({
                district,
                date,
                waste_type: wasteType
            })
        });
        return await response.json();
    }

    static async deleteEvent(district, date, type) {
        const response = await fetch('/api/events/delete', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({district, date, type})
        });
        return await response.json();
    }

    static async moveEvent(district, oldDate, newDate, type) {
        const response = await fetch('/api/events/move', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({
                district,
                old_date: oldDate,
                new_date: newDate,
                type
            })
        });
        return await response.json();
    }

    static async commitCalendar() {
        const response = await fetch('/api/calendar/commit', {
            method: 'POST'
        });
        if (!response.ok) {
            const error = await response.text();
            throw new Error(error);
        }
        return await response.json();
    }

    static async revertCalendar() {
        const response = await fetch('/api/calendar/revert', {
            method: 'POST'
        });
        if (!response.ok) {
            const error = await response.text();
            throw new Error(error);
        }
        return await response.json();
    }

    static async getCalendarStatus() {
        const response = await fetch('/api/calendar/status');
        return await response.json();
    }

    static getDownloadURL(district, year) {
        return `/download/${district}/${year}`;
    }
}
