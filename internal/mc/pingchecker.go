package mc

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/Tnze/go-mc/bot"
)

// Struktura do odczytania odpowiedzi JSON z serwera
type StatusResponse struct {
	Version struct {
		Name     string `json:"name"`
		Protocol int    `json:"protocol"`
	} `json:"version"`
	Players struct {
		Max    int `json:"max"`
		Online int `json:"online"`
	} `json:"players"`
	Description json.RawMessage `json:"description"`
}

func Ping(host string, port string) (StatusResponse, time.Duration, error) {
	// 2. Rozwiązywanie SRV (ważne dla domen bez portu, np. hypixel.net)
	// Minecraft automatycznie szuka rekordu _minecraft._tcp.domena
	_, addrs, err := net.LookupSRV("minecraft", "tcp", host)
	if err == nil && len(addrs) > 0 {
		// Znaleziono rekord SRV, podmieniamy host i port
		host = addrs[0].Target
		port = fmt.Sprintf("%d", addrs[0].Port)
	}
	
	address := net.JoinHostPort(host, port)

	// 3. Właściwy Ping (pakiet bot)
	// PingAndList zwraca surowe bajty (data), opóźnienie (delay) i błąd
	_, delay, err := bot.PingAndList(address)
	if err != nil {
		return StatusResponse{}, 0, err
	}

	return StatusResponse{}, delay, nil

	// 5. Wyniki
	// fmt.Println("--- Sukces ---")
	// fmt.Printf("Realny Ping (RTT): %v\n", delay)
	// fmt.Printf("Wersja: %s (Protokół: %d)\n", status.Version.Name, status.Version.Protocol)
	// fmt.Printf("Gracze: %d / %d\n", status.Players.Online, status.Players.Max)
}
