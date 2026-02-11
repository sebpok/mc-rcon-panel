package ui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"sebpok/mc-rcon-tui/internal/mc"
	"sebpok/mc-rcon-tui/internal/rcon"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tickMsg time.Time

type Colors struct {
	// usage / in what theme
	textDark         string
	textActiveDark   string
	textDimmedDark   string
	borderDark       string
	borderActiveDark string

	textBright         string
	textActiveBright   string
	textDimmedBright   string
	borderBright       string
	borderActiveBright string

	green  string
	yellow string
	red    string
}

type PopupOptions struct {
	label string
	cmd string
	color string
}

type PlayerSnapshot struct {
	Nickname   string
	Pos        mc.Vec3
	Health     float64
	Food       int
	XPLevel    int
	XPProgress float64
	Dimension  string
	Facing     string
	HeldItem   mc.SelectedItem
}

type Popup struct {
	width  int
	height int

	shown bool

	player PlayerSnapshot

	options []PopupOptions
	activeOptionIndex int
}

type Model struct {
	rcon *rcon.Client

	host string
	port string

	prevPlayers []string
	players     []string
	playerActiveIndex int



	tabActiveIndex    int
	tabs 			[]string

	pingMs  int64
	version string
	slots   string
	motd    string

	logs    []string
	logBlockYSize int
	err     error

	input textinput.Model
	popup *Popup

	hasProperResolution bool

	refreshRate int
	refreshIn   int

	colors *Colors

	width  int
	height int
}

func NewModel(client *rcon.Client, host string, refreshRateInSeconds int) Model {
	c := &Colors{
		textDark:         "#eebefa",
		borderDark:       "#666666",
		borderActiveDark: "#da77f2",
		textDimmedDark:   "#555555",
		yellow:           "#f59f00",
		red:              "#f03e3e",
		green:            "#37b24d",
	}

	p := &Popup{
		width:   60,
		height:  5,
		shown:   false,
		options: []PopupOptions{
			{
				label: "kick",
				cmd:   "kick %s",
				color: c.textDimmedDark,
			},
			{
				label: "ban",
				cmd:   "ban %s",
				color: c.red,
			},
		},
		activeOptionIndex: 0,

		player: PlayerSnapshot{},
	}

	ti := textinput.New()
	ti.Placeholder = "Type commands here"
	ti.Prompt = "/ "
	ti.CharLimit = 200
	ti.Width = 40

	return Model{
		rcon:        client,
		logs:        make([]string, 0),
		colors:      c,
		refreshRate: refreshRateInSeconds,
		refreshIn:   0,
		host:        host,
		input:       ti,
		playerActiveIndex: 0,

		tabs: []string{"players", "cmds"},
		tabActiveIndex: 0,

		popup: p,
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Init() tea.Cmd {
	return tickCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		if m.popup.shown {
			m.FetchPlayerDetails()
		}

		m.refreshIn--
		if m.refreshIn <= 0 {
			// PLAYERS FETCH
			resp, err := m.rcon.Exec("list")
			if err != nil {
				m.err = err
				return m, tickCmd()
			}
			current := mc.ParsePlayers(resp)
			joined := mc.DiffAdded(m.prevPlayers, current)
			left := mc.DiffRemoved(m.prevPlayers, current)
			for _, p := range joined {
				m.AppendLog(p + " joined the game")
			}
			for _, p := range left {
				m.AppendLog(p + " left the game")
			}
			m.prevPlayers = current
			m.players = current

			// FETCH MC SPECIFIC REQUEST DATA
			data, ping, err := mc.Ping(m.host, "25565")
			if err != nil {
				m.err = err
				return m, tickCmd()
			}
			m.pingMs = ping.Milliseconds()
			m.version = data.Version.Name
			m.slots = fmt.Sprintf("%d/%d", data.Players.Online, data.Players.Max)

			var motd string
			err = json.Unmarshal(data.Description, &motd)
			if err != nil {
				m.err = err
				return m, tickCmd()
			}
			m.motd = motd

			m.refreshIn = m.refreshRate
		}
		return m, tickCmd()

	case tea.KeyMsg:
		if m.input.Focused() {
			m.input, cmd = m.input.Update(msg)
		}

		switch msg.String() {

		case "tab":
			m.tabActiveIndex++
			if m.tabActiveIndex >= len(m.tabs) {
				m.tabActiveIndex = 0
			}

			if m.tabs[m.tabActiveIndex] == "cmds" {
				m.input.Focus()
			} else {
				m.input.Blur()
			}

		case "up", "k":
			if m.tabActiveIndex == 0 && len(m.players) > 0 && !m.popup.shown {
				m.playerActiveIndex--
				if m.playerActiveIndex < 0 {
					m.playerActiveIndex = len(m.players) - 1
				}
			}

		case "down", "j":
			if m.tabActiveIndex == 0 && len(m.players) > 0 && !m.popup.shown {
				m.playerActiveIndex++
				if m.playerActiveIndex >= len(m.players) {
					m.playerActiveIndex = 0
				}
			}

		case "left", "h":
			if m.popup.shown {
				m.popup.activeOptionIndex--
				if m.popup.activeOptionIndex < 0 {
					m.popup.activeOptionIndex = len(m.popup.options) - 1
				}
			}

		case "right", "l":
			if m.popup.shown {
				m.popup.activeOptionIndex++
				if m.popup.activeOptionIndex >= len(m.popup.options) {
					m.popup.activeOptionIndex = 0
				}
			}

		case "enter":
			switch m.tabs[m.tabActiveIndex] {
			case "cmds":
				if m.input.Value() != "" {
					currentCmd := m.input.Value()
					// dodaj do logów
					m.AppendLog("> " + m.input.Value())
					m.input.SetValue("")

					resp, err := m.rcon.Exec(currentCmd)
					if err != nil {
						m.err = err
					}
					m.AppendLog(resp)
				}

			case "players":
				if !m.popup.shown && len(m.players) > 0 {
					m.popup.shown = true
					m.FetchPlayerDetails()
				} else {
					if len(m.players) > 0 {
						player := m.players[m.playerActiveIndex]
						option := m.popup.options[m.popup.activeOptionIndex]
						cmd := fmt.Sprintf(option.cmd, player)

						m.AppendLog("> " + cmd)

						m.popup.shown = false

						resp, err := m.rcon.Exec(cmd)
						if err != nil {
							m.err = err
						}
						m.AppendLog(resp)
					}
				}
			}
			return m, nil

		case "ctrl+l":
			m.logs = nil

		case "ctrl+c", "esc":
			if m.popup.shown {
				m.popup.shown = false
				return m, nil
			}
			return m, tea.Quit
		}
	}

	return m, cmd
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// check windows size
	if m.width < 75 || m.height < 21 {
		return lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Bold(true).
			Foreground(lipgloss.Color(m.colors.yellow)).
			AlignVertical(lipgloss.Center).
			AlignHorizontal(lipgloss.Center).
			Render("window is too small for proper rendering")
	}

	// ------------- header ------------------
	headerWidth := m.width
	titleBox := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(m.colors.textDark)).
		Width(headerWidth / 3).
		Align(lipgloss.Center).
		SetString("Minecraft RCON console")
	refreshBox := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(m.colors.textDark)).
		Width(headerWidth / 3).
		Align(lipgloss.Right).
		SetString(fmt.Sprintf("Refresh in: %d", m.refreshIn))
	versionBox := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.colors.textDimmedDark)).
		Width(headerWidth / 3).
		SetString("v0.1")
	headerBox := lipgloss.JoinHorizontal(
		lipgloss.Center,
		versionBox.Render(),
		titleBox.Render(),
		refreshBox.Render(),
	)

	// ------------- footer ------------------
	var footerBox lipgloss.Style
	if m.err != nil {
		footerBox = lipgloss.NewStyle().
			SetString(m.err.Error()).Foreground(lipgloss.Color(m.colors.red))
	} else {
		footerBox = lipgloss.NewStyle().
			SetString("[esc] Quit | [tab] Switch tabs | [ctrl+l] Clear logs | [arrows] Nav").Foreground(lipgloss.Color(m.colors.textDimmedDark))
	}

	// ------------- main content ------------------
	contentHeight := m.height - 4
	leftColumnWidth := int(float64(m.width) * 0.3)
	rightColumnWidth := m.width - leftColumnWidth - 4

	infoBoxHeight := int(float64(contentHeight) * 0.4)
	playerBoxHeight := contentHeight - infoBoxHeight - 2

	// ---------- right column ------------
	infoBox := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.colors.borderDark)).
		PaddingLeft(1).
		PaddingRight(1).
		Width(leftColumnWidth).
		Height(infoBoxHeight)

	infoItemLabel := lipgloss.NewStyle().
		Width(leftColumnWidth/2 - 2).
		Align(lipgloss.Left)
	infoItemValue := lipgloss.NewStyle().
		Bold(true).
		Width(leftColumnWidth / 2).
		Align(lipgloss.Right)
	separatorStyle := lipgloss.NewStyle().
		Width(leftColumnWidth).
		Height(1).
		Foreground(lipgloss.Color(m.colors.textDimmedDark))


	versionInfoBoxContent := lipgloss.JoinHorizontal(
		lipgloss.Left,
		infoItemLabel.Render("Ver:"),
		infoItemValue.Render(m.version),
	)

	slotsInfoBoxContent := lipgloss.JoinHorizontal(
		lipgloss.Left,
		infoItemLabel.Render("Slots:"),
		infoItemValue.Render(m.slots),
	)

	pingInfoBoxContent := lipgloss.JoinHorizontal(
		lipgloss.Left,
		infoItemLabel.Render("Ping:"),
		infoItemValue.Render(fmt.Sprintf("%d ms", m.pingMs)),
	)

	motdInfoBoxContent := lipgloss.NewStyle().
		Width(leftColumnWidth - 2).
		Align(lipgloss.Left).
		Foreground(lipgloss.Color(m.colors.textDimmedDark))

	infoBoxContent := lipgloss.JoinVertical(
		lipgloss.Top,
		versionInfoBoxContent,

		separatorStyle.Render(strings.Repeat("-", leftColumnWidth-2)),

		slotsInfoBoxContent,
		pingInfoBoxContent,
		motdInfoBoxContent.Render("MOTD: " + m.motd),
	)
	
	// players
	playerPopup := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.colors.borderDark)).
		Padding(1, 2).
		Width(m.popup.width).
		Height(m.popup.height)

	playerBox := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		PaddingLeft(1).
		PaddingRight(1).
		Width(leftColumnWidth).
		Height(playerBoxHeight)
	
	if m.tabActiveIndex == 0 {
		playerBox = playerBox.BorderForeground(lipgloss.Color(m.colors.borderActiveDark))
	} else {
		playerBox = playerBox.BorderForeground(lipgloss.Color(m.colors.borderDark))
	}
	
	playerBoxItem := lipgloss.NewStyle().
		Width(leftColumnWidth - 2).
		Bold(true).Align(lipgloss.Left)

	playersMaxLines := playerBoxHeight - 2
	playerLines := []string{}

	playerLines = append(playerLines, playerBoxItem.Render("Online:"))
	playerLines = append(playerLines, separatorStyle.Render(strings.Repeat("-", leftColumnWidth-2)))

	player_start := 0
	if len(m.players) > playersMaxLines {
		player_start = len(m.players) - playersMaxLines
	}
	for i, p := range m.players[player_start:] {
		if i == m.playerActiveIndex && m.tabActiveIndex == 0 {
			playerLines = append(playerLines, playerBoxItem.Background(lipgloss.Color(m.colors.borderDark)).Render("- " + p))
		} else {
			playerLines = append(playerLines, playerBoxItem.Render("- " + p))
		}
	}

	leftColumn := lipgloss.JoinVertical(
		lipgloss.Top,
		infoBox.Render(infoBoxContent),
		playerBox.Render(lipgloss.JoinVertical(
			lipgloss.Top,
			playerLines...,
		)),
	)

	// ---------- right column ------------
	logBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.colors.borderDark)).
		PaddingLeft(1).
		PaddingRight(1).
		Width(rightColumnWidth - 2).
		Height(contentHeight - 3)

	logBoxItem := lipgloss.NewStyle().
		Bold(true).
		Width(rightColumnWidth - 2)

	maxLines := contentHeight - 5 // border top/bottom
	if maxLines < 1 {
		maxLines = 1
	}
	lines := []string{}
	start := 0
	if len(m.logs) > maxLines {
		start = len(m.logs) - maxLines
	}
	for _, l := range m.logs[start:] {
		lines = append(lines, logBoxItem.Render(l))
	}

	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		PaddingLeft(1).
		PaddingRight(1).
		Width(rightColumnWidth - 2).Height(1)
	
	if m.tabActiveIndex == 1 {
		inputStyle = inputStyle.BorderForeground(lipgloss.Color(m.colors.borderActiveDark))
	} else {
		inputStyle = inputStyle.BorderForeground(lipgloss.Color(m.colors.borderDark))
	}
	inputView := inputStyle.Render(m.input.View())

	logBlock := lipgloss.JoinVertical(
		lipgloss.Top,
		lines...,
	)

	// detect log overflow - in future change way of displaying logs
	m.logBlockYSize = lipgloss.Height(logBlock)
	if m.logBlockYSize > contentHeight - 5 {
		return lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Bold(true).
			Foreground(lipgloss.Color(m.colors.red)).
			AlignVertical(lipgloss.Center).
			AlignHorizontal(lipgloss.Center).
			Render("Logs overflowed, please clear logs with [ctrl+l]")
	}

	rightColumn := lipgloss.JoinVertical(
		lipgloss.Top,
		logBox.Render(logBlock),
		inputView,
	)

	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftColumn,
		rightColumn,
	)

	if !m.popup.shown {
		return lipgloss.JoinVertical(
			lipgloss.Center,
			headerBox,
			body,
			footerBox.Render(),
		)
	}

	playerPopupNickname := lipgloss.NewStyle().
		Bold(true).
		Width(m.popup.width - 4).
		Foreground(lipgloss.Color(m.colors.textDark)).
		Align(lipgloss.Center).
		Render(m.players[m.playerActiveIndex])
	
	playerPopupSeparator := lipgloss.NewStyle().
		Width(m.popup.width - 4).
		Height(1).
		Foreground(lipgloss.Color(m.colors.textDimmedDark)).
		Render(strings.Repeat("-", m.popup.width - 4))

	playerPopupStatLabel := lipgloss.NewStyle().
		Width((m.popup.width - 4) / 2).
		Align(lipgloss.Left)
	playerPopupStatValue := lipgloss.NewStyle().
		Bold(true).
		Width((m.popup.width - 4) / 2).
		Align(lipgloss.Right)

	playerPopupStatsPos := lipgloss.JoinHorizontal(
		lipgloss.Left,
		playerPopupStatLabel.Render("Position XYZ:"),
		playerPopupStatValue.
			Foreground(lipgloss.Color(m.colors.textDark)).
			Render(
				fmt.Sprintf(
					"[%.1f, %.1f, %.1f]", 
					m.popup.player.Pos.X,
					m.popup.player.Pos.Y, 
					m.popup.player.Pos.Z,
				),
			),
	)

	playerPopupStatsHealth := lipgloss.JoinHorizontal(
		lipgloss.Left,
		playerPopupStatLabel.Render("Health:"),
		playerPopupStatValue.Foreground(
			lipgloss.Color(m.colors.red)).
			Render(AsciiBar(m.popup.player.Health/20, 20, "█", "░"),
		),
	)

	playerPopupStatsFood := lipgloss.JoinHorizontal(
		lipgloss.Left,
		playerPopupStatLabel.Render("Food:"),
		playerPopupStatValue.
			Foreground(lipgloss.Color("173")).
			Render(AsciiBar(float64(m.popup.player.Food)/20, 20, "█", "░"),
		),
	)

	playerPopupStatsXP := lipgloss.JoinHorizontal(
		lipgloss.Left,
		playerPopupStatLabel.Render("XP:"),
		playerPopupStatValue.
			Foreground(lipgloss.Color("114")).
			Render(
				fmt.Sprintf(
					"%dlvl + (%.0f%%)", 
					m.popup.player.XPLevel, 
					m.popup.player.XPProgress*100,
				),
			),
	)

	playerPopupStatsDimension := lipgloss.JoinHorizontal(
		lipgloss.Left,
		playerPopupStatLabel.Render("Dimension:"),
		playerPopupStatValue.Render(m.popup.player.Dimension),
	)

	playerPopupStatsSelectedItem := lipgloss.JoinHorizontal(
		lipgloss.Left,
		playerPopupStatLabel.Render("Held Item:"),
		playerPopupStatValue.Render(m.popup.player.HeldItem.ID),
	)

	playerPopupStats := lipgloss.JoinVertical(
		lipgloss.Top,
		playerPopupStatsPos,
		playerPopupStatsHealth,
		playerPopupStatsFood,
		playerPopupStatsXP,
		playerPopupStatsDimension,
		playerPopupStatsSelectedItem,
	)
	
	playerPopupOption := lipgloss.NewStyle().
		Bold(true).
		Width((m.popup.width - 4) / len(m.popup.options)).
		MarginTop(1).
		Align(lipgloss.Center)
	
	optionsLines := []string{}
	for i, o := range m.popup.options {
		if m.popup.activeOptionIndex == i {
			optionsLines = append(
				optionsLines, playerPopupOption.Background(lipgloss.Color(o.color)).Render(o.label),
			)
		} else {
			optionsLines = append(
				optionsLines, playerPopupOption.Render(o.label),
			)
		}
	}

	playerPopupOptions := lipgloss.JoinHorizontal(
		lipgloss.Center,	
		optionsLines...,
	)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		playerPopup.Render(
			lipgloss.JoinVertical(
				lipgloss.Top,
				playerPopupNickname,
				playerPopupSeparator,
				playerPopupStats,
				playerPopupOptions,
			),
		),
	)
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}

func (m *Model) AppendLog(log string) {
	now := time.Now().Format("15:04:05")
	m.logs = append(
		m.logs,
		fmt.Sprintf("[%s] %s", now, log),
	)
}

func (m *Model) FetchPlayerDetails() {
	playerName := m.players[m.playerActiveIndex]

	//check if player is still online
	resp, err := m.rcon.Exec("list")
	if err != nil {
		m.err = err
		return
	}
	var isPlayerOnline bool
	players := mc.ParsePlayers(resp)
	for _, p := range players {
		if p == playerName {
			isPlayerOnline = true
			break
		}
	}
	if !isPlayerOnline {
		m.popup.shown = false
		m.AppendLog(fmt.Sprintf("%s is no longer online", playerName))
		return
	}

	// position
	resp, err = m.rcon.Exec(fmt.Sprintf("data get entity %s Pos", playerName))
	if err != nil {
		m.err = err
		return
	}
	pos, err := mc.ParsePosition(resp)
	if err != nil {
		m.err = err
		return
	}
	m.popup.player.Pos = pos

	// health
	resp, err = m.rcon.Exec(fmt.Sprintf("data get entity %s Health", playerName))
	if err != nil {
		m.err = err
		return
	}
	health, err := mc.ParseHealth(resp)
	if err != nil {
		m.err = err
		return
	}
	m.popup.player.Health = health
	
	// food level
	resp, err = m.rcon.Exec(fmt.Sprintf("data get entity %s foodLevel", playerName))
	if err != nil {
		m.err = err
		return
	}
	food, err := mc.ParseFoodLevel(resp)
	if err != nil {
		m.err = err
		return
	}
	m.popup.player.Food = food
	
	// xp level
	resp, err = m.rcon.Exec(fmt.Sprintf("data get entity %s XpLevel", playerName))
	if err != nil {
		m.err = err
		return
	}
	xpLevel, err := mc.ParseXPLevel(resp)
	if err != nil {
		m.err = err
		return
	}
	m.popup.player.XPLevel = xpLevel
	
	// xp progress
	resp, err = m.rcon.Exec(fmt.Sprintf("data get entity %s XpP", playerName))
	if err != nil {
		m.err = err
		return
	}
	xpProgress, err := mc.ParseXPProgress(resp)
	if err != nil {
		m.err = err
		return
	}
	m.popup.player.XPProgress = xpProgress
	
	// dimension
	resp, err = m.rcon.Exec(fmt.Sprintf("data get entity %s Dimension", playerName))
	if err != nil {
		m.err = err
		return
	}
	dimension, err := mc.ParseDimension(resp)
	if err != nil {
		m.err = err
		return
	}
	m.popup.player.Dimension = dimension

	// held item
	resp, err = m.rcon.Exec(fmt.Sprintf("data get entity %s SelectedItem", playerName))
	if err != nil {
		m.err = err
		return
	}
	selectedItem, err := mc.ParseSelectedItem(resp)
	if err != nil {
		m.err = err
		return
	}
	m.popup.player.HeldItem = selectedItem
}
