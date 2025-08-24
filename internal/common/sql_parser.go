package common

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// SQLParser 统一的SQL解析器
type SQLParser struct{}

// NewSQLParser 创建新的SQL解析器
func NewSQLParser() *SQLParser {
	return &SQLParser{}
}

// ParseValue 解析字符串值为适当的类型
func (p *SQLParser) ParseValue(valueStr string) interface{} {
	if valueStr == "" {
		return ""
	}

	// 移除首尾引号
	if (strings.HasPrefix(valueStr, "'") && strings.HasSuffix(valueStr, "'")) ||
		(strings.HasPrefix(valueStr, "\"") && strings.HasSuffix(valueStr, "\"")) {
		return valueStr[1 : len(valueStr)-1]
	}

	// 尝试解析为数字
	if intVal, err := strconv.Atoi(valueStr); err == nil {
		return intVal
	}

	if floatVal, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return floatVal
	}

	// 解析布尔值
	if boolVal, err := strconv.ParseBool(valueStr); err == nil {
		return boolVal
	}

	// 默认作为字符串处理
	return valueStr
}

// ParseSelectSQL 解析SELECT语句
func (p *SQLParser) ParseSelectSQL(sql string, cmd *SQLCommand) (*SQLCommand, error) {
	cmd.Type = CommandSelect

	// SELECT fields FROM table [WHERE condition] [LIMIT number]
	re := regexp.MustCompile(`(?i)SELECT\s+(.*?)\s+FROM\s+([^\s;]+)(?:\s+WHERE\s+(.*?))?(?:\s+LIMIT\s+(\d+))?(?:\s*;\s*)?$`)
	matches := re.FindStringSubmatch(sql)

	if len(matches) < 3 {
		return nil, fmt.Errorf("SELECT 语法错误，正确格式: SELECT fields FROM table [WHERE condition] [LIMIT number]")
	}

	// 解析字段列表
	fieldsStr := strings.TrimSpace(matches[1])
	if err := ValidateNotEmpty(fieldsStr, "SELECT 字段列表"); err != nil {
		return nil, err
	}

	// 检查是否包含聚合函数
	aggregateRe := regexp.MustCompile(`(?i)(COUNT|SUM|AVG|MIN|MAX)\s*\(\s*(\*|[^\)]+)\s*\)`)
	aggregateMatches := aggregateRe.FindStringSubmatch(fieldsStr)

	if len(aggregateMatches) >= 3 {
		// 这是一个聚合查询
		cmd.IsAggregate = true
		cmd.AggregateFunction = strings.ToUpper(aggregateMatches[1])
		cmd.AggregateField = strings.TrimSpace(aggregateMatches[2])
		cmd.Fields = []string{fieldsStr}
	} else if fieldsStr == "*" {
		cmd.Fields = []string{"*"}
	} else {
		fields := p.parseFieldList(fieldsStr)
		if len(fields) == 0 {
			return nil, fmt.Errorf("SELECT 字段列表解析失败")
		}
		cmd.Fields = fields
	}

	// 解析表名
	cmd.Table = strings.TrimSpace(matches[2])
	if err := ValidateNotEmpty(cmd.Table, "表名"); err != nil {
		return nil, err
	}

	// 解析 WHERE 条件
	if len(matches) > 3 && matches[3] != "" {
		whereClause := strings.TrimSpace(matches[3])
		cmd.Where = whereClause
		conditions, err := p.parseWhereClause(whereClause)
		if err != nil {
			return nil, fmt.Errorf("WHERE 条件解析失败: %w", err)
		}
		cmd.Condition = conditions
	}

	// 解析 LIMIT 子句
	if len(matches) > 4 && matches[4] != "" {
		limitStr := strings.TrimSpace(matches[4])
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			return nil, fmt.Errorf("LIMIT 值必须是数字: %s", limitStr)
		}
		if limit < 0 {
			return nil, fmt.Errorf("LIMIT 值不能为负数: %d", limit)
		}
		cmd.Limit = limit
	}

	return cmd, nil
}

// ParseInsertSQL 解析INSERT语句
func (p *SQLParser) ParseInsertSQL(sql string, cmd *SQLCommand) (*SQLCommand, error) {
	cmd.Type = CommandInsert

	// INSERT INTO table (field1, field2) VALUES (value1, value2)
	re := regexp.MustCompile(`(?i)INSERT\s+INTO\s+([^\s\(;]+)\s*\((.*?)\)\s*VALUES\s*\((.*?)\)`)
	matches := re.FindStringSubmatch(sql)

	if len(matches) < 4 {
		return nil, fmt.Errorf("INSERT 语法错误，正确格式: INSERT INTO table (field1, field2) VALUES (value1, value2)")
	}

	// 解析表名
	cmd.Table = strings.TrimSpace(matches[1])
	if err := ValidateNotEmpty(cmd.Table, "表名"); err != nil {
		return nil, err
	}

	// 解析字段列表
	fieldsStr := strings.TrimSpace(matches[2])
	if err := ValidateNotEmpty(fieldsStr, "字段列表"); err != nil {
		return nil, err
	}

	fields := p.parseFieldList(fieldsStr)
	if len(fields) == 0 {
		return nil, fmt.Errorf("字段列表解析失败")
	}
	cmd.Fields = fields

	// 解析值列表
	valuesStr := strings.TrimSpace(matches[3])
	if err := ValidateNotEmpty(valuesStr, "值列表"); err != nil {
		return nil, err
	}

	values := p.parseValueList(valuesStr)
	if len(fields) != len(values) {
		return nil, fmt.Errorf("字段数量(%d)与值数量(%d)不匹配", len(fields), len(values))
	}

	// 将字段和值组合成映射
	for i, field := range fields {
		cmd.Values[field] = values[i]
	}

	return cmd, nil
}

// ParseUpdateSQL 解析UPDATE语句
func (p *SQLParser) ParseUpdateSQL(sql string, cmd *SQLCommand) (*SQLCommand, error) {
	cmd.Type = CommandUpdate

	// UPDATE table SET field1=value1, field2=value2 [WHERE condition]
	var re *regexp.Regexp
	var matches []string

	if strings.Contains(strings.ToUpper(sql), " WHERE ") {
		re = regexp.MustCompile(`(?i)UPDATE\s+([^\s;]+)\s+SET\s+(.*?)\s+WHERE\s+(.*)`)
		matches = re.FindStringSubmatch(sql)
	} else {
		re = regexp.MustCompile(`(?i)UPDATE\s+([^\s;]+)\s+SET\s+(.*)`)
		matches = re.FindStringSubmatch(sql)
	}

	if len(matches) < 3 {
		return nil, fmt.Errorf("UPDATE 语法错误，正确格式: UPDATE table SET field1=value1 [WHERE condition]")
	}

	// 解析表名
	cmd.Table = strings.TrimSpace(matches[1])
	if err := ValidateNotEmpty(cmd.Table, "表名"); err != nil {
		return nil, err
	}

	// 解析 SET 子句
	setClause := strings.TrimSpace(matches[2])
	if err := ValidateNotEmpty(setClause, "SET 子句"); err != nil {
		return nil, err
	}

	updateFields, err := p.parseSetClause(setClause)
	if err != nil {
		return nil, fmt.Errorf("SET 子句解析失败: %w", err)
	}
	cmd.UpdateFields = updateFields
	// 同时设置到 Values 中以保持兼容性
	for k, v := range updateFields {
		cmd.SetValue(k, v)
	}

	// 解析 WHERE 条件
	if len(matches) > 3 && matches[3] != "" {
		whereClause := strings.TrimSpace(matches[3])
		cmd.Where = whereClause
		conditions, err := p.parseWhereClause(whereClause)
		if err != nil {
			return nil, fmt.Errorf("WHERE 条件解析失败: %w", err)
		}
		cmd.Condition = conditions
	}

	return cmd, nil
}

// ParseDeleteSQL 解析DELETE语句
func (p *SQLParser) ParseDeleteSQL(sql string, cmd *SQLCommand) (*SQLCommand, error) {
	cmd.Type = CommandDelete

	// DELETE FROM table [WHERE condition]
	re := regexp.MustCompile(`(?i)DELETE\s+FROM\s+(\w+)(?:\s+WHERE\s+(.+))?`)
	matches := re.FindStringSubmatch(sql)

	if len(matches) < 2 {
		return nil, fmt.Errorf("DELETE 语法错误，正确格式: DELETE FROM table [WHERE condition]")
	}

	cmd.Table = matches[1]
	if err := ValidateNotEmpty(cmd.Table, "表名"); err != nil {
		return nil, err
	}

	if len(matches) > 2 && matches[2] != "" {
		whereClause := strings.TrimSpace(matches[2])
		cmd.Where = whereClause
		conditions, err := p.parseWhereClause(whereClause)
		if err != nil {
			return nil, fmt.Errorf("WHERE 条件解析失败: %w", err)
		}
		cmd.Condition = conditions
	}

	return cmd, nil
}

// parseFieldList 解析字段列表
func (p *SQLParser) parseFieldList(fieldsStr string) []string {
	var fields []string
	var current strings.Builder
	inQuotes := false
	quoteChar := byte(0)

	for i := 0; i < len(fieldsStr); i++ {
		char := fieldsStr[i]

		if !inQuotes {
			if char == '\'' || char == '"' {
				inQuotes = true
				quoteChar = char
			} else if char == ',' {
				field := strings.TrimSpace(current.String())
				if field != "" {
					fields = append(fields, field)
				}
				current.Reset()
				continue
			}
		} else {
			if char == quoteChar {
				inQuotes = false
				quoteChar = 0
			}
		}

		current.WriteByte(char)
	}

	// 添加最后一个字段
	if current.Len() > 0 {
		field := strings.TrimSpace(current.String())
		if field != "" {
			fields = append(fields, field)
		}
	}

	return fields
}

// parseValueList 解析值列表
func (p *SQLParser) parseValueList(valuesStr string) []interface{} {
	var values []interface{}
	var current strings.Builder
	inQuotes := false
	quoteChar := byte(0)

	for i := 0; i < len(valuesStr); i++ {
		char := valuesStr[i]

		if !inQuotes {
			if char == '\'' || char == '"' {
				inQuotes = true
				quoteChar = char
				continue
			} else if char == ',' {
				value := strings.TrimSpace(current.String())
				values = append(values, p.parseValue(value))
				current.Reset()
				continue
			}
		} else {
			if char == quoteChar {
				inQuotes = false
				quoteChar = 0
				continue
			}
		}

		current.WriteByte(char)
	}

	// 添加最后一个值
	if current.Len() > 0 {
		value := strings.TrimSpace(current.String())
		values = append(values, p.parseValue(value))
	}

	return values
}

// parseValue 解析单个值
func (p *SQLParser) parseValue(value string) interface{} {
	value = strings.TrimSpace(value)

	// 检查是否为 NULL
	if strings.ToUpper(value) == "NULL" {
		return nil
	}

	// 尝试解析为数字
	if num, err := strconv.ParseFloat(value, 64); err == nil {
		return num
	}

	// 尝试解析为布尔值
	if b, err := strconv.ParseBool(value); err == nil {
		return b
	}

	// 默认作为字符串
	return value
}

// parseSetClause 解析SET子句
func (p *SQLParser) parseSetClause(setClause string) (map[string]interface{}, error) {
	updateFields := make(map[string]interface{})
	assignments := strings.Split(setClause, ",")

	for _, assignment := range assignments {
		assignment = strings.TrimSpace(assignment)
		parts := strings.SplitN(assignment, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("无效的赋值表达式: %s", assignment)
		}

		field := strings.TrimSpace(parts[0])
		valueStr := strings.TrimSpace(parts[1])

		// 移除值周围的引号
		if (strings.HasPrefix(valueStr, "'") && strings.HasSuffix(valueStr, "'")) ||
			(strings.HasPrefix(valueStr, "\"") && strings.HasSuffix(valueStr, "\"")) {
			valueStr = valueStr[1 : len(valueStr)-1]
		}

		updateFields[field] = p.parseValue(valueStr)
	}

	return updateFields, nil
}

// parseWhereClause 解析WHERE子句
func (p *SQLParser) parseWhereClause(whereClause string) (map[string]interface{}, error) {
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

		// 移除值周围的引号
		if (strings.HasPrefix(valueStr, "'") && strings.HasSuffix(valueStr, "'")) ||
			(strings.HasPrefix(valueStr, "\"") && strings.HasSuffix(valueStr, "\"")) {
			valueStr = valueStr[1 : len(valueStr)-1]
		}

		// 为 LIKE 操作添加特殊标记
		return map[string]interface{}{
			field:                p.parseValue(valueStr),
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

		// 移除值周围的引号
		if (strings.HasPrefix(valueStr, "'") && strings.HasSuffix(valueStr, "'")) ||
			(strings.HasPrefix(valueStr, "\"") && strings.HasSuffix(valueStr, "\"")) {
			valueStr = valueStr[1 : len(valueStr)-1]
		}

		result := map[string]interface{}{field: p.parseValue(valueStr)}
		// 如果不是等号，添加操作符信息
		if operator != "=" {
			result["_operator_"+field] = operator
		}

		return result, nil
	}

	// 如果都不匹配，返回错误
	return nil, fmt.Errorf("WHERE 条件格式错误，支持的格式: field = value, field LIKE 'pattern', field > value, field < value, field >= value, field <= value, field != value")
}

// DefaultSQLParser 默认的SQL解析器实例
var DefaultSQLParser = NewSQLParser()
