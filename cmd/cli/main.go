package main

import (
	"fmt"
	"os"

	"github.com/XingfenD/tag_file/cmd/cli/commands"
	"github.com/XingfenD/tag_file/internal/bootstrap"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tagfile",
	Short: "A command-line tool for tagging and searching files",
	Long: `tagfile is a powerful command-line utility that allows you to tag files
with custom labels and search for files based on those tags. It helps you
organize your files more efficiently without changing their physical location.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func initCmd() {
	rootCmd.AddCommand(commands.CategoryCmd)
	rootCmd.AddCommand(commands.TagCmd)
	rootCmd.AddCommand(commands.FileCmd)
	rootCmd.AddCommand(commands.SearchCmd)
}

func main() {
	if err := bootstrap.InitDB(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	initCmd()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
