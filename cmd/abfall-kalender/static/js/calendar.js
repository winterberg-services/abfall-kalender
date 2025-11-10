// Calendar Component for Abfallkalender

class Calendar {
    constructor(container, options = {}) {
        this.container = typeof container === 'string'
            ? document.querySelector(container)
            : container;

        this.options = {
            editMode: false,
            showDeleteButtons: false,
            enableDragDrop: false,
            enableClick: true,
            onEventClick: null,
            onDayClick: null,
            holidays: {},
            ...options
        };

        this.draggedEvent = null;
        this.WEEKDAYS = ['Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa', 'So'];
        this.WEEKDAYS_FULL = ['Sonntag', 'Montag', 'Dienstag', 'Mittwoch', 'Donnerstag', 'Freitag', 'Samstag'];
        this.MONTHS = ['Januar', 'Februar', 'März', 'April', 'Mai', 'Juni',
                       'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember'];

        // Create tooltip element
        this.tooltip = document.createElement('div');
        this.tooltip.className = 'day-tooltip';
        document.body.appendChild(this.tooltip);
    }

    // Format date as YYYY-MM-DD in local timezone (no UTC conversion)
    formatDateLocal(date) {
        const year = date.getFullYear();
        const month = String(date.getMonth() + 1).padStart(2, '0');
        const day = String(date.getDate()).padStart(2, '0');
        return `${year}-${month}-${day}`;
    }

    // Show tooltip with day information
    showTooltip(date, events, x, y) {
        const dayOfWeek = this.WEEKDAYS_FULL[date.getDay()];
        const day = date.getDate();
        const month = this.MONTHS[date.getMonth()];
        const year = date.getFullYear();
        const isoDate = this.formatDateLocal(date);
        const holiday = this.options.holidays[isoDate];

        // Check if it's today
        const today = new Date();
        today.setHours(0, 0, 0, 0);
        const checkDate = new Date(date);
        checkDate.setHours(0, 0, 0, 0);
        const isToday = checkDate.getTime() === today.getTime();

        let html = `<div class="tooltip-date">${dayOfWeek}, ${day}. ${month} ${year}`;
        if (isToday) {
            html += ` <span style="font-style: italic; color: #007bff; font-size: 12px;">(Heute)</span>`;
        }
        html += `</div>`;

        if (holiday) {
            html += `<div class="tooltip-holiday">${holiday}</div>`;
        }

        if (events.length > 0) {
            html += '<div class="tooltip-events">';
            events.forEach(event => {
                const colorClass = event.type;
                html += `<div class="tooltip-event">`;
                html += `<div class="tooltip-event-color ${colorClass}"></div>`;
                html += `<span>${event.description}</span>`;
                html += `</div>`;
            });
            html += '</div>';
        }

        this.tooltip.innerHTML = html;
        this.tooltip.style.left = (x + 10) + 'px';
        this.tooltip.style.top = (y + 10) + 'px';
        this.tooltip.classList.add('show');
    }

    // Hide tooltip
    hideTooltip() {
        this.tooltip.classList.remove('show');
    }

    render(year, events) {
        this.container.innerHTML = '';

        // Group events by date
        const eventsByDate = {};
        events.forEach(event => {
            if (!eventsByDate[event.date]) {
                eventsByDate[event.date] = [];
            }
            eventsByDate[event.date].push(event);
        });

        const today = new Date();
        today.setHours(0, 0, 0, 0);

        // Render each month
        for (let month = 0; month < 12; month++) {
            const monthDiv = this.createMonthDiv(year, month, eventsByDate, today);
            this.container.appendChild(monthDiv);
        }
    }

    createMonthDiv(year, month, eventsByDate, today) {
        const monthDiv = document.createElement('div');
        monthDiv.className = 'month';

        // Header
        const header = document.createElement('div');
        header.className = 'month-header';
        header.textContent = this.MONTHS[month] + ' ' + year;
        monthDiv.appendChild(header);

        // Weekdays
        const weekdaysDiv = document.createElement('div');
        weekdaysDiv.className = 'weekdays';
        this.WEEKDAYS.forEach(wd => {
            const wdDiv = document.createElement('div');
            wdDiv.className = 'weekday';
            wdDiv.textContent = wd;
            weekdaysDiv.appendChild(wdDiv);
        });
        monthDiv.appendChild(weekdaysDiv);

        // Days
        const daysDiv = document.createElement('div');
        daysDiv.className = 'days';

        const firstDay = new Date(year, month, 1);
        const lastDay = new Date(year, month + 1, 0);
        // Convert to Monday=0, Sunday=6 (instead of Sunday=0, Saturday=6)
        const startingDayOfWeek = (firstDay.getDay() + 6) % 7;

        // Previous month days
        const prevMonthLastDay = new Date(year, month, 0).getDate();
        for (let i = startingDayOfWeek - 1; i >= 0; i--) {
            const dayDiv = this.createDayDiv(
                prevMonthLastDay - i,
                new Date(year, month - 1, prevMonthLastDay - i),
                eventsByDate,
                today,
                true
            );
            daysDiv.appendChild(dayDiv);
        }

        // Current month days
        for (let day = 1; day <= lastDay.getDate(); day++) {
            const date = new Date(year, month, day);
            const dayDiv = this.createDayDiv(day, date, eventsByDate, today, false);

            if (date.getTime() === today.getTime()) {
                dayDiv.classList.add('today');
            }

            daysDiv.appendChild(dayDiv);
        }

        // Next month days
        const remainingDays = 42 - daysDiv.children.length;
        for (let day = 1; day <= remainingDays; day++) {
            const dayDiv = this.createDayDiv(
                day,
                new Date(year, month + 1, day),
                eventsByDate,
                today,
                true
            );
            daysDiv.appendChild(dayDiv);
        }

        monthDiv.appendChild(daysDiv);
        return monthDiv;
    }

    createDayDiv(dayNumber, date, eventsByDate, today, otherMonth) {
        const dayDiv = document.createElement('div');
        dayDiv.className = 'day';
        if (otherMonth) dayDiv.classList.add('other-month');

        // Mark weekend (Saturday=6, Sunday=0)
        const dayOfWeek = date.getDay();
        if (dayOfWeek === 0 || dayOfWeek === 6) {
            dayDiv.classList.add('weekend');
        }

        const isoDate = this.formatDateLocal(date);
        dayDiv.dataset.date = isoDate;

        // Mark holidays
        if (this.options.holidays[isoDate]) {
            dayDiv.classList.add('holiday');
            dayDiv.title = this.options.holidays[isoDate];
        }

        // Day number
        const dayNumberDiv = document.createElement('div');
        dayNumberDiv.className = 'day-number';
        dayNumberDiv.textContent = dayNumber;
        dayDiv.appendChild(dayNumberDiv);

        // Events
        const eventsDiv = document.createElement('div');
        eventsDiv.className = 'day-events';

        const dayEvents = eventsByDate[isoDate] || [];
        dayEvents.forEach(event => {
            const eventEl = this.createEventElement(event, isoDate);
            eventsDiv.appendChild(eventEl);
        });

        dayDiv.appendChild(eventsDiv);

        // Make day droppable (edit mode)
        if (this.options.enableDragDrop) {
            dayDiv.addEventListener('dragover', (e) => this.handleDragOver(e));
            dayDiv.addEventListener('drop', (e) => this.handleDrop(e));
            dayDiv.addEventListener('dragleave', (e) => this.handleDragLeave(e));
        }

        // Click handler
        if (this.options.enableClick && this.options.onDayClick) {
            dayDiv.addEventListener('click', (e) => {
                if (e.target === dayDiv || e.target === dayNumberDiv || e.target === eventsDiv) {
                    this.options.onDayClick(isoDate);
                }
            });
        }

        // Tooltip on hover
        if (!otherMonth) {
            dayDiv.addEventListener('mouseenter', (e) => {
                this.showTooltip(date, dayEvents, e.clientX, e.clientY);
            });
            dayDiv.addEventListener('mousemove', (e) => {
                this.tooltip.style.left = (e.clientX + 10) + 'px';
                this.tooltip.style.top = (e.clientY + 10) + 'px';
            });
            dayDiv.addEventListener('mouseleave', () => {
                this.hideTooltip();
            });
        }

        return dayDiv;
    }

    createEventElement(event, isoDate) {
        const eventEl = document.createElement('div');

        if (this.options.editMode) {
            // Edit mode: large pills with text
            eventEl.className = `event-pill ${event.type}`;
            eventEl.title = event.description;
            eventEl.dataset.type = event.type;
            eventEl.dataset.date = isoDate;

            // Label
            const label = document.createElement('span');
            label.className = 'event-label';
            label.textContent = this.getShortName(event.type);
            eventEl.appendChild(label);

            // Delete button
            if (this.options.showDeleteButtons && this.options.onEventDelete) {
                const deleteBtn = document.createElement('div');
                deleteBtn.className = 'delete-btn';
                deleteBtn.textContent = '×';
                deleteBtn.onclick = (e) => {
                    e.stopPropagation();
                    this.options.onEventDelete(isoDate, event.type);
                };
                eventEl.appendChild(deleteBtn);
            }

            // Drag & drop
            if (this.options.enableDragDrop) {
                eventEl.draggable = true;
                eventEl.addEventListener('dragstart', (e) => this.handleEventDragStart(e));
            }
        } else {
            // Serve mode: thin stripes
            eventEl.className = `event-stripe ${event.type}`;
            eventEl.title = event.description;
            eventEl.dataset.type = event.type;
            eventEl.dataset.date = isoDate;

            // Click handler
            if (this.options.onEventClick) {
                eventEl.style.cursor = 'pointer';
                eventEl.addEventListener('click', (e) => {
                    e.stopPropagation();
                    this.options.onEventClick(event, isoDate);
                });
            }
        }

        return eventEl;
    }

    getShortName(type) {
        const names = {
            restmuell: 'R',
            biotonne: 'B',
            papiertonne: 'P',
            gelber_sack: 'G',
            sondermuell: 'S',
            altkleider: 'A'
        };
        return names[type] || type;
    }

    // Drag & Drop handlers
    handleEventDragStart(e) {
        this.draggedEvent = {
            date: e.target.dataset.date,
            type: e.target.dataset.type
        };
        e.dataTransfer.effectAllowed = 'move';
    }

    handleDragOver(e) {
        e.preventDefault();
        e.currentTarget.classList.add('drag-over');
        e.dataTransfer.dropEffect = 'move';
    }

    handleDragLeave(e) {
        e.currentTarget.classList.remove('drag-over');
    }

    async handleDrop(e) {
        e.preventDefault();
        e.currentTarget.classList.remove('drag-over');

        if (!this.draggedEvent) return;

        const newDate = e.currentTarget.dataset.date;
        if (newDate === this.draggedEvent.date) return;

        if (this.options.onEventMove) {
            await this.options.onEventMove(
                this.draggedEvent.date,
                newDate,
                this.draggedEvent.type
            );
        }

        this.draggedEvent = null;
    }
}
