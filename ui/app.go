package ui

import (
	"raco/http"
	"raco/metrics"
	"raco/model"
	protocol2 "raco/protocol"
	"raco/storage"
	"raco/ui/func/command"
	"raco/ui/func/helper"
	"raco/ui/func/render"
	"raco/ui/func/render/modal"
	"raco/ui/notification"
	"raco/util"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type viewMode int

const (
	viewSidebar viewMode = iota
	viewPanel
	viewResponse
	viewDashboard
	viewStream
	viewCommandPalette
)

type inputField int

const (
	inputMethod inputField = iota
	inputURL
	inputHeaderKey
	inputHeaderValue
	inputBody
)

type Model struct {
	width            int
	height           int
	mode             viewMode
	collections      []*model.Collection
	currentRequest   *model.Request
	currentResponse  *model.Response
	httpClient       *http.Client
	storage          *storage.Storage
	activeEnv        *model.Environment
	selectedIndex    int
	expandedIndex    int
	headers          map[string]string
	headerKeys       []string
	selectedHeader   int
	focusedInput     inputField
	methodInput      textinput.Model
	urlInput         textinput.Model
	headerKeyInput   textinput.Model
	headerValueInput textinput.Model
	bodyInput        textarea.Model
	responseViewport viewport.Model
	notification     notification.State
	sidebarScroll    int
	collectionInput  textinput.Model
	requestNameInput textinput.Model
	showCreateCollection bool
	showSaveRequest bool
	metricsCollector *metrics.Collector
	streamClient        protocol2.StreamHandler
	streamMessages      []model.StreamMessage
	streamActive        bool
	streamInput         textinput.Model
	commandPaletteInput textinput.Model
	commandPaletteItems []string
	commandPaletteIndex int
	assertionResults    []model.AssertionResult
	history            []*model.HistoryEntry
	historyExpanded    bool
}

func NewModel(storagePath string) Model {
	methodInput := textinput.New()
	methodInput.Placeholder = "GET"
	methodInput.SetValue("GET")
	methodInput.Width = 15
	methodInput.CharLimit = 10

	urlInput := textinput.New()
	urlInput.Placeholder = "https://api.example.com/endpoint"
	urlInput.Width = 80

	headerKeyInput := textinput.New()
	headerKeyInput.Placeholder = "Content-Type"
	headerKeyInput.Width = 30

	headerValueInput := textinput.New()
	headerValueInput.Placeholder = "application/json"
	headerValueInput.Width = 40

	bodyInput := textarea.New()
	bodyInput.Placeholder = "Request body (JSON, XML, etc.)"
	bodyInput.SetWidth(80)
	bodyInput.SetHeight(6)
	responseViewport := viewport.New(80, 20)

	collectionInput := textinput.New()
	collectionInput.Placeholder = "Collection name"
	collectionInput.Width = 40

	requestNameInput := textinput.New()
	requestNameInput.Placeholder = "Request name"
	requestNameInput.Width = 40

	streamInput := textinput.New()
	streamInput.Placeholder = "Type message and press Enter to send..."
	streamInput.Width = 60

	commandPaletteInput := textinput.New()
	commandPaletteInput.Placeholder = "Search requests..."
	commandPaletteInput.Width = 60

	return Model{
		mode:             viewSidebar,
		httpClient:       http.NewClient(),
		storage:          storage.NewStorage(storagePath),
		collections:      make([]*model.Collection, 0),
		headers:          make(map[string]string),
		headerKeys:       make([]string, 0),
		selectedHeader:   -1,
		focusedInput:     inputURL,
		methodInput:      methodInput,
		urlInput:         urlInput,
		headerKeyInput:   headerKeyInput,
		headerValueInput: headerValueInput,
		bodyInput:        bodyInput,
		responseViewport: responseViewport,
		notification:     notification.New(),
		selectedIndex:    0,
		expandedIndex:    -1,
		sidebarScroll:    0,
		collectionInput:  collectionInput,
		requestNameInput: requestNameInput,
		showCreateCollection: false,
		showSaveRequest: false,
		metricsCollector: metrics.NewCollector(100),
		streamMessages:   make([]model.StreamMessage, 0),
		streamActive:     false,
		streamInput:      streamInput,
		commandPaletteInput: commandPaletteInput,
		commandPaletteItems: make([]string, 0),
		commandPaletteIndex: 0,
		assertionResults:    make([]model.AssertionResult, 0),
		history:            make([]*model.HistoryEntry, 0),
		historyExpanded:    true,
	}
}

func (m *Model) Init() tea.Cmd {
	return command.Load(m.storage)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateDimensions()
		return m, nil

	case command.CollectionsLoadedMsg:
		m.collections = msg.Collections
		if len(m.collections) > 0 {
			m.expandedIndex = 0
		}
		return m, nil

	case command.RequestExecutedMsg:
		if msg.Error != "" {
			m.metricsCollector.Record(metrics.RequestMetric{
				Timestamp:  time.Now(),
				Duration:   0,
				StatusCode: 0,
				Success:    false,
				Protocol:   "HTTP",
			})
			m.addHistoryEntry()
			return m, notification.ShowCmd("Request failed: " + msg.Error)
		}

		m.currentResponse = msg.Response
		m.assertionResults = msg.AssertionResults
		m.mode = viewResponse

		if msg.Response != nil {
			isSuccess := msg.Response.StatusCode >= 200 && msg.Response.StatusCode < 300
			m.metricsCollector.Record(metrics.RequestMetric{
				Timestamp:  msg.Response.Timestamp,
				Duration:   msg.Response.Duration,
				StatusCode: msg.Response.StatusCode,
				Success:    isSuccess,
				Protocol:   "HTTP",
			})
		}

		m.addHistoryEntry()
		return m, nil

	case command.StreamConnectedMsg:
		if msg.Success {
			m.streamActive = true
			if m.streamClient == nil {
				return m, notification.ShowCmd("Stream client is nil")
			}
			return m, tea.Batch(
				notification.ShowCmd("Connected successfully"),
				command.ListenStream(m.streamClient),
			)
		}
		m.streamActive = false
		return m, notification.ShowCmd("Connection failed: " + msg.Error)

	case command.StreamMessageReceivedMsg:
		maxMessages := 1000
		m.streamMessages = append(m.streamMessages, model.StreamMessage{
			Type:      msg.Message.Type,
			Data:      msg.Message.Data,
			Timestamp: msg.Message.Timestamp,
			Direction: msg.Message.Direction,
		})
		if len(m.streamMessages) > maxMessages {
			m.streamMessages = m.streamMessages[len(m.streamMessages)-maxMessages:]
		}
		if m.streamClient == nil {
			return m, nil
		}
		return m, command.ListenStream(m.streamClient)

	case command.StreamDisconnectedMsg:
		m.streamActive = false
		if msg.Error != "" {
			return m, notification.ShowCmd("Disconnected: " + msg.Error)
		}
		return m, notification.ShowCmd("Connection closed")

	case command.StreamClosedMsg:
		m.streamActive = false
		m.streamClient = nil
		if msg.Error != "" {
			return m, notification.ShowCmd("Close error: " + msg.Error)
		}
		return m, notification.ShowCmd("Disconnected")

	case notification.Msg:
		m.notification.Show(string(msg))
		return m, notification.HideCmd()

	case notification.HideMsg:
		m.notification.Hide()
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.MouseMsg:
		return m.handleMouseEvent(msg)
	}

	return m, nil
}

func (m *Model) View() string {
	if m.width == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("Initializing raco...")
	}

	sidebarWidth := m.width / 4
	if sidebarWidth < 30 {
		sidebarWidth = 30
	}
	mainWidth := m.width - sidebarWidth

	statusBar := render.StatusBar(m.width)
	contentHeight := m.height - 2

	sidebarView := render.Sidebar(sidebarWidth, contentHeight, m.mode == viewSidebar, m.collections, m.selectedIndex, m.expandedIndex, m.history, m.historyExpanded)
	
	var mainView string
	
	if m.mode == viewDashboard {
		stats := m.metricsCollector.GetStats()
		recent := m.metricsCollector.GetRecent(20)
		
		durations := make([]time.Duration, len(recent))
		for i, r := range recent {
			durations[i] = r.Duration
		}
		
		dashStats := render.DashboardStats{
			TotalRequests:  stats.TotalRequests,
			SuccessCount:   stats.SuccessCount,
			FailureCount:   stats.FailureCount,
			SuccessRate:    stats.SuccessRate,
			AvgDuration:    stats.AverageDuration.String(),
			MinDuration:    stats.MinDuration.String(),
			MaxDuration:    stats.MaxDuration.String(),
			Sparkline:      render.Sparkline(durations, mainWidth-20),
			SuccessRateBar: render.SuccessRateBar(stats.SuccessCount, stats.TotalRequests, mainWidth-20),
		}
		
		mainView = render.Dashboard(mainWidth, contentHeight, dashStats)
	}
	
	if m.mode == viewStream {
		protocol := "WebSocket"
		if m.streamClient != nil {
			protocol = "Stream"
		}
		mainView = render.Stream(
			mainWidth,
			contentHeight,
			convertStreamMessages(m.streamMessages),
			protocol,
			m.streamActive,
			m.streamInput,
		)
	}
	
	if m.mode == viewResponse && m.currentResponse != nil {
		responseView := render.Response(mainWidth, contentHeight, m.mode == viewResponse, m.currentResponse, &m.responseViewport)
		if len(m.assertionResults) > 0 {
			assertionView := render.Assertions(m.assertionResults, mainWidth)
			mainView = lipgloss.JoinVertical(lipgloss.Left, responseView, assertionView)
		}
		if len(m.assertionResults) == 0 {
			mainView = responseView
		}
	}

	if m.mode == viewCommandPalette {
		mainView = render.CommandPalette(mainWidth, contentHeight, m.commandPaletteInput, m.commandPaletteItems, m.commandPaletteIndex)
	}
	
	if m.mode != viewResponse && m.mode != viewDashboard && m.mode != viewStream && m.mode != viewCommandPalette {
		panelInputs := render.PanelInputs{
			MethodInput:      m.methodInput,
			URLInput:         m.urlInput,
			HeaderKeyInput:   m.headerKeyInput,
			HeaderValueInput: m.headerValueInput,
			BodyInput:        m.bodyInput,
			HeaderKeys:       m.headerKeys,
			SelectedHeader:   m.selectedHeader,
		}
		mainView = render.Panel(mainWidth, contentHeight, m.mode == viewPanel, m.headers, panelInputs)
	}

	baseView := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, mainView)
	baseView += "\n" + statusBar
	baseView += m.notification.Render(m.width)

	if m.showCreateCollection {
		baseView += modal.Collection(m.collectionInput)
	}

	if m.showSaveRequest {
		baseView += modal.Request(m.requestNameInput, m.collections, m.expandedIndex)
	}

	return baseView
}

func (m *Model) updateDimensions() {
	inputs := helper.DimensionInputs{
		URLInput:         &m.urlInput,
		BodyInput:        &m.bodyInput,
		ResponseViewport: &m.responseViewport,
	}
	helper.Dimensions(m.width, m.height, inputs)
}

func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showCreateCollection {
		return m.handleCreateCollectionInput(msg)
	}

	if m.showSaveRequest {
		return m.handleSaveRequestInput(msg)
	}

	if m.mode == viewCommandPalette {
		return m.handleCommandPaletteInput(msg)
	}

	if m.mode == viewStream && m.streamActive {
		return m.handleStreamInput(msg)
	}

	if m.mode == viewPanel && m.isInputFocused() {
		key := msg.String()
		if key == "ctrl+c" || key == "tab" || key == "esc" || key == "ctrl+r" || key == "ctrl+s" || key == "ctrl+d" {
			return m.handleGlobalKeys(msg)
		}
		if m.focusedInput == inputMethod {
			return m.handleMethodInput(msg)
		}
		return m.updateInputs(msg)
	}

	if m.mode == viewResponse {
		if msg.String() == "j" || msg.String() == "k" || msg.String() == "down" || msg.String() == "up" {
			var cmd tea.Cmd
			m.responseViewport, cmd = m.responseViewport.Update(msg)
			return m, cmd
		}
	}

	return m.handleGlobalKeys(msg)
}

func (m *Model) handleGlobalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "tab":
		return m.handleTabNavigation(), nil

	case "esc":
		m.unfocusAllInputs()
		if m.mode == viewResponse {
			m.mode = viewSidebar
		}
		return m, nil

	case "j", "down":
		return m.handleDownNavigation(), nil

	case "k", "up":
		return m.handleUpNavigation(), nil

	case "enter":
		return m.handleEnterKey(), nil

	case "ctrl+r":
		return m, m.executeCurrentRequest()

	case "ctrl+s":
		return m.handleHeaderAdd()

	case "ctrl+d":
		return m.handleHeaderDelete()

	case "ctrl+n":
		m.showCreateCollection = true
		m.collectionInput.Focus()
		return m, nil

	case "ctrl+w":
		m.showSaveRequest = true
		m.requestNameInput.Focus()
		return m, nil
	


	case "ctrl+q":
		if m.streamActive && m.streamClient != nil {
			return m, command.DisconnectStream(m.streamClient)
		}
		return m, nil

	case "ctrl+p":
		m.mode = viewCommandPalette
		m.commandPaletteInput.Focus()
		m.buildCommandPaletteItems()
		return m, nil

	case "f1":
		m.mode = viewDashboard
		return m, nil
	}

	return m, nil
}

func (m *Model) handleTabNavigation() *Model {
	if m.mode == viewSidebar {
		m.mode = viewPanel
		m.focusedInput = inputMethod
		m.unfocusAllInputs()
		m.focusInput(m.focusedInput)
		return m
	}

	if m.mode == viewPanel {
		if !m.isInputFocused() {
			m.focusedInput = inputMethod
			m.unfocusAllInputs()
			m.focusInput(m.focusedInput)
			return m
		}
		m.unfocusAllInputs()
		m.focusedInput = (m.focusedInput + 1) % 5
		m.focusInput(m.focusedInput)
		return m
	}

	if m.mode == viewResponse {
		m.mode = viewSidebar
		return m
	}

	if m.mode == viewDashboard {
		m.mode = viewSidebar
		return m
	}

	if m.mode == viewStream {
		m.mode = viewSidebar
		return m
	}

	return m
}

func (m *Model) handleDownNavigation() *Model {
	if m.mode == viewSidebar {
		totalItems := helper.TotalSidebarItems(m.collections, m.expandedIndex, m.history, m.historyExpanded)
		if m.selectedIndex < totalItems-1 {
			m.selectedIndex++
		}
		return m
	}

	if m.mode == viewPanel && !m.isInputFocused() && len(m.headerKeys) > 0 {
		if m.selectedHeader < len(m.headerKeys)-1 {
			m.selectedHeader++
		}
	}
	return m
}

func (m *Model) handleUpNavigation() *Model {
	if m.mode == viewSidebar {
		if m.selectedIndex > 0 {
			m.selectedIndex--
		}
		return m
	}

	if m.mode == viewPanel && !m.isInputFocused() && len(m.headerKeys) > 0 {
		if m.selectedHeader > 0 {
			m.selectedHeader--
		}
	}
	return m
}

func (m *Model) handleEnterKey() *Model {
	if m.mode == viewSidebar {
		m.handleSidebarSelection()
	}
	return m
}

func (m *Model) handleHeaderAdd() (*Model, tea.Cmd) {
	if m.mode != viewPanel {
		return m, nil
	}

	key := m.headerKeyInput.Value()
	value := m.headerValueInput.Value()

	if key == "" {
		return m, notification.ShowCmd("Header key required")
	}

	if _, exists := m.headers[key]; !exists {
		m.headerKeys = append(m.headerKeys, key)
	}

	m.headers[key] = value
	m.headerKeyInput.SetValue("")
	m.headerValueInput.SetValue("")
	m.selectedHeader = len(m.headerKeys) - 1
	m.focusedInput = inputHeaderKey
	m.unfocusAllInputs()
	m.focusInput(inputHeaderKey)
	return m, notification.ShowCmd("Header added: " + key)
}

func (m *Model) handleHeaderDelete() (*Model, tea.Cmd) {
	if m.mode != viewPanel {
		return m, nil
	}

	if len(m.headerKeys) == 0 {
		return m, notification.ShowCmd("No headers to delete")
	}

	if m.selectedHeader < 0 || m.selectedHeader >= len(m.headerKeys) {
		m.selectedHeader = 0
	}

	key := m.headerKeys[m.selectedHeader]
	delete(m.headers, key)
	m.headerKeys = append(m.headerKeys[:m.selectedHeader], m.headerKeys[m.selectedHeader+1:]...)

	if m.selectedHeader >= len(m.headerKeys) {
		m.selectedHeader = len(m.headerKeys) - 1
	}

	return m, notification.ShowCmd("Header deleted: " + key)
}



func (m *Model) handleSidebarSelection() {
	currentIdx := 0
	for colIdx, col := range m.collections {
		if col == nil {
			continue
		}
		if currentIdx == m.selectedIndex {
			if m.expandedIndex == colIdx {
				m.expandedIndex = -1
			}
			if m.expandedIndex != colIdx {
				m.expandedIndex = colIdx
			}
			return
		}
		currentIdx++

		if m.expandedIndex == colIdx {
			for _, req := range col.Requests {
				if req == nil {
					continue
				}
				if currentIdx == m.selectedIndex {
					m.loadRequest(req)
					m.mode = viewPanel
					return
				}
				currentIdx++
			}
		}
	}

	if currentIdx == m.selectedIndex {
		m.historyExpanded = !m.historyExpanded
		return
	}
	currentIdx++

	if m.historyExpanded {
		for i := len(m.history) - 1; i >= 0; i-- {
			entry := m.history[i]
			if entry == nil {
				continue
			}
			if currentIdx == m.selectedIndex {
				m.loadHistoryEntry(entry)
				m.mode = viewPanel
				return
			}
			currentIdx++
		}
	}
}

func (m *Model) loadRequest(req *model.Request) {
	m.currentRequest = req
	m.methodInput.SetValue(req.Method)
	m.urlInput.SetValue(req.URL)
	m.headers = make(map[string]string)
	m.headerKeys = make([]string, 0, len(req.Headers))
	for k, v := range req.Headers {
		m.headers[k] = v
		m.headerKeys = append(m.headerKeys, k)
	}
	m.selectedHeader = -1
	if len(m.headerKeys) > 0 {
		m.selectedHeader = 0
	}
	m.bodyInput.SetValue(req.Body)
}

func (m *Model) executeCurrentRequest() tea.Cmd {
	protocol := m.methodInput.Value()
	url := m.urlInput.Value()

	if url == "" {
		return notification.ShowCmd("URL required")
	}

	if protocol == "WS" {
		if m.streamClient != nil {
			m.streamClient.Close()
		}
		m.streamClient = protocol2.NewWebSocketClient(url)
		m.streamMessages = make([]model.StreamMessage, 0)
		m.mode = viewStream
		m.addHistoryEntryWithProtocol("WS")
		return command.ConnectStream(m.streamClient)
	}

	if protocol == "GRPC" {
		if m.streamClient != nil {
			m.streamClient.Close()
		}
		m.streamClient = protocol2.NewGRPCClient(url)
		m.streamMessages = make([]model.StreamMessage, 0)
		m.mode = viewStream
		m.addHistoryEntryWithProtocol("GRPC")
		return command.ConnectStream(m.streamClient)
	}

	bodyContent := m.bodyInput.Value()

	req := &model.Request{
		Method:     protocol,
		URL:        url,
		Headers:    m.headers,
		Body:       bodyContent,
		Assertions: make([]model.Assertion, 0),
		Extractors: make([]model.Extractor, 0),
	}
	
	if m.currentRequest != nil {
		req.Assertions = m.currentRequest.Assertions
		req.Extractors = m.currentRequest.Extractors
	}

	if !util.ValidateURL(req.URL) {
		return notification.ShowCmd("Invalid URL")
	}

	return command.Execute(m.httpClient, req, m.activeEnv)
}

func (m *Model) handleStreamInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == "ctrl+c" || key == "ctrl+q" {
		return m.handleGlobalKeys(msg)
	}

	if key == "esc" {
		m.streamInput.Blur()
		return m, nil
	}

	if key == "enter" {
		message := m.streamInput.Value()
		if message != "" {
			m.streamInput.SetValue("")
			return m, command.SendStreamMessage(m.streamClient, message)
		}
		return m, nil
	}

	if !m.streamInput.Focused() {
		m.streamInput.Focus()
	}

	var cmd tea.Cmd
	m.streamInput, cmd = m.streamInput.Update(msg)
	return m, cmd
}

func (m *Model) handleMethodInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "WS", "GRPC"}
	currentValue := m.methodInput.Value()
	currentIdx := 0

	for i, method := range methods {
		if method == currentValue {
			currentIdx = i
			break
		}
	}

	if key == "left" || key == "h" {
		currentIdx--
		if currentIdx < 0 {
			currentIdx = len(methods) - 1
		}
		m.methodInput.SetValue(methods[currentIdx])
		return m, nil
	}

	if key == "right" || key == "l" {
		currentIdx++
		if currentIdx >= len(methods) {
			currentIdx = 0
		}
		m.methodInput.SetValue(methods[currentIdx])
		return m, nil
	}

	return m.updateInputs(msg)
}

func (m *Model) handleMouseEvent(msg tea.MouseMsg) (*Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress {
		return m, nil
	}

	sidebarWidth := m.width / 4
	if sidebarWidth < 30 {
		sidebarWidth = 30
	}

	if msg.X < sidebarWidth {
		m.mode = viewSidebar
		m.unfocusAllInputs()
		itemIdx := msg.Y - 3
		if itemIdx >= 0 && itemIdx < helper.TotalSidebarItems(m.collections, m.expandedIndex, m.history, m.historyExpanded) {
			oldIndex := m.selectedIndex
			m.selectedIndex = itemIdx
			if oldIndex == itemIdx {
				m.handleSidebarSelection()
			}
		}
		return m, nil
	}

	if m.mode == viewResponse || m.mode == viewDashboard || m.mode == viewStream {
		return m, nil
	}

	m.mode = viewPanel
	panelX := msg.X - sidebarWidth
	panelY := msg.Y

	if panelY == 4 {
		m.unfocusAllInputs()
		m.focusedInput = inputMethod
		m.focusInput(m.focusedInput)
		return m, nil
	}

	if panelY == 7 {
		m.unfocusAllInputs()
		m.focusedInput = inputURL
		m.focusInput(m.focusedInput)
		return m, nil
	}

	headerStartY := 10
	headerEndY := headerStartY + len(m.headerKeys)
	if panelY >= headerStartY && panelY < headerEndY && len(m.headerKeys) > 0 {
		m.selectedHeader = panelY - headerStartY
		m.unfocusAllInputs()
		return m, nil
	}

	headerInputY := headerEndY + 2
	if len(m.headerKeys) == 0 {
		headerInputY = 12
	}
	if panelY == headerInputY {
		m.unfocusAllInputs()
		if panelX < 35 {
			m.focusedInput = inputHeaderKey
		}
		if panelX >= 35 {
			m.focusedInput = inputHeaderValue
		}
		m.focusInput(m.focusedInput)
		return m, nil
	}

	bodyStartY := headerInputY + 3
	if panelY >= bodyStartY && panelY < bodyStartY+8 {
		m.unfocusAllInputs()
		m.focusedInput = inputBody
		m.focusInput(m.focusedInput)
		return m, nil
	}

	return m, nil
}

func (m *Model) isInputFocused() bool {
	return m.methodInput.Focused() || m.urlInput.Focused() ||
		m.headerKeyInput.Focused() || m.headerValueInput.Focused() ||
		m.bodyInput.Focused()
}

func (m *Model) focusInput(field inputField) {
	switch field {
	case inputMethod:
		m.methodInput.Focus()
	case inputURL:
		m.urlInput.Focus()
	case inputBody:
		m.bodyInput.Focus()
	case inputHeaderKey:
		m.headerKeyInput.Focus()
	case inputHeaderValue:
		m.headerValueInput.Focus()
	}
}

func (m *Model) unfocusAllInputs() {
	m.methodInput.Blur()
	m.urlInput.Blur()
	m.bodyInput.Blur()
	m.headerKeyInput.Blur()
	m.headerValueInput.Blur()
}

func (m *Model) updateInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.focusedInput {
	case inputMethod:
		m.methodInput, cmd = m.methodInput.Update(msg)
		return m, cmd
	case inputURL:
		m.urlInput, cmd = m.urlInput.Update(msg)
		return m, cmd
	case inputBody:
		m.bodyInput, cmd = m.bodyInput.Update(msg)
		return m, cmd
	case inputHeaderKey:
		m.headerKeyInput, cmd = m.headerKeyInput.Update(msg)
		return m, cmd
	case inputHeaderValue:
		m.headerValueInput, cmd = m.headerValueInput.Update(msg)
		return m, cmd
	}

	return m, cmd
}

func (m *Model) handleCreateCollectionInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "enter" {
		name := m.collectionInput.Value()
		if name != "" {
			return m.createCollection(name)
		}
		m.showCreateCollection = false
		m.collectionInput.SetValue("")
		m.collectionInput.Blur()
		return m, nil
	}

	if msg.String() == "esc" {
		m.showCreateCollection = false
		m.collectionInput.SetValue("")
		m.collectionInput.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.collectionInput, cmd = m.collectionInput.Update(msg)
	return m, cmd
}

func (m *Model) handleSaveRequestInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "enter" {
		name := m.requestNameInput.Value()
		if name != "" {
			return m.saveRequestToCollection(name)
		}
		m.showSaveRequest = false
		m.requestNameInput.SetValue("")
		m.requestNameInput.Blur()
		return m, nil
	}

	if msg.String() == "esc" {
		m.showSaveRequest = false
		m.requestNameInput.SetValue("")
		m.requestNameInput.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.requestNameInput, cmd = m.requestNameInput.Update(msg)
	return m, cmd
}

func (m *Model) createCollection(name string) (*Model, tea.Cmd) {
	col := &model.Collection{
		ID:       util.GenerateID(),
		Name:     name,
		Requests: make([]*model.Request, 0),
	}

	if err := m.storage.SaveCollection(col); err != nil {
		m.showCreateCollection = false
		m.collectionInput.SetValue("")
		m.collectionInput.Blur()
		return m, notification.ShowCmd("Failed to create collection")
	}

	m.collections = append(m.collections, col)
	m.showCreateCollection = false
	m.collectionInput.SetValue("")
	m.collectionInput.Blur()
	return m, notification.ShowCmd("Collection created: " + name)
}

func (m *Model) saveRequestToCollection(name string) (*Model, tea.Cmd) {
	if len(m.collections) == 0 {
		m.showSaveRequest = false
		m.requestNameInput.SetValue("")
		m.requestNameInput.Blur()
		return m, notification.ShowCmd("No collection available. Create one first (Ctrl+N)")
	}

	req := &model.Request{
		ID:         util.GenerateID(),
		Name:       name,
		Method:     m.methodInput.Value(),
		URL:        m.urlInput.Value(),
		Headers:    make(map[string]string),
		Body:       m.bodyInput.Value(),
		Assertions: make([]model.Assertion, 0),
		Extractors: make([]model.Extractor, 0),
	}

	for k, v := range m.headers {
		req.Headers[k] = v
	}
	
	if m.currentRequest != nil {
		req.Assertions = m.currentRequest.Assertions
		req.Extractors = m.currentRequest.Extractors
	}

	targetColIdx := 0
	if m.expandedIndex >= 0 && m.expandedIndex < len(m.collections) {
		targetColIdx = m.expandedIndex
	}

	targetCol := m.collections[targetColIdx]
	targetCol.Requests = append(targetCol.Requests, req)

	if err := m.storage.SaveCollection(targetCol); err != nil {
		m.showSaveRequest = false
		m.requestNameInput.SetValue("")
		m.requestNameInput.Blur()
		return m, notification.ShowCmd("Failed to save request")
	}

	reloadedCol, err := m.storage.LoadCollection(targetCol.ID)
	if err == nil {
		m.collections[targetColIdx] = reloadedCol
	}

	m.showSaveRequest = false
	m.requestNameInput.SetValue("")
	m.requestNameInput.Blur()
	return m, notification.ShowCmd("Request saved to " + targetCol.Name)
}

func convertStreamMessages(msgs []model.StreamMessage) []render.StreamMessage {
	result := make([]render.StreamMessage, len(msgs))
	for i, msg := range msgs {
		result[i] = render.StreamMessage{
			Type:      msg.Type,
			Data:      msg.Data,
			Timestamp: msg.Timestamp,
			Direction: msg.Direction,
		}
	}
	return result
}

func (m *Model) buildCommandPaletteItems() {
	m.commandPaletteItems = make([]string, 0)

	for _, col := range m.collections {
		if col == nil {
			continue
		}

		for _, req := range col.Requests {
			if req == nil {
				continue
			}

			item := col.Name + " → " + req.Name + " (" + req.Method + " " + req.URL + ")"
			m.commandPaletteItems = append(m.commandPaletteItems, item)
		}
	}
}

func (m *Model) handleCommandPaletteInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.mode = viewSidebar
		m.commandPaletteInput.SetValue("")
		m.commandPaletteInput.Blur()
		m.commandPaletteIndex = 0
		return m, nil
	}

	if msg.String() == "enter" {
		if len(m.commandPaletteItems) > 0 {
			if m.commandPaletteIndex >= 0 && m.commandPaletteIndex < len(m.commandPaletteItems) {
				m.selectCommandPaletteItem(m.commandPaletteIndex)
			}
		}
		m.mode = viewPanel
		m.commandPaletteInput.SetValue("")
		m.commandPaletteInput.Blur()
		m.commandPaletteIndex = 0
		return m, nil
	}

	if msg.String() == "down" || msg.String() == "j" {
		if m.commandPaletteIndex < len(m.commandPaletteItems)-1 {
			m.commandPaletteIndex++
		}
		return m, nil
	}

	if msg.String() == "up" || msg.String() == "k" {
		if m.commandPaletteIndex > 0 {
			m.commandPaletteIndex--
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.commandPaletteInput, cmd = m.commandPaletteInput.Update(msg)

	query := m.commandPaletteInput.Value()
	if query != "" {
		allItems := make([]string, 0)
		for _, col := range m.collections {
			if col == nil {
				continue
			}
			for _, req := range col.Requests {
				if req == nil {
					continue
				}
				item := col.Name + " → " + req.Name + " (" + req.Method + " " + req.URL + ")"
				allItems = append(allItems, item)
			}
		}

		matches := util.FuzzyMatch(query, allItems)
		m.commandPaletteItems = make([]string, 0)
		for _, match := range matches {
			m.commandPaletteItems = append(m.commandPaletteItems, match.Text)
		}
		m.commandPaletteIndex = 0
	}
	if query == "" {
		m.buildCommandPaletteItems()
		m.commandPaletteIndex = 0
	}

	return m, cmd
}

func (m *Model) selectCommandPaletteItem(index int) {
	if index < 0 || index >= len(m.commandPaletteItems) {
		return
	}

	selectedIdx := 0
	for _, col := range m.collections {
		if col == nil {
			continue
		}
		for _, req := range col.Requests {
			if req == nil {
				continue
			}
			if selectedIdx == index {
				m.loadRequest(req)
				return
			}
			selectedIdx++
		}
	}
}

func (m *Model) addHistoryEntry() {
	m.addHistoryEntryWithProtocol("")
}

func (m *Model) addHistoryEntryWithProtocol(protocol string) {
	method := m.methodInput.Value()
	url := m.urlInput.Value()
	body := m.bodyInput.Value()

	headersCopy := make(map[string]string)
	for k, v := range m.headers {
		headersCopy[k] = v
	}

	proto := protocol
	if proto == "" {
		proto = "HTTP"
	}

	entry := model.NewHistoryEntry(method, url, headersCopy, body, proto)
	m.history = append(m.history, entry)

	maxHistory := 100
	if len(m.history) > maxHistory {
		m.history = m.history[len(m.history)-maxHistory:]
	}
}

func (m *Model) loadHistoryEntry(entry *model.HistoryEntry) {
	m.methodInput.SetValue(entry.Method)
	m.urlInput.SetValue(entry.URL)
	m.headers = make(map[string]string)
	m.headerKeys = make([]string, 0, len(entry.Headers))
	for k, v := range entry.Headers {
		m.headers[k] = v
		m.headerKeys = append(m.headerKeys, k)
	}
	m.selectedHeader = -1
	if len(m.headerKeys) > 0 {
		m.selectedHeader = 0
	}
	m.bodyInput.SetValue(entry.Body)
}

