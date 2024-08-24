package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

var (
	db          *sql.DB
	upgrader    = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	currentRSI  float64
	currentMACD float64
)

type Alert struct {
	ID        int     `json:"id"`
	Value     float64 `json:"value" binding:"required"`
	Direction string  `json:"direction" binding:"required"`
	Indicator string  `json:"indicator" binding:"required"`
	Status    string  `json:"status"`
}

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./data.db")
	if err != nil {
		log.Fatal(err)
	}

	statement, err := db.Prepare(`
		CREATE TABLE IF NOT EXISTS alerts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			value REAL,
			direction TEXT,
			indicator TEXT,
			status TEXT DEFAULT 'pending'
		)`)
	if err != nil {
		log.Fatal(err)
	}
	statement.Exec()
}

func fetchAlerts() ([]Alert, error) {
	rows, err := db.Query("SELECT * FROM alerts")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var alert Alert
		if err := rows.Scan(&alert.ID, &alert.Value, &alert.Direction, &alert.Indicator, &alert.Status); err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

func updateAlertStatus(alertID int, status string) error {
	statement, err := db.Prepare("UPDATE alerts SET status = ? WHERE id = ?")
	if err != nil {
		return err
	}
	defer statement.Close()

	_, err = statement.Exec(status, alertID)
	return err
}

func alertWebSocket(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer ws.Close()

	for {
		alerts, err := fetchAlerts()
		if err != nil {
			log.Println("Fetch alerts error:", err)
			return
		}

		for _, alert := range alerts {
			triggered := false

			if alert.Status == "pending" {

				switch alert.Indicator {
				case "RSI":
					fmt.Printf("currentRSI: %.6f | alertRSI: %.6f\n", currentRSI, alert.Value)
					if (alert.Direction == "up" && currentRSI > alert.Value) || (alert.Direction == "down" && currentRSI < alert.Value) {
						triggered = true
					}
				case "MACD":
					fmt.Printf("currentMACD: %.6f | alertMACD: %.6f\n", currentMACD, alert.Value)
					if (alert.Direction == "up" && currentMACD > alert.Value) || (alert.Direction == "down" && currentMACD < alert.Value) {
						triggered = true
					}
				}
			}

			if triggered {
				alert.Status = "completed"
				err := updateAlertStatus(alert.ID, alert.Status)
				if err != nil {
					log.Println("Update alert status error:", err)
				}
			}
		}

		if err := ws.WriteJSON(alerts); err != nil {
			log.Println("Write error:", err)
			break
		}

		//delay to control the WebSocket message frequency
		select {
		case <-c.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}

func main() {
	router := gin.Default()

	initDB()

	go indicatorCompute(true)

	router.POST("/alert", func(c *gin.Context) {
		var input Alert

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		statement, err := db.Prepare("INSERT INTO alerts (value, direction, indicator) VALUES (?, ?, ?)")
		if err != nil {
			log.Fatal(err)
		}

		_, err = statement.Exec(input.Value, input.Direction, input.Indicator)
		if err != nil {
			log.Fatal(err)
		}

		c.JSON(http.StatusOK, gin.H{"message": "Alert created"})
	})

	router.GET("/alerts", func(c *gin.Context) {
		rows, err := db.Query("SELECT * FROM alerts")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		var alerts []Alert
		for rows.Next() {
			var alert Alert
			err := rows.Scan(&alert.ID, &alert.Value, &alert.Direction, &alert.Indicator, &alert.Status)
			if err != nil {
				log.Fatal(err)
			}
			alerts = append(alerts, alert)
		}

		c.JSON(http.StatusOK, alerts)
	})

	router.GET("/ws", alertWebSocket)

	router.Run(":8080")
}
