package main

import (
	"fmt"
	"os"

	"github.com/XingfenD/tag_file/cmd/cli/commands/attribute"
	"github.com/XingfenD/tag_file/cmd/cli/commands/category"
	"github.com/XingfenD/tag_file/cmd/cli/commands/config"
	"github.com/XingfenD/tag_file/cmd/cli/commands/file"
	"github.com/XingfenD/tag_file/cmd/cli/commands/search"
	"github.com/XingfenD/tag_file/cmd/cli/commands/tag"
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
	rootCmd.CompletionOptions.DisableDefaultCmd = false
	rootCmd.AddCommand(category.CategoryCmd)
	rootCmd.AddCommand(tag.TagCmd)
	rootCmd.AddCommand(attribute.AttrCmd)
	rootCmd.AddCommand(file.FileCmd)
	rootCmd.AddCommand(search.SearchCmd)
	rootCmd.AddCommand(config.ConfigCmd)
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
