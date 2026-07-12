package search

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/XingfenD/tag_file/cmd/cli/commands/util"
	svcsearch "github.com/XingfenD/tag_file/internal/service/search"
	"github.com/spf13/cobra"
)

var SearchCmd = &cobra.Command{
	Use:   "search",
	Short: "按标签搜索文件",
	Long: `根据标签搜索已注册的文件。

搜索模式:
  默认 (OR):  文件拥有任一指定标签即匹配
  --and:      文件必须拥有所有指定标签
  --not:      排除拥有指定标签的文件
  --include + --not:  包含某些标签同时排除另一些标签`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var (
	searchAndFlag bool
	searchOrFlag  bool
	searchNotTags []string
)

var searchByTagCmd = &cobra.Command{
	Use:   "tag <标签名...>",
	Short: "按标签搜索文件",
	Long:  `根据标签名搜索文件，支持 AND/OR/NOT 逻辑。`,
	Args:  cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		include := args
		exclude := searchNotTags

		if len(include) == 0 && len(exclude) == 0 {
			fmt.Fprintln(os.Stderr, "请指定要搜索的标签")
			os.Exit(1)
		}

		logic := "or"
		if searchAndFlag {
			logic = "and"
		} else if searchOrFlag {
			logic = "or"
		}

		svc := svcsearch.NewService()
		results, err := svc.SearchByTags(include, exclude, logic)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}

		if len(results) == 0 {
			fmt.Println("未找到匹配的文件")
			return
		}

		condParts := []string{}
		if len(include) > 0 {
			condParts = append(condParts, fmt.Sprintf("包含: %s (%s)", strings.Join(include, ", "), logic))
		}
		if len(exclude) > 0 {
			condParts = append(condParts, fmt.Sprintf("排除: %s", strings.Join(exclude, ", ")))
		}
		fmt.Printf("搜索条件: %s\n", strings.Join(condParts, ", "))

		for _, r := range results {
			fmt.Printf("\n%s  (%s)\n", r.File.FilePath, util.FormatSize(r.File.FileSize))
			for _, t := range r.Tags {
				fmt.Printf("  [%s] %s\n", t.Category.CategoryName, t.TagName)
			}
		}
	},
}

var searchQueryCmd = &cobra.Command{
	Use:   "query [file]",
	Short: "使用结构化查询搜索文件",
	Long: `使用结构化 JSON 查询搜索文件。支持文件名、标签、标签值、文件大小、文件类型、修改日期等条件。

查询格式为 JSON，可从文件读取或通过 stdin 传入。
支持 AND/OR 逻辑和嵌套条件组。`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var data []byte
		var err error

		if len(args) > 0 {
			data, err = os.ReadFile(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "读取查询文件失败: %v\n", err)
				os.Exit(1)
			}
		} else {
			data, err = io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "读取 stdin 失败: %v\n", err)
				os.Exit(1)
			}
		}

		if len(strings.TrimSpace(string(data))) == 0 {
			fmt.Fprintln(os.Stderr, "查询内容为空")
			os.Exit(1)
		}

		var query svcsearch.SearchQuery
		if err := json.Unmarshal(data, &query); err != nil {
			fmt.Fprintf(os.Stderr, "无效的查询 JSON: %v\n", err)
			os.Exit(1)
		}

		svc := svcsearch.NewService()
		results, err := svc.SearchByQuery(query)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}

		if len(results) == 0 {
			fmt.Println("未找到匹配的文件")
			return
		}

		fmt.Printf("找到 %d 个文件:\n", len(results))
		for _, r := range results {
			fmt.Printf("\n%s  (%s)\n", r.File.FilePath, util.FormatSize(r.File.FileSize))
			for _, t := range r.Tags {
				fmt.Printf("  [%s] %s\n", t.Category.CategoryName, t.TagName)
			}
		}
	},
}

func init() {
	searchByTagCmd.Flags().BoolVar(&searchAndFlag, "and", false, "文件必须拥有所有指定标签 (AND)")
	searchByTagCmd.Flags().BoolVar(&searchOrFlag, "or", false, "文件拥有任一指定标签 (OR，默认)")
	searchByTagCmd.Flags().StringSliceVar(&searchNotTags, "not", nil, "排除拥有指定标签的文件")

	SearchCmd.AddCommand(searchByTagCmd)
	SearchCmd.AddCommand(searchQueryCmd)
}
