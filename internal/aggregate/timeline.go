package aggregate

import (
	"time"

	"github.com/tdawn0-0/git-analyzer/internal/model"
)

// Timeline builds daily buckets of Change Intensity and added/deleted lines.
// Days with no activity between the first and last change are filled with zeros.
// If there are no changes, the result is empty.
func Timeline(changes []model.ChangeUnit) []model.DailyBucket {
	if len(changes) == 0 {
		return nil
	}

	byDay := map[string]*model.DailyBucket{}
	var minDay, maxDay time.Time
	first := true

	for _, c := range changes {
		if c.Timestamp.IsZero() {
			continue
		}
		day := truncateUTC(c.Timestamp)
		key := day.Format("2006-01-02")
		b, ok := byDay[key]
		if !ok {
			b = &model.DailyBucket{Date: day}
			byDay[key] = b
		}
		b.TotalScore += c.Score.Final
		b.AddedLines += c.AddedLines
		b.DeletedLines += c.DeletedLines
		b.ChangeCount++

		if first || day.Before(minDay) {
			minDay = day
		}
		if first || day.After(maxDay) {
			maxDay = day
		}
		first = false
	}
	if first {
		return nil
	}

	var out []model.DailyBucket
	for d := minDay; !d.After(maxDay); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		if b, ok := byDay[key]; ok {
			out = append(out, *b)
		} else {
			out = append(out, model.DailyBucket{Date: d})
		}
	}
	return out
}

// DeveloperTimeline builds daily buckets for a single developer name.
func DeveloperTimeline(changes []model.ChangeUnit, developer string) []model.DailyBucket {
	var filtered []model.ChangeUnit
	for _, c := range changes {
		if c.Author.Name == developer {
			filtered = append(filtered, c)
		}
	}
	return Timeline(filtered)
}

func truncateUTC(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}
