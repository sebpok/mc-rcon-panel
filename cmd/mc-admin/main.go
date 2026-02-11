package main

import (
	"flag"
	"fmt"
	"os"

	"sebpok/mc-rcon-tui/internal/rcon"
	"sebpok/mc-rcon-tui/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	addr := flag.String("addr", "localhost:25575", "RCON address")
	pass := flag.String("pass", "kopbes", "RCON password")
	flag.Parse()

	if *pass == "" {
		fmt.Println("RCON password required")
		os.Exit(1)
	}

	client, err := rcon.Connect(*addr, *pass)
	if err != nil {
		fmt.Println("RCON connection error:", err)
		os.Exit(1)
	}
	defer client.Close()

	p := tea.NewProgram(
		ui.NewModel(client, "localhost", 9),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
