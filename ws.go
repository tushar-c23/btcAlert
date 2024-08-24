package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

// Data from binance
type KlineData struct {
	E         uint64 `json:"E"` // Event time
	S         string `json:"s"` // Symbol
	K         Kline  `json:"k"` // Kline struct
	EventType string `json:"e"`
}

type Kline struct {
	StartTime int64  `json:"t"` // Kline start time
	CloseTime int64  `json:"T"` // Kline close time
	Symbol    string `json:"s"` // Symbol
	Interval  string `json:"i"` // Interval
	Close     string `json:"c"` // Close price
	IsFinal   bool   `json:"x"` // Is this the final kline of the interval
}

func parseFloat(s string) float64 {
	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Fatalf("ParseFloat error: %v", err)
	}
	return value
}

func calcEMA(prices []float64, period int) float64 {

	k := 2.0 / (float64(period) + 1.0)
	ema := prices[0]

	for i := 1; i < len(prices); i++ {
		ema = prices[i]*k + ema*(k-1)
	}

	return ema
}

func calcRSI(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}

	gains, losses := 0.0, 0.0

	// initial gains and losses
	for i := 1; i < period; i++ {
		difference := prices[i] - prices[i-1]
		if difference > 0 {
			gains += difference
		} else {
			losses -= difference
		}
	}

	averageGain := gains / float64(period)
	averageLoss := losses / float64(period)

	// Calculate the RSI
	for i := period; i < len(prices); i++ {
		difference := prices[i] - prices[i-1]
		if difference > 0 {
			averageGain = ((averageGain * float64(period-1)) + difference) / float64(period)
			averageLoss = (averageLoss * float64(period-1)) / float64(period)
		} else {
			averageGain = (averageGain * float64(period-1)) / float64(period)
			averageLoss = ((averageLoss * float64(period-1)) - difference) / float64(period)
		}
	}

	if averageLoss == 0 {
		return 100
	}

	rs := averageGain / averageLoss
	rsi := 100 - (100 / (1 + rs))

	return rsi
}

func indicatorCompute(
	// ws *websocket.Conn,
	notOnlyFinal bool,
) {
	// BTCUSDT 1m kline
	const binanceWS = "wss://fstream.binance.com/ws/btcusdt@kline_1m"

	c, _, err := websocket.DefaultDialer.Dial(binanceWS, nil)
	if err != nil {
		log.Fatal("Dial error:", err)
	}
	defer c.Close()

	var closePrices []float64

	// Listen for messages from the WebSocket
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			return
		}

		var klineData KlineData
		if err := json.Unmarshal(message, &klineData); err != nil {
			log.Println("JSON Unmarshal error:", err)
			continue
		}

		if notOnlyFinal || klineData.K.IsFinal {
			closePrice := parseFloat(klineData.K.Close)
			closePrices = append(closePrices, closePrice)

			// We need at least 26 periods to calculate the MACD
			if len(closePrices) >= 26 {
				ema12 := calcEMA(closePrices[len(closePrices)-12:], 12)
				ema26 := calcEMA(closePrices[len(closePrices)-26:], 26)

				macd := ema12 - ema26

				rsi := calcRSI(closePrices, 14)

				fmt.Printf("Time: %s\n", time.Unix(0, klineData.K.CloseTime*int64(time.Millisecond)))
				fmt.Printf("MACD: %.6f | RSI: %.6f\n", macd, rsi)

				currentRSI = rsi
				currentMACD = macd

			} else {
				fmt.Println("Data insufficient...")
			}
		} else {
			fmt.Println("Non final kline candlestick...")
		}
	}
}
