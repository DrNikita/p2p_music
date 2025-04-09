package model

import (
	"fmt"
	"p2p-music/internal/song"

	tea "github.com/charmbracelet/bubbletea"
)

type SongList struct {
	sngs     []song.Song
	cursor   int
	selected map[int]struct{}
}

func (sl SongList) Init() tea.Cmd {
	return nil
}

func (sl SongList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Is it a key press?
	case tea.KeyMsg:

		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return sl, tea.Quit

		// The "up" and "k" keys move the cursor up
		case "up", "k":
			if sl.cursor > 0 {
				sl.cursor--
			}

		// The "down" and "j" keys move the cursor down
		case "down", "j":
			if sl.cursor < len(sl.sngs)-1 {
				sl.cursor++
			}

		// The "enter" key and the spacebar (a literal space) toggle
		// the selected state for the item that the cursor is pointing at.
		case "enter", " ":
			_, ok := sl.selected[sl.cursor]
			if ok {
				delete(sl.selected, sl.cursor)
			} else {
				sl.selected[sl.cursor] = struct{}{}
			}
		}
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return sl, nil
}

func (sl SongList) View() string {
	// The header
	s := "What should we buy at the market?\n\n"

	// Iterate over our choices
	for i, choice := range sl.sngs {

		// Is the cursor pointing at this choice?
		cursor := " " // no cursor
		if sl.cursor == i {
			cursor = ">" // cursor!
		}

		// Is this choice selected?
		checked := " " // not selected
		if _, ok := sl.selected[i]; ok {
			checked = "x" // selected!
		}

		// Render the row
		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
	}

	// The footer
	s += "\nPress q to quit.\n"

	// Send the UI for rendering
	return s
}
