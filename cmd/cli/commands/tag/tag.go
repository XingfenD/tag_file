package tag

import (
	"fmt"
	"os"

	"github.com/XingfenD/tag_file/internal/service"
	"github.com/spf13/cobra"
)

var TagCmd = &cobra.Command{
	Use:   "tag",
	Short: "管理标签",
	Long:  `对标签进行增删查操作，标签必须归属于某个分类项。`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var tagAddCmd = &cobra.Command{
	Use:   "add <分类项> <标签名>",
	Short: "新增标签",
	Long:  `在指定分类项下新增一个标签，标签名在同分类项下唯一。`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		categoryName := args[0]
		tagName := args[1]
		svc := service.NewTagService()
		tag, err := svc.AddTag(categoryName, tagName, "")
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("标签 '%s' 已添加到分类项 '%s' (ID: %d)\n", tag.TagName, categoryName, tag.TagID)
	},
}

var tagRemoveCmd = &cobra.Command{
	Use:   "remove <标签名>",
	Short: "删除标签",
	Long:  `删除指定名称的标签。若多个分类项下有同名标签，将全部删除。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		svc := service.NewTagService()
		count, err := svc.RemoveTag(name)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("已删除 %d 个标签 '%s'\n", count, name)
	},
}

var tagListCmd = &cobra.Command{
	Use:   "list [分类项]",
	Short: "列出标签",
	Long:  `列出标签，可选按分类项过滤。支持 --limit/--offset 分页。`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		var categoryName string
		if len(args) > 0 {
			categoryName = args[0]
		}
		svc := service.NewTagService()
		list, total, err := svc.ListTags(categoryName, offset, limit)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		if len(list) == 0 {
			fmt.Println("暂无标签")
			return
		}
		if categoryName != "" {
			fmt.Printf("%-4s %-20s %s\n", "ID", "名称", "描述")
			for _, t := range list {
				fmt.Printf("%-4d %-20s %s\n", t.TagID, t.TagName, t.Description)
			}
		} else {
			fmt.Printf("%-4s %-20s %-20s %s\n", "ID", "名称", "分类项", "描述")
			for _, t := range list {
				fmt.Printf("%-4d %-20s %-20s %s\n", t.TagID, t.TagName, t.Category.CategoryName, t.Description)
			}
		}
		if limit > 0 {
			fmt.Printf("--- %d/%d ---\n", len(list), total)
		}
	},
}

func init() {
	tagListCmd.Flags().Int("limit", 0, "分页条数")
	tagListCmd.Flags().Int("offset", 0, "分页偏移")
	TagCmd.AddCommand(tagAddCmd)
	TagCmd.AddCommand(tagRemoveCmd)
	TagCmd.AddCommand(tagListCmd)
}
