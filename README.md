## Overview
Basic server that connects with the Binance web socket to fetch kline data of BTCUSDT and calculates MACD and RSI values for the data.

## Technical Details
### Routes
- GET /alerts : returns a list of all alerts in the database.
- POST /alert : Creates an alert. 
    ```JSON
    {
    "value":25.0,
    "direction":"up",
    "indicator":"RSI"
    }
    ```
- WEBSOCKET /ws : Websocket that streams the alerts with alert status.

## How to
### Run server
- Clone the repository.
    ```bash
    git clone https://github.com/tushar-c23/btcAlert
    ```
- Run the server
    ```bash
    cd btcAlert
    go run .
    ```