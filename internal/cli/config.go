package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ag9920/basesql/internal/common"
)

// configTemplate 配置文件模板
// 包含所有必要的配置项和说明
const configTemplate = `# BaseSQL 配置文件
# 用于连接飞书多维表格的应用配置
#
# 获取这些配置的步骤：
# 1. 访问飞书开放平台: https://open.feishu.cn/
# 2. 创建企业自建应用
# 3. 获取 App ID 和 App Secret
# 4. 在多维表格中获取 App Token
#
# 注意：请妥善保管这些敏感信息，不要提交到版本控制系统

# 飞书应用 ID（必需）
FEISHU_APP_ID=your_app_id

# 飞书应用密钥（必需）
FEISHU_APP_SECRET=your_app_secret

# 飞书多维表格 App Token（必需）
FEISHU_APP_TOKEN=your_app_token

# 调试模式（可选，默认为 false）
# 启用后会显示详细的调试信息
# DEBUG=false

# 连接超时时间（可选，默认为 30 秒）
# TIMEOUT=30
`

// InitConfig 初始化配置文件
// 在用户主目录下创建 BaseSQL 配置文件
// 返回:
//   - error: 初始化错误信息
func InitConfig() error {
	// 获取用户主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("获取用户主目录失败: %w", err)
	}

	// 创建配置目录
	configDir := filepath.Join(homeDir, ".basesql")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 配置文件路径
	configFile := filepath.Join(configDir, "config.env")

	// 检查文件是否已存在
	if _, err := os.Stat(configFile); err == nil {
		fmt.Printf("⚠️  配置文件已存在: %s\n", configFile)
		fmt.Println("💡 如需重新创建，请先删除现有配置文件")
		return nil
	}

	// 创建配置文件
	if err := os.WriteFile(configFile, []byte(configTemplate), 0600); err != nil {
		return fmt.Errorf("创建配置文件失败: %w", err)
	}

	fmt.Printf("✅ 配置文件已创建: %s\n", configFile)
	fmt.Println("")
	fmt.Println("📝 下一步操作:")
	fmt.Println("1. 编辑配置文件并填入您的飞书应用信息")
	fmt.Println("2. 确保应用具有多维表格的读写权限")
	fmt.Println("3. 使用 'basesql connect' 测试连接")

	return nil
}

// ShowConfig 显示当前配置信息
// 敏感信息会被遮盖显示
// 返回:
//   - error: 显示错误信息
func ShowConfig() error {
	// 获取配置文件路径
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("获取用户主目录失败: %w", err)
	}

	configFile := filepath.Join(homeDir, ".basesql", "config.env")

	fmt.Println("📋 BaseSQL 配置信息")
	fmt.Println("")
	fmt.Printf("📁 配置文件位置: %s\n", configFile)

	// 检查配置文件是否存在
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		fmt.Println("⚠️  配置文件不存在")
		fmt.Println("💡 使用 'basesql config init' 创建配置文件")
		fmt.Println("")
	}

	fmt.Println("🔧 当前配置值:")
	fmt.Printf("  飞书应用 ID:     %s\n", maskSensitive(common.GetEnv("FEISHU_APP_ID", "")))
	fmt.Printf("  飞书应用密钥:    %s\n", maskSensitive(common.GetEnv("FEISHU_APP_SECRET", "")))
	fmt.Printf("  多维表格 Token:  %s\n", maskSensitive(common.GetEnv("FEISHU_APP_TOKEN", "")))
	fmt.Printf("  调试模式:        %s\n", common.GetEnv("DEBUG", "false"))
	fmt.Printf("  连接超时:        %s 秒\n", common.GetEnv("TIMEOUT", "30"))
	fmt.Println("")

	// 检查配置完整性
	if err := validateConfigCompleteness(); err != nil {
		fmt.Printf("❌ 配置验证失败: %v\n", err)
		fmt.Println("💡 请检查并完善配置信息")
	} else {
		fmt.Println("✅ 配置验证通过")
	}

	return nil
}

// maskSensitive 遮盖敏感信息
// 参数:
//   - value: 需要遮盖的值
//
// 返回:
//   - string: 遮盖后的值
func maskSensitive(value string) string {
	if value == "" {
		return "<未设置>"
	}
	if len(value) <= 8 {
		return strings.Repeat("*", len(value))
	}
	return value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
}

// validateConfigCompleteness 验证配置完整性
// 返回:
//   - error: 验证错误信息
func validateConfigCompleteness() error {
	requiredConfigs := map[string]string{
		"FEISHU_APP_ID":     "飞书应用 ID",
		"FEISHU_APP_SECRET": "飞书应用密钥",
		"FEISHU_APP_TOKEN":  "多维表格 Token",
	}

	var missing []string
	for key, name := range requiredConfigs {
		if common.GetEnv(key, "") == "" {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("缺少必要配置: %s", strings.Join(missing, ", "))
	}

	return nil
}
