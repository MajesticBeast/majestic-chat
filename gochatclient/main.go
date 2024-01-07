package main

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/majesticbeast/gochat/gochatclient/proto"
	"github.com/muesli/reflow/wordwrap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"strings"
	"time"
)

type model struct {
	viewport    viewport.Model
	textarea    textarea.Model
	senderStyle lipgloss.Style
	username    string
	clientID    uuid.UUID
	messages    []string
	err         error
	stream      gochat.ChatService_JoinChatClient
	client      gochat.ChatServiceClient
	appState    appState
}

type appState int

const (
	username appState = iota
	chat
)

type (
	errMsg  struct{ error }
	chatMsg *gochat.Message
)

func getMessages(m model) tea.Cmd {
	return func() tea.Msg {
		incomingMsg, err := m.stream.Recv()
		if err != nil {
			log.Printf("Error receiving message: %v", err)
			return nil
		}
		chatMessage := chatMsg(incomingMsg)
		return chatMessage
	}
}

func initialModel() model {
	// Connect to server
	creds := insecure.NewCredentials()
	conn, err := grpc.Dial("localhost:3001", grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatal(err)
	}

	// Create client
	client := gochat.NewChatServiceClient(conn)
	clientId := uuid.New()
	// Create stream
	stream, err := client.JoinChat(context.Background(), &gochat.JoinRequest{ClientId: clientId.String()})
	if err != nil {
		log.Fatal(err)
	}

	// Chat entry area
	ta := textarea.New()
	ta.Placeholder = "Type a message and hit enter to send..."
	ta.Focus()
	ta.Prompt = "┃ "
	ta.CharLimit = 280
	ta.SetWidth(50)
	ta.SetHeight(3)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	// Chat display area
	vp := viewport.New(50, 5)
	vp.SetContent("Welcome to the lobby!")

	return model{
		client:      client,
		stream:      stream,
		username:    "DamnitTemp",
		clientID:    clientId,
		textarea:    ta,
		messages:    []string{},
		viewport:    vp,
		senderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		err:         nil,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, getMessages(m), tea.SetWindowTitle("MajesticBeast's GoChat Client"))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch m.appState {
	case username:
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.viewport.Width = msg.Width - 10
			m.viewport.Height = msg.Height - 10
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyCtrlC, tea.KeyEsc:
				fmt.Println(m.textarea.Value())
				return m, tea.Quit
			case tea.KeyEnter:
				m.username = m.textarea.Value()
				m.textarea.Reset()
				m.appState = chat

			}
		case errMsg:
			m.err = msg
			log.Println(m.err)
			return m, nil
		}

	case chat:
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 10
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyCtrlC, tea.KeyEsc:
				fmt.Println(m.textarea.Value())
				return m, tea.Quit
			case tea.KeyEnter:
				if m.textarea.Value() == "" {
					return m, nil
				}
				msg := &gochat.Message{
					ClientId:  m.clientID.String(),
					Username:  m.username,
					Content:   m.textarea.Value(),
					Timestamp: time.Now().Unix(),
				}
				m.textarea.Reset()

				_, err := m.client.SendMessage(context.Background(), msg)
				if err != nil {
					return m, func() tea.Msg { return errMsg{err} }
				}
			}
		case errMsg:
			m.err = msg
			log.Println(m.err)
			return m, nil

		case chatMsg:
			chatMessage := (*gochat.Message)(msg)
			unixTime := chatMessage.GetTimestamp()
			t := time.Unix(unixTime, 0)
			newMessage := fmt.Sprintf("[%v] %s: %s\n", t.UTC().Format("2006-01-02 15:04:05"), chatMessage.GetUsername(), chatMessage.GetContent())
			m.messages = append(m.messages, newMessage)
			m.viewport.SetContent(strings.Join(m.messages, ""))
			m.viewport.GotoBottom()
			return m, getMessages(m)
		}

	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m model) View() string {
	switch m.appState {
	case username:
		m.textarea.Placeholder = "Enter a username and hit enter to join..."
		return wordwrap.String(fmt.Sprintf(
			"%s\n\n%s\n",
			m.viewport.View(),
			m.textarea.View(),
		)+"\n\n", m.viewport.Width-50)

	case chat:
		return wordwrap.String(fmt.Sprintf(
			"%s\n\n%s\n",
			m.viewport.View(),
			m.textarea.View(),
		)+"\n\n", m.viewport.Width-50)
	default:
		return "something went wrong"
	}
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
