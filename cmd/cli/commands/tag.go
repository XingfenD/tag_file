package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var TagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Tag a file with custom labels",
	Long: `Tag command allows you to add, remove, or list tags for a file.

Usage examples:
  tagfile tag add /path/to/file.txt work important
  tagfile tag remove /path/to/file.txt work
  tagfile tag list /path/to/file.txt`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var tagAddCmd = &cobra.Command{
	Use:   "add <file> <tags...>",
	Short: "Add tags to a file",
	Long:  `Add one or more tags to a specified file.`,
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		file := args[0]
		tags := args[1:]
		fmt.Printf("Adding tags %v to file: %s\n", tags, file)
	},
}

var tagRemoveCmd = &cobra.Command{
	Use:   "remove <file> <tags...>",
	Short: "Remove tags from a file",
	Long:  `Remove one or more tags from a specified file.`,
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		file := args[0]
		tags := args[1:]
		fmt.Printf("Removing tags %v from file: %s\n", tags, file)
	},
}

var tagListCmd = &cobra.Command{
	Use:   "list <file>",
	Short: "List tags for a file",
	Long:  `List all tags associated with a specified file.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		file := args[0]
		fmt.Printf("Listing tags for file: %s\n", file)
	},
}

func init() {
	TagCmd.AddCommand(tagAddCmd)
	TagCmd.AddCommand(tagRemoveCmd)
	TagCmd.AddCommand(tagListCmd)
}
