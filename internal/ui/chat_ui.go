package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-coding/free-agent/internal/messaging"
)

var (
	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#62E884")).
			Bold(true)

	aiStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6366F1"))

	systemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B"))

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3B82F6"))

	borderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#4B5563"))

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")).
			Bold(true)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FBBF24")).
				Bold(true)
)

type Message struct {
	Role      string
	Content   string
	Timestamp time.Time
}

type ConversationRound struct {
	ID        int
	UserInput string
	Summary   string
	Messages  []Message
	Timestamp time.Time
}

type ChatModel struct {
	viewport          viewport.Model
	textarea          textarea.Model
	conversationList  list.Model
	aiResponse        string
	currentRound      *ConversationRound
	allRounds         []ConversationRound
	systemStatus      string
	width             int
	height            int
	isThinking        bool
	thinkingStart     time.Time
	charCount         int
	charCountTotal    int
	contextInfo       string
	currentAgent      string
	llmName           string
	thinkingTime      float64
	inputChan         chan string
	responseChan      chan string
	errorChan         chan error
	focusList         bool
	listWidth         int
	pentestMode       bool
	projectName       string
	saveChan          chan string
	quitConfirm       bool
}

func NewChatModel(width, height int, inputChan, responseChan chan string, errorChan chan error, saveChan chan string, pentestMode bool) *ChatModel {
	listWidth := 35
	contentWidth := width - listWidth - 4

	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Focus()
	ta.SetWidth(contentWidth)
	ta.SetHeight(1)
	ta.CharLimit = 4096
	ta.ShowLineNumbers = false

	vp := viewport.New(contentWidth, height-12)
	vp.SetContent("Welcome to Free Agent Chat!\n\n")

	items := []list.Item{}
	l := list.New(items, list.NewDefaultDelegate(), listWidth, height-12)
	l.Title = "Conversations"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	return &ChatModel{
		viewport:         vp,
		textarea:         ta,
		conversationList: l,
		width:            width,
		height:           height,
		systemStatus:     "Ready",
		inputChan:        inputChan,
		responseChan:     responseChan,
		errorChan:        errorChan,
		charCountTotal:   0,
		listWidth:        listWidth,
		focusList:        false,
		pentestMode:      pentestMode,
		saveChan:         saveChan,
	}
}

func (m *ChatModel) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.listenForResponses())
}

func (m *ChatModel) listenForResponses() tea.Cmd {
	return func() tea.Msg {
		select {
		case response := <-m.responseChan:
			return ResponseMsg{Content: response}
		case err := <-m.errorChan:
			return ErrorMsg{Err: err}
		default:
			return nil
		}
	}
}

func (m *ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.focusList = !m.focusList
			if !m.focusList {
				m.textarea.Focus()
			}
			return m, nil
		case "enter":
			if !m.focusList && m.textarea.Value() != "" {
				input := m.textarea.Value()
				trimmedInput := strings.TrimSpace(input)
				
				if m.quitConfirm {
					if trimmedInput == "" {
						m.quitConfirm = false
						m.systemStatus = "Quit cancelled"
					} else {
						m.projectName = trimmedInput
						m.saveChan <- trimmedInput
						return m, tea.Quit
					}
					m.textarea.Reset()
					return m, nil
				}
				
				if trimmedInput == "/quit" {
					if m.projectName != "" {
						m.saveChan <- m.projectName
						return m, tea.Quit
					}
					m.quitConfirm = true
					m.systemStatus = "Save project name before quit? (Enter name or press Enter to cancel, /forcequit to exit without saving)"
					m.textarea.Reset()
					return m, nil
				}
				
				if strings.HasPrefix(trimmedInput, "/save ") {
					projectName := strings.TrimSpace(strings.TrimPrefix(trimmedInput, "/save "))
					if projectName != "" {
						m.projectName = projectName
						m.saveChan <- projectName
						m.systemStatus = fmt.Sprintf("Project saved as: %s", projectName)
					} else {
						m.systemStatus = "Please provide a project name: /save <name>"
					}
					m.textarea.Reset()
					return m, nil
				}
				
				if trimmedInput == "/forcequit" {
					return m, tea.Quit
				}
				
				if !m.isThinking {
					m.startNewRound(input)
					m.textarea.Reset()
					m.inputChan <- input
				}
			} else if m.focusList {
				if selected := m.conversationList.SelectedItem(); selected != nil {
					roundItem := selected.(ConversationRoundItem)
					m.selectRound(roundItem.RoundID)
				}
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		contentWidth := msg.Width - m.listWidth - 4
		m.viewport.Width = contentWidth
		m.viewport.Height = msg.Height - 12
		m.textarea.SetWidth(contentWidth)
		m.conversationList.SetSize(m.listWidth, msg.Height-12)
	case ResponseMsg:
		m.handleResponse(msg.Content)
	case ErrorMsg:
		if strings.Contains(msg.Err.Error(), "quit requested") {
			return m, tea.Quit
		}
		m.systemStatus = fmt.Sprintf("ERROR: %v", msg.Err)
		m.isThinking = false
	case StatusMsg:
		m.systemStatus = msg.Message
	case tea.MouseMsg:
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.focusList {
		m.conversationList, cmd = m.conversationList.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(append(cmds, m.listenForResponses())...)
}

func (m *ChatModel) startNewRound(input string) {
	round := ConversationRound{
		ID:        len(m.allRounds) + 1,
		UserInput: input,
		Summary:   input[:min(30, len(input))] + "...",
		Messages: []Message{
			{Role: "user", Content: input, Timestamp: time.Now()},
		},
		Timestamp: time.Now(),
	}

	m.currentRound = &round
	m.allRounds = append(m.allRounds, round)

	m.conversationList.SetItems(append(m.conversationList.Items(), ConversationRoundItem{
		RoundID: round.ID,
		Summary: round.Summary,
	}))
	m.conversationList.Select(len(m.allRounds) - 1)

	m.charCount = len(input)
	m.charCountTotal += len(input)
	m.isThinking = true
	m.thinkingStart = time.Now()
	m.systemStatus = "Sending..."

	m.updateViewport()
}

func (m *ChatModel) handleResponse(content string) {
	content = filterAdvertisement(content, m.pentestMode)
	m.aiResponse = content
	m.isThinking = false

	thinkingTime := time.Since(m.thinkingStart).Seconds()
	m.thinkingTime = thinkingTime

	responseChars := len(content)
	m.charCount = responseChars
	m.charCountTotal += responseChars

	if m.currentRound != nil {
		m.currentRound.Messages = append(m.currentRound.Messages, Message{
			Role:      "assistant",
			Content:   content,
			Timestamp: time.Now(),
		})

		m.currentRound.Summary = m.generateSummary(m.currentRound)

		items := m.conversationList.Items()
		if len(items) > 0 {
			items[len(items)-1] = ConversationRoundItem{
				RoundID: m.currentRound.ID,
				Summary: m.currentRound.Summary,
			}
			m.conversationList.SetItems(items)
		}

		m.contextInfo = m.buildContext()
	}

	m.systemStatus = fmt.Sprintf("Received: %.2fs, Chars: %d/%d", thinkingTime, responseChars, m.charCountTotal)

	m.updateViewport()
}

func (m *ChatModel) generateSummary(round *ConversationRound) string {
	if len(round.Messages) < 2 {
		return round.UserInput[:min(30, len(round.UserInput))] + "..."
	}

	response := round.Messages[len(round.Messages)-1].Content
	summary := round.UserInput[:min(20, len(round.UserInput))] + " -> "
	
	if len(response) > 40 {
		summary += response[:40] + "..."
	} else {
		summary += response
	}
	
	return summary[:min(60, len(summary))]
}

func (m *ChatModel) buildContext() string {
	var contexts []string
	for _, round := range m.allRounds {
		if round.Summary != "" {
			contexts = append(contexts, fmt.Sprintf("#%d: %s", round.ID, round.Summary[:min(20, len(round.Summary))]))
		}
	}
	
	if len(contexts) > 5 {
		contexts = contexts[len(contexts)-5:]
	}
	
	return strings.Join(contexts, "; ")
}

func (m *ChatModel) selectRound(roundID int) {
	for i := range m.allRounds {
		if m.allRounds[i].ID == roundID {
			m.currentRound = &m.allRounds[i]
			m.updateViewport()
			m.systemStatus = fmt.Sprintf("Viewing round #%d", roundID)
			return
		}
	}
}

func (m *ChatModel) updateViewport() {
	var content strings.Builder
	content.WriteString(headerStyle.Render("=== Free Agent Chat ===\n\n"))

	if m.currentRound != nil {
		for _, msg := range m.currentRound.Messages {
			role := "User"
			if msg.Role == "assistant" {
				role = "AI"
			}

			content.WriteString(fmt.Sprintf("[%s] %s:\n", msg.Timestamp.Format("15:04:05"), role))

			if msg.Role == "assistant" {
				rendered := m.renderMarkdown(msg.Content)
				content.WriteString(rendered)
			} else {
				content.WriteString(msg.Content)
			}
			content.WriteString("\n\n")
		}
	}

	if m.isThinking {
		content.WriteString(aiStyle.Render("AI is thinking..."))
	}

	m.viewport.SetContent(content.String())
	m.viewport.GotoBottom()
}

func (m *ChatModel) View() string {
	thinkingInfo := ""
	if m.isThinking {
		thinkingTime := time.Since(m.thinkingStart).Seconds()
		thinkingInfo = fmt.Sprintf(" | Thinking: %.1fs", thinkingTime)
	}

	roundInfo := fmt.Sprintf("Round: %d", len(m.allRounds))
	charsInfo := fmt.Sprintf("Chars: %d/%d", m.charCount, m.charCountTotal)
	
	systemBar := systemStyle.Render(fmt.Sprintf("LLM: %s | Agent: %s | Context: %s | %s | %s%s | %s",
		m.llmName, m.currentAgent, m.contextInfo, roundInfo, charsInfo, thinkingInfo, m.systemStatus))

	listView := borderStyle.Render(m.conversationList.View())
	contentView := lipgloss.JoinVertical(lipgloss.Left,
		borderStyle.Render(m.viewport.View()),
		lipgloss.NewStyle().Width(m.width-m.listWidth-4).Render(m.textarea.View()),
	)

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, listView, contentView)

	return lipgloss.JoinVertical(lipgloss.Left,
		mainContent,
		systemBar,
	)
}

func (m *ChatModel) renderMarkdown(content string) string {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithWordWrap(m.width-m.listWidth-20),
		glamour.WithStandardStyle("dark"),
	)
	if err != nil {
		return content
	}

	rendered, err := renderer.Render(content)
	if err != nil {
		return content
	}

	return rendered
}

func (m *ChatModel) SetContextInfo(info string) {
	m.contextInfo = info
}

func (m *ChatModel) SetCurrentAgent(agent string) {
	m.currentAgent = agent
}

func (m *ChatModel) SetLLMName(name string) {
	m.llmName = name
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func filterAdvertisement(content string, pentestMode bool) string {
	var processor *messaging.MessageProcessor
	if pentestMode {
		processor = messaging.NewMessageProcessorWithConfig(messaging.PentestConfig())
	} else {
		processor = messaging.NewMessageProcessor()
	}
	return processor.CleanMessage(content)
}

type ConversationRoundItem struct {
	RoundID int
	Summary string
}

func (i ConversationRoundItem) FilterValue() string {
	return i.Summary
}

func (i ConversationRoundItem) Title() string {
	return fmt.Sprintf("#%d: %s", i.RoundID, i.Summary)
}

func (i ConversationRoundItem) Description() string {
	return ""
}

type ResponseMsg struct {
	Content string
}

type ErrorMsg struct {
	Err error
}

type StatusMsg struct {
	Message string
}
