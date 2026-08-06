package config

// Default working window used when technicians have no explicit schedule rows.
// Format: HH:MM:SS to match DB TIME formatting.
const (
	DefaultWorkingStart = "08:00:00"
	DefaultWorkingEnd   = "18:00:00"
)
