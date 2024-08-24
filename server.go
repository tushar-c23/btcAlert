package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

type Alert struct {
	ID        int     `json:"id"`
	Value     float64 `json:"value" binding:"required"`
	Direction string  `json:"direction" binding:"required"`
	Indicator string  `json:"indicator" binding:"required"`
	Status    string  `json:"status"`
}

func main() {
	router := gin.Default()

	db, err := sql.Open("sqlite3", "./data.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

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

	router.Run(":8080")
}
