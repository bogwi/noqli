package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/peterh/liner"
)

// CommandHistory manages command history with namespace support
type CommandHistory struct {
	// Map of namespaces to command histories
	// Namespace is in format "db" or "db:table"
	histories map[string][]string
	// Current namespace
	currentNamespace string
	// Maximum history entries per namespace
	maxHistoryEntries int
	// History file path
	historyFile string
}

// NewCommandHistory creates a new command history manager
func NewCommandHistory(maxEntries int) *CommandHistory {
	// Create history directory if it doesn't exist
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Warning: Could not determine home directory for history file:", err)
		homeDir = "."
	}

	historyDir := filepath.Join(homeDir, ".noqli")
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		fmt.Println("Warning: Could not create history directory:", err)
	}

	return &CommandHistory{
		histories:         make(map[string][]string),
		maxHistoryEntries: maxEntries,
		historyFile:       filepath.Join(historyDir, "history.txt"),
	}
}

// UpdateNamespace updates the current namespace based on db and table
func (h *CommandHistory) UpdateNamespace(db, table string) {
	if db == "" {
		h.currentNamespace = "global"
	} else if table == "" {
		h.currentNamespace = db
	} else {
		h.currentNamespace = fmt.Sprintf("%s:%s", db, table)
	}
}

// AddHistory adds a command to the current namespace's history
func (h *CommandHistory) AddHistory(cmd string) {
	// Don't add empty commands or duplicates at the end
	if cmd == "" {
		return
	}

	// Get current namespace history
	history := h.histories[h.currentNamespace]

	// Skip if this command is a duplicate of the last one
	if len(history) > 0 && history[len(history)-1] == cmd {
		return
	}

	// Add command to history
	history = append(history, cmd)

	// Trim history to max entries
	if len(history) > h.maxHistoryEntries {
		history = history[len(history)-h.maxHistoryEntries:]
	}

	// Update the map
	h.histories[h.currentNamespace] = history
}

// GetHistory returns the current namespace's history
func (h *CommandHistory) GetHistory() []string {
	return h.histories[h.currentNamespace]
}

// LoadHistory loads command history from the history file
func (h *CommandHistory) LoadHistory() {
	file, err := os.Open(h.historyFile)
	if err != nil {
		// It's okay if the file doesn't exist yet
		return
	}
	defer file.Close()

	// Create a liner for reading the history file
	line := liner.NewLiner()
	defer line.Close()

	line.ReadHistory(file)

	// Extract namespaced history entries from liner's flat history
	// liner.State doesn't provide direct access to history, so we'll manually read each line
	// and parse it
	var history []string

	// Create a temporary file to store the history
	tempFile, err := os.CreateTemp("", "noqli-history-")
	if err == nil {
		defer os.Remove(tempFile.Name())
		defer tempFile.Close()

		// Write history to temp file
		line.WriteHistory(tempFile)
		tempFile.Seek(0, 0)

		// Read history from temp file
		data, err := os.ReadFile(tempFile.Name())
		if err == nil {
			history = strings.Split(string(data), "\n")
		}
	}

	// Process each history entry
	for _, cmd := range history {
		if cmd == "" {
			continue
		}

		// Format is "namespace::command"
		parts := strings.SplitN(cmd, "::", 2)
		if len(parts) != 2 {
			continue
		}

		namespace := parts[0]
		command := parts[1]

		h.histories[namespace] = append(h.histories[namespace], command)
	}
}

// SaveHistory saves command history to the history file
func (h *CommandHistory) SaveHistory() {
	file, err := os.Create(h.historyFile)
	if err != nil {
		fmt.Println("Error saving history:", err)
		return
	}
	defer file.Close()

	// Create a liner for writing the history file
	line := liner.NewLiner()
	defer line.Close()

	// Flatten namespaced history into a single history
	// with namespace prefixes
	for namespace, commands := range h.histories {
		for _, cmd := range commands {
			// Format is "namespace::command"
			line.AppendHistory(fmt.Sprintf("%s::%s", namespace, cmd))
		}
	}

	line.WriteHistory(file)
}

// SetupLiner configures a liner instance with the command history
func (h *CommandHistory) SetupLiner() *liner.State {
	line := liner.NewLiner()

	// Enable tab completion for common commands
	line.SetCompleter(func(line string) (c []string) {
		commands := []string{"USE", "CREATE", "GET", "UPDATE", "DELETE", "EXIT"}

		for _, cmd := range commands {
			if strings.HasPrefix(strings.ToUpper(cmd), strings.ToUpper(line)) {
				c = append(c, cmd)
			}
		}
		return
	})

	// Configure history
	line.SetCtrlCAborts(true)

	// Add history to liner
	for _, cmd := range h.GetHistory() {
		line.AppendHistory(cmd)
	}

	return line
}
