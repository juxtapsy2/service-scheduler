package service

import (
    "context"
    "testing"
    "time"

    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
    "github.com/stretchr/testify/require"

    "juxtapsy2/service-scheduler-backend/internal/database"
    "juxtapsy2/service-scheduler-backend/internal/models"
    "juxtapsy2/service-scheduler-backend/internal/repository"
)

// NOTE: these are lightweight integration-style tests and expect a local test DB configured via POSTGRES_* env vars
func TestBook_NoTechnician(t *testing.T) {
    db, err := database.NewFromEnv()
    require.NoError(t, err)
    repo := repository.NewBookingRepository(db)
    svc := NewBookingService(repo)

    req := &models.BookingRequest{
        CustomerID:    "00000000-0000-0000-0000-000000000000",
        VehicleID:     "00000000-0000-0000-0000-000000000000",
        DealershipID:  "11111111-1111-1111-1111-111111111111",
        ServiceTypeID: "20000000-0000-0000-0000-000000000001",
        DesiredStart:  time.Date(2099, 1, 1, 3, 0, 0, 0, time.UTC),
    }

    _, err = svc.Book(context.Background(), req)
    require.Error(t, err)
}
