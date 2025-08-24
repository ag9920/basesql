package common

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// ProgressBar 进度条结构体
type ProgressBar struct {
	Total   int
	Current int
	Width   int
	Prefix  string
	Suffix  string
}

// NewProgressBar 创建新的进度条
func NewProgressBar(total int, prefix string) *ProgressBar {
	return &ProgressBar{
		Total:  total,
		Width:  50,
		Prefix: prefix,
	}
}

// Update 更新进度条
func (pb *ProgressBar) Update(current int) {
	pb.Current = current
	pb.render()
}

// Increment 增加进度
func (pb *ProgressBar) Increment() {
	pb.Current++
	pb.render()
}

// Finish 完成进度条
func (pb *ProgressBar) Finish() {
	pb.Current = pb.Total
	pb.render()
	fmt.Println() // 换行
}

// render 渲染进度条
func (pb *ProgressBar) render() {
	percent := float64(pb.Current) / float64(pb.Total)
	filledWidth := int(percent * float64(pb.Width))

	bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", pb.Width-filledWidth)
	percentStr := fmt.Sprintf("%.1f%%", percent*100)

	fmt.Printf("\r%s [%s] %s (%d/%d)", pb.Prefix, bar, percentStr, pb.Current, pb.Total)
}

// UserFriendlyError 用户友好的错误信息
type UserFriendlyError struct {
	OriginalError error
	UserMessage   string
	Suggestions   []string
	ErrorCode     string
}

func (e *UserFriendlyError) Error() string {
	return e.UserMessage
}

// NewUserFriendlyError 创建用户友好的错误
func NewUserFriendlyError(originalErr error, userMsg string, suggestions ...string) *UserFriendlyError {
	return &UserFriendlyError{
		OriginalError: originalErr,
		UserMessage:   userMsg,
		Suggestions:   suggestions,
	}
}

// FormatUserError 格式化用户友好的错误信息
func FormatUserError(err error) string {
	if err == nil {
		return ""
	}

	// 检查是否为用户友好错误
	if ufErr, ok := err.(*UserFriendlyError); ok {
		var result strings.Builder
		result.WriteString(fmt.Sprintf("❌ %s\n", ufErr.UserMessage))

		if len(ufErr.Suggestions) > 0 {
			result.WriteString("\n💡 建议解决方案:\n")
			for i, suggestion := range ufErr.Suggestions {
				result.WriteString(fmt.Sprintf("   %d. %s\n", i+1, suggestion))
			}
		}

		return result.String()
	}

	// 处理常见错误类型
	errorMsg := err.Error()
	errorMsgLower := strings.ToLower(errorMsg)

	// 连接错误
	if strings.Contains(errorMsgLower, "connection") || strings.Contains(errorMsgLower, "连接") {
		return fmt.Sprintf("❌ 连接失败\n\n💡 建议解决方案:\n   1. 检查网络连接是否正常\n   2. 验证飞书应用配置是否正确\n   3. 确认 App Token 是否有效\n\n🔧 原始错误: %s", errorMsg)
	}

	// 认证错误
	if strings.Contains(errorMsgLower, "auth") || strings.Contains(errorMsgLower, "认证") || strings.Contains(errorMsgLower, "credential") {
		return fmt.Sprintf("❌ 认证失败\n\n💡 建议解决方案:\n   1. 检查 App ID 和 App Secret 是否正确\n   2. 确认应用是否已启用\n   3. 验证 App Token 是否匹配对应的多维表格\n\n🔧 原始错误: %s", errorMsg)
	}

	// SQL 语法错误
	if strings.Contains(errorMsgLower, "syntax") || strings.Contains(errorMsgLower, "语法") || strings.Contains(errorMsgLower, "parse") {
		return fmt.Sprintf("❌ SQL 语法错误\n\n💡 建议解决方案:\n   1. 检查 SQL 语句的语法是否正确\n   2. 确认表名和字段名是否存在\n   3. 参考帮助文档中的 SQL 语法示例\n\n🔧 原始错误: %s", errorMsg)
	}

	// 权限错误
	if strings.Contains(errorMsgLower, "permission") || strings.Contains(errorMsgLower, "权限") {
		return fmt.Sprintf("❌ 权限不足\n\n💡 建议解决方案:\n   1. 确认应用是否有访问该多维表格的权限\n   2. 检查 App Token 对应的表格是否正确\n   3. 联系表格管理员授予相应权限\n\n🔧 原始错误: %s", errorMsg)
	}

	// 表不存在错误
	if strings.Contains(errorMsgLower, "table") && (strings.Contains(errorMsgLower, "not found") || strings.Contains(errorMsgLower, "不存在")) {
		return fmt.Sprintf("❌ 表不存在\n\n💡 建议解决方案:\n   1. 使用 'SHOW TABLES' 命令查看可用的表\n   2. 检查表名拼写是否正确\n   3. 确认是否连接到正确的多维表格\n\n🔧 原始错误: %s", errorMsg)
	}

	// 默认错误格式
	return fmt.Sprintf("❌ 操作失败: %s", errorMsg)
}

// ShowSpinner 显示加载动画
func ShowSpinner(message string, duration time.Duration) {
	spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0
	start := time.Now()

	for time.Since(start) < duration {
		fmt.Printf("\r%s %s", spinners[i%len(spinners)], message)
		time.Sleep(100 * time.Millisecond)
		i++
	}

	fmt.Printf("\r✅ %s\n", message)
}

// PrintSuccess 打印成功信息
func PrintSuccess(message string) {
	fmt.Printf("✅ %s\n", message)
}

// PrintWarning 打印警告信息
func PrintWarning(message string) {
	fmt.Printf("⚠️  %s\n", message)
}

// PrintInfo 打印信息
func PrintInfo(message string) {
	fmt.Printf("ℹ️  %s\n", message)
}

// PrintError 打印错误信息
func PrintError(message string) {
	fmt.Fprintf(os.Stderr, "❌ %s\n", message)
}

// ConfirmAction 确认用户操作
func ConfirmAction(message string) bool {
	fmt.Printf("❓ %s (y/N): ", message)
	var response string
	fmt.Scanln(&response)
	return strings.ToLower(strings.TrimSpace(response)) == "y"
}

// FormatTableOutput 格式化表格输出
func FormatTableOutput(headers []string, rows [][]string) string {
	if len(headers) == 0 || len(rows) == 0 {
		return "📭 没有数据"
	}

	// 计算每列的最大宽度
	colWidths := make([]int, len(headers))
	for i, header := range headers {
		colWidths[i] = GetDisplayWidth(header)
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) {
				if width := GetDisplayWidth(cell); width > colWidths[i] {
					colWidths[i] = width
				}
			}
		}
	}

	var result strings.Builder

	// 打印表头
	result.WriteString("┌")
	for i, width := range colWidths {
		result.WriteString(strings.Repeat("─", width+2))
		if i < len(colWidths)-1 {
			result.WriteString("┬")
		}
	}
	result.WriteString("┐\n")

	// 打印表头内容
	result.WriteString("│")
	for i, header := range headers {
		result.WriteString(fmt.Sprintf(" %s ", PadString(header, colWidths[i])))
		if i < len(headers)-1 {
			result.WriteString("│")
		}
	}
	result.WriteString("│\n")

	// 打印分隔线
	result.WriteString("├")
	for i, width := range colWidths {
		result.WriteString(strings.Repeat("─", width+2))
		if i < len(colWidths)-1 {
			result.WriteString("┼")
		}
	}
	result.WriteString("┤\n")

	// 打印数据行
	for _, row := range rows {
		result.WriteString("│")
		for i, cell := range row {
			if i < len(colWidths) {
				result.WriteString(fmt.Sprintf(" %s ", PadString(cell, colWidths[i])))
				if i < len(row)-1 && i < len(colWidths)-1 {
					result.WriteString("│")
				}
			}
		}
		result.WriteString("│\n")
	}

	// 打印底部边框
	result.WriteString("└")
	for i, width := range colWidths {
		result.WriteString(strings.Repeat("─", width+2))
		if i < len(colWidths)-1 {
			result.WriteString("┴")
		}
	}
	result.WriteString("┘\n")

	return result.String()
}

// GetHelpText 获取帮助文本
func GetHelpText() string {
	return `🚀 BaseSQL CLI 帮助文档

📋 基本命令:
  connect                    测试与飞书多维表格的连接
  query "SQL"                执行 SELECT 查询
  exec "SQL"                 执行 INSERT/UPDATE/DELETE 操作
  shell                      启动交互式 SQL shell
  config init                初始化配置文件
  config show                显示当前配置

📝 SQL 语法示例:
  查询数据:
    SELECT * FROM users
    SELECT name, email FROM users WHERE age > 18
    SELECT COUNT(*) FROM users
  
  插入数据:
    INSERT INTO users (name, email) VALUES ('张三', 'zhang@example.com')
  
  更新数据:
    UPDATE users SET email = 'new@example.com' WHERE name = '张三'
  
  删除数据:
    DELETE FROM users WHERE age < 18
  
  查看表结构:
    SHOW TABLES
    SHOW COLUMNS FROM users

🔧 配置说明:
  配置文件位置: ~/.basesql/config.env
  环境变量:
    FEISHU_APP_ID      飞书应用 ID
    FEISHU_APP_SECRET  飞书应用密钥
    FEISHU_APP_TOKEN   多维表格 App Token

💡 使用说明:
  • 使用 --debug 参数查看详细的请求信息
  • 在交互式模式下使用 Tab 键自动补全
  • 使用上下箭头键浏览命令历史
  • SQL 语句可以不加分号结尾

🆘 常见问题:
  • 连接失败: 检查网络和应用配置
  • 认证失败: 验证 App ID、Secret 和 Token
  • 表不存在: 使用 SHOW TABLES 查看可用表
  • 权限不足: 确认应用有访问表格的权限

📚 更多信息请访问: https://github.com/ag9920/basesql
`
}
