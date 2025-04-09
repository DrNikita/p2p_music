package model

import (
	"fmt"
	"p2p-music/internal/song"

	tea "github.com/charmbracelet/bubbletea"
)

type Domain interface{}

type Tea struct {
	choices  []string
	cursor   int
	selected map[int]struct{}
}

func InitTea() Tea {
	return Tea{
		choices:  StartMenueChoice,
		selected: make(map[int]struct{}),
	}
}

func (t Tea) Init() tea.Cmd {
	return nil
}

func (t Tea) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Is it a key press?
	case tea.KeyMsg:

		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return t, tea.Quit

		// The "up" and "k" keys move the cursor up
		case "up", "k":
			if t.cursor > 0 {
				t.cursor--
			}

		// The "down" and "j" keys move the cursor down
		case "down", "j":
			if t.cursor < len(t.choices)-1 {
				t.cursor++
			}

		// The "enter" key and the spacebar (a literal space) toggle
		// the selected state for the item that the cursor is pointing at.
		case "enter", " ":
			// _, ok := t.selected[t.cursor]
			// if ok {
			// 	delete(t.selected, t.cursor)
			// } else {
			// 	t.selected[t.cursor] = struct{}{}
			// }

			switch t.choices[t.cursor] {
			case "Songs list":
				return SongList{
					sngs: []song.Song{
						song.Song{
							Title: "SONG",
						}, song.Song{
							Title: "SONG0",
						}, song.Song{
							Title: "SONG1",
						},
					},
				}, nil
			}

		}
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return t, nil
}

func (t Tea) View() string {
	// The header
	s := "What should we buy at the market?\n\n"

	// Iterate over our choices
	for i, choice := range t.choices {

		// Is the cursor pointing at this choice?
		cursor := " " // no cursor
		if t.cursor == i {
			cursor = ">" // cursor!
		}

		// Is this choice selected?
		checked := " " // not selected
		if _, ok := t.selected[i]; ok {
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
