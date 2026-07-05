package code

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Contribution is a per-day commit count for a GitHub user.
type Contribution struct {
	ID        uint      `gorm:"primarykey" json:"-"`
	User      string    `gorm:"index:idx_user_date,unique" json:"user"`
	Date      time.Time `gorm:"index:idx_user_date,unique;type:date" json:"date"`
	Count     int       `json:"count"`
	UpdatedAt time.Time `json:"-"`
}

// Upsert inserts or updates daily counts, keyed on (user, date).
func Upsert(ctx context.Context, db *gorm.DB, rows []Contribution) error {
	if len(rows) == 0 {
		return nil
	}
	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user"}, {Name: "date"}},
		DoUpdates: clause.AssignmentColumns([]string{"count", "updated_at"}),
	}).Create(&rows).Error
}

// ForAllTime returns every stored day (as "2006-01-02") mapped to its count.
func ForAllTime(ctx context.Context, db *gorm.DB, user string) (map[string]int64, error) {
	var rows []Contribution
	if err := db.WithContext(ctx).Where(`"user" = ?`, user).Order("date asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, r := range rows {
		out[r.Date.Format("2006-01-02")] = int64(r.Count)
	}
	return out, nil
}

// ForYear returns counts for one calendar year, grouped by ISO week key.
func ForYear(ctx context.Context, db *gorm.DB, user string, year int) (map[string]int64, error) {
	from := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(1, 0, 0)
	var rows []Contribution
	if err := db.WithContext(ctx).
		Where(`"user" = ? AND date >= ? AND date < ?`, user, from, to).
		Order("date asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]int64{}
	for _, r := range rows {
		out[weekKey(r.Date)] += int64(r.Count)
	}
	return out, nil
}

func weekKey(t time.Time) string {
	y, w := t.ISOWeek()
	return fmt.Sprintf("%04d-W%02d", y, w)
}
