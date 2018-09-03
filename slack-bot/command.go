package main

import (
	"github.com/shomali11/commander"
	"github.com/shomali11/proper"
)

// NewBotCommand creates a new bot command object
func NewBotCommand(usage string, description string, handler func(inputText string, parameters *proper.Properties) (outputText string)) BotCommand {
	command := commander.NewCommand(usage)
	return &botCommand{usage: usage, description: description, handler: handler, command: command}
}

// BotCommand interface
type BotCommand interface {
	Usage() string
	Description() string

	Match(text string) (*proper.Properties, bool)
	Tokenize() []*commander.Token
	Execute(inputText string, parameters *proper.Properties) (outputText string)
}

// botCommand structure contains the bot's command, description and handler
type botCommand struct {
	usage       string
	description string
	handler     func(inputText string, parameters *proper.Properties) (outputText string)
	command     *commander.Command
}

// Usage returns the command usage
func (c *botCommand) Usage() string {
	return c.usage
}

// Description returns the command description
func (c *botCommand) Description() string {
	return c.description
}

// Match determines whether the bot should respond based on the text received
func (c *botCommand) Match(text string) (*proper.Properties, bool) {
	return c.command.Match(text)
}

// Tokenize returns the command format's tokens
func (c *botCommand) Tokenize() []*commander.Token {
	return c.command.Tokenize()
}

// Execute executes the handler logic
func (c *botCommand) Execute(inputText string, parameters *proper.Properties) (outputText string) {
	return c.handler(inputText, parameters)
}
