let config = null;
let calendarComponent = null;
let selectedFormat = 'ics';
let selectedWasteTypes = new Set();

// Init
document.addEventListener('DOMContentLoaded', async () => {
    await loadConfig();
    await reloadCalendar();

    document.getElementById('ortsteil').addEventListener('change', () => {
        reloadCalendar();
        updateSubscribeLink();
    });
    document.getElementById('year').addEventListener('change', () => {
        reloadCalendar();
        // No need to update subscribe link - it's not year-dependent
    });

    // Initialize subscribe link
    updateSubscribeLink();
});

async function loadConfig() {
    config = await API.getConfig();

    // Fill districts
    const districtsSelect = document.getElementById('ortsteil');
    config.districts.forEach(district => {
        const option = document.createElement('option');
        option.value = district;
        option.textContent = district;
        districtsSelect.appendChild(option);
    });

    // Fill years - get all available years from calendar data
    const calendarData = await API.getCalendar();
    const yearSelect = document.getElementById('year');

    // For now, just use current year - we'll enhance this when we have multi-year data
    const currentYear = calendarData.year || new Date().getFullYear();
    const option = document.createElement('option');
    option.value = currentYear;
    option.textContent = currentYear;
    option.selected = true;
    yearSelect.appendChild(option);

    // Fill waste types for filter (legend style)
    const wasteFilter = document.getElementById('waste-filter');
    Object.entries(config.wasteTypes).forEach(([key, name]) => {
        const item = document.createElement('div');
        item.className = 'waste-legend-item';
        item.dataset.type = key;
        item.onclick = () => toggleWasteType(key);

        item.innerHTML = `
            <input type="checkbox" id="waste-${key}" checked>
            <div class="waste-color-box legend-color ${key}"></div>
            <label for="waste-${key}">${name}</label>
        `;

        wasteFilter.appendChild(item);
        selectedWasteTypes.add(key);
    });

    // Fill legend
    const legend = document.getElementById('legend');
    Object.entries(config.wasteTypes).forEach(([key, name]) => {
        const item = document.createElement('div');
        item.className = 'legend-item';

        const color = document.createElement('div');
        color.className = `legend-color ${key}`;
        item.appendChild(color);

        const label = document.createElement('span');
        label.textContent = name;
        item.appendChild(label);

        legend.appendChild(item);
    });
}

async function reloadCalendar() {
    const district = document.getElementById('ortsteil').value;
    const year = parseInt(document.getElementById('year').value);

    const data = await API.getDistrictCalendar(district);

    // Create calendar component with thin stripes
    if (!calendarComponent) {
        calendarComponent = new Calendar('#calendar', {
            editMode: false,
            enableClick: false,
            enableDragDrop: false,
            showDeleteButtons: false,
            holidays: config.holidays || {}
        });
    }

    calendarComponent.render(year, data.events || []);
}

function toggleWasteType(type) {
    const checkbox = document.getElementById(`waste-${type}`);
    const item = checkbox.closest('.waste-legend-item');

    checkbox.checked = !checkbox.checked;

    if (checkbox.checked) {
        selectedWasteTypes.add(type);
        item.classList.remove('inactive');
    } else {
        selectedWasteTypes.delete(type);
        item.classList.add('inactive');
    }

    updateFilterToggle();
    updateSubscribeLink();
}

function toggleAllWaste() {
    const allSelected = selectedWasteTypes.size === Object.keys(config.wasteTypes).length;

    document.querySelectorAll('.waste-legend-item').forEach(item => {
        const checkbox = item.querySelector('input');
        const type = item.dataset.type;

        if (allSelected) {
            // Deselect all
            checkbox.checked = false;
            item.classList.add('inactive');
            selectedWasteTypes.delete(type);
        } else {
            // Select all
            checkbox.checked = true;
            item.classList.remove('inactive');
            selectedWasteTypes.add(type);
        }
    });

    updateFilterToggle();
    updateSubscribeLink();
}

function updateFilterToggle() {
    const toggle = document.getElementById('filter-toggle');
    const allSelected = selectedWasteTypes.size === Object.keys(config.wasteTypes).length;

    if (allSelected) {
        toggle.textContent = 'Alle abwählen';
    } else {
        toggle.textContent = 'Alle auswählen';
    }
}

function onFormatChange(format) {
    selectedFormat = format;

    // Show/hide reminders section
    const remindersSection = document.getElementById('reminders-section');
    if (selectedFormat === 'ics') {
        remindersSection.classList.remove('hidden');
    } else {
        remindersSection.classList.add('hidden');
    }

    updateDownloadText();
}

function updateDownloadText() {
    const text = {
        ics: 'Herunterladen (.ics)',
        csv: 'Herunterladen (.csv)',
        json: 'Herunterladen (.json)'
    };
    document.getElementById('download-text').textContent = text[selectedFormat];
}

function updateReminderState() {
    // Update visual state of reminder items based on checkbox
    const reminders = [
        { checkbox: 'reminder-2days', item: 'reminder-item-2days' },
        { checkbox: 'reminder-1day', item: 'reminder-item-1day' },
        { checkbox: 'reminder-same', item: 'reminder-item-same' }
    ];

    reminders.forEach(reminder => {
        const checkbox = document.getElementById(reminder.checkbox);
        const item = document.getElementById(reminder.item);

        if (checkbox.checked) {
            item.classList.remove('opacity-50', 'pointer-events-none');
        } else {
            item.classList.add('opacity-50', 'pointer-events-none');
        }
    });
}

function downloadCalendar() {
    const district = document.getElementById('ortsteil').value;
    const year = document.getElementById('year').value;

    if (selectedWasteTypes.size === 0) {
        alert('Bitte mindestens einen Abfalltyp auswählen');
        return;
    }

    // Build URL with parameters
    const params = new URLSearchParams({
        district,
        year,
        format: selectedFormat,
        wasteTypes: Array.from(selectedWasteTypes).join(',')
    });

    // Add reminders for ICS
    if (selectedFormat === 'ics') {
        if (document.getElementById('reminder-2days').checked) {
            const time = document.getElementById('time-2days').value;
            params.append('reminder2Days', 'true');
            params.append('time2Days', time);
        }
        if (document.getElementById('reminder-1day').checked) {
            const time = document.getElementById('time-1day').value;
            params.append('reminder1Day', 'true');
            params.append('time1Day', time);
        }
        if (document.getElementById('reminder-same').checked) {
            const time = document.getElementById('time-same').value;
            params.append('reminderSameDay', 'true');
            params.append('timeSameDay', time);
        }
    }

    window.location.href = `/api/download?${params.toString()}`;
}

function updateSubscribeLink() {
    const district = document.getElementById('ortsteil').value;

    if (!district) {
        return;
    }

    // Build subscribe URL (no year parameter - backend returns last year + future)
    const params = new URLSearchParams();

    // Add waste types filter if not all selected
    if (selectedWasteTypes.size > 0 && selectedWasteTypes.size < Object.keys(config.wasteTypes).length) {
        params.append('wasteTypes', Array.from(selectedWasteTypes).join(','));
    }

    // Use webcal:// protocol for calendar subscriptions (works for both HTTP and HTTPS)
    const host = window.location.host;
    const queryString = params.toString();
    const url = `webcal://${host}/api/subscribe/${encodeURIComponent(district)}${queryString ? '?' + queryString : ''}`;

    // Update the subscribe link href
    document.getElementById('subscribe-link').href = url;
}

function toggleOptions() {
    const options = document.getElementById('advanced-options');
    const icon = document.getElementById('toggle-icon');

    if (options.style.maxHeight && options.style.maxHeight !== '0px') {
        options.style.maxHeight = '0px';
        icon.style.transform = 'rotate(0deg)';
    } else {
        options.style.maxHeight = options.scrollHeight + 'px';
        icon.style.transform = 'rotate(180deg)';
    }
}
