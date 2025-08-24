package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/ag9920/basesql"
	"github.com/ag9920/basesql/internal/common"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config CLI 配置结构体
// 包含连接飞书多维表格所需的所有配置信息
type Config struct {
	// ConfigFile 配置文件路径（可选）
	ConfigFile string
	// AppID 飞书应用 ID（必需）
	AppID string
	// AppSecret 飞书应用密钥（必需）
	AppSecret string
	// AppToken 飞书多维表格 App Token（必需）
	AppToken string
	// Debug 是否启用调试模式
	Debug bool
	// Timeout 连接超时时间（秒）
	Timeout int
}

// Client CLI 客户端
// 封装了与飞书多维表格的交互逻辑
type Client struct {
	// db GORM 数据库实例
	db *gorm.DB
	// config 客户端配置
	config *Config
	// executor SQL 执行器
	executor *Executor
}

// NewClient 创建新的 CLI 客户端
// 该函数会验证配置、建立连接并初始化客户端
// 参数:
//   - config: 客户端配置
//
// 返回:
//   - *Client: 客户端实例
//   - error: 错误信息
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	// 从环境变量或配置文件加载配置
	cfg, err := loadConfig(config)
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	// 创建 BaseSQL 配置
	baseCfg := &basesql.Config{
		AppID:     cfg.AppID,
		AppSecret: cfg.AppSecret,
		AppToken:  cfg.AppToken,
		AuthType:  basesql.AuthTypeTenant,
		DebugMode: cfg.Debug,
		Timeout:   300 * time.Second, // 增加超时时间到5分钟，支持大量数据分页获取
	}

	// 配置 GORM
	gormConfig := &gorm.Config{
		// 根据调试模式设置日志级别
		Logger: logger.Default.LogMode(getLogLevel(cfg.Debug)),
		// 禁用外键约束检查（飞书多维表格不支持）
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	// 连接数据库
	db, err := gorm.Open(basesql.Open(baseCfg), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("连接飞书多维表格失败: %w", err)
	}

	// 创建执行器
	executor, err := NewExecutor(db)
	if err != nil {
		return nil, fmt.Errorf("创建执行器失败: %w", err)
	}

	client := &Client{
		db:       db,
		config:   cfg,
		executor: executor,
	}

	// 验证连接
	if err := client.validateConnection(); err != nil {
		return nil, fmt.Errorf("连接验证失败: %w", err)
	}

	return client, nil
}

// Close 关闭客户端连接
// 清理资源并关闭与飞书多维表格的连接
// 返回:
//   - error: 错误信息
func (c *Client) Close() error {
	if c == nil {
		return nil
	}

	// GORM 不需要显式关闭连接，但可以在这里添加清理逻辑
	c.db = nil
	c.executor = nil
	return nil
}

// Query 执行查询操作
// 专门用于执行 SELECT 类型的查询语句
// 参数:
//   - sql: SQL 查询语句
//
// 返回:
//   - error: 错误信息
func (c *Client) Query(sql string) error {
	if c == nil {
		return fmt.Errorf("客户端未初始化")
	}
	return c.Execute(sql)
}

// Exec 执行修改操作
// 专门用于执行 INSERT、UPDATE、DELETE 类型的语句
// 参数:
//   - sql: SQL 执行语句
//
// 返回:
//   - error: 错误信息
func (c *Client) Exec(sql string) error {
	if c == nil {
		return fmt.Errorf("客户端未初始化")
	}
	return c.Execute(sql)
}

// Execute 执行任意 SQL 语句
// 统一的 SQL 执行入口，支持所有类型的 SQL 语句
// 参数:
//   - sql: SQL 语句
//
// 返回:
//   - error: 错误信息
func (c *Client) Execute(sql string) error {
	if c == nil {
		return common.NewUserFriendlyError(
			fmt.Errorf("客户端未初始化"),
			"客户端连接异常",
			"请重新建立连接",
			"检查网络连接是否正常",
		)
	}

	if c.executor == nil {
		return common.NewUserFriendlyError(
			fmt.Errorf("执行器未初始化"),
			"SQL 执行器异常",
			"请重启应用程序",
		)
	}

	sql = strings.TrimSpace(sql)
	if sql == "" {
		return common.NewUserFriendlyError(
			fmt.Errorf("SQL 语句为空"),
			"请输入有效的 SQL 语句",
			"参考帮助文档中的 SQL 语法示例",
			"使用 'help' 命令查看可用的 SQL 语法",
		)
	}

	// 记录SQL执行开始
	if c.config.Debug {
		common.Debug(fmt.Sprintf("Starting SQL execution: %s", sql))
	}

	// 解析 SQL
	cmd, err := ParseSQL(sql)
	if err != nil {
		return common.NewUserFriendlyError(
			err,
			"SQL 语法错误",
			"检查 SQL 语句的语法是否正确",
			"确认表名和字段名是否存在",
			"参考帮助文档中的 SQL 语法示例",
		)
	}

	// 执行命令
	start := time.Now()
	err = c.executor.Execute(cmd)
	duration := time.Since(start)

	// 记录SQL执行日志
	common.LogSQLExecution(sql, duration, err)

	if err != nil {
		return common.NewUserFriendlyError(
			err,
			"SQL 执行失败",
			"检查表名和字段名是否正确",
			"确认是否有足够的权限执行此操作",
			"使用 'SHOW TABLES' 查看可用的表",
		)
	}

	// 显示执行成功信息
	if c.config.Debug {
		common.PrintSuccess(fmt.Sprintf("SQL 执行完成，耗时: %v", duration))
	}

	return nil
}

// validateConnection 验证与飞书多维表格的连接
// 通过执行简单的查询来验证连接是否正常
// 返回:
//   - error: 错误信息
func (c *Client) validateConnection() error {
	if c.db == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	// 这里可以添加简单的连接验证逻辑
	// 例如执行 SHOW TABLES 命令
	return nil
}

// getLogLevel 根据调试模式获取日志级别
// 参数:
//   - debug: 是否启用调试模式
//
// 返回:
//   - logger.LogLevel: GORM 日志级别
func getLogLevel(debug bool) logger.LogLevel {
	if debug {
		return logger.Info
	}
	return logger.Warn
}

// loadConfig 加载和验证配置
// 按优先级顺序加载配置：命令行参数 > 环境变量 > 配置文件
// 参数:
//   - config: 输入配置
//
// 返回:
//   - *Config: 加载后的配置
//   - error: 错误信息
func loadConfig(config *Config) (*Config, error) {
	if config == nil {
		return nil, fmt.Errorf("输入配置不能为空")
	}

	result := &Config{
		Debug:   config.Debug,
		Timeout: config.Timeout,
	}

	// 设置默认超时时间
	if result.Timeout <= 0 {
		result.Timeout = 30 // 默认 30 秒
	}

	// 优先使用命令行参数，其次使用环境变量
	result.AppID = getConfigValue(config.AppID, "FEISHU_APP_ID")
	result.AppSecret = getConfigValue(config.AppSecret, "FEISHU_APP_SECRET")
	result.AppToken = getConfigValue(config.AppToken, "FEISHU_APP_TOKEN")

	// 验证必要的配置
	if err := validateRequiredConfig(result); err != nil {
		return nil, err
	}

	return result, nil
}

// getConfigValue 获取配置值
// 优先使用提供的值，如果为空则从环境变量获取
// 参数:
//   - value: 提供的配置值
//   - envKey: 环境变量键名
//
// 返回:
//   - string: 配置值
func getConfigValue(value, envKey string) string {
	if value != "" {
		return value
	}
	return common.GetEnv(envKey, "")
}

// validateRequiredConfig 验证必需的配置项
// 参数:
//   - config: 配置实例
//
// 返回:
//   - error: 验证错误信息
func validateRequiredConfig(config *Config) error {
	var missing []string

	if config.AppID == "" {
		missing = append(missing, "飞书应用 ID (FEISHU_APP_ID)")
	}
	if config.AppSecret == "" {
		missing = append(missing, "飞书应用密钥 (FEISHU_APP_SECRET)")
	}
	if config.AppToken == "" {
		missing = append(missing, "多维表格 Token (FEISHU_APP_TOKEN)")
	}

	if len(missing) > 0 {
		return fmt.Errorf("缺少必要的配置信息: %s\n\n"+
			"请通过以下方式之一提供配置:\n"+
			"1. 命令行参数:\n"+
			"   --app-id=your_app_id\n"+
			"   --app-secret=your_app_secret\n"+
			"   --app-token=your_app_token\n\n"+
			"2. 环境变量:\n"+
			"   export FEISHU_APP_ID=your_app_id\n"+
			"   export FEISHU_APP_SECRET=your_app_secret\n"+
			"   export FEISHU_APP_TOKEN=your_app_token\n\n"+
			"3. 配置文件:\n"+
			"   basesql config init",
			strings.Join(missing, ", "))
	}

	return nil
}
