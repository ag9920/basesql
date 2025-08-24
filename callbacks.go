package basesql

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/ag9920/basesql/internal/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// parseValue 解析字符串值为适当的类型
func parseValue(value string) interface{} {
	return common.DefaultSQLParser.ParseValue(value)
}

// getTableID 通过表名获取表 ID
// 这个函数会调用飞书多维表格 API 获取应用下的所有表，然后根据表名查找对应的表 ID
// 参数:
//   - dialector: BaseSQL 的方言器实例，包含客户端和配置信息
//   - tableName: 要查找的表名
//
// 返回:
//   - string: 表 ID
//   - error: 查找过程中的错误
func getTableID(dialector *Dialector, tableName string) (string, error) {
	if dialector == nil {
		return "", fmt.Errorf("dialector 不能为 nil")
	}
	if tableName == "" {
		return "", fmt.Errorf("表名不能为空")
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 调用飞书 API 获取表列表
	apiReq := &APIRequest{
		Method: "GET",
		Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables", dialector.Config.AppToken),
	}

	resp, err := dialector.Client.DoRequest(ctx, apiReq)
	if err != nil {
		return "", fmt.Errorf("获取表列表失败: %w", err)
	}

	// 解析飞书API的完整响应结构
	var apiResp ListTablesAPIResponse
	if err := json.Unmarshal(resp.Body, &apiResp); err != nil {
		return "", fmt.Errorf("解析API响应失败: %w", err)
	}

	// 检查API响应码
	if apiResp.Code != 0 {
		return "", fmt.Errorf("API请求失败: code=%d, msg=%s", apiResp.Code, apiResp.Msg)
	}

	// 检查数据是否存在
	if apiResp.Data == nil {
		return "", fmt.Errorf("API响应中没有数据")
	}

	listResp := apiResp.Data

	// 查找表名对应的 ID（精确匹配）
	for _, table := range listResp.Items {
		if table.Name == tableName {
			return table.TableID, nil
		}
	}

	// 如果精确匹配失败，尝试不区分大小写的匹配
	for _, table := range listResp.Items {
		if strings.EqualFold(table.Name, tableName) {

			return table.TableID, nil
		}
	}

	return "", fmt.Errorf("未找到表 '%s'，请检查表名是否正确", tableName)
}

// getTableFields 获取表的所有字段信息
// 这个函数会先获取表 ID，然后调用飞书多维表格 API 获取该表的所有字段定义
// 参数:
//   - dialector: BaseSQL 的方言器实例，包含客户端和配置信息
//   - tableName: 表名
//
// 返回:
//   - []*Field: 字段信息列表
//   - error: 获取过程中的错误
func getTableFields(dialector *Dialector, tableName string) ([]*Field, error) {
	if dialector == nil {
		return nil, fmt.Errorf("dialector 不能为 nil")
	}
	if tableName == "" {
		return nil, fmt.Errorf("表名不能为空")
	}

	// 先获取表 ID
	tableID, err := getTableID(dialector, tableName)
	if err != nil {
		return nil, fmt.Errorf("获取表 ID 失败: %w", err)
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 调用飞书 API 获取字段列表
	apiReq := &APIRequest{
		Method: "GET",
		Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/fields", dialector.Config.AppToken, tableID),
	}

	resp, err := dialector.Client.DoRequest(ctx, apiReq)
	if err != nil {

		return nil, fmt.Errorf("获取字段列表失败: %w", err)
	}

	var apiResp ListFieldsAPIResponse
	if err := json.Unmarshal(resp.Body, &apiResp); err != nil {

		return nil, fmt.Errorf("解析字段列表响应失败: %w", err)
	}

	if apiResp.Code != 0 {

		return nil, fmt.Errorf("API返回错误: %s", apiResp.Msg)
	}

	if apiResp.Data == nil {

		return nil, fmt.Errorf("API响应数据为空")
	}

	return apiResp.Data.Items, nil
}

// RegisterCallbacks 注册 GORM 回调函数
// 这个函数会替换 GORM 的默认回调处理器，将 SQL 操作转换为飞书多维表格 API 调用
// 参数:
//   - db: GORM 数据库实例
//   - dialector: BaseSQL 的方言器实例
func RegisterCallbacks(db *gorm.DB, dialector *Dialector) {
	if db == nil {
		panic("GORM 数据库实例不能为 nil")
	}
	if dialector == nil {
		panic("dialector 不能为 nil")
	}

	// 替换查询处理器 - 处理 SELECT 语句
	db.Callback().Query().Replace("gorm:query", func(db *gorm.DB) {
		if err := queryCallback(db, dialector); err != nil {
			// 在事务中的查询失败会影响整个事务
			if db.Statement.ConnPool != nil {
				common.Warnf("事务中的查询操作失败，由于飞书多维表格不支持回滚，可能导致数据不一致: %v", err)
			}
			db.AddError(fmt.Errorf("查询操作失败: %w", err))
		}
	})

	// 替换行查询处理器 - 处理单行查询
	db.Callback().Row().Replace("gorm:row", func(db *gorm.DB) {
		if err := queryCallback(db, dialector); err != nil {
			// 在事务中的查询失败会影响整个事务
			if db.Statement.ConnPool != nil {
				common.Warnf("事务中的行查询操作失败，由于飞书多维表格不支持回滚，可能导致数据不一致: %v", err)
			}
			db.AddError(fmt.Errorf("行查询操作失败: %w", err))
		}
	})

	// 替换原始查询处理器 - 处理原生 SQL 语句
	db.Callback().Raw().Replace("gorm:raw", func(db *gorm.DB) {
		if err := rawCallback(db, dialector); err != nil {
			// 在事务中的原生SQL失败会影响整个事务
			if db.Statement.ConnPool != nil {
				common.Warnf("事务中的原生SQL操作失败，由于飞书多维表格不支持回滚，已执行的操作无法撤销: %v", err)
			}
			db.AddError(fmt.Errorf("原生 SQL 操作失败: %w", err))
		}
	})

	// 替换创建回调 - 处理 INSERT 语句
	db.Callback().Create().Replace("gorm:create", func(db *gorm.DB) {
		if err := createCallback(db, dialector); err != nil {
			// 在事务中的创建失败会影响整个事务
			if db.Statement.ConnPool != nil {
				common.Warnf("事务中的创建操作失败，由于飞书多维表格不支持回滚，已创建的数据无法撤销: %v", err)
			}
			db.AddError(fmt.Errorf("创建操作失败: %w", err))
		}
	})

	// 替换更新回调 - 处理 UPDATE 语句
	db.Callback().Update().Replace("gorm:update", func(db *gorm.DB) {
		if err := updateCallback(db, dialector); err != nil {
			// 在事务中的更新失败会影响整个事务
			if db.Statement.ConnPool != nil {
				common.Warnf("事务中的更新操作失败，由于飞书多维表格不支持回滚，已更新的数据无法撤销: %v", err)
			}
			db.AddError(fmt.Errorf("更新操作失败: %w", err))
		}
	})

	// 替换删除回调 - 处理 DELETE 语句
	db.Callback().Delete().Replace("gorm:delete", func(db *gorm.DB) {
		if err := deleteCallback(db, dialector); err != nil {
			// 检查是否是'record ID not found'错误
			if err.Error() == "record ID not found" {
				// 对于记录不存在的情况，静默处理，不显示警告
				db.AddError(err)
				return
			}

			// 在事务中的删除失败会影响整个事务
			if db.Statement.ConnPool != nil {
				common.Warnf("事务中的删除操作失败，由于飞书多维表格不支持回滚，已删除的数据无法恢复: %v", err)
			}
			db.AddError(fmt.Errorf("删除操作失败: %w", err))
		}
	})
}

// createCallback 创建回调函数
// 处理 GORM 的 CREATE 操作，将其转换为飞书多维表格的创建记录 API 调用
// 参数:
//   - db: GORM 数据库实例
//   - dialector: BaseSQL 的方言器实例
//
// 返回:
//   - error: 创建过程中的错误
func createCallback(db *gorm.DB, dialector *Dialector) error {
	// 检查是否已有错误
	if db.Error != nil {

		return db.Error
	}

	// 检查模式是否存在
	if db.Statement.Schema == nil {

		return fmt.Errorf("未找到数据模式定义")
	}

	// 检查表名是否存在
	if db.Statement.Table == "" {

		return fmt.Errorf("表名不能为空")
	}

	// 获取表名和表 ID
	tableName := db.Statement.Table

	tableID, err := getTableID(dialector, tableName)
	if err != nil {

		return fmt.Errorf("获取表 ID 失败: %w", err)
	}

	// 获取表的字段信息
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
	for _, field := range db.Statement.Schema.Fields {
		// 跳过主键、自增字段和自动时间字段
		if field.PrimaryKey || field.AutoIncrement {
			continue
		}

		// 跳过自动时间字段，这些字段应该由飞书系统自动处理
		if field.AutoCreateTime == schema.UnixTime || field.AutoUpdateTime == schema.UnixTime {
			continue
		}

		// 尝试获取字段值
		value, ok := field.ValueOf(db.Statement.Context, db.Statement.ReflectValue)

		// 如果 ValueOf 失败，尝试直接从反射值获取
		if !ok || value == nil {
			value, ok = extractFieldValueByReflection(db.Statement.ReflectValue, field)
		}

		// 如果仍然无法获取值，跳过该字段
		if !ok || value == nil {
			continue
		}

		// 对于时间类型，如果是零值时间，跳过该字段
		if t, ok := value.(time.Time); ok && t.IsZero() {
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

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(db.Statement.Context, 30*time.Second)
	defer cancel()

	// 调用飞书 API
	apiReq := &APIRequest{
		Method: "POST",
		Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records", dialector.Config.AppToken, tableID),
		Body:   req,
	}

	resp, err := dialector.Client.DoRequest(ctx, apiReq)
	if err != nil {

		return fmt.Errorf("创建记录 API 调用失败: %w", err)
	}

	// 解析飞书API的完整响应结构
	var apiResp CreateRecordAPIResponse
	if err := json.Unmarshal(resp.Body, &apiResp); err != nil {

		return fmt.Errorf("解析API响应失败: %w", err)
	}

	// 检查API响应码
	if apiResp.Code != 0 {

		return fmt.Errorf("API请求失败: code=%d, msg=%s", apiResp.Code, apiResp.Msg)
	}

	// 检查数据是否存在
	if apiResp.Data == nil {

		return fmt.Errorf("API响应中没有数据")
	}

	// 检查记录是否存在
	if apiResp.Data.Record == nil {

		return fmt.Errorf("API响应中没有记录信息")
	}

	createResp := apiResp.Data

	// 设置主键值（记录 ID）
	if db.Statement.Schema.PrioritizedPrimaryField != nil {

		if err := db.Statement.Schema.PrioritizedPrimaryField.Set(db.Statement.Context, db.Statement.ReflectValue, createResp.Record.RecordID); err != nil {

			return fmt.Errorf("设置主键值失败: %w", err)
		}

	}

	// 设置影响的行数
	db.RowsAffected = 1

	return nil
}

// extractFieldValueByReflection 通过反射提取字段值
// 这是一个辅助函数，当 GORM 的 ValueOf 方法失败时使用
// 参数:
//   - reflectValue: 反射值
//   - field: GORM 字段定义
//
// 返回:
//   - interface{}: 字段值
//   - bool: 是否成功获取值
func extractFieldValueByReflection(reflectValue reflect.Value, field *schema.Field) (interface{}, bool) {
	var structValue reflect.Value

	// 处理指针类型
	if reflectValue.Kind() == reflect.Ptr {
		if reflectValue.IsNil() {
			return nil, false
		}
		structValue = reflectValue.Elem()
	} else if reflectValue.Kind() == reflect.Struct {
		structValue = reflectValue
	} else {
		return nil, false
	}

	// 检查结构体是否有效
	if !structValue.IsValid() || structValue.Kind() != reflect.Struct {
		return nil, false
	}

	// 通过字段名获取字段值
	fieldValue := structValue.FieldByName(field.Name)
	if !fieldValue.IsValid() || !fieldValue.CanInterface() {
		return nil, false
	}

	return fieldValue.Interface(), true
}

// 使用公共的 SQLCommand 类型
type SQLCommand = common.SQLCommand

// rawCallback 处理原生 SQL 语句的回调函数
// 这个函数负责解析和执行用户提供的原生 SQL 语句，并将其转换为飞书多维表格的 API 调用
// 参数:
//   - db: GORM 数据库实例
//   - dialector: BaseSQL 方言实例
//
// 返回:
//   - error: 执行过程中的错误
func rawCallback(db *gorm.DB, dialector *Dialector) error {
	// 参数校验
	if db == nil {
		return fmt.Errorf("数据库实例不能为空")
	}
	if dialector == nil {
		return fmt.Errorf("方言实例不能为空")
	}

	// 检查是否已有错误
	if db.Error != nil {
		return db.Error
	}

	// 获取原生 SQL 语句
	rawSQL := db.Statement.SQL.String()
	if rawSQL == "" {
		return fmt.Errorf("SQL 语句不能为空")
	}

	// 解析 SQL 语句
	cmd, err := parseRawSQL(rawSQL)
	if err != nil {
		return common.FormatError("解析 SQL 语句失败", err)
	}

	// 根据命令类型执行相应操作
	switch cmd.Type {
	case "SELECT":
		// SELECT查询由Row回调处理，这里只做验证
		if cmd.Table == "" {
			return fmt.Errorf("表名不能为空")
		}
		// 验证表是否存在
		_, err := getTableID(dialector, cmd.Table)
		if err != nil {
			return fmt.Errorf("表不存在: %w", err)
		}
		return nil
	case "UPDATE":
		return executeRawUpdate(db, dialector, cmd)
	case "INSERT":
		return executeRawInsert(db, dialector, cmd)
	case "DELETE":
		return executeRawDelete(db, dialector, cmd)
	default:
		return fmt.Errorf("不支持的 SQL 命令类型: %s", cmd.Type)
	}
}

// parseRawSQL 解析原生 SQL 语句
func parseRawSQL(sql string) (*SQLCommand, error) {
	if err := common.ValidateNotEmpty(sql, "SQL 语句"); err != nil {
		return nil, err
	}

	// 预处理 SQL
	sql = common.PreprocessSQL(sql)

	cmd := &SQLCommand{
		RawSQL: sql,
		Values: make(map[string]interface{}),
	}

	// 转换为大写进行匹配
	upperSQL := strings.ToUpper(sql)

	switch {
	case strings.HasPrefix(upperSQL, "SELECT"):
		cmd.Type = "SELECT"
		return parseSelectSQL(sql, cmd)
	case strings.HasPrefix(upperSQL, "UPDATE"):
		cmd.Type = "UPDATE"
		return parseUpdateSQL(sql, cmd)
	case strings.HasPrefix(upperSQL, "INSERT"):
		cmd.Type = "INSERT"
		return parseInsertSQL(sql, cmd)
	case strings.HasPrefix(upperSQL, "DELETE"):
		cmd.Type = "DELETE"
		return parseDeleteSQL(sql, cmd)
	default:
		return nil, fmt.Errorf("不支持的 SQL 命令: %s", sql)
	}
}

// parseUpdateSQL 解析 UPDATE 命令
func parseUpdateSQL(sql string, cmd *SQLCommand) (*SQLCommand, error) {
	cmd.Type = "UPDATE"

	// UPDATE table SET field1=value1, field2=value2 WHERE condition
	var re *regexp.Regexp
	var matches []string

	if strings.Contains(strings.ToUpper(sql), " WHERE ") {
		re = regexp.MustCompile(`(?i)UPDATE\s+([^\s]+)\s+SET\s+(.*?)\s+WHERE\s+(.*)`)
		matches = re.FindStringSubmatch(sql)
	} else {
		re = regexp.MustCompile(`(?i)UPDATE\s+([^\s]+)\s+SET\s+(.*)`)
		matches = re.FindStringSubmatch(sql)
	}

	if len(matches) < 3 {
		return nil, fmt.Errorf("UPDATE 语法错误")
	}

	cmd.Table = matches[1]
	if err := common.ValidateNotEmpty(cmd.Table, "表名"); err != nil {
		return nil, err
	}

	// 解析 SET 子句
	setClause := strings.TrimSpace(matches[2])
	setClause = strings.TrimSuffix(setClause, ";")

	setPairs := strings.Split(setClause, ",")
	for _, pair := range setPairs {
		pair = strings.TrimSpace(pair)
		parts := strings.Split(pair, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("SET 子句语法错误")
		}
		field := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, "'\"") // 移除引号
		cmd.Values[field] = parseValue(value)
	}

	// 处理 WHERE 子句
	if len(matches) > 3 && matches[3] != "" {
		whereClause := strings.TrimSpace(matches[3])
		whereClause = strings.TrimSuffix(whereClause, ";")
		cmd.Where = whereClause
	}

	return cmd, nil
}

// executeRawUpdate 执行原生 UPDATE 命令
// 这个函数负责将 UPDATE SQL 语句转换为飞书多维表格的更新记录 API 调用
// 参数:
//   - db: GORM 数据库实例
//   - dialector: BaseSQL 方言实例
//   - cmd: 解析后的 SQL 命令
//
// 返回:
//   - error: 执行过程中的错误
func executeRawUpdate(db *gorm.DB, dialector *Dialector, cmd *SQLCommand) error {
	// 参数校验
	if db == nil {
		return fmt.Errorf("数据库实例不能为空")
	}
	if dialector == nil {
		return fmt.Errorf("方言实例不能为空")
	}
	if cmd == nil {
		return fmt.Errorf("SQL 命令不能为空")
	}
	if cmd.Table == "" {
		return fmt.Errorf("表名不能为空")
	}
	if len(cmd.Values) == 0 {
		return fmt.Errorf("没有字段需要更新")
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 获取表 ID
	tableID, err := getTableID(dialector, cmd.Table)
	if err != nil {
		return fmt.Errorf("获取表 ID 失败: %w", err)
	}

	// 获取表字段信息并转换字段值
	tableFieldsList, err := getTableFields(dialector, cmd.Table)
	if err == nil {
		// 将字段列表转换为map以便查找
		tableFields := make(map[string]*Field)
		for _, tableField := range tableFieldsList {
			tableFields[tableField.FieldName] = tableField
		}

		// 转换字段值
		for fieldName, value := range cmd.Values {
			if tableField, exists := tableFields[fieldName]; exists {
				cmd.Values[fieldName] = fmt.Sprintf("%v", tableField.ConvertFromGoValue(value))
			}
		}
	} else {
		// 如果获取字段信息失败，记录警告但继续执行

	}

	// 构建更新请求的字段
	fields := make(map[string]interface{})
	for field, value := range cmd.Values {
		fields[field] = value
	}

	if cmd.Where == "" {
		// 没有 WHERE 条件，更新所有记录（符合 SQL 标准）

		// 首先获取所有记录
		apiReq := &APIRequest{
			Method: "GET",
			Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records", dialector.Config.AppToken, tableID),
		}

		resp, err := dialector.Client.DoRequest(ctx, apiReq)
		if err != nil {
			return fmt.Errorf("获取所有记录失败: %w", err)
		}

		var listResp ListRecordsResponse
		if err := json.Unmarshal(resp.Body, &listResp); err != nil {
			return fmt.Errorf("解析记录列表响应失败: %w", err)
		}

		if len(listResp.Items) == 0 {
			db.RowsAffected = 0
			return nil
		}

		// 逐条更新所有记录（避免批量更新 API 格式问题）
		updateReq := &UpdateRecordRequest{
			Fields: fields,
		}

		var successCount int64
		for _, record := range listResp.Items {
			apiReq := &APIRequest{
				Method: "PUT",
				Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records/%s", dialector.Config.AppToken, tableID, record.RecordID),
				Body:   updateReq,
			}

			_, err = dialector.Client.DoRequest(ctx, apiReq)
			if err != nil {
				return fmt.Errorf("更新记录 %s 失败: %w", record.RecordID, err)
			}
			successCount++
		}

		db.RowsAffected = successCount
	} else {
		// 有 WHERE 条件，先查询符合条件的记录，然后更新
		filter := buildFilterFromWhere(cmd.Where)

		// 构建查询请求体
		listReq := &ListRecordsRequest{
			Filter: filter,
		}

		apiReq := &APIRequest{
			Method: "POST",
			Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records/search", dialector.Config.AppToken, tableID),
			Body:   listReq,
		}

		resp, err := dialector.Client.DoRequest(ctx, apiReq)
		if err != nil {
			return fmt.Errorf("查询符合条件的记录失败: %w", err)
		}

		var listResp ListRecordsResponse
		if err := json.Unmarshal(resp.Body, &listResp); err != nil {
			return fmt.Errorf("解析查询响应失败: %w", err)
		}

		if len(listResp.Items) == 0 {
			db.RowsAffected = 0
			return nil
		}

		// 逐条更新符合条件的记录
		updateReq := &UpdateRecordRequest{
			Fields: fields,
		}

		var successCount int64
		for _, record := range listResp.Items {
			apiReq := &APIRequest{
				Method: "PUT",
				Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records/%s", dialector.Config.AppToken, tableID, record.RecordID),
				Body:   updateReq,
			}

			_, err = dialector.Client.DoRequest(ctx, apiReq)
			if err != nil {
				return fmt.Errorf("更新记录 %s 失败: %w", record.RecordID, err)
			}
			successCount++
		}

		db.RowsAffected = successCount
	}

	return nil
}

// buildFilterFromWhere 从 WHERE 条件构建过滤器
// 这个函数负责将 SQL WHERE 子句转换为飞书多维表格的过滤条件格式
// 支持的操作符：=、!=、>、>=、<、<=、LIKE
// 参数:
//   - whereClause: WHERE 子句字符串
//
// 返回:
//   - *FilterRequest: 飞书多维表格的过滤请求，如果无法解析则返回 nil
func buildFilterFromWhere(whereClause string) *FilterRequest {
	if whereClause == "" {
		return nil
	}

	// 清理 WHERE 子句
	whereClause = strings.TrimSpace(whereClause)
	if whereClause == "" {
		return nil
	}

	var conditions []*FilterCondition

	// 支持多种操作符的正则表达式
	operatorPatterns := []struct {
		pattern  string
		operator string
	}{
		{`(?i)(\w+)\s+IS\s+NOT\s+NULL`, "isNotEmpty"},
		{`(?i)(\w+)\s+IS\s+NULL`, "isEmpty"},
		{`(?i)(\w+)\s*!=\s*['"]*([^'"]+)['"]*`, "isNot"},
		{`(?i)(\w+)\s*>=\s*['"]*([^'"]+)['"]*`, "isGreaterEqual"},
		{`(?i)(\w+)\s*<=\s*['"]*([^'"]+)['"]*`, "isLessEqual"},
		{`(?i)(\w+)\s*>\s*['"]*([^'"]+)['"]*`, "isGreater"},
		{`(?i)(\w+)\s*<\s*['"]*([^'"]+)['"]*`, "isLess"},
		{`(?i)(\w+)\s*=\s*['"]*([^'"]+)['"]*`, "is"},
		{`(?i)(\w+)\s+LIKE\s+['"]*([^'"]+)['"]*`, "contains"},
		{`(?i)(\w+)\s+IN\s*\(([^)]+)\)`, "isAnyOf"},
	}

	// 尝试匹配各种操作符
	for _, op := range operatorPatterns {
		re := regexp.MustCompile(op.pattern)
		matches := re.FindStringSubmatch(whereClause)

		if len(matches) >= 2 {
			field := strings.TrimSpace(matches[1])

			// 验证字段名不为空
			if field == "" {
				continue
			}

			// 对于NULL操作符，不需要值
			if op.operator == "isEmpty" || op.operator == "isNotEmpty" {
				conditions = append(conditions, &FilterCondition{
					FieldName: field,
					Operator:  op.operator,
					Value:     []interface{}{},
				})
				break
			}

			// 对于其他操作符，需要值
			if len(matches) < 3 {
				continue
			}

			value := strings.TrimSpace(matches[2])
			if value == "" {
				continue
			}

			// 处理不同操作符的值
			var values []interface{}

			if op.operator == "isAnyOf" {
				// 处理 IN 操作符的多个值
				// 分割逗号分隔的值列表
				valueList := strings.Split(value, ",")
				for _, v := range valueList {
					v = strings.TrimSpace(v)
					// 移除引号
					v = strings.Trim(v, "'\"")
					if v != "" {
						// 转换值类型
						if strings.ToLower(v) == "true" {
							values = append(values, true)
						} else if strings.ToLower(v) == "false" {
							values = append(values, false)
						} else {
							values = append(values, v)
						}
					}
				}
			} else {
				// 处理单个值的操作符
				if op.operator == "contains" {
					// 移除 SQL LIKE 的通配符 % 和 _
					value = strings.ReplaceAll(value, "%", "")
					value = strings.ReplaceAll(value, "_", "")
				}

				// 转换值类型，特别处理布尔值
				var convertedValue interface{}
				if strings.ToLower(value) == "true" {
					convertedValue = true
				} else if strings.ToLower(value) == "false" {
					convertedValue = false
				} else {
					convertedValue = value
				}
				values = []interface{}{convertedValue}
			}

			// 确保有有效值
			if len(values) == 0 {
				continue
			}

			conditions = append(conditions, &FilterCondition{
				FieldName: field,
				Operator:  op.operator,
				Value:     values,
			})
			break // 找到匹配的操作符后停止
		}
	}

	if len(conditions) == 0 {
		return nil
	}

	return &FilterRequest{
		Conjunction: "and",
		Conditions:  conditions,
	}
}

// parseSelectSQL 解析SELECT语句
func parseSelectSQL(sql string, cmd *SQLCommand) (*SQLCommand, error) {
	return common.DefaultSQLParser.ParseSelectSQL(sql, cmd)
}

func parseInsertSQL(sql string, cmd *SQLCommand) (*SQLCommand, error) {
	return common.DefaultSQLParser.ParseInsertSQL(sql, cmd)
}

// parseValue 函数已移至 common.SQLParser 中

func parseDeleteSQL(sql string, cmd *SQLCommand) (*SQLCommand, error) {
	return common.DefaultSQLParser.ParseDeleteSQL(sql, cmd)
}

func executeRawSelect(db *gorm.DB, dialector *Dialector, cmd *SQLCommand) error {
	if cmd.Table == "" {
		return fmt.Errorf("表名不能为空")
	}

	// 验证表是否存在
	_, err := getTableID(dialector, cmd.Table)
	if err != nil {
		return fmt.Errorf("表不存在: %w", err)
	}

	// Raw SELECT 查询暂不支持，建议使用 GORM 的标准查询方法
	return fmt.Errorf("Raw SELECT 查询暂不支持，请使用 db.Where().Find() 等 GORM 标准查询方法")
}

func executeRawInsert(db *gorm.DB, dialector *Dialector, cmd *SQLCommand) error {
	// 获取表 ID
	tableID, err := getTableID(dialector, cmd.Table)
	if err != nil {
		return err
	}

	// 获取表字段信息并转换字段值
	tableFieldsList, err := getTableFields(dialector, cmd.Table)
	if err == nil {
		// 将字段列表转换为map以便查找
		tableFields := make(map[string]*Field)
		for _, tableField := range tableFieldsList {
			tableFields[tableField.FieldName] = tableField
		}

		// 转换字段值
		for fieldName, value := range cmd.Values {
			if tableField, exists := tableFields[fieldName]; exists {
				// 直接使用转换后的值，不要再转换为字符串
				cmd.Values[fieldName] = tableField.ConvertFromGoValue(value)
			}
		}
	}

	// 构建创建请求的字段
	fields := make(map[string]interface{})
	for field, value := range cmd.Values {
		fields[field] = value
	}

	// 创建记录
	createReq := &CreateRecordRequest{
		Fields: fields,
	}

	apiReq := &APIRequest{
		Method: "POST",
		Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records", dialector.Config.AppToken, tableID),
		Body:   createReq,
	}

	_, err = dialector.Client.DoRequest(context.Background(), apiReq)
	if err != nil {
		return err
	}

	db.RowsAffected = 1
	return nil
}

func executeRawDelete(db *gorm.DB, dialector *Dialector, cmd *SQLCommand) error {
	// 获取表 ID
	tableID, err := getTableID(dialector, cmd.Table)
	if err != nil {
		return err
	}

	if cmd.Where == "" {
		// 没有 WHERE 条件，删除所有记录（符合 SQL 标准）

		// 首先获取所有记录
		apiReq := &APIRequest{
			Method: "GET",
			Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records", dialector.Config.AppToken, tableID),
		}

		resp, err := dialector.Client.DoRequest(context.Background(), apiReq)
		if err != nil {
			return err
		}

		var listResp ListRecordsResponse
		if err := json.Unmarshal(resp.Body, &listResp); err != nil {
			return err
		}

		if len(listResp.Items) == 0 {
			db.RowsAffected = 0
			return nil
		}

		// 逐条删除所有记录
		for _, record := range listResp.Items {
			apiReq := &APIRequest{
				Method: "DELETE",
				Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records/%s", dialector.Config.AppToken, tableID, record.RecordID),
			}

			_, err = dialector.Client.DoRequest(context.Background(), apiReq)
			if err != nil {
				return err
			}
		}

		db.RowsAffected = int64(len(listResp.Items))
	} else {
		// 有 WHERE 条件，先查询符合条件的记录，然后删除
		filter := buildFilterFromWhere(cmd.Where)

		reqBody := &ListRecordsRequest{
			Filter: filter,
		}

		bodyData, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}

		apiReq := &APIRequest{
			Method: "POST",
			Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records/search", dialector.Config.AppToken, tableID),
			Body:   bodyData,
		}

		resp, err := dialector.Client.DoRequest(context.Background(), apiReq)
		if err != nil {
			return err
		}

		var listResp ListRecordsResponse
		if err := json.Unmarshal(resp.Body, &listResp); err != nil {
			return err
		}

		if len(listResp.Items) == 0 {
			db.RowsAffected = 0
			return nil
		}

		// 逐条删除符合条件的记录
		for _, record := range listResp.Items {
			apiReq := &APIRequest{
				Method: "DELETE",
				Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records/%s", dialector.Config.AppToken, tableID, record.RecordID),
			}

			_, err = dialector.Client.DoRequest(context.Background(), apiReq)
			if err != nil {
				return err
			}
		}

		db.RowsAffected = int64(len(listResp.Items))
	}

	return nil
}

// isValidFieldName 验证字段名是否有效
// 过滤掉GORM内部生成的特殊标识符和无效字段名
func isValidFieldName(fieldName string) bool {
	if fieldName == "" {
		return false
	}

	// 过滤掉包含特殊字符的无效字段名
	if strings.Contains(fieldName, "~~~") {
		return false
	}

	// 过滤掉纯特殊字符的字段名
	if strings.HasPrefix(fieldName, "~") || strings.HasSuffix(fieldName, "~") {
		return false
	}

	// 过滤掉包含非法字符的字段名
	for _, char := range fieldName {
		if char < 32 || char > 126 { // 非可打印ASCII字符
			return false
		}
	}

	return true
}

// queryCallback 查询回调
func queryCallback(db *gorm.DB, dialector *Dialector) error {
	if db.Error != nil {

		return db.Error
	}

	if db.Statement.Schema == nil {

		return fmt.Errorf("schema not found")
	}

	// 获取表名和表 ID
	tableName := db.Statement.Table

	tableID, err := getTableID(dialector, tableName)
	if err != nil {

		return err
	}

	// 构建查询请求
	req := &ListRecordsRequest{
		// 不指定字段名，让API返回所有字段（像CLI一样）
		FieldNames: make([]string, 0),
	}

	// 注释掉字段名添加逻辑，避免InvalidFieldNames错误
	// for _, field := range db.Statement.Schema.Fields {
	//     req.FieldNames = append(req.FieldNames, field.DBName)
	// }

	// 处理 WHERE 条件
	if whereClause, ok := db.Statement.Clauses["WHERE"]; ok {

		if where, ok := whereClause.Expression.(clause.Where); ok && len(where.Exprs) > 0 {

			// 打印每个表达式的详细信息

			converter := &SQLConverter{}
			filter := converter.buildFilter(where.Exprs)

			req.Filter = filter
		}
	}

	// 处理排序
	if orderClause, ok := db.Statement.Clauses["ORDER BY"]; ok {
		if orderBy, ok := orderClause.Expression.(clause.OrderBy); ok && len(orderBy.Columns) > 0 {

			sort := make([]string, 0)
			for _, column := range orderBy.Columns {
				columnName := column.Column.Name
				// 验证字段名有效性，过滤掉无效的字段名
				if columnName != "" && isValidFieldName(columnName) {
					if column.Desc {
						sort = append(sort, fmt.Sprintf("-%s", columnName))
					} else {
						sort = append(sort, columnName)
					}
				} else if columnName != "" {
					// 记录无效的字段名用于调试
					common.Debugf("跳过无效的排序字段名: %q", columnName)
				}
			}
			if len(sort) > 0 {
				req.Sort = sort
			}
		}
	}

	// 如果有过滤条件，使用 POST 请求
	var apiReq *APIRequest
	if req.Filter != nil {
		// 使用 POST 请求发送 filter
		apiReq = &APIRequest{
			Method: "POST",
			Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records/search", dialector.Config.AppToken, tableID),
			Body:   req,
		}
	} else {
		// 没有过滤条件，使用 GET 请求
		// 不传递任何查询参数，让API返回所有字段
		apiReq = &APIRequest{
			Method: "GET",
			Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records", dialector.Config.AppToken, tableID),
			// 不设置QueryParams，获取所有字段
		}
	}

	resp, err := dialector.Client.DoRequest(context.Background(), apiReq)
	if err != nil {
		return err
	}

	// 解析响应
	var apiResp ListRecordsAPIResponse
	if err := json.Unmarshal(resp.Body, &apiResp); err != nil {
		return err
	}

	// 检查API调用是否成功
	if apiResp.Code != 0 {
		return fmt.Errorf("API返回错误: code=%d, msg=%s", apiResp.Code, apiResp.Msg)
	}

	if apiResp.Data == nil {
		return fmt.Errorf("API响应数据为空")
	}

	listResp := apiResp.Data

	// 设置结果
	if len(listResp.Items) > 0 {
		// 判断是否是查询单个记录
		if db.Statement.ReflectValue.Kind() == reflect.Slice {
			// 查询多个记录
			sliceValue := reflect.MakeSlice(db.Statement.ReflectValue.Type(), 0, len(listResp.Items))
			for _, record := range listResp.Items {
				elemValue := reflect.New(db.Statement.Schema.ModelType).Elem()
				if err := setRecordToStruct(elemValue, record, db.Statement.Schema, dialector); err != nil {
					return err
				}
				sliceValue = reflect.Append(sliceValue, elemValue)
			}
			db.Statement.ReflectValue.Set(sliceValue)
		} else {
			// 查询单个记录
			if len(listResp.Items) > 0 {
				if err := setRecordToStruct(db.Statement.ReflectValue, listResp.Items[0], db.Statement.Schema, dialector); err != nil {
					return err
				}
			}
		}
	}

	db.RowsAffected = int64(len(listResp.Items))
	return nil
}

// updateCallback 更新回调
func updateCallback(db *gorm.DB, dialector *Dialector) error {
	if db.Error != nil {

		return db.Error
	}

	if db.Statement.Schema == nil {

		return fmt.Errorf("schema not found")
	}

	// 获取表名、表 ID 和记录 ID
	tableName := db.Statement.Table

	tableID, err := getTableID(dialector, tableName)
	if err != nil {

		return err
	}

	var recordID string
	if db.Statement.Schema.PrioritizedPrimaryField != nil {
		// 首先尝试从 Model 中获取主键值
		if db.Statement.Model != nil {

			modelValue := reflect.ValueOf(db.Statement.Model)

			if modelValue.Kind() == reflect.Ptr && !modelValue.IsNil() {
				modelValue = modelValue.Elem()
			}

			if modelValue.Kind() == reflect.Struct {
				// 尝试使用 GORM 的方法
				if value, ok := db.Statement.Schema.PrioritizedPrimaryField.ValueOf(db.Statement.Context, reflect.ValueOf(db.Statement.Model)); ok {
					recordID = fmt.Sprintf("%v", value)

				} else {
					// 如果 GORM 方法失败，直接通过反射获取
					primaryFieldName := db.Statement.Schema.PrioritizedPrimaryField.Name
					if idField := modelValue.FieldByName(primaryFieldName); idField.IsValid() && idField.CanInterface() {
						recordID = fmt.Sprintf("%v", idField.Interface())

					}
				}
			}
		}

		// 如果从 Model 中没有获取到，尝试从 ReflectValue 获取
		if recordID == "" {

			if db.Statement.ReflectValue.Kind() == reflect.Struct {
				if value, ok := db.Statement.Schema.PrioritizedPrimaryField.ValueOf(db.Statement.Context, db.Statement.ReflectValue); ok {
					recordID = fmt.Sprintf("%v", value)

				}
			} else if db.Statement.ReflectValue.Kind() == reflect.Ptr {
				if !db.Statement.ReflectValue.IsNil() {
					if value, ok := db.Statement.Schema.PrioritizedPrimaryField.ValueOf(db.Statement.Context, db.Statement.ReflectValue); ok {
						recordID = fmt.Sprintf("%v", value)

					}
				}
			}
		}

		// 最后尝试从 WHERE 子句中获取主键值
		if recordID == "" {

			if whereClause, ok := db.Statement.Clauses["WHERE"]; ok {
				if where, ok := whereClause.Expression.(clause.Where); ok {
					for _, expr := range where.Exprs {
						if eq, ok := expr.(clause.Eq); ok {
							if column, ok := eq.Column.(clause.Column); ok {
								if column.Name == db.Statement.Schema.PrioritizedPrimaryField.DBName {
									recordID = fmt.Sprintf("%v", eq.Value)

									break
								}
							}
						}
					}
				}
			}
		}
	}
	if recordID == "" {

		return fmt.Errorf("record ID not found")
	}

	// 获取更新字段
	fields := make(map[string]interface{})
	if setClause, ok := db.Statement.Clauses["SET"]; ok {
		if set, ok := setClause.Expression.(clause.Set); ok {
			for _, assign := range set {
				fields[assign.Column.Name] = assign.Value

			}
		}
	}

	// 如果没有 SET 子句，使用结构体字段
	if len(fields) == 0 && db.Statement.ReflectValue.Kind() == reflect.Struct {

		for _, field := range db.Statement.Schema.Fields {
			if field.PrimaryKey || field.AutoIncrement {
				continue
			}

			value, ok := field.ValueOf(db.Statement.Context, db.Statement.ReflectValue)
			if !ok {
				continue
			}

			fields[field.DBName] = value

		}
	}

	if len(fields) == 0 {

		return nil
	}

	// 获取表字段信息并转换字段值
	tableFieldsList, err := getTableFields(dialector, tableName)
	if err == nil {
		// 将字段列表转换为map以便查找
		tableFields := make(map[string]*Field)
		for _, tableField := range tableFieldsList {
			tableFields[tableField.FieldName] = tableField
		}

		// 转换字段值
		for fieldName, value := range fields {
			if tableField, exists := tableFields[fieldName]; exists {
				fields[fieldName] = tableField.ConvertFromGoValue(value)
			}
		}
	} else {

	}

	// 更新记录请求
	req := &UpdateRecordRequest{
		Fields: fields,
	}

	// 调用 API
	apiReq := &APIRequest{
		Method: "PUT",
		Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records/%s", dialector.Config.AppToken, tableID, recordID),
		Body:   req,
	}

	_, err = dialector.Client.DoRequest(context.Background(), apiReq)
	if err != nil {

		return err
	}

	db.RowsAffected = 1

	return nil
}

// deleteCallback 删除回调
func deleteCallback(db *gorm.DB, dialector *Dialector) error {
	if db.Error != nil {
		return db.Error
	}

	if db.Statement.Schema == nil {
		return fmt.Errorf("schema not found")
	}

	// 获取表名、表 ID 和记录 ID
	tableName := db.Statement.Table
	tableID, err := getTableID(dialector, tableName)
	if err != nil {
		return err
	}

	var recordID string
	if db.Statement.Schema.PrioritizedPrimaryField != nil {
		// 首先尝试从 Model 中获取主键值
		if db.Statement.Model != nil {
			modelValue := reflect.ValueOf(db.Statement.Model)

			if modelValue.Kind() == reflect.Ptr && !modelValue.IsNil() {
				modelValue = modelValue.Elem()
			}

			if modelValue.Kind() == reflect.Struct {
				// 尝试使用 GORM 的方法
				if value, ok := db.Statement.Schema.PrioritizedPrimaryField.ValueOf(db.Statement.Context, reflect.ValueOf(db.Statement.Model)); ok {
					recordID = fmt.Sprintf("%v", value)
				} else {
					// 如果 GORM 方法失败，直接通过反射获取
					primaryFieldName := db.Statement.Schema.PrioritizedPrimaryField.Name
					if idField := modelValue.FieldByName(primaryFieldName); idField.IsValid() && idField.CanInterface() {
						recordID = fmt.Sprintf("%v", idField.Interface())
					}
				}
			}
		}

		// 如果从 Model 中没有获取到，尝试从 ReflectValue 获取
		if recordID == "" {
			if value, ok := db.Statement.Schema.PrioritizedPrimaryField.ValueOf(db.Statement.Context, db.Statement.ReflectValue); ok {
				recordID = fmt.Sprintf("%v", value)
			}
		}
	}

	if recordID == "" {
		return fmt.Errorf("record ID not found")
	}

	// 调用 API
	apiReq := &APIRequest{
		Method: "DELETE",
		Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records/%s", dialector.Config.AppToken, tableID, recordID),
	}

	_, err = dialector.Client.DoRequest(context.Background(), apiReq)
	if err != nil {
		return err
	}

	db.RowsAffected = 1
	return nil
}

// setRecordToStruct 将记录设置到结构体
func setRecordToStruct(structValue reflect.Value, record *Record, schema *schema.Schema, dialector *Dialector) error {
	if record == nil {
		return fmt.Errorf("记录不能为空")
	}
	if schema == nil {
		return fmt.Errorf("schema不能为空")
	}

	// 获取表字段信息用于类型转换
	tableName := schema.Table
	tableFieldsList, err := getTableFields(dialector, tableName)
	if err != nil {
		// 如果获取字段信息失败，记录警告并继续使用原始逻辑

		for _, field := range schema.Fields {
			// 处理主键字段：主键值来自 record.RecordID
			if field.PrimaryKey {
				if record.RecordID != "" {

					if err := field.Set(context.Background(), structValue, record.RecordID); err != nil {
						return fmt.Errorf("设置主键字段 %s 失败: %w", field.DBName, err)
					}
				}
				continue
			}

			// 处理普通字段：值来自 record.Fields
			if value, ok := record.Fields[field.DBName]; ok {

				if err := field.Set(context.Background(), structValue, value); err != nil {
					return fmt.Errorf("设置字段 %s 失败: %w", field.DBName, err)
				}
			}
		}
		return nil
	}

	// 将字段列表转换为map以便查找
	tableFields := make(map[string]*Field)
	for _, tableField := range tableFieldsList {
		tableFields[tableField.FieldName] = tableField
	}

	// 遍历结构体字段并设置值
	for _, field := range schema.Fields {
		// 处理主键字段：主键值来自 record.RecordID
		if field.PrimaryKey {
			if record.RecordID != "" {
				if err := field.Set(context.Background(), structValue, record.RecordID); err != nil {
					return fmt.Errorf("设置主键字段 %s 失败: %w", field.DBName, err)
				}
			}
			continue
		}

		// 处理普通字段：值来自 record.Fields
		if value, ok := record.Fields[field.DBName]; ok {
			// 查找对应的表字段信息
			if tableField, exists := tableFields[field.DBName]; exists {
				// 使用 ConvertToGoValue 进行类型转换
				convertedValue := tableField.ConvertToGoValue(value)
				if err := field.Set(context.Background(), structValue, convertedValue); err != nil {
					return fmt.Errorf("设置字段 %s 失败: %w", field.DBName, err)
				}
			} else {
				// 如果找不到字段信息，直接设置原始值
				if err := field.Set(context.Background(), structValue, value); err != nil {
					return fmt.Errorf("设置字段 %s 失败: %w", field.DBName, err)
				}
			}
		}
	}

	return nil
}
