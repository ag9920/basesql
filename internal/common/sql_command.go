package common

import (
	"fmt"
	"strings"
)

// SQLCommandType SQL 命令类型枚举
type SQLCommandType string

const (
	// CommandSelect SELECT 查询命令
	CommandSelect SQLCommandType = "SELECT"
	// CommandInsert INSERT 插入命令
	CommandInsert SQLCommandType = "INSERT"
	// CommandUpdate UPDATE 更新命令
	CommandUpdate SQLCommandType = "UPDATE"
	// CommandDelete DELETE 删除命令
	CommandDelete SQLCommandType = "DELETE"
	// CommandShow SHOW 显示命令
	CommandShow SQLCommandType = "SHOW"
	// CommandCreate CREATE 创建命令
	CommandCreate SQLCommandType = "CREATE"
	// CommandDrop DROP 删除命令
	CommandDrop SQLCommandType = "DROP"
	// CommandDescribe DESCRIBE 描述命令
	CommandDescribe SQLCommandType = "DESCRIBE"
	// CommandUnknown 未知命令
	CommandUnknown SQLCommandType = "UNKNOWN"
)

// String 返回命令类型的字符串表示
func (c SQLCommandType) String() string {
	return string(c)
}

// IsValid 检查命令类型是否有效
func (c SQLCommandType) IsValid() bool {
	switch c {
	case CommandSelect, CommandInsert, CommandUpdate, CommandDelete,
		CommandShow, CommandCreate, CommandDrop, CommandDescribe:
		return true
	default:
		return false
	}
}

// SQLCommand 统一的 SQL 命令结构体
// 用于表示解析后的 SQL 命令及其相关信息
type SQLCommand struct {
	// Type 命令类型
	Type SQLCommandType `json:"type"`

	// Table 目标表名
	Table string `json:"table,omitempty"`

	// Fields 字段列表（用于 SELECT、INSERT 等）
	Fields []string `json:"fields,omitempty"`

	// Values 值映射（用于 INSERT、UPDATE）
	// 键为字段名，值为对应的值
	Values map[string]interface{} `json:"values,omitempty"`

	// Condition 查询条件（用于 SELECT、UPDATE、DELETE）
	Condition map[string]interface{} `json:"condition,omitempty"`

	// UpdateFields 更新字段映射（用于 UPDATE，与 Values 功能重叠但保持兼容性）
	UpdateFields map[string]interface{} `json:"update_fields,omitempty"`

	// Where WHERE 条件字符串（原始格式）
	Where string `json:"where,omitempty"`

	// RawSQL 原始 SQL 语句
	RawSQL string `json:"raw_sql,omitempty"`

	// ShowType SHOW 命令的子类型（TABLES、COLUMNS 等）
	ShowType string `json:"show_type,omitempty"`

	// OrderBy 排序字段
	OrderBy []string `json:"order_by,omitempty"`

	// Limit 限制返回记录数
	Limit int `json:"limit,omitempty"`

	// Offset 偏移量
	Offset int `json:"offset,omitempty"`

	// AggregateFunction 聚合函数信息
	AggregateFunction string `json:"aggregate_function,omitempty"`

	// AggregateField 聚合函数作用的字段
	AggregateField string `json:"aggregate_field,omitempty"`

	// IsAggregate 是否是聚合查询
	IsAggregate bool `json:"is_aggregate,omitempty"`
}

// NewSQLCommand 创建新的 SQL 命令
// 参数:
//   - cmdType: 命令类型
//
// 返回:
//   - *SQLCommand: 新的 SQL 命令实例
func NewSQLCommand(cmdType SQLCommandType) *SQLCommand {
	return &SQLCommand{
		Type:         cmdType,
		Values:       make(map[string]interface{}),
		Condition:    make(map[string]interface{}),
		UpdateFields: make(map[string]interface{}),
		Fields:       make([]string, 0),
		OrderBy:      make([]string, 0),
	}
}

// Validate 验证 SQL 命令的有效性
// 返回:
//   - error: 验证错误，如果验证通过返回 nil
func (cmd *SQLCommand) Validate() error {
	// 验证命令类型
	if !cmd.Type.IsValid() {
		return fmt.Errorf("无效的命令类型: %s", cmd.Type)
	}

	// 根据命令类型进行特定验证
	switch cmd.Type {
	case CommandSelect:
		return cmd.validateSelect()
	case CommandInsert:
		return cmd.validateInsert()
	case CommandUpdate:
		return cmd.validateUpdate()
	case CommandDelete:
		return cmd.validateDelete()
	case CommandShow:
		return cmd.validateShow()
	case CommandCreate:
		return cmd.validateCreate()
	case CommandDrop:
		return cmd.validateDrop()
	case CommandDescribe:
		return cmd.validateDescribe()
	default:
		return fmt.Errorf("不支持的命令类型: %s", cmd.Type)
	}
}

// validateSelect 验证 SELECT 命令
func (cmd *SQLCommand) validateSelect() error {
	if err := ValidateNotEmpty(cmd.Table, "表名"); err != nil {
		return err
	}
	if len(cmd.Fields) == 0 {
		return NewValidationError("字段列表", "不能为空")
	}
	return nil
}

// validateInsert 验证 INSERT 命令
func (cmd *SQLCommand) validateInsert() error {
	if err := ValidateNotEmpty(cmd.Table, "表名"); err != nil {
		return err
	}
	if len(cmd.Values) == 0 {
		return NewValidationError("插入值", "不能为空")
	}
	return nil
}

// validateUpdate 验证 UPDATE 命令
func (cmd *SQLCommand) validateUpdate() error {
	if err := ValidateNotEmpty(cmd.Table, "表名"); err != nil {
		return err
	}
	if len(cmd.Values) == 0 && len(cmd.UpdateFields) == 0 {
		return NewValidationError("更新字段", "不能为空")
	}
	return nil
}

// validateDelete 验证 DELETE 命令
func (cmd *SQLCommand) validateDelete() error {
	if err := ValidateNotEmpty(cmd.Table, "表名"); err != nil {
		return err
	}
	// DELETE 命令可以没有 WHERE 条件（删除所有记录），但应该给出警告
	return nil
}

// validateShow 验证 SHOW 命令
func (cmd *SQLCommand) validateShow() error {
	if cmd.ShowType == "" {
		return NewValidationError("SHOW 类型", "不能为空")
	}

	// 对于 SHOW COLUMNS，需要表名
	if strings.ToUpper(cmd.ShowType) == "COLUMNS" {
		if err := ValidateNotEmpty(cmd.Table, "表名"); err != nil {
			return err
		}
	}
	return nil
}

// validateCreate 验证 CREATE 命令
func (cmd *SQLCommand) validateCreate() error {
	if err := ValidateNotEmpty(cmd.Table, "表名"); err != nil {
		return err
	}
	return nil
}

// validateDrop 验证 DROP 命令
func (cmd *SQLCommand) validateDrop() error {
	if err := ValidateNotEmpty(cmd.Table, "表名"); err != nil {
		return err
	}
	return nil
}

// validateDescribe 验证 DESCRIBE 命令
func (cmd *SQLCommand) validateDescribe() error {
	if err := ValidateNotEmpty(cmd.Table, "表名"); err != nil {
		return err
	}
	return nil
}

// HasWhere 检查是否有 WHERE 条件
// 返回:
//   - bool: 是否有 WHERE 条件
func (cmd *SQLCommand) HasWhere() bool {
	return cmd.Where != "" || len(cmd.Condition) > 0
}

// GetEffectiveValues 获取有效的值映射
// 优先使用 Values，如果为空则使用 UpdateFields
// 返回:
//   - map[string]interface{}: 有效的值映射
func (cmd *SQLCommand) GetEffectiveValues() map[string]interface{} {
	if len(cmd.Values) > 0 {
		return cmd.Values
	}
	return cmd.UpdateFields
}

// SetValue 设置字段值
// 参数:
//   - field: 字段名
//   - value: 字段值
func (cmd *SQLCommand) SetValue(field string, value interface{}) {
	if cmd.Values == nil {
		cmd.Values = make(map[string]interface{})
	}
	cmd.Values[field] = value
}

// GetValue 获取字段值
// 参数:
//   - field: 字段名
//
// 返回:
//   - interface{}: 字段值
//   - bool: 是否存在
func (cmd *SQLCommand) GetValue(field string) (interface{}, bool) {
	value, exists := cmd.Values[field]
	if !exists {
		value, exists = cmd.UpdateFields[field]
	}
	return value, exists
}

// SetCondition 设置查询条件
// 参数:
//   - field: 字段名
//   - value: 条件值
func (cmd *SQLCommand) SetCondition(field string, value interface{}) {
	if cmd.Condition == nil {
		cmd.Condition = make(map[string]interface{})
	}
	cmd.Condition[field] = value
}

// GetCondition 获取查询条件
// 参数:
//   - field: 字段名
//
// 返回:
//   - interface{}: 条件值
//   - bool: 是否存在
func (cmd *SQLCommand) GetCondition(field string) (interface{}, bool) {
	value, exists := cmd.Condition[field]
	return value, exists
}

// AddField 添加字段
// 参数:
//   - field: 字段名
func (cmd *SQLCommand) AddField(field string) {
	if !Contains(cmd.Fields, field) {
		cmd.Fields = append(cmd.Fields, field)
	}
}

// HasField 检查是否包含指定字段
// 参数:
//   - field: 字段名
//
// 返回:
//   - bool: 是否包含
func (cmd *SQLCommand) HasField(field string) bool {
	return Contains(cmd.Fields, field)
}

// IsSelectAll 检查是否是 SELECT * 查询
// 返回:
//   - bool: 是否是 SELECT * 查询
func (cmd *SQLCommand) IsSelectAll() bool {
	return cmd.Type == CommandSelect && len(cmd.Fields) == 1 && cmd.Fields[0] == "*"
}

// String 返回命令的字符串表示
// 返回:
//   - string: 命令的字符串表示
func (cmd *SQLCommand) String() string {
	if cmd.RawSQL != "" {
		return cmd.RawSQL
	}

	switch cmd.Type {
	case CommandSelect:
		fields := "*"
		if len(cmd.Fields) > 0 && !cmd.IsSelectAll() {
			fields = strings.Join(cmd.Fields, ", ")
		}
		sql := fmt.Sprintf("SELECT %s FROM %s", fields, cmd.Table)
		if cmd.HasWhere() {
			sql += " WHERE " + cmd.Where
		}
		return sql

	case CommandInsert:
		fields := make([]string, 0, len(cmd.Values))
		values := make([]string, 0, len(cmd.Values))
		for field, value := range cmd.Values {
			fields = append(fields, field)
			values = append(values, fmt.Sprintf("'%v'", value))
		}
		return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			cmd.Table, strings.Join(fields, ", "), strings.Join(values, ", "))

	case CommandUpdate:
		effectiveValues := cmd.GetEffectiveValues()
		setClauses := make([]string, 0, len(effectiveValues))
		for field, value := range effectiveValues {
			setClauses = append(setClauses, fmt.Sprintf("%s='%v'", field, value))
		}
		sql := fmt.Sprintf("UPDATE %s SET %s", cmd.Table, strings.Join(setClauses, ", "))
		if cmd.HasWhere() {
			sql += " WHERE " + cmd.Where
		}
		return sql

	case CommandDelete:
		sql := fmt.Sprintf("DELETE FROM %s", cmd.Table)
		if cmd.HasWhere() {
			sql += " WHERE " + cmd.Where
		}
		return sql

	default:
		return fmt.Sprintf("%s %s", cmd.Type, cmd.Table)
	}
}
