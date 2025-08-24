package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ag9920/basesql/internal/cli"
	"github.com/ag9920/basesql/internal/common"
	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
)

var (
	// 版本信息，在构建时可以通过 ldflags 设置
	// 使用方式: go build -ldflags "-X main.version=v1.2.3"
	version = "1.0.0"

	// 全局配置选项，这些选项可以通过命令行参数或环境变量设置
	configFile string // 配置文件路径，默认为 ~/.basesql/config.env
	appID      string // 飞书应用 ID，用于身份认证
	appSecret  string // 飞书应用密钥，用于身份认证
	appToken   string // 多维表格 App Token，用于访问特定的多维表格
	debug      bool   // 调试模式开关，启用后显示详细的请求和响应信息
)

// main 函数是 CLI 工具的入口点
// 负责初始化命令行界面、设置全局选项和执行用户命令
// 该函数会创建根命令、设置全局标志、添加子命令，并处理执行过程中的错误
func main() {
	// 确保在程序退出时清理资源
	defer func() {
		if err := common.ShutdownGlobalResourceManager(); err != nil {
			common.Warnf("清理全局资源时出错: %v", err)
		}
	}()

	// 创建根命令，定义 CLI 工具的基本信息和行为
	rootCmd := &cobra.Command{
		Use:   "basesql",
		Short: "BaseSQL CLI - 使用 SQL 操作飞书多维表格",
		Long: `BaseSQL CLI 是一个命令行工具，让你可以使用标准 SQL 语法来操作飞书多维表格。

支持的功能：
  • SELECT 查询数据
  • INSERT 插入数据
  • UPDATE 更新数据
  • DELETE 删除数据
  • SHOW TABLES 显示表列表
  • 交互式 SQL Shell
  • 配置管理

使用示例：
  basesql query "SELECT * FROM users"
  basesql exec "INSERT INTO users (name, email) VALUES ('张三', 'zhangsan@example.com')"
  basesql shell
  basesql config init`,
		Version: version,
		// 禁用默认的补全命令，我们提供自定义的补全功能
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		// 设置示例用法
		Example: `  # 测试连接
  basesql connect

  # 查询数据
  basesql query "SELECT * FROM users LIMIT 10"

  # 插入数据
  basesql exec "INSERT INTO users (name, email) VALUES ('李四', 'lisi@example.com')"

  # 启动交互式 Shell
  basesql shell

  # 初始化配置
  basesql config init`,
		// 静默使用信息，避免在错误时显示使用帮助
		SilenceUsage: true,
	}

	// 设置全局标志
	setupGlobalFlags(rootCmd)

	// 添加子命令
	addSubCommands(rootCmd)

	// 执行命令并处理错误
	if err := rootCmd.Execute(); err != nil {
		// 根据错误类型设置不同的退出码，便于脚本判断错误类型
		exitCode := getExitCode(err)
		// 使用用户友好的错误格式
		errorMsg := common.FormatUserError(err)
		fmt.Fprint(os.Stderr, errorMsg)
		os.Exit(exitCode)
	}
}

// setupGlobalFlags 设置全局命令行标志
// 这些标志在所有子命令中都可用，提供统一的配置方式
// 参数:
//   - cmd: 根命令实例
func setupGlobalFlags(cmd *cobra.Command) {
	// 配置文件路径标志
	cmd.PersistentFlags().StringVarP(&configFile, "config", "c", "",
		"配置文件路径 (默认: ~/.basesql/config.env)")

	// 飞书应用认证相关标志
	cmd.PersistentFlags().StringVar(&appID, "app-id", "",
		"飞书应用 ID，用于身份认证")
	cmd.PersistentFlags().StringVar(&appSecret, "app-secret", "",
		"飞书应用密钥，用于身份认证")
	cmd.PersistentFlags().StringVar(&appToken, "app-token", "",
		"多维表格 App Token，用于访问特定的多维表格")

	// 调试模式标志
	cmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false,
		"启用调试模式，显示详细的请求和响应信息")

	// 注意：配置文件标志已设置
}

// addSubCommands 添加所有子命令
// 该函数负责将各个功能模块的命令添加到根命令中
// 参数:
//   - cmd: 根命令实例
func addSubCommands(cmd *cobra.Command) {
	// 连接测试命令
	cmd.AddCommand(newConnectCmd())

	// SQL 执行命令
	cmd.AddCommand(newQueryCmd())
	cmd.AddCommand(newExecCmd())

	// 交互式 Shell 命令
	cmd.AddCommand(newInteractiveCmd())

	// 配置管理命令
	cmd.AddCommand(newConfigCmd())
}

// getExitCode 根据错误类型返回适当的退出码
// 不同的退出码可以帮助脚本和自动化工具判断错误类型
// 参数:
//   - err: 错误信息
//
// 返回:
//   - int: 退出码
//   - 0: 成功
//   - 1: 一般错误
//   - 2: 配置错误
//   - 3: 连接错误
//   - 4: SQL 语法错误
//   - 5: 权限错误
func getExitCode(err error) int {
	if err == nil {
		return 0
	}

	errorMsg := strings.ToLower(err.Error())

	// 配置相关错误
	if strings.Contains(errorMsg, "配置") || strings.Contains(errorMsg, "config") {
		return 2
	}

	// 连接相关错误
	if strings.Contains(errorMsg, "连接") || strings.Contains(errorMsg, "connect") ||
		strings.Contains(errorMsg, "网络") || strings.Contains(errorMsg, "network") {
		return 3
	}

	// SQL 语法错误
	if strings.Contains(errorMsg, "语法") || strings.Contains(errorMsg, "syntax") ||
		strings.Contains(errorMsg, "解析") || strings.Contains(errorMsg, "parse") {
		return 4
	}

	// 权限相关错误
	if strings.Contains(errorMsg, "权限") || strings.Contains(errorMsg, "permission") ||
		strings.Contains(errorMsg, "认证") || strings.Contains(errorMsg, "auth") {
		return 5
	}

	// 默认为一般错误
	return 1
}

// newConnectCmd 创建连接测试命令
// 该命令用于测试与飞书多维表格的连接是否正常
// 返回:
//   - *cobra.Command: 连接测试命令实例
func newConnectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connect",
		Short: "测试与飞书多维表格的连接",
		Long: `测试与飞书多维表格的连接是否正常。

该命令会验证配置的飞书应用信息是否正确，
并尝试连接到指定的多维表格。`,
		Example: `  # 使用命令行参数测试连接
  basesql connect --app-id=your_app_id --app-secret=your_secret --app-token=your_token

  # 使用环境变量测试连接
  export FEISHU_APP_ID=your_app_id
  export FEISHU_APP_SECRET=your_secret
  export FEISHU_APP_TOKEN=your_token
  basesql connect`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("🔗 正在测试连接...")

			client, err := cli.NewClient(getConfig())
			if err != nil {
				return fmt.Errorf("连接失败: %w", err)
			}
			defer client.Close()

			fmt.Println("✅ 连接成功！")
			fmt.Println("📋 可以开始使用 BaseSQL 操作飞书多维表格了")
			return nil
		},
	}
	return cmd
}

// newQueryCmd 创建查询命令
// 该命令用于执行 SELECT 查询语句
// 返回:
//   - *cobra.Command: 查询命令实例
func newQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query [SQL]",
		Short: "执行 SELECT 查询语句",
		Long: `执行 SELECT 查询语句，从飞书多维表格中检索数据。

支持的查询语法：
  • SELECT * FROM table_name
  • SELECT field1, field2 FROM table_name
  • SELECT * FROM table_name WHERE condition
  • SHOW TABLES
  • SHOW COLUMNS FROM table_name`,
		Args: cobra.ExactArgs(1),
		Example: `  # 查询所有数据
  basesql query "SELECT * FROM users"

  # 查询特定字段
  basesql query "SELECT name, email FROM users"

  # 带条件查询
  basesql query "SELECT * FROM users WHERE name = '张三'"

  # 显示所有表
  basesql query "SHOW TABLES"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] == "" {
				return fmt.Errorf("SQL 查询语句不能为空")
			}

			client, err := cli.NewClient(getConfig())
			if err != nil {
				return fmt.Errorf("连接失败: %w", err)
			}
			defer client.Close()

			return client.Query(args[0])
		},
	}
	return cmd
}

// newExecCmd 创建执行命令
// 该命令用于执行 INSERT、UPDATE、DELETE 等数据修改操作
// 返回:
//   - *cobra.Command: 执行命令实例
func newExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec [SQL]",
		Short: "执行 INSERT、UPDATE、DELETE 等数据修改操作",
		Long: `执行数据修改操作，包括插入、更新和删除数据。

支持的操作语法：
  • INSERT INTO table (field1, field2) VALUES (value1, value2)
  • UPDATE table SET field1=value1 WHERE condition
  • DELETE FROM table WHERE condition`,
		Args: cobra.ExactArgs(1),
		Example: `  # 插入数据
  basesql exec "INSERT INTO users (name, email) VALUES ('张三', 'zhangsan@example.com')"

  # 更新数据
  basesql exec "UPDATE users SET email = 'new@example.com' WHERE name = '张三'"

  # 删除数据
  basesql exec "DELETE FROM users WHERE name = '张三'"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] == "" {
				return fmt.Errorf("SQL 执行语句不能为空")
			}

			client, err := cli.NewClient(getConfig())
			if err != nil {
				return fmt.Errorf("连接失败: %w", err)
			}
			defer client.Close()

			return client.Exec(args[0])
		},
	}
	return cmd
}

// newInteractiveCmd 创建交互式 Shell 命令
// 该命令启动一个交互式的 SQL shell，支持命令历史和自动补全
// 返回:
//   - *cobra.Command: 交互式命令实例
func newInteractiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "shell",
		Aliases: []string{"interactive", "i"},
		Short:   "启动交互式 SQL shell",
		Long: `启动交互式 SQL shell，提供便捷的命令行界面。

功能特性：
  • 支持多行 SQL 语句输入
  • 命令历史记录
  • 自动补全功能
  • 语法高亮显示
  • 内置帮助命令`,
		Example: `  # 启动交互式 shell
  basesql shell

  # 在 shell 中执行命令
  basesql> SELECT * FROM users;
  basesql> SHOW TABLES;
  basesql> exit`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cli.NewClient(getConfig())
			if err != nil {
				return fmt.Errorf("连接失败: %w", err)
			}
			defer client.Close()

			// 配置 readline
			rl, err := readline.NewEx(&readline.Config{
				Prompt:          "basesql> ",
				HistoryFile:     os.ExpandEnv("$HOME/.basesql_history"),
				AutoComplete:    newCompleter(),
				InterruptPrompt: "^C",
				EOFPrompt:       "exit",
			})
			if err != nil {
				return fmt.Errorf("初始化 readline 失败: %w", err)
			}
			defer rl.Close()

			// 显示欢迎信息
			fmt.Println("🚀 BaseSQL 交互式 Shell")
			fmt.Println("📝 输入 SQL 语句，使用 \\q 退出")
			fmt.Println("💡 使用上下箭头键浏览命令历史，Tab 键自动补全")
			fmt.Println("---")

			for {
				line, err := rl.Readline()
				if err != nil {
					if err.Error() == "Interrupt" {
						fmt.Println("\n👋 再见！")
					}
					break
				}

				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}

				// 处理内置命令
				switch strings.ToLower(line) {
				case "\\q", "quit", "exit":
					fmt.Println("👋 再见！")
					return nil
				case "help", "\\h":
					printShellHelp()
					continue
				case "clear", "\\c":
					fmt.Print("\033[2J\033[H") // 清屏
					continue
				}

				// 执行 SQL 命令
				if err := client.Execute(line); err != nil {
					errorMsg := common.FormatUserError(err)
					fmt.Print(errorMsg)
				} else {
					common.PrintSuccess("命令执行成功")
				}
				fmt.Println() // 添加空行分隔
			}

			return nil
		},
	}
	return cmd
}

// newConfigCmd 创建配置管理命令
// 该命令提供配置文件的初始化和查看功能
// 返回:
//   - *cobra.Command: 配置管理命令实例
func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "配置文件管理",
		Long: `管理 BaseSQL 的配置文件。

配置文件位置: ~/.basesql/config.env
支持的配置项：
  • FEISHU_APP_ID: 飞书应用 ID
  • FEISHU_APP_SECRET: 飞书应用密钥
  • FEISHU_APP_TOKEN: 飞书多维表格 Token`,
		Example: `  # 初始化配置文件
  basesql config init

  # 查看当前配置
  basesql config show`,
	}

	// 初始化配置子命令
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "初始化配置文件",
		Long: `在用户主目录下创建 BaseSQL 配置文件。

该命令会在 ~/.basesql/ 目录下创建 config.env 文件，
包含所有必要的配置项模板。`,
		Example: `  # 初始化配置文件
  basesql config init

  # 然后编辑配置文件
  vim ~/.basesql/config.env`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("📝 正在初始化配置文件...")
			if err := cli.InitConfig(); err != nil {
				return fmt.Errorf("初始化配置失败: %w", err)
			}
			fmt.Println("✅ 配置文件初始化成功！")
			fmt.Println("📁 配置文件位置: ~/.basesql/config.env")
			fmt.Println("💡 请编辑配置文件并填入您的飞书应用信息")
			return nil
		},
	}

	// 显示配置子命令
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "显示当前配置信息",
		Long: `显示当前的配置信息，敏感信息会被遮盖。

该命令会读取配置文件和环境变量，
并显示当前生效的配置值。`,
		Example: `  # 显示当前配置
  basesql config show`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("📋 当前配置信息:")
			if err := cli.ShowConfig(); err != nil {
				return fmt.Errorf("显示配置失败: %w", err)
			}
			return nil
		},
	}

	cmd.AddCommand(initCmd, showCmd)
	return cmd
}

// printShellHelp 显示交互式 Shell 的帮助信息
func printShellHelp() {
	fmt.Println("📚 BaseSQL 交互式 Shell 帮助")
	fmt.Println("")
	fmt.Println("🔧 内置命令:")
	fmt.Println("  help, \\h     显示此帮助信息")
	fmt.Println("  exit, quit, \\q  退出 Shell")
	fmt.Println("  clear, \\c    清屏")
	fmt.Println("")
	fmt.Println("📝 SQL 命令示例:")
	fmt.Println("  SHOW TABLES;")
	fmt.Println("  SHOW COLUMNS FROM table_name;")
	fmt.Println("  SELECT * FROM table_name;")
	fmt.Println("  SELECT field1, field2 FROM table_name WHERE condition;")
	fmt.Println("  INSERT INTO table (field1, field2) VALUES (value1, value2);")
	fmt.Println("  UPDATE table SET field1=value1 WHERE condition;")
	fmt.Println("  DELETE FROM table WHERE condition;")
	fmt.Println("")
	fmt.Println("💡 提示:")
	fmt.Println("  • 使用上下箭头键浏览命令历史")
	fmt.Println("  • 使用 Tab 键进行自动补全")
	fmt.Println("  • SQL 语句可以不加分号结尾")
	fmt.Println("")
}

// getConfig 从命令行参数和环境变量获取配置
// 该函数会按优先级顺序获取配置：命令行参数 > 环境变量 > 配置文件
// 返回:
//   - *cli.Config: 配置实例
func getConfig() *cli.Config {
	// 初始化日志系统
	if err := common.InitializeLogging(debug, ""); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 日志系统初始化失败: %v\n", err)
	}

	// 优先使用命令行参数
	config := &cli.Config{
		ConfigFile: configFile,
		AppID:      appID,
		AppSecret:  appSecret,
		AppToken:   appToken,
		Debug:      debug,
	}

	// 如果命令行参数为空，尝试从环境变量获取
	if config.AppID == "" {
		config.AppID = common.GetEnv("FEISHU_APP_ID", "")
	}
	if config.AppSecret == "" {
		config.AppSecret = common.GetEnv("FEISHU_APP_SECRET", "")
	}
	if config.AppToken == "" {
		config.AppToken = common.GetEnv("FEISHU_APP_TOKEN", "")
	}

	return config
}

// newCompleter 创建自动补全器
func newCompleter() readline.AutoCompleter {
	return readline.NewPrefixCompleter(
		readline.PcItem("SELECT",
			readline.PcItem("*"),
			readline.PcItem("COUNT(*)"),
		),
		readline.PcItem("SHOW",
			readline.PcItem("TABLES"),
			readline.PcItem("DATABASES"),
			readline.PcItem("COLUMNS"),
		),
		readline.PcItem("DESC"),
		readline.PcItem("DESCRIBE"),
		readline.PcItem("INSERT",
			readline.PcItem("INTO"),
		),
		readline.PcItem("UPDATE"),
		readline.PcItem("DELETE",
			readline.PcItem("FROM"),
		),
		readline.PcItem("CREATE",
			readline.PcItem("TABLE"),
		),
		readline.PcItem("DROP",
			readline.PcItem("TABLE"),
		),
		readline.PcItem("FROM"),
		readline.PcItem("WHERE"),
		readline.PcItem("ORDER",
			readline.PcItem("BY"),
		),
		readline.PcItem("GROUP",
			readline.PcItem("BY"),
		),
		readline.PcItem("LIMIT"),
		readline.PcItem("\\q"),
		readline.PcItem("quit"),
		readline.PcItem("exit"),
	)
}
