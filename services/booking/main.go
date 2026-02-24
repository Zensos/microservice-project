package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v3"
	"github.com/zensos/microservice-project/internal/common"
)

// รับคำขอจอง + ตรวจข้อมูล
type CreateBookingRequest struct {
	EventID int      `json:"event_id"`
	UserID  string   `json:"user_id"`
	SeatIDs []string `json:"seat_ids"`
}

func main() {
	app := fiber.New()

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Booking Service")
	})

	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "booking"})
	})

	// Note: Dont touch this naja
	consulClient, serviceID, err := common.RegisterService(common.ServiceConfig{
		Name: "booking",
		Port: 3001,
	})
	if err != nil {
		log.Printf("couldn't register with consul: %v", err)
	}

	// TODO: replace it with real business logic
	app.Post("/bookings", func(c fiber.Ctx) error {
		if consulClient == nil {
			return c.Status(503).JSON(fiber.Map{"error": "service discovery is not available right now"})
		}

		// json
		var req CreateBookingRequest
		if err := c.Bind().Body(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid json body"})
		}

		//เช็คค่าเบสิก
		if req.EventID <= 0 {
			return c.Status(400).JSON(fiber.Map{"error": "event_id must be > 0"})
		}
		if req.UserID == "" {
			return c.Status(400).JSON(fiber.Map{"error": "user_id is required"})
		}
		if len(req.SeatIDs) == 0 {
			return c.Status(400).JSON(fiber.Map{"error": "seat_ids must not be empty"})
		}

		//เช็ค seat list ไม่ให้ว่างและซ้ำ
		seen := map[string]bool{}
		for _, seat := range req.SeatIDs {
			if seat == "" {
				return c.Status(400).JSON(fiber.Map{"error": "seat_id must not be empty"})
			}
			if seen[seat] {
				return c.Status(400).JSON(fiber.Map{"error": "seat_ids must not contain duplicates"})
			}
			seen[seat] = true
		}

		// เช็คว่าเจอ event มั้ย
		eventAddr, err := common.DiscoverService(consulClient, "event")
		if err != nil {
			return c.Status(502).JSON(fiber.Map{"error": fmt.Sprintf("couldn't find the event service: %v", err)})
		}

		// ช็คeventว่ารันอยู่มั้ยและค่อยไปเช็คstatus
		// NOTE: ตอนต่อ DB จริง ควรใช้ http.Client พร้อม timeout (กันค้าง)
		eventURL := fmt.Sprintf("http://%s/events/%d", eventAddr, req.EventID)
		eventResp, err := http.Get(eventURL)
		if err != nil {
			return c.Status(502).JSON(fiber.Map{"error": fmt.Sprintf("couldn't reach the event service: %v", err)})
		}
		defer eventResp.Body.Close()

		if eventResp.StatusCode == 404 {
			return c.Status(404).JSON(fiber.Map{"error": "event not found"})
		}
		if eventResp.StatusCode != 200 {
			return c.Status(502).JSON(fiber.Map{"error": "event service error"})
		}

		var eventData map[string]any
		eventBody, _ := io.ReadAll(eventResp.Body)
		if err := json.Unmarshal(eventBody, &eventData); err != nil {
			return c.Status(502).JSON(fiber.Map{"error": "got a bad response from the event service"})
		}

		bookingID := "book_123" //replace with real id

		return c.Status(201).JSON(fiber.Map{
			"booking_id": bookingID,
			"event":      eventData,
			"user_id":    req.UserID,
			"seat_ids":   req.SeatIDs,
			"status":     "PENDING", // อาจเป็น PENDING ก่อน แล้วค่อย CONFIRMED หลัง payment
		})

	})

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		if consulClient != nil {
			if err := common.DeregisterService(consulClient, serviceID); err != nil {
				log.Printf("couldn't deregister from consul: %v", err)
			}
		}
		if err := app.Shutdown(); err != nil {
			log.Printf("something went wrong while shutting down: %v", err)
		}
	}()

	log.Fatal(app.Listen(":3001"))
}
