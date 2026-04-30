package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var SearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search files by tags",
	Long: `Search command allows you to find files based on their tags.

Usage examples:
  tagfile search work
  tagfile search --and work important
  tagfile search --or work personal
  tagfile search --not draft`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var (
	andFlag bool
	orFlag  bool
	notFlag bool
)

var searchByTagCmd = &cobra.Command{
	Use:   "tag <tags...>",
	Short: "Search files by tags",
	Long:  `Search for files that match the specified tags.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		tags := args
		fmt.Printf("Searching files with tags: %v\n", tags)
		if andFlag {
			fmt.Println("Using AND logic")
		} else if orFlag {
			fmt.Println("Using OR logic")
		}
		if notFlag {
			fmt.Println("Excluding files with specified tags")
		}
	},
}

func init() {
	searchByTagCmd.Flags().BoolVar(&andFlag, "and", false, "Match files that have ALL specified tags")
	searchByTagCmd.Flags().BoolVar(&orFlag, "or", false, "Match files that have ANY specified tag")
	searchByTagCmd.Flags().BoolVar(&notFlag, "not", false, "Exclude files with specified tags")

	SearchCmd.AddCommand(searchByTagCmd)
}
