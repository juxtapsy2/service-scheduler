package models

type ServiceType struct {
    ID string `db:"id" json:"id"`
    Name string `db:"name" json:"name"`
    DurationMinutes int `db:"duration_minutes" json:"duration_minutes"`
}
