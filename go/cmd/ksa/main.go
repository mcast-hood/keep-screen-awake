package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mcast-hood/keep-screen-awake/internal/transport"
)

func main() {
	root := &cobra.Command{
		Use:   "ksa",
		Short: "Keep Screen Awake CLI",
		Long:  "ksa communicates with the ksad daemon to control sleep prevention.",
	}

	root.AddCommand(
		statusCmd(),
		onCmd(),
		offCmd(),
		modeCmd(),
		logsCmd(),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current daemon status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return send(transport.Request{Command: transport.CmdStatus}, printStatus)
		},
	}
}

func onCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "on",
		Short: "Enable sleep prevention",
		RunE: func(cmd *cobra.Command, args []string) error {
			return send(transport.Request{Command: transport.CmdOn}, printOK)
		},
	}
}

func offCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "off",
		Short: "Disable sleep prevention",
		RunE: func(cmd *cobra.Command, args []string) error {
			return send(transport.Request{Command: transport.CmdOff}, printOK)
		},
	}
}

func modeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mode <always|toggle|schedule>",
		Short: "Set the operating mode",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return send(transport.Request{Command: transport.CmdMode, Mode: args[0]}, printOK)
		},
	}
}

func logsCmd() *cobra.Command {
	var lines int
	c := &cobra.Command{
		Use:   "logs",
		Short: "Show recent daemon log lines",
		RunE: func(cmd *cobra.Command, args []string) error {
			return send(transport.Request{Command: transport.CmdLogs, Lines: lines}, printLogs)
		},
	}
	c.Flags().IntVar(&lines, "lines", 50, "number of log lines to show")
	return c
}

// send creates a client, sends req, and calls the display function on success.
func send(req transport.Request, display func(transport.Response)) error {
	client := newClient()
	defer client.Close()

	resp, err := client.Send(req)
	if err != nil {
		return fmt.Errorf("send command: %w", err)
	}

	if !resp.OK {
		return fmt.Errorf("daemon error: %s", resp.Error)
	}

	display(resp)
	return nil
}

func printOK(_ transport.Response) {
	fmt.Println("OK")
}

func printStatus(resp transport.Response) {
	if resp.Data == nil {
		fmt.Println("(no data)")
		return
	}

	// Re-marshal data into StatusData for typed access.
	raw, _ := json.Marshal(resp.Data)
	var sd transport.StatusData
	if err := json.Unmarshal(raw, &sd); err != nil {
		fmt.Printf("Status: %s\n", string(raw))
		return
	}

	awakeStr := "no"
	if sd.AwakeActive {
		awakeStr = "yes"
	}
	displayStr := "no"
	if sd.DisplayOnly {
		displayStr = "yes"
	}

	fmt.Printf("Mode:         %s\n", sd.Mode)
	fmt.Printf("Awake active: %s\n", awakeStr)
	fmt.Printf("Display only: %s\n", displayStr)
	if len(sd.Schedule) > 0 {
		fmt.Println("Schedule:")
		for _, w := range sd.Schedule {
			fmt.Printf("  %s-%s [%s]\n", w.Start, w.End, strings.Join(w.Days, ","))
		}
	}
}

func printLogs(resp transport.Response) {
	if resp.Data == nil {
		fmt.Println("(no logs)")
		return
	}

	raw, _ := json.Marshal(resp.Data)
	var ld transport.LogsData
	if err := json.Unmarshal(raw, &ld); err != nil {
		fmt.Println(string(raw))
		return
	}
	for _, line := range ld.Lines {
		fmt.Println(line)
	}
}
