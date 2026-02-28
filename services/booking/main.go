package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/zensos/microservice-project/internal/common"
)

// TODO(DB): replace in-memory seat reservation with DB transaction:
// 1) insert bookings
// 2) insert booking_seats (UNIQUE(event_id, seat_id) to prevent oversell)
// 3) on unique violation -> return 409 and rollback

// รับคำขอจอง + ตรวจข้อมูล
type CreateBookingRequest struct {
	EventID  int      `json:"event_id"`
	MemberID string   `json:"member_id"`
	SeatIDs  []string `json:"seat_ids"`
}

type Ticket struct {
	TicketID  string `json:"ticket_id"`
	BookingID string `json:"booking_id"`
	EventID   int    `json:"event_id"`
	MemberID  string `json:"member_id"`
	SeatID    string `json:"seat_id"`
}

// เก็บ seat ที่ถูกจอง, ตั๋ว
var (
	seatMu        = sync.Mutex{}
	reservedSeats = map[string]bool{} // key = fmt.Sprintf("%d:%s", eventID, seatID)

	ticketMu    = sync.Mutex{}
	userTickets = map[string][]Ticket{} // member_id -> tickets[]
)

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
		if req.MemberID == "" {
			return c.Status(400).JSON(fiber.Map{"error": "member_id is required"})
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

		//ใส่ timeout ให้ event/member call (กันค้าง)
		client := &http.Client{Timeout: 3 * time.Second}
		// ช็คeventว่ารันอยู่มั้ยและค่อยไปเช็คstatus
		// NOTE: ตอนต่อ DB จริง ควรใช้ http.Client พร้อม timeout (กันค้าง)
		eventURL := fmt.Sprintf("http://%s/events/%d", eventAddr, req.EventID)
		eventResp, err := client.Get(eventURL)
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

		log.Println("checking member service...")

		memberAddr, err := common.DiscoverService(consulClient, "member")
		if err != nil {
			return c.Status(502).JSON(fiber.Map{"error": fmt.Sprintf("couldn't find the member service: %v", err)})
		}

		memberURL := fmt.Sprintf("http://%s/members/%s", memberAddr, req.MemberID)
		memberResp, err := client.Get(memberURL)
		if err != nil {
			return c.Status(502).JSON(fiber.Map{"error": fmt.Sprintf("couldn't reach the member service: %v", err)})
		}
		defer memberResp.Body.Close()

		if memberResp.StatusCode == 404 {
			return c.Status(404).JSON(fiber.Map{"error": "user not found"})
		}
		if memberResp.StatusCode != 200 {
			return c.Status(502).JSON(fiber.Map{"error": "member service error"})
		}

		//ส่วนเช็คสถานะทีนั่ง
		seatMu.Lock()

		// check availability
		for _, seat := range req.SeatIDs {
			key := fmt.Sprintf("%d:%s", req.EventID, seat)
			if reservedSeats[key] {
				seatMu.Unlock()
				return c.Status(409).JSON(fiber.Map{"error": fmt.Sprintf("seat %s is not available", seat)})
			}
		}
		// reserve
		for _, seat := range req.SeatIDs {
			key := fmt.Sprintf("%d:%s", req.EventID, seat)
			reservedSeats[key] = true
		}

		seatMu.Unlock()

		// TODO(DB): generate booking_id (UUID) and persist to DB
		bookingID := fmt.Sprintf("book_%d", time.Now().UnixNano())

		base := time.Now().UnixNano()
		//สร้างตั๋วเก็บตั๋ว
		ticketMu.Lock()
		for i, seat := range req.SeatIDs {
			t := Ticket{
				TicketID:  fmt.Sprintf("t_%d_%d", base, i),
				BookingID: bookingID,
				EventID:   req.EventID,
				MemberID:  req.MemberID,
				SeatID:    seat,
			}
			userTickets[req.MemberID] = append(userTickets[req.MemberID], t)
		}
		ticketMu.Unlock()

		return c.Status(201).JSON(fiber.Map{
			"booking_id": bookingID,
			"event":      eventData,
			"member_id":  req.MemberID,
			"seat_ids":   req.SeatIDs,
			"status":     "PENDING", // อาจเป็น PENDING ก่อน แล้วค่อย CONFIRMED หลัง payment
		})

	})

	app.Get("/users/:member_id/tickets", func(c fiber.Ctx) error {
		MemberID := c.Params("member_id")
		if MemberID == "" {
			return c.Status(400).JSON(fiber.Map{"error": "member_id is required"})
		}

		ticketMu.Lock()
		tickets, ok := userTickets[MemberID]
		ticketMu.Unlock()

		if !ok {
			tickets = []Ticket{}
		}

		return c.JSON(fiber.Map{
			"member_id": MemberID,
			"tickets":   tickets,
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
