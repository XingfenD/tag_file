package commands

import (
	"fmt"
	"os"
	"strconv"

	"github.com/XingfenD/tag_file/internal/service"
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
		svc := service.NewAttrService()
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
		svc := service.NewAttrService()
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
	Long:  `列出属性定义，可选按标签名过滤。`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var tagName string
		if len(args) > 0 {
			tagName = args[0]
		}
		svc := service.NewAttrService()
		attrs, err := svc.ListAttributes(tagName)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		if len(attrs) == 0 {
			fmt.Println("暂无属性定义")
			return
		}
		fmt.Printf("%-6s %-20s %-20s %s\n", "ID", "属性名", "标签", "描述")
		for _, a := range attrs {
			fmt.Printf("%-6d %-20s %-20s %s\n", a.AttributeID, a.AttributeName, a.Tag.TagName, a.Description)
		}
	},
}

func init() {
	AttrCmd.AddCommand(attrAddCmd)
	AttrCmd.AddCommand(attrRemoveCmd)
	AttrCmd.AddCommand(attrListCmd)
}
