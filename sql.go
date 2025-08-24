package basesql

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ag9920/basesql/internal/common"
)

// SQLConverter SQL 转换器
// 负责将 GORM 的 SQL 语句转换为飞书多维表格 API 调用
// 支持 CREATE、SELECT、UPDATE、DELETE 等基本操作
type SQLConverter struct {
	client *Client // 飞书 API 客户端
	config *Config // 配置信息
}

// NewSQLConverter 创建 SQL 转换器实例
// 参数:
//   - client: 已初始化的飞书 API 客户端
//   - config: 配置信息
//
// 返回:
//   - *SQLConverter: SQL 转换器实例
func NewSQLConverter(client *Client, config *Config) *SQLConverter {
	if client == nil {
		panic("client 不能为 nil")
	}
	if config == nil {
		panic("config 不能为 nil")
	}
	return &SQLConverter{
		client: client,
		config: config,
	}
}

// ConvertCreate 转换 CREATE 语句为飞书多维表格的创建记录 API 调用
// 参数:
//   - ctx: 上下文，用于控制请求超时和取消
//   - stmt: GORM 语句对象，包含要创建的数据信息
//
// 返回:
//   - error: 转换或执行过程中的错误
func (c *SQLConverter) ConvertCreate(ctx context.Context, stmt *gorm.Statement) error {
	if stmt == nil {
		return fmt.Errorf("GORM 语句不能为 nil")
	}
	if stmt.Table == "" {
		return fmt.Errorf("表名不能为空")
	}

	// 获取表名
	tableName := stmt.Table

	// 获取表的字段信息
	dialector := &Dialector{
		Client: c.client,
		Config: c.config,
	}
	tableFields, err := getTableFields(dialector, tableName)
	if err != nil {
		return fmt.Errorf("获取表字段信息失败: %w", err)
	}

	// 创建字段名到字段信息的映射，提高查找效率
	fieldMap := make(map[string]*Field, len(tableFields))
	for _, f := range tableFields {
		fieldMap[f.FieldName] = f
	}

	// 获取字段值并进行类型转换
	fields := make(map[string]interface{})
	for _, field := range stmt.Schema.Fields {
		value, ok := field.ValueOf(ctx, stmt.ReflectValue)
		if !ok || value == nil {
			continue
		}

		// 使用字段的转换方法进行类型转换
		if tableField, exists := fieldMap[field.DBName]; exists {
			convertedValue := tableField.ConvertFromGoValue(value)
			if convertedValue != nil {
				fields[field.DBName] = convertedValue
			}
		} else {
			// 如果找不到字段信息，直接使用原值（向后兼容）
			fields[field.DBName] = value
		}
	}

	// 检查是否有字段需要创建
	if len(fields) == 0 {
		return fmt.Errorf("没有找到需要创建的字段")
	}

	// 创建记录请求
	req := &CreateRecordRequest{
		Fields: fields,
	}

	// 调用飞书 API
	apiReq := &APIRequest{
		Method: "POST",
		Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records", c.config.AppToken, tableName),
		Body:   req,
	}

	resp, err := c.client.DoRequest(ctx, apiReq)
	if err != nil {
		return fmt.Errorf("创建记录 API 调用失败: %w", err)
	}

	// 解析响应
	var createResp CreateRecordResponse
	if err := json.Unmarshal(resp.Body, &createResp); err != nil {
		return fmt.Errorf("解析创建记录响应失败: %w", err)
	}

	// 设置主键值（记录 ID）
	if stmt.Schema.PrioritizedPrimaryField != nil {
		if err := stmt.Schema.PrioritizedPrimaryField.Set(ctx, stmt.ReflectValue, createResp.Record.RecordID); err != nil {
			return fmt.Errorf("设置主键值失败: %w", err)
		}
	}

	return nil
}

// ConvertQuery 转换 SELECT 语句为飞书多维表格的查询记录 API 调用
// 支持字段选择、WHERE 条件过滤、ORDER BY 排序等功能
// 参数:
//   - ctx: 上下文，用于控制请求超时和取消
//   - stmt: GORM 语句对象，包含查询条件和字段信息
//
// 返回:
//   - error: 转换或执行过程中的错误
func (c *SQLConverter) ConvertQuery(ctx context.Context, stmt *gorm.Statement) error {
	if err := common.ValidateStatement(stmt); err != nil {
		return err
	}

	// 获取表名
	tableName := stmt.Table

	// 构建查询请求
	req := &ListRecordsRequest{
		FieldNames: make([]string, 0, len(stmt.Schema.Fields)),
	}

	// 添加需要查询的字段名
	for _, field := range stmt.Schema.Fields {
		if field.DBName != "" {
			req.FieldNames = append(req.FieldNames, field.DBName)
		}
	}

	// 处理 WHERE 条件
	if whereClause, ok := stmt.Clauses["WHERE"]; ok {
		if where, ok := whereClause.Expression.(clause.Where); ok && len(where.Exprs) > 0 {
			filter := c.buildFilter(where.Exprs)
			if filter != nil {
				req.Filter = filter
			}
		}
	}

	// 处理 ORDER BY 排序
	if orderBy, ok := stmt.Clauses["ORDER BY"]; ok {
		if orderExprs, ok := orderBy.Expression.(clause.OrderBy); ok {
			sort := make([]string, 0, len(orderExprs.Columns))
			for _, expr := range orderExprs.Columns {
				columnName := expr.Column.Name
				if columnName != "" {
					if expr.Desc {
						sort = append(sort, fmt.Sprintf("-%s", columnName))
					} else {
						sort = append(sort, columnName)
					}
				}
			}
			if len(sort) > 0 {
				req.Sort = sort
			}
		}
	}

	// 处理 LIMIT 限制
	if limitClause, ok := stmt.Clauses["LIMIT"]; ok {
		if limit, ok := limitClause.Expression.(clause.Limit); ok {
			if limit.Limit != nil {
				req.PageSize = int(*limit.Limit)
			}
		}
	}

	// 调用飞书 API
	apiReq := &APIRequest{
		Method: "GET",
		Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records", c.config.AppToken, tableName),
		Body:   req,
	}

	resp, err := c.client.DoRequest(ctx, apiReq)
	if err != nil {
		return fmt.Errorf("查询记录 API 调用失败: %w", err)
	}

	// 解析响应
	var listResp ListRecordsResponse
	if err := json.Unmarshal(resp.Body, &listResp); err != nil {
		return fmt.Errorf("解析查询记录响应失败: %w", err)
	}

	// 设置查询结果到 GORM 语句中
	for _, record := range listResp.Items {
		result := reflect.New(stmt.Schema.ModelType).Elem()
		for _, field := range stmt.Schema.Fields {
			if value, ok := record.Fields[field.DBName]; ok {
				if err := field.Set(ctx, result, value); err != nil {
					return fmt.Errorf("设置字段 %s 值失败: %w", field.DBName, err)
				}
			}
		}

		// 这里可能需要使用 stmt.Dest 或其他方式来设置查询结果
	}

	return nil
}

// ConvertUpdate 转换 UPDATE 语句为飞书多维表格的更新记录 API 调用
// 参数:
//   - ctx: 上下文，用于控制请求超时和取消
//   - stmt: GORM 语句对象，包含更新条件和字段信息
//
// 返回:
//   - error: 转换或执行过程中的错误
func (c *SQLConverter) ConvertUpdate(ctx context.Context, stmt *gorm.Statement) error {
	if err := common.ValidateStatement(stmt); err != nil {
		return err
	}

	// 获取表名和记录 ID
	tableName := stmt.Table

	// 获取记录 ID
	recordID, err := common.GetRecordID(ctx, stmt)
	if err != nil {
		return err
	}

	// 获取更新字段
	fields := make(map[string]interface{})
	if setClause, ok := stmt.Clauses["SET"]; ok {
		if set, ok := setClause.Expression.(clause.Set); ok {
			for _, assign := range set {
				if assign.Column.Name != "" {
					fields[assign.Column.Name] = assign.Value
				}
			}
		}
	}

	// 检查是否有字段需要更新
	if len(fields) == 0 {
		return fmt.Errorf("没有找到需要更新的字段")
	}

	// 更新记录请求
	req := &UpdateRecordRequest{
		Fields: fields,
	}

	// 调用飞书 API
	apiReq := &APIRequest{
		Method: "PUT",
		Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records/%s", c.config.AppToken, tableName, recordID),
		Body:   req,
	}

	_, err = c.client.DoRequest(ctx, apiReq)
	if err != nil {
		return fmt.Errorf("更新记录 API 调用失败: %w", err)
	}

	return nil
}

// ConvertDelete 转换 DELETE 语句为飞书多维表格的删除记录 API 调用
// 参数:
//   - ctx: 上下文，用于控制请求超时和取消
//   - stmt: GORM 语句对象，包含删除条件信息
//
// 返回:
//   - error: 转换或执行过程中的错误
func (c *SQLConverter) ConvertDelete(ctx context.Context, stmt *gorm.Statement) error {
	if err := common.ValidateStatement(stmt); err != nil {
		return err
	}

	// 获取表名和记录 ID
	tableName := stmt.Table

	// 获取记录 ID
	recordID, err := common.GetRecordID(ctx, stmt)
	if err != nil {
		return err
	}

	// 调用飞书 API
	apiReq := &APIRequest{
		Method: "DELETE",
		Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records/%s", c.config.AppToken, tableName, recordID),
	}

	_, err = c.client.DoRequest(ctx, apiReq)
	if err != nil {
		return fmt.Errorf("删除记录 API 调用失败: %w", err)
	}

	return nil
}

// buildFilter 构建过滤条件，将 GORM 的 WHERE 条件转换为飞书多维表格的过滤条件
// 支持的操作符：=、!=、>、>=、<、<=、LIKE
// 参数:
//   - exprs: GORM 的条件表达式列表
//
// 返回:
//   - *FilterRequest: 飞书多维表格的过滤请求，如果没有有效条件则返回 nil
func (c *SQLConverter) buildFilter(exprs []clause.Expression) *FilterRequest {
	if len(exprs) == 0 {
		return nil
	}

	var conditions []*FilterCondition

	for _, expr := range exprs {
		switch e := expr.(type) {
		case clause.Eq:
			if condition := c.buildEqualCondition(e); condition != nil {
				conditions = append(conditions, condition)
			}
		case clause.Neq:
			if condition := c.buildNotEqualCondition(e); condition != nil {
				conditions = append(conditions, condition)
			}
		case clause.Gt:
			if condition := c.buildGreaterThanCondition(e); condition != nil {
				conditions = append(conditions, condition)
			}
		case clause.Gte:
			if condition := c.buildGreaterEqualCondition(e); condition != nil {
				conditions = append(conditions, condition)
			}
		case clause.Lt:
			if condition := c.buildLessThanCondition(e); condition != nil {
				conditions = append(conditions, condition)
			}
		case clause.Lte:
			if condition := c.buildLessEqualCondition(e); condition != nil {
				conditions = append(conditions, condition)
			}
		case clause.Like:
			if condition := c.buildLikeCondition(e); condition != nil {
				conditions = append(conditions, condition)
			}
		case clause.IN:
			if condition := c.buildInCondition(e); condition != nil {
				conditions = append(conditions, condition)
			}
		case clause.Expr:
			// 处理 clause.Expr 类型，如 "active = ?" 这样的表达式
			if condition := c.buildExprCondition(e); condition != nil {
				conditions = append(conditions, condition)
			}
		}
	}

	if len(conditions) == 0 {
		return nil
	}

	return &FilterRequest{
		Conjunction: "and", // 默认使用 AND 连接多个条件
		Conditions:  conditions,
	}
}

// buildEqualCondition 构建等于条件
func (c *SQLConverter) buildEqualCondition(eq clause.Eq) *FilterCondition {
	column, ok := eq.Column.(clause.Column)
	if !ok || column.Name == "" {
		return nil
	}

	// 对于布尔值，使用实际的布尔值而不是字符串
	var value []interface{}
	if b, ok := eq.Value.(bool); ok {
		// 布尔值保持原始类型
		value = []interface{}{b}
	} else {
		value = []interface{}{eq.Value}
	}

	return &FilterCondition{
		FieldName: column.Name,
		Operator:  "is",
		Value:     value,
	}
}

// buildNotEqualCondition 构建不等于条件
func (c *SQLConverter) buildNotEqualCondition(neq clause.Neq) *FilterCondition {
	column, ok := neq.Column.(clause.Column)
	if !ok || column.Name == "" {
		return nil
	}

	// 对于布尔值，使用实际的布尔值而不是字符串
	var value []interface{}
	if b, ok := neq.Value.(bool); ok {
		// 布尔值保持原始类型
		value = []interface{}{b}
	} else {
		value = []interface{}{neq.Value}
	}

	return &FilterCondition{
		FieldName: column.Name,
		Operator:  "isNot",
		Value:     value,
	}
}

// buildGreaterThanCondition 构建大于条件
func (c *SQLConverter) buildGreaterThanCondition(gt clause.Gt) *FilterCondition {
	column, ok := gt.Column.(clause.Column)
	if !ok || column.Name == "" {
		return nil
	}
	return &FilterCondition{
		FieldName: column.Name,
		Operator:  "isGreater",
		Value:     []interface{}{gt.Value},
	}
}

// buildGreaterEqualCondition 构建大于等于条件
func (c *SQLConverter) buildGreaterEqualCondition(gte clause.Gte) *FilterCondition {
	column, ok := gte.Column.(clause.Column)
	if !ok || column.Name == "" {
		return nil
	}
	return &FilterCondition{
		FieldName: column.Name,
		Operator:  "isGreaterEqual",
		Value:     []interface{}{gte.Value},
	}
}

// buildLessThanCondition 构建小于条件
func (c *SQLConverter) buildLessThanCondition(lt clause.Lt) *FilterCondition {
	column, ok := lt.Column.(clause.Column)
	if !ok || column.Name == "" {
		return nil
	}
	return &FilterCondition{
		FieldName: column.Name,
		Operator:  "isLess",
		Value:     []interface{}{lt.Value},
	}
}

// buildLessEqualCondition 构建小于等于条件
func (c *SQLConverter) buildLessEqualCondition(lte clause.Lte) *FilterCondition {
	column, ok := lte.Column.(clause.Column)
	if !ok || column.Name == "" {
		return nil
	}
	return &FilterCondition{
		FieldName: column.Name,
		Operator:  "isLessEqual",
		Value:     []interface{}{lte.Value},
	}
}

// buildLikeCondition 构建模糊匹配条件
func (c *SQLConverter) buildLikeCondition(like clause.Like) *FilterCondition {
	column, ok := like.Column.(clause.Column)
	if !ok || column.Name == "" {
		return nil
	}
	return &FilterCondition{
		FieldName: column.Name,
		Operator:  "contains",
		Value:     []interface{}{like.Value},
	}
}

// buildInCondition 构建IN条件
func (c *SQLConverter) buildInCondition(in clause.IN) *FilterCondition {
	column, ok := in.Column.(clause.Column)
	if !ok || column.Name == "" {
		return nil
	}

	// 转换IN条件的值列表
	var values []interface{}
	for _, val := range in.Values {
		if b, ok := val.(bool); ok {
			// 布尔值保持原始类型
			values = append(values, b)
		} else {
			values = append(values, val)
		}
	}

	if len(values) == 0 {
		return nil
	}

	return &FilterCondition{
		FieldName: column.Name,
		Operator:  "isAnyOf",
		Value:     values,
	}
}

// buildExprCondition 构建表达式条件，处理 clause.Expr 类型
func (c *SQLConverter) buildExprCondition(expr clause.Expr) *FilterCondition {
	// 解析 SQL 表达式，如 "active = ?"
	sql := expr.SQL
	vars := expr.Vars

	// 简单的 SQL 解析，支持常见的操作符
	// 解析字段名和操作符
	var fieldName, operator string

	// 处理NULL条件（不需要参数）
	if strings.Contains(strings.ToUpper(sql), " IS NOT NULL") {
		fieldName = strings.TrimSpace(strings.Split(strings.ToUpper(sql), " IS NOT NULL")[0])
		operator = "isNotEmpty"
		return &FilterCondition{
			FieldName: fieldName,
			Operator:  operator,
			Value:     []interface{}{},
		}
	} else if strings.Contains(strings.ToUpper(sql), " IS NULL") {
		fieldName = strings.TrimSpace(strings.Split(strings.ToUpper(sql), " IS NULL")[0])
		operator = "isEmpty"
		return &FilterCondition{
			FieldName: fieldName,
			Operator:  operator,
			Value:     []interface{}{},
		}
	}

	// 对于其他操作符，需要参数
	if len(vars) == 0 {
		return nil
	}

	// 处理等于操作
	if strings.Contains(sql, " = ?") {
		fieldName = strings.TrimSpace(strings.Split(sql, " = ?")[0])
		operator = "is"
	} else if strings.Contains(sql, " != ?") || strings.Contains(sql, " <> ?") {
		fieldName = strings.TrimSpace(strings.Split(sql, " != ?")[0])
		if fieldName == "" {
			fieldName = strings.TrimSpace(strings.Split(sql, " <> ?")[0])
		}
		operator = "isNot"
	} else if strings.Contains(sql, " > ?") {
		fieldName = strings.TrimSpace(strings.Split(sql, " > ?")[0])
		operator = "isGreater"
	} else if strings.Contains(sql, " >= ?") {
		fieldName = strings.TrimSpace(strings.Split(sql, " >= ?")[0])
		operator = "isGreaterEqual"
	} else if strings.Contains(sql, " < ?") {
		fieldName = strings.TrimSpace(strings.Split(sql, " < ?")[0])
		operator = "isLess"
	} else if strings.Contains(sql, " <= ?") {
		fieldName = strings.TrimSpace(strings.Split(sql, " <= ?")[0])
		operator = "isLessEqual"
	} else if strings.Contains(sql, " LIKE ?") || strings.Contains(sql, " like ?") {
		fieldName = strings.TrimSpace(strings.Split(sql, " LIKE ?")[0])
		if fieldName == "" {
			fieldName = strings.TrimSpace(strings.Split(sql, " like ?")[0])
		}
		operator = "contains"
	} else {
		// 不支持的操作符
		return nil
	}

	if fieldName == "" {
		return nil
	}

	// 转换值
	var value []interface{}
	if len(vars) > 0 {
		if b, ok := vars[0].(bool); ok {
			// 布尔值保持原始类型
			value = []interface{}{b}
		} else {
			value = []interface{}{vars[0]}
		}
	}

	return &FilterCondition{
		FieldName: fieldName,
		Operator:  operator,
		Value:     value,
	}
}
