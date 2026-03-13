package editor

import "fmt"

// EditorContext provides commands with access to the editor state.
type EditorContext interface {
	ActiveBuffer() interface{ Text() string }
	ExecuteCommand(name string) error
}

// CommandFunc is a function that executes a command.
type CommandFunc func(ctx EditorContext) error

// Command represents a registered editor command.
type Command struct {
	Name        string
	Description string
	Execute     CommandFunc
}

// CommandRegistry manages named commands.
type CommandRegistry struct {
	commands map[string]*Command
}

// NewCommandRegistry creates an empty command registry.
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]*Command),
	}
}

// Register adds a command to the registry.
func (r *CommandRegistry) Register(cmd *Command) {
	r.commands[cmd.Name] = cmd
}

// Get returns a command by name, or nil if not found.
func (r *CommandRegistry) Get(name string) *Command {
	return r.commands[name]
}

// Execute runs a named command with the given context.
func (r *CommandRegistry) Execute(name string, ctx EditorContext) error {
	cmd := r.commands[name]
	if cmd == nil {
		return fmt.Errorf("unknown command: %s", name)
	}
	return cmd.Execute(ctx)
}

// List returns all registered command names.
func (r *CommandRegistry) List() []string {
	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		names = append(names, name)
	}
	return names
}

// Commands returns all registered commands.
func (r *CommandRegistry) Commands() []*Command {
	cmds := make([]*Command, 0, len(r.commands))
	for _, cmd := range r.commands {
		cmds = append(cmds, cmd)
	}
	return cmds
}
