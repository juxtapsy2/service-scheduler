package main

import (
    "log"

    "github.com/gin-gonic/gin"

    "juxtapsy2/service-scheduler-backend/internal/database"
    "juxtapsy2/service-scheduler-backend/internal/handler"
    "juxtapsy2/service-scheduler-backend/internal/repository"
    "juxtapsy2/service-scheduler-backend/internal/service"
)

func main() {
    db, err := database.NewFromEnv()
    if err != nil {
        log.Fatalf("db connect: %v", err)
    }

    repo := repository.NewBookingRepository(db)
    svc := service.NewBookingService(repo)
    h := handler.NewBookingHandler(svc)

    r := gin.Default()
    h.RegisterRoutes(r)

    if err := r.Run(); err != nil {
        log.Fatalf("server error: %v", err)
    }
}
