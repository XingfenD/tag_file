package commands

import (
	"fmt"
	"os"

	"github.com/XingfenD/tag_file/internal/service"
	"github.com/spf13/cobra"
)

var CategoryCmd = &cobra.Command{
	Use:   "category",
	Short: "管理分类项",
	Long:  `对标签分类项进行增删查操作。`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var categoryAddCmd = &cobra.Command{
	Use:   "add <名称>",
	Short: "新增分类项",
	Long:  `新增一个标签分类项，名称全局唯一。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		svc := service.NewCategoryService()
		category, err := svc.AddCategory(name, "")
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("分类项 '%s' 已创建 (ID: %d)\n", category.CategoryName, category.CategoryID)
	},
}

var categoryRemoveCmd = &cobra.Command{
	Use:   "remove <名称>",
	Short: "删除分类项",
	Long:  `删除指定名称的分类项，需要先删除该分类项下的所有标签。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		svc := service.NewCategoryService()
		if err := svc.RemoveCategory(name); err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("分类项 '%s' 已删除\n", name)
	},
}

var categoryListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有分类项",
	Long:  `列出当前数据库中的所有标签分类项。支持 --limit/--offset 分页。`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		svc := service.NewCategoryService()
		list, total, err := svc.ListCategories(offset, limit)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		if len(list) == 0 {
			fmt.Println("暂无分类项")
			return
		}
		fmt.Printf("%-4s %-20s %s\n", "ID", "名称", "描述")
		for _, c := range list {
			fmt.Printf("%-4d %-20s %s\n", c.CategoryID, c.CategoryName, c.Description)
		}
		if limit > 0 {
			fmt.Printf("--- %d/%d ---\n", len(list), total)
		}
	},
}

func init() {
	categoryListCmd.Flags().Int("limit", 0, "分页条数")
	categoryListCmd.Flags().Int("offset", 0, "分页偏移")
	CategoryCmd.AddCommand(categoryAddCmd)
	CategoryCmd.AddCommand(categoryRemoveCmd)
	CategoryCmd.AddCommand(categoryListCmd)
}
