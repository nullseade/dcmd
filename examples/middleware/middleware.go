package main

/*
This example provides provides examples for middlwares in containers
*/

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dcmd"
	"log"
	"os"
	"sync"
)

func main() {
	system := dcmd.NewStandardSystem("[")
	system.Root.AddCommands(&StaticCmd{
		CmdNames:    []string{"Hello", "Hey"},
		Response:    "Hey there buddy",
		Description: "Greets you",
	}, &StaticCmd{
		CmdNames:    []string{"Bye", "Bai"},
		Response:    "Bye friendo!",
		Description: "Parting words",
	})

	container := system.Root.Sub("container", "c")
	container.Description = "Some extra seperated commands"
	container.AddCommands(&StaticCmd{
		CmdNames:    []string{"Hello", "Hey"},
		Response:    "Hey there buddy",
		Description: "Greets you",
	}, &StaticCmd{
		CmdNames:    []string{"Bye", "Bai"},
		Response:    "Bye friendo!",
		Description: "Parting words",
	})

	tracker := &CommandsStatTracker{
		CommandUsages: make(map[string]int),
	}

	system.Root.AddMidlewares(tracker.MiddleWare)
	system.Root.AddCommands(tracker, dcmd.NewStdHelpCommand("help"))

	session, err := discordgo.New(os.Getenv("DG_TOKEN"))
	if err != nil {
		log.Fatal("Failed setting up session:", err)
	}

	session.AddHandler(system.HandleMessageCreate)

	err = session.Open()
	if err != nil {
		log.Fatal("Failed opening gateway connection:", err)
	}
	log.Println("Running, Ctrl-c to stop.")
	select {}
}

// Same commands as used in the simple example
type StaticCmd struct {
	Response    string
	CmdNames    []string
	Description string
}

// Compilie time assertions, will not compiled unless StaticCmd implements these interfaces
var _ dcmd.Cmd = (*StaticCmd)(nil)
var _ dcmd.CmdWithDescriptions = (*StaticCmd)(nil)

func (s *StaticCmd) Names() []string { return s.CmdNames }

// Descriptions should return a short description (used in the overall help overiview) and one long descriptions for targetted help
func (s *StaticCmd) Descriptions() (string, string) { return s.Description, "" }

func (e *StaticCmd) Run(data *dcmd.Data) (interface{}, error) {
	return e.Response, nil
}

// Using this middleware, command usages in the container (and all sub containers) will be counted

type CommandsStatTracker struct {
	CommandUsages     map[string]int
	CommandUsagesLock sync.RWMutex
}

func (c *CommandsStatTracker) MiddleWare(inner dcmd.RunFunc) dcmd.RunFunc {
	return func(d *dcmd.Data) (interface{}, error) {
		// Using the container chain to generate a unique name for this command
		// The container chain is just a slice of all the containers the command is in, the first will always be the root
		name := ""
		for _, c := range d.ContainerChain {
			if len(c.Names()) < 1 || c.Names()[0] == "" {
				continue
			}
			name += c.Names()[0] + " "
		}

		// Finally append the actual command name
		name += d.Cmd.Names()[0]

		c.CommandUsagesLock.Lock()
		c.CommandUsages[name]++
		c.CommandUsagesLock.Unlock()

		return inner(d)
	}
}

// Also have the CommandStatTracker implement the Cmd interface to easily add a command that dumps the stats
func (c *CommandsStatTracker) Names() []string {
	return []string{"CommandStats", "CmdStats"}
}

func (c *CommandsStatTracker) Descriptions() (string, string) {
	return "Shows command usage stats", ""
}

// Sort and dump the stats
func (c *CommandsStatTracker) Run(d *dcmd.Data) (interface{}, error) {
	c.CommandUsagesLock.RLock()
	defer c.CommandUsagesLock.RUnlock()

	out := "```\n"

	for cmdName, usages := range c.CommandUsages {
		out += fmt.Sprintf("%15s: %d\n", cmdName, usages)
	}

	out += "```"
	return out, nil
}
