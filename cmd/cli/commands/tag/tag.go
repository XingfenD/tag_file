package tag

import (
	"fmt"
	"os"
	"strconv"

	svctag "github.com/XingfenD/tag_file/internal/service/tag"
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
	Long:  `在指定分类项下新增一个标签，标签名在同分类项下唯一。--parent 可指定父标签。`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		categoryName := args[0]
		tagName := args[1]
		parentTag, _ := cmd.Flags().GetString("parent")
		svc := svctag.NewService()
		tag, err := svc.AddTag(categoryName, tagName, "", parentTag)
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
		svc := svctag.NewService()
		count, err := svc.RemoveTag(name)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("已删除 %d 个标签 '%s'\n", count, name)
	},
}

var tagUpdateCmd = &cobra.Command{
	Use:   "update <标签ID>",
	Short: "更新标签",
	Long:  `更新标签的名称、描述或父标签。--name 新名称，--desc 新描述，--parent 新父标签名。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id, err := strconv.ParseUint(args[0], 10, 32)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误: 无效的标签ID")
			os.Exit(1)
		}
		newName, _ := cmd.Flags().GetString("name")
		newDesc, _ := cmd.Flags().GetString("desc")
		newParent, _ := cmd.Flags().GetString("parent")
		var newParentPtr *string
		if cmd.Flags().Changed("parent") {
			newParentPtr = &newParent
		}
		svc := svctag.NewService()
		tag, err := svc.UpdateTag(uint(id), newName, newDesc, newParentPtr)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("标签 '%s' (ID: %d) 已更新\n", tag.TagName, tag.TagID)
	},
}

var tagListCmd = &cobra.Command{
	Use:   "list [分类项]",
	Short: "列出标签",
	Long:  `列出标签，可选按分类项过滤。--tree 以树形展示。支持 --limit/--offset 分页。`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		tree, _ := cmd.Flags().GetBool("tree")
		var categoryName string
		if len(args) > 0 {
			categoryName = args[0]
		}
		svc := svctag.NewService()

		if tree {
			nodes, err := svc.ListTagTree(categoryName)
			if err != nil {
				fmt.Fprintln(os.Stderr, "错误:", err)
				os.Exit(1)
			}
			if len(nodes) == 0 {
				fmt.Println("暂无标签")
				return
			}
			printTagTree(nodes, "")
			return
		}

		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
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

func printTagTree(nodes []svctag.TagTreeNode, prefix string) {
	for i, n := range nodes {
		isLast := i == len(nodes)-1
		branch := "├── "
		nextPrefix := "│   "
		if isLast {
			branch = "└── "
			nextPrefix = "    "
		}
		fmt.Printf("%s%s%s\n", prefix, branch, n.Tag.TagName)
		if len(n.Children) > 0 {
			printTagTree(n.Children, prefix+nextPrefix)
		}
	}
}

func init() {
	tagAddCmd.Flags().String("parent", "", "父标签名")
	tagUpdateCmd.Flags().String("name", "", "新标签名")
	tagUpdateCmd.Flags().String("desc", "", "新描述")
	tagUpdateCmd.Flags().String("parent", "", "新父标签名")
	tagListCmd.Flags().Bool("tree", false, "树形展示")
	tagListCmd.Flags().Int("limit", 0, "分页条数")
	tagListCmd.Flags().Int("offset", 0, "分页偏移")
	TagCmd.AddCommand(tagAddCmd)
	TagCmd.AddCommand(tagUpdateCmd)
	TagCmd.AddCommand(tagRemoveCmd)
	TagCmd.AddCommand(tagListCmd)
	TagCmd.AddCommand(tagValueCmd)
}

var tagValueCmd = &cobra.Command{
	Use:   "value",
	Short: "管理标签值",
	Long:  `管理标签的预定义值列表，如 "作者" 标签的值 "张三"。`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var tagValueAddCmd = &cobra.Command{
	Use:   "add <标签名> <值>",
	Short: "添加标签值",
	Long:  `为指定标签添加一个可选值。`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		svc := svctag.NewService()
		val, err := svc.AddTagValue(args[0], args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("标签值 '%s' 已添加 (ID: %d)\n", val.Value, val.ValueID)
	},
}

var tagValueRemoveCmd = &cobra.Command{
	Use:   "remove <标签名> <值>",
	Short: "删除标签值",
	Long:  `删除指定标签的一个值。`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		svc := svctag.NewService()
		if err := svc.RemoveTagValue(args[0], args[1]); err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("标签值 '%s' 已删除\n", args[1])
	},
}

var tagValueListCmd = &cobra.Command{
	Use:   "list <标签名>",
	Short: "列出标签值",
	Long:  `列出指定标签的所有可选值。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc := svctag.NewService()
		vals, err := svc.ListTagValues(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		if len(vals) == 0 {
			fmt.Println("暂无标签值")
			return
		}
		fmt.Printf("%-4s %s\n", "ID", "值")
		for _, v := range vals {
			fmt.Printf("%-4d %s\n", v.ValueID, v.Value)
		}
	},
}

func init() {
	tagValueCmd.AddCommand(tagValueAddCmd)
	tagValueCmd.AddCommand(tagValueRemoveCmd)
	tagValueCmd.AddCommand(tagValueListCmd)
}
