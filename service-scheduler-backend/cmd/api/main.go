package main

import (
    "log"
    "os"
    "time"

    "github.com/gin-contrib/cors"
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
    bookingHandler := handler.NewBookingHandler(svc)
    serviceTypesHandler := handler.NewServiceTypesHandler(repo)

    r := gin.Default()

    // Configure CORS (allow frontend dev server by default)
    frontendOrigin := os.Getenv("FRONTEND_ORIGIN")
    if frontendOrigin == "" {
        frontendOrigin = "http://localhost:5173"
    }
    r.Use(cors.New(cors.Config{
        AllowOrigins:     []string{frontendOrigin},
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
        ExposeHeaders:    []string{"Content-Length"},
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    }))

    bookingHandler.RegisterRoutes(r)
    serviceTypesHandler.RegisterRoutes(r)
    quickHandler := handler.NewQuickBookingHandler(repo, svc)
    quickHandler.RegisterRoutes(r)

    if err := r.Run(); err != nil {
        log.Fatalf("server error: %v", err)
    }
}
