package config

import (
	"fmt"
	"os"

	svcconfig "github.com/XingfenD/tag_file/internal/service/config"
	"github.com/spf13/cobra"
)

var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "管理配置项",
	Long:  `管理 tagfile 的全局配置项，如 import_dir 等。`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "设置配置项",
	Long:  `设置一个配置项的值，若不存在则自动创建。`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		svc := svcconfig.NewService()
		if err := svc.Set(args[0], args[1]); err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("配置 '%s' = '%s'\n", args[0], args[1])
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "查看配置项",
	Long:  `查看指定配置项的值。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc := svcconfig.NewService()
		cfg, err := svc.Get(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Println(cfg.ConfigValue)
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有配置项",
	Long:  `列出当前数据库中的所有配置项。支持 --limit/--offset 分页。`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		svc := svcconfig.NewService()
		list, total, err := svc.List(offset, limit)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		if len(list) == 0 {
			fmt.Println("暂无配置项")
			return
		}
		fmt.Printf("%-20s %-30s %s\n", "Key", "Value", "描述")
		for _, c := range list {
			fmt.Printf("%-20s %-30s %s\n", c.ConfigName, c.ConfigValue, c.Description)
		}
		if limit > 0 {
			fmt.Printf("--- %d/%d ---\n", len(list), total)
		}
	},
}

var configUnsetCmd = &cobra.Command{
	Use:   "unset <key>",
	Short: "删除配置项",
	Long:  `删除指定配置项。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc := svcconfig.NewService()
		if err := svc.Delete(args[0]); err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			os.Exit(1)
		}
		fmt.Printf("配置 '%s' 已删除\n", args[0])
	},
}

func init() {
	configListCmd.Flags().Int("limit", 0, "分页条数")
	configListCmd.Flags().Int("offset", 0, "分页偏移")
	ConfigCmd.AddCommand(configSetCmd)
	ConfigCmd.AddCommand(configGetCmd)
	ConfigCmd.AddCommand(configListCmd)
	ConfigCmd.AddCommand(configUnsetCmd)
}
