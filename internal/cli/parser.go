package cli

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ag9920/basesql/internal/common"
)

// 使用公共的 SQL 命令类型
type SQLCommandType = common.SQLCommandType
type SQLCommand = common.SQLCommand

// 导出常量以保持向后兼容性
const (
	CommandSelect   = common.CommandSelect
	CommandInsert   = common.CommandInsert
	CommandUpdate   = common.CommandUpdate
	CommandDelete   = common.CommandDelete
	CommandShow     = common.CommandShow
	CommandCreate   = common.CommandCreate
	CommandDrop     = common.CommandDrop
	CommandDescribe = common.CommandDescribe
	CommandUnknown  = common.CommandUnknown
)

// ParseSQL 解析 SQL 语句
// 将 SQL 字符串解析为结构化的 SQLCommand 对象
// 参数:
//   - sql: 要解析的 SQL 语句
//
// 返回:
//   - *SQLCommand: 解析后的命令对象
//   - error: 解析错误信息
func ParseSQL(sql string) (*common.SQLCommand, error) {
	if err := common.ValidateNotEmpty(sql, "SQL 语句"); err != nil {
		return nil, err
	}

	// 预处理 SQL 语句
	sql = common.PreprocessSQL(sql)

	// 识别命令类型
	cmdType := identifyCommandType(sql)
	cmd := common.NewSQLCommand(cmdType)
	cmd.RawSQL = sql

	// 根据命令类型进行解析
	switch cmdType {
	case common.CommandSelect:
		return parseSelect(sql, cmd)
	case common.CommandInsert:
		return parseInsert(sql, cmd)
	case common.CommandUpdate:
		return parseUpdate(sql, cmd)
	case common.CommandDelete:
		return parseDelete(sql, cmd)
	case common.CommandShow:
		return parseShow(sql, cmd)
	default:
		return nil, fmt.Errorf("不支持的 SQL 命令类型: %s", string(cmdType))
	}
}

// identifyCommandType 识别 SQL 命令类型
func identifyCommandType(sql string) common.SQLCommandType {
	upperSQL := strings.ToUpper(sql)

	switch {
	case strings.HasPrefix(upperSQL, "SELECT"):
		return common.CommandSelect
	case strings.HasPrefix(upperSQL, "INSERT"):
		return common.CommandInsert
	case strings.HasPrefix(upperSQL, "UPDATE"):
		return common.CommandUpdate
	case strings.HasPrefix(upperSQL, "DELETE"):
		return common.CommandDelete
	case strings.HasPrefix(upperSQL, "SHOW"):
		return common.CommandShow
	case strings.HasPrefix(upperSQL, "CREATE"):
		return common.CommandCreate
	case strings.HasPrefix(upperSQL, "DROP"):
		return common.CommandDrop
	case strings.HasPrefix(upperSQL, "DESCRIBE") || strings.HasPrefix(upperSQL, "DESC"):
		return common.CommandDescribe
	default:
		return common.CommandUnknown
	}
}

// parseShow 解析 SHOW 命令
// 支持 SHOW TABLES、SHOW COLUMNS FROM table 等命令
// 参数:
//   - sql: SQL 语句
//   - cmd: 命令对象
//
// 返回:
//   - *SQLCommand: 解析后的命令
//   - error: 解析错误
func parseShow(sql string, cmd *common.SQLCommand) (*common.SQLCommand, error) {
	upperSQL := strings.ToUpper(sql)

	switch {
	case strings.Contains(upperSQL, "TABLES"):
		cmd.ShowType = "TABLES"
		return cmd, nil

	case strings.Contains(upperSQL, "DATABASES"):
		cmd.ShowType = "DATABASES"
		return cmd, nil

	case strings.Contains(upperSQL, "COLUMNS"):
		// SHOW COLUMNS FROM table_name
		re := regexp.MustCompile(`(?i)SHOW\s+COLUMNS\s+FROM\s+([^\s;]+)`)
		matches := re.FindStringSubmatch(sql)
		if len(matches) < 2 {
			return nil, fmt.Errorf("SHOW COLUMNS 语法错误，正确格式: SHOW COLUMNS FROM table_name")
		}
		cmd.ShowType = "COLUMNS"
		cmd.Table = strings.TrimSpace(matches[1])
		return cmd, nil

	default:
		return nil, fmt.Errorf("不支持的 SHOW 命令: %s，支持的命令: SHOW TABLES, SHOW COLUMNS FROM table", sql)
	}
}

// parseSelect 解析 SELECT 命令
// 支持 SELECT fields FROM table [WHERE condition] 语法
// 支持聚合函数如 COUNT(*), SUM(field), AVG(field) 等
// 参数:
//   - sql: SQL 语句
//   - cmd: 命令对象
//
// 返回:
//   - *SQLCommand: 解析后的命令
//   - error: 解析错误
func parseSelect(sql string, cmd *SQLCommand) (*SQLCommand, error) {
	return common.DefaultSQLParser.ParseSelectSQL(sql, cmd)
}

// parseInsert 解析 INSERT 命令
// 支持 INSERT INTO table (fields) VALUES (values) 语法
// 参数:
//   - sql: SQL 语句
//   - cmd: 命令对象
//
// 返回:
//   - *SQLCommand: 解析后的命令
//   - error: 解析错误
func parseInsert(sql string, cmd *SQLCommand) (*SQLCommand, error) {
	return common.DefaultSQLParser.ParseInsertSQL(sql, cmd)
}

// parseUpdate 解析 UPDATE 命令
// 支持 UPDATE table SET field1=value1, field2=value2 [WHERE condition] 语法
// 参数:
//   - sql: SQL 语句
//   - cmd: 命令对象
//
// 返回:
//   - *SQLCommand: 解析后的命令
//   - error: 解析错误
func parseUpdate(sql string, cmd *SQLCommand) (*SQLCommand, error) {
	return common.DefaultSQLParser.ParseUpdateSQL(sql, cmd)
}

// parseDelete 解析 DELETE 命令
// 支持 DELETE FROM table [WHERE condition] 语法
// 参数:
//   - sql: SQL 语句
//   - cmd: 命令对象
//
// 返回:
//   - *SQLCommand: 解析后的命令
//   - error: 解析错误
func parseDelete(sql string, cmd *SQLCommand) (*SQLCommand, error) {
	// DELETE FROM table [WHERE condition]
	re := regexp.MustCompile(`(?i)DELETE\s+FROM\s+([^\s;]+)(?:\s+WHERE\s+(.*))?	`)
	matches := re.FindStringSubmatch(sql)

	if len(matches) < 2 {
		return nil, fmt.Errorf("DELETE 语法错误，正确格式: DELETE FROM table [WHERE condition]")
	}

	// 解析表名
	cmd.Table = strings.TrimSpace(matches[1])
	if err := common.ValidateNotEmpty(cmd.Table, "表名"); err != nil {
		return nil, err
	}

	// 解析 WHERE 条件
	if len(matches) > 2 && matches[2] != "" {
		whereClause := strings.TrimSpace(matches[2])
		cmd.Where = whereClause
		conditions, err := parseWhereClause(whereClause)
		if err != nil {
			return nil, common.FormatError("WHERE 条件解析", err)
		}
		cmd.Condition = conditions
	}

	return cmd, nil
}

// parseFieldList 解析字段列表
// 参数:
//   - fieldsStr: 字段列表字符串，如 "field1, field2, field3"
//
// 返回:
//   - []string: 解析后的字段列表
func parseFieldList(fieldsStr string) []string {
	if fieldsStr == "" {
		return nil
	}

	fields := strings.Split(fieldsStr, ",")
	result := make([]string, 0, len(fields))

	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			result = append(result, field)
		}
	}

	return result
}

// parseValueList 解析值列表
// 支持带引号的字符串值和数字值
// 参数:
//   - valuesStr: 值列表字符串，如 "'value1', 123, 'value2'"
//
// 返回:
//   - []interface{}: 解析后的值列表
//   - error: 解析错误
func parseValueList(valuesStr string) ([]interface{}, error) {
	if valuesStr == "" {
		return nil, fmt.Errorf("值列表不能为空")
	}

	var values []interface{}
	var current strings.Builder
	var inQuotes bool
	var quoteChar rune

	for _, char := range valuesStr {
		switch char {
		case '\'', '"':
			if !inQuotes {
				inQuotes = true
				quoteChar = char
			} else if char == quoteChar {
				inQuotes = false
			}
			// 不包含引号字符本身
		case ',':
			if !inQuotes {
				value, err := parseValue(strings.TrimSpace(current.String()))
				if err != nil {
					return nil, err
				}
				values = append(values, value)
				current.Reset()
			} else {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
	}

	// 添加最后一个值
	if current.Len() > 0 {
		value, err := parseValue(strings.TrimSpace(current.String()))
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}

	return values, nil
}

// parseValue 解析单个值
// 参数:
//   - valueStr: 值字符串
//
// 返回:
//   - interface{}: 解析后的值
//   - error: 解析错误
func parseValue(valueStr string) (interface{}, error) {
	if valueStr == "" {
		return nil, fmt.Errorf("值不能为空")
	}

	// 移除首尾引号
	if (strings.HasPrefix(valueStr, "'") && strings.HasSuffix(valueStr, "'")) ||
		(strings.HasPrefix(valueStr, "\"") && strings.HasSuffix(valueStr, "\"")) {
		return valueStr[1 : len(valueStr)-1], nil
	}

	// 尝试解析为数字
	if intVal, err := strconv.Atoi(valueStr); err == nil {
		return intVal, nil
	}

	if floatVal, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return floatVal, nil
	}

	// 解析布尔值
	if boolVal, err := strconv.ParseBool(valueStr); err == nil {
		return boolVal, nil
	}

	// 默认作为字符串处理
	return valueStr, nil
}

// parseSetClause 解析 SET 子句
// 参数:
//   - setClause: SET 子句字符串，如 "field1=value1, field2=value2"
//
// 返回:
//   - map[string]interface{}: 字段值映射
//   - error: 解析错误
func parseSetClause(setClause string) (map[string]interface{}, error) {
	if setClause == "" {
		return nil, fmt.Errorf("SET 子句不能为空")
	}

	result := make(map[string]interface{})
	setPairs := strings.Split(setClause, ",")

	for _, pair := range setPairs {
		pair = strings.TrimSpace(pair)
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("SET 子句格式错误: %s", pair)
		}

		field := strings.TrimSpace(parts[0])
		valueStr := strings.TrimSpace(parts[1])

		if field == "" {
			return nil, fmt.Errorf("字段名不能为空")
		}

		value, err := parseValue(valueStr)
		if err != nil {
			return nil, fmt.Errorf("解析字段 %s 的值失败: %w", field, err)
		}

		result[field] = value
	}

	return result, nil
}

// parseWhereClause 解析 WHERE 条件子句
// 参数:
//   - whereClause: WHERE 条件字符串
//
// 返回:
//   - map[string]interface{}: 条件映射
//   - error: 解析错误
func parseWhereClause(whereClause string) (map[string]interface{}, error) {
	if whereClause == "" {
		return nil, fmt.Errorf("WHERE 条件不能为空")
	}

	// 支持多种操作符：=, LIKE, >, <, >=, <=, !=
	// 优先匹配 LIKE 操作符（不区分大小写）
	likeRe := regexp.MustCompile(`(?i)([^\s]+)\s+LIKE\s+(.+)`)
	likeMatches := likeRe.FindStringSubmatch(whereClause)

	if len(likeMatches) >= 3 {
		field := strings.TrimSpace(likeMatches[1])
		valueStr := strings.TrimSpace(likeMatches[2])

		value, err := parseValue(valueStr)
		if err != nil {
			return nil, fmt.Errorf("解析 WHERE 条件值失败: %w", err)
		}

		// 为 LIKE 操作添加特殊标记
		return map[string]interface{}{
			field:                value,
			"_operator_" + field: "LIKE",
		}, nil
	}

	// 匹配其他比较操作符
	compareRe := regexp.MustCompile(`([^\s<>=!]+)\s*(>=|<=|!=|>|<|=)\s*(.+)`)
	compareMatches := compareRe.FindStringSubmatch(whereClause)

	if len(compareMatches) >= 4 {
		field := strings.TrimSpace(compareMatches[1])
		operator := strings.TrimSpace(compareMatches[2])
		valueStr := strings.TrimSpace(compareMatches[3])

		value, err := parseValue(valueStr)
		if err != nil {
			return nil, fmt.Errorf("解析 WHERE 条件值失败: %w", err)
		}

		result := map[string]interface{}{field: value}
		// 如果不是等号，添加操作符信息
		if operator != "=" {
			result["_operator_"+field] = operator
		}

		return result, nil
	}

	// 如果都不匹配，返回错误
	return nil, fmt.Errorf("WHERE 条件格式错误，支持的格式: field = value, field LIKE 'pattern', field > value, field < value, field >= value, field <= value, field != value")
}
