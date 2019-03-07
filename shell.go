package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/duoflow/smallshell/api"
	"github.com/duoflow/smallshell/cmd"
)

var (
	reCmd = regexp.MustCompile(`\S+`)
)

type Goshell struct {
	ctx        context.Context
	pluginsDir string
	commands   map[string]api.Command
}

func New() *Goshell {
	return &Goshell{
		pluginsDir: api.PluginsDir,
		commands:   make(map[string]api.Command),
	}
}

func (shell *Goshell) Init(ctx context.Context) error {
	shell.ctx = ctx
	shell.printSplash()
	return shell.loadCommands()
}

func (shell *Goshell) loadCommands() error {
	// load sys commands
	for name, cmd := range cmd.Commands.Registry() {
		shell.commands[name] = cmd
	}
	// write context
	shell.ctx = context.WithValue(shell.ctx, "shell.commands", shell.commands)
	return nil
}

// TODO delegate splash to a plugin
func (shell *Goshell) printSplash() {
	fmt.Println(`

      _    _    _    _    _    _    _    _  
     / \  / \  / \  / \  / \  / \  / \  / \ 
    ( B )( R )( A )( S )( _ )( M )( O )( N )
     \_/  \_/  \_/  \_/  \_/  \_/  \_/  \_/    
    
	`)
}

func (shell *Goshell) handle(ctx context.Context, cmdLine string) (context.Context, error) {
	line := strings.TrimSpace(cmdLine)
	if line == "" {
		return ctx, nil
	}
	args := reCmd.FindAllString(line, -1)
	if args != nil {
		cmdName := args[0]
		cmd, ok := shell.commands[cmdName]
		if !ok {
			return ctx, errors.New(fmt.Sprintf("command not found: %s", cmdName))
		}
		return cmd.Exec(ctx, args)
	}
	return ctx, errors.New(fmt.Sprintf("unable to parse command line: %s", line))
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx = context.WithValue(ctx, "shell.prompt", api.DefaultPrompt)
	ctx = context.WithValue(ctx, "shell.stdout", os.Stdout)
	ctx = context.WithValue(ctx, "shell.stderr", os.Stderr)
	ctx = context.WithValue(ctx, "shell.stdin", os.Stdin)

	shell := New()
	if err := shell.Init(ctx); err != nil {
		fmt.Print("\n\nFailed to initialize:", err)
		os.Exit(1)
	}

	// prompt for help
	cmdCount := len(shell.commands)
	if cmdCount > 0 {
		if _, ok := shell.commands["help"]; ok {
			fmt.Printf("\nLoaded %d command(s)...", cmdCount)
			fmt.Println("\nType help for available commands")
			fmt.Print("\n")
		}
	} else {
		fmt.Print("\n\nNo commands found")
	}

	// shell loop
	go func(shellCtx context.Context, shell *Goshell) {
		lineReader := bufio.NewReader(os.Stdin)
		loopCtx := shellCtx
		for {
			fmt.Printf("%s ", api.GetPrompt(loopCtx))
			line, err := lineReader.ReadString('\n')
			if err != nil {
				fmt.Println(err)
				return
			}
			// TODO: future enhancement is to capture input key by key
			// to give command granular notification of key events.
			// This could be used to implement command autocompletion.
			c, err := shell.handle(loopCtx, line)
			loopCtx = c
			if err != nil {
				fmt.Printf("%v\n", err)
			}
		}
	}(shell.ctx, shell)

	// wait
	// TODO: sig handling
	select {}
}
