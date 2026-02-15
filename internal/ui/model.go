package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"sebpok/mc-rcon-tui/internal/mc"
	"sebpok/mc-rcon-tui/internal/rcon"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/muesli/reflow/wordwrap"
)

type tickMsg time.Time

type Styles struct {
	borderStyle 	 lipgloss.Border

	borderColor       lipgloss.Color
	borderColorActive lipgloss.Color
	textDark          lipgloss.Color
	textDimmedDark    lipgloss.Color

	separator lipgloss.Style

	box lipgloss.Style

	inputField     lipgloss.Style
	title          lipgloss.Style
	refreshInfo	   lipgloss.Style
	programVersion lipgloss.Style

	playersTitle lipgloss.Style
	playerLabel lipgloss.Style
	playerLabelSelected lipgloss.Style
}

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
	cmd   string
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

	options           []PopupOptions
	activeOptionIndex int
}

type playerItem string
func (p playerItem) Title() string       { return string(p) }
func (p playerItem) Description() string { return "" }
func (p playerItem) FilterValue() string { return string(p) }

type customDelegate struct{
	playerInactiveStyle lipgloss.Style
	playerActiveStyle lipgloss.Style
	labelWidth int
}
func (d customDelegate) Height() int { return 1 }  // only one line
func (d customDelegate) Spacing() int { return 0 }
func (d customDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
    return nil
}
func (d customDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
    i := listItem.(playerItem)

	var s string
    if index == m.Index() {
        s = d.playerActiveStyle.Width(d.labelWidth).Render(i.Title())
    } else {
        s = d.playerInactiveStyle.Width(d.labelWidth).Render(i.Title())
    }

    fmt.Fprint(w, "- " + s)
}

type Model struct {
	rcon *rcon.Client

	host string
	port string

	players           list.Model
	playerActiveIndex int

	tabActiveIndex int
	tabs           []string

	pingMs  int64
	version string
	slots   string
	motd    string

	err error

	input    textinput.Model
	popup    *Popup
	viewport viewport.Model

	logs []string

	hasProperResolution bool

	refreshRate int
	refreshIn   int

	colors *Colors
	styles Styles

	width  int
	height int

	contentHeight    int
	rightColumnWidth int
	leftColumnWidth  int

	ready bool
}

func DefaultStyles() Styles {
	s := Styles{
		borderStyle: 	   lipgloss.RoundedBorder(),

		borderColor:       lipgloss.Color("#666666"),
		borderColorActive: lipgloss.Color("#da77f2"),
		textDark:          lipgloss.Color("#eebefa"),
		textDimmedDark:    lipgloss.Color("#555555"),
	}

	s.separator = lipgloss.NewStyle().
		Height(1).
		Foreground(lipgloss.Color(s.textDimmedDark))

	s.inputField = lipgloss.NewStyle().
		Border(s.borderStyle).
		PaddingLeft(1).
		PaddingRight(1).
		BorderForeground(s.borderColor).
		Height(1)

	s.box = lipgloss.NewStyle().
		BorderStyle(s.borderStyle).
		BorderForeground(s.borderColor).
		PaddingLeft(1).
		PaddingRight(1)
	
	s.title = lipgloss.NewStyle().
		Bold(true).
		Foreground(s.textDark).
		Align(lipgloss.Center)
	
	s.refreshInfo = lipgloss.NewStyle().
		Foreground(lipgloss.Color(s.textDimmedDark)).
		Align(lipgloss.Right)

	s.programVersion = lipgloss.NewStyle().
		Foreground(s.textDimmedDark)

	s.playersTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(s.textDark).
		Padding(0, 0).
		Margin(0, 0, 0, 0)

	s.playerLabel = lipgloss.NewStyle().
		Bold(true)
	
	s.playerLabelSelected = lipgloss.NewStyle().
		Bold(true).
		Background(lipgloss.Color(s.textDimmedDark))

	return s
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
		width:  60,
		height: 5,
		shown:  false,
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
		rcon:              client,
		colors:            c,
		refreshRate:       refreshRateInSeconds,
		refreshIn:         refreshRateInSeconds,
		host:              host,
		input:             ti,
		playerActiveIndex: 0,
		styles:            DefaultStyles(),

		tabs:           []string{"players", "cmds"},
		tabActiveIndex: 0,

		popup: p,
	}
}

type initMsg struct{}

func initCmd() tea.Cmd {
	return func() tea.Msg {
		return initMsg{}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Init() tea.Cmd {
	return initCmd()
}

func (m *Model) FetchData() {
	// PLAYERS FETCH
	resp, err := m.rcon.Exec("list")
	if err != nil {
		m.err = err
	}

	players := mc.ParsePlayers(resp)
	playersForList := make([]list.Item, len(players))
	for i, p := range players {
		playersForList[i] = playerItem(p)
	}
	m.players.SetItems(playersForList)

	// ------------ FETCH MC SPECIFIC REQUEST DATA ------------
	data, ping, err := mc.Ping(m.host, "25565")
	if err != nil {
		m.err = err
	}
	m.pingMs = ping.Milliseconds()
	m.version = data.Version.Name
	m.slots = fmt.Sprintf("%d/%d", data.Players.Online, data.Players.Max)

	var motd string
	err = json.Unmarshal(data.Description, &motd)
	if err != nil {
		m.err = err
	}
	m.motd = motd
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {

	case initMsg:
		return m, tickCmd()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.leftColumnWidth = m.width/3 - 2
		m.rightColumnWidth = m.width - m.leftColumnWidth - 2
		m.contentHeight = m.height - 4

		logBoxWidth := m.rightColumnWidth
		logBoxHeight := m.contentHeight - 1

		frameWidth, frameHeight := m.styles.box.GetFrameSize()
		viewportWidth := logBoxWidth - frameWidth
		viewportHeight := logBoxHeight - frameHeight

		playersWidth := m.leftColumnWidth - frameWidth
		playersHeight := m.contentHeight - 1 - int(float64(m.contentHeight) * 0.4) - frameHeight

		if !m.ready {
			m.viewport = viewport.New(viewportWidth, viewportHeight)

			l := list.New([]list.Item{}, customDelegate{playerInactiveStyle: m.styles.playerLabel, playerActiveStyle: m.styles.playerLabelSelected, labelWidth: playersWidth}, playersWidth, playersHeight)
			l.SetShowStatusBar(false)
			l.SetShowHelp(false)
			l.SetFilteringEnabled(false)
			l.SetShowTitle(false)
			m.players = l

			m.FetchData()
			m.ready = true
		} else {
			// logs
			m.viewport.Width = viewportWidth
			m.viewport.Height = viewportHeight
			content := strings.Join(m.logs, "\n")
			content = wordwrap.String(content, m.viewport.Width)
			m.viewport.SetContent(content)
			m.viewport.GotoBottom()

			// players list
			m.players.SetWidth(playersWidth)
			m.players.SetHeight(playersHeight)
		}

		return m, nil

	case tickMsg:
		if m.popup.shown {
			m.FetchPlayerDetails()
		}

		if m.refreshIn <= 0 {
			m.FetchData()
			m.refreshIn = m.refreshRate
		} else {
			m.refreshIn--
		}
		return m, tickCmd()

	case tea.KeyMsg:
		if m.input.Focused() {
			m.input, cmd = m.input.Update(msg)
		}

		switch msg.String() {

		case "tab":
			if !m.popup.shown {
				m.tabActiveIndex++
				if m.tabActiveIndex >= len(m.tabs) {
					m.tabActiveIndex = 0
				}

				if m.tabs[m.tabActiveIndex] == "cmds" {
					m.input.Focus()
				} else {
					m.input.Blur()
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
				if !m.popup.shown && len(m.players.Items()) > 0 {
					m.popup.shown = true
					m.FetchPlayerDetails()
				} else {
					if len(m.players.Items()) > 0 {
						player := m.players.SelectedItem()
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
			m.viewport.SetContent("")

		case "ctrl+c", "esc":
			if m.popup.shown {
				m.popup.shown = false
				return m, nil
			}
			return m, tea.Quit
		}
	}

	if m.tabs[m.tabActiveIndex] == "players" {
		m.players, cmd = m.players.Update(msg)
	}
	return m, cmd
}

func (m Model) View() string {
	if !m.ready {
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
	titleBox := m.styles.title.Width(m.width / 3)
	programVersionBox := m.styles.programVersion.Width(m.width / 3)
	refreshBox := m.styles.refreshInfo.Width(m.width / 3)
	headerBox := lipgloss.JoinHorizontal(
		lipgloss.Center,
		programVersionBox.Render("v1.0"),
		titleBox.Render("Minecraft RCON Console"),
		refreshBox.Render(fmt.Sprintf("Refresh in: %d", m.refreshIn)),
	)

	// ------------- footer ------------------
	var footerBox lipgloss.Style
	if m.err != nil {
		footerBox = lipgloss.NewStyle().
			SetString(m.err.Error()).Foreground(lipgloss.Color(m.colors.red))
	} else {
		footerBox = lipgloss.NewStyle().
			SetString("[esc] Quit | [tab] Switch tabs | [ctrl+l] Clear logs").Foreground(lipgloss.Color(m.colors.textDimmedDark))
	}

	// ------------- main content ------------------
	infoBoxHeight := int(float64(m.contentHeight) * 0.4)
	playerBoxHeight := m.contentHeight - infoBoxHeight - 2

	// ---------- left column ------------
	infoBox := m.styles.box.Width(m.leftColumnWidth).Height(infoBoxHeight)

	infoItemLabel := lipgloss.NewStyle().
		Width(m.leftColumnWidth/2 - 2).
		Align(lipgloss.Left)
	infoItemValue := lipgloss.NewStyle().
		Bold(true).
		Width(m.leftColumnWidth / 2).
		Align(lipgloss.Right)

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

	var pingColor string
	if m.pingMs < 30 {
		pingColor = m.colors.green
	} else {
		pingColor = m.colors.yellow
	}
	pingInfoBoxContent := lipgloss.JoinHorizontal(
		lipgloss.Left,
		infoItemLabel.Render("Ping:"),
		infoItemValue.Foreground(lipgloss.Color(pingColor)).Render(fmt.Sprintf("%d ms", m.pingMs)),
	)

	motdInfoBoxContent := lipgloss.NewStyle().
		Width(m.leftColumnWidth - 2).
		Align(lipgloss.Left).
		Foreground(lipgloss.Color(m.colors.textDimmedDark))

	infoBoxContent := lipgloss.JoinVertical(
		lipgloss.Top,
		versionInfoBoxContent,

		m.styles.separator.Render(strings.Repeat("-", m.leftColumnWidth-2)),

		slotsInfoBoxContent,
		pingInfoBoxContent,
		motdInfoBoxContent.Render("MOTD: "+m.motd),
	)

	// players
	playerPopup := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.colors.borderDark)).
		Padding(1, 2).
		Width(m.popup.width).
		Height(m.popup.height)

	var playerBoxBorder lipgloss.Color
	if m.tabs[m.tabActiveIndex] == "players" {
		playerBoxBorder = m.styles.borderColorActive
	} else {
		playerBoxBorder = m.styles.borderColor
	}

	playerBox := m.styles.box.
		Width(m.leftColumnWidth).
		Height(playerBoxHeight).
		BorderForeground(playerBoxBorder)

	leftColumn := lipgloss.JoinVertical(
		lipgloss.Top,
		infoBox.Render(infoBoxContent),
		playerBox.Render("Online:\n" + m.players.View()),
	)

	// ---------- input ------------
	var inputStyle lipgloss.Style
	if m.tabs[m.tabActiveIndex] == "cmds" {
		inputStyle = m.styles.inputField.
			Width(m.rightColumnWidth - 2).
			BorderForeground(lipgloss.Color(m.styles.borderColorActive))
	} else {
		inputStyle = m.styles.inputField.Width(m.rightColumnWidth - 2)
	}
	inputView := inputStyle.Render(m.input.View())

	// ---------- logs  ------------

	// ---------- right column assembly  ------------
	rightColumn := lipgloss.JoinVertical(
		lipgloss.Top,
		m.styles.box.
			Width(m.rightColumnWidth - 2).
			Height(infoBoxHeight - 6).
			Render(
				m.viewport.View(),
			),
		inputView,
	)

	// ---------- body with all elements ------------
	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftColumn,
		rightColumn,
	)

	// ---------- render body if popup is hidden ------------
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
		Render(m.popup.player.Nickname)

	playerPopupSeparator := lipgloss.NewStyle().
		Width(m.popup.width - 4).
		Height(1).
		Foreground(lipgloss.Color(m.colors.textDimmedDark)).
		Render(strings.Repeat("-", m.popup.width-4))

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
			Render(AsciiBar(m.popup.player.Health/20, 20, "█", "░")),
	)

	playerPopupStatsFood := lipgloss.JoinHorizontal(
		lipgloss.Left,
		playerPopupStatLabel.Render("Food:"),
		playerPopupStatValue.
			Foreground(lipgloss.Color("173")).
			Render(AsciiBar(float64(m.popup.player.Food)/20, 20, "█", "░")),
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
	log = mc.RemoveColorCodes(log)

	m.logs = append(
		m.logs,
		fmt.Sprintf("[%s] %s", now, log),
	)

	content := strings.Join(m.logs, "\n")

	content = wordwrap.String(content, m.viewport.Width)
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

func (m *Model) FetchPlayerDetails() {
	selected := m.players.SelectedItem()
	playerName := string(selected.(playerItem))
	m.popup.player.Nickname = playerName

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
