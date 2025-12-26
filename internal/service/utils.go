package service

import "time"

func maxDate(a, b time.Time) time.Time {
	// выбираем познию дату
	if a.After(b) {
		return a
	}
	return b
}

func minDate(a, b time.Time) time.Time {
	// берем ту что раньше
	if a.Before(b) {
		return a
	}
	return b
}

func countMonths(start, end time.Time) int {
	if start.After(end) {
		return 0
	}

	years := end.Year() - start.Year()
	months := int(end.Month()) - int(start.Month())

	// инклюзивно считаем месяцы, +1 чтоб текущий тоже зашел
	return years*12 + months + 1
}
