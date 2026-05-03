package attribute

import (
	"fmt"
	"os"
	"strconv"

	svcattr "github.com/XingfenD/tag_file/internal/service/attr"
	"github.com/spf13/cobra"
)

var AttrCmd = &cobra.Command{
	Use:   "attribute",
	Short: "管理标签属性",
	Long:  `管理标签的属性定义和属性值。属性归属于标签，如 "角色", "颜色" 等。`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var attrAddCmd = &cobra.Command{
	Use:   "add <标签名> <属性名>",
	Short: "为标签添加属性定义",
	Long:  `为指定标签添加一个属性定义，属性名在同标签下唯一。`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		svc := svcattr.NewService()
		attr, err := svc.AddAttribute(args[0], args[1], "")
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("属性 '%s' 已添加到标签 (ID: %d)\n", attr.AttributeName, attr.AttributeID)
	},
}

var attrRemoveCmd = &cobra.Command{
	Use:   "remove <属性ID>",
	Short: "删除属性定义",
	Long:  `删除指定 ID 的属性定义，需先删除其下所有属性值。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id, err := strconv.ParseUint(args[0], 10, 32)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误: 无效的属性ID")
			os.Exit(1)
		}
		svc := svcattr.NewService()
		if err := svc.RemoveAttribute(uint(id)); err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("属性 ID %d 已删除\n", id)
	},
}

var attrListCmd = &cobra.Command{
	Use:   "list [标签名]",
	Short: "列出属性定义",
	Long:  `列出属性定义，可选按标签名过滤。支持 --limit/--offset 分页。`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		var tagName string
		if len(args) > 0 {
			tagName = args[0]
		}
		svc := svcattr.NewService()
		list, total, err := svc.ListAttributes(tagName, offset, limit)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		if len(list) == 0 {
			fmt.Println("暂无属性定义")
			return
		}
		fmt.Printf("%-6s %-20s %-20s %s\n", "ID", "属性名", "标签", "描述")
		for _, a := range list {
			fmt.Printf("%-6d %-20s %-20s %s\n", a.AttributeID, a.AttributeName, a.Tag.TagName, a.Description)
		}
		if limit > 0 {
			fmt.Printf("--- %d/%d ---\n", len(list), total)
		}
	},
}

func init() {
	attrListCmd.Flags().Int("limit", 0, "分页条数")
	attrListCmd.Flags().Int("offset", 0, "分页偏移")
	AttrCmd.AddCommand(attrAddCmd)
	AttrCmd.AddCommand(attrRemoveCmd)
	AttrCmd.AddCommand(attrListCmd)
	AttrCmd.AddCommand(attrValueCmd)
}

var attrValueCmd = &cobra.Command{
	Use:   "value",
	Short: "管理属性值",
	Long:  `管理属性的可选值，如属性 "品种" 的值 "英短"、"布偶猫" 等。`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var attrValueAddCmd = &cobra.Command{
	Use:   "add <属性ID> <值>",
	Short: "添加属性值",
	Long:  `为指定属性添加一个可选值。`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		id, err := strconv.ParseUint(args[0], 10, 32)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误: 无效的属性ID")
			os.Exit(1)
		}
		svc := svcattr.NewService()
		val, err := svc.AddValue(uint(id), args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("属性值 '%s' 已添加 (ID: %d)\n", val.ValueName, val.ValueID)
	},
}

var attrValueRemoveCmd = &cobra.Command{
	Use:   "remove <值ID>",
	Short: "删除属性值",
	Long:  `删除指定 ID 的属性值。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id, err := strconv.ParseUint(args[0], 10, 32)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误: 无效的属性值ID")
			os.Exit(1)
		}
		svc := svcattr.NewService()
		if err := svc.RemoveValue(uint(id)); err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("属性值 ID %d 已删除\n", id)
	},
}

var attrValueListCmd = &cobra.Command{
	Use:   "list <属性ID>",
	Short: "列出属性值",
	Long:  `列出指定属性的所有可选值。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id, err := strconv.ParseUint(args[0], 10, 32)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误: 无效的属性ID")
			os.Exit(1)
		}
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		svc := svcattr.NewService()
		list, total, err := svc.ListValues(uint(id), offset, limit)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		if len(list) == 0 {
			fmt.Println("暂无属性值")
			return
		}
		fmt.Printf("%-6s %s\n", "ID", "值")
		for _, v := range list {
			fmt.Printf("%-6d %s\n", v.ValueID, v.ValueName)
		}
		if limit > 0 {
			fmt.Printf("--- %d/%d ---\n", len(list), total)
		}
	},
}

func init() {
	attrValueListCmd.Flags().Int("limit", 0, "分页条数")
	attrValueListCmd.Flags().Int("offset", 0, "分页偏移")
	attrValueCmd.AddCommand(attrValueAddCmd)
	attrValueCmd.AddCommand(attrValueRemoveCmd)
	attrValueCmd.AddCommand(attrValueListCmd)
}
