package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	basesql "github.com/ag9920/basesql"
	"github.com/ag9920/basesql/internal/common"
	"github.com/ag9920/basesql/internal/security"
	"gorm.io/gorm"
)

// Executor SQL 执行器
// 负责执行各种 SQL 命令并与飞书多维表格 API 交互
type Executor struct {
	db       *gorm.DB        // GORM 数据库连接
	client   *basesql.Client // BaseSQL 客户端
	appToken string          // 飞书应用 Token
	timeout  time.Duration   // 请求超时时间
}

// NewExecutor 创建新的 SQL 执行器
// 参数:
//   - db: GORM 数据库连接
//
// 返回:
//   - *Executor: 执行器实例
//   - error: 创建错误
func NewExecutor(db *gorm.DB) (*Executor, error) {
	if db == nil {
		return nil, fmt.Errorf("数据库连接不能为空")
	}

	// 从 GORM Dialector 中提取 BaseSQL 客户端信息
	dialector, ok := db.Dialector.(*basesql.Dialector)
	if !ok {
		return nil, fmt.Errorf("不支持的数据库类型，需要 BaseSQL Dialector")
	}

	return &Executor{
		db:       db,
		client:   dialector.Client,
		appToken: dialector.Config.AppToken,
		timeout:  dialector.Config.Timeout, // 使用配置中的超时时间
	}, nil
}

// Execute 执行 SQL 命令
// 根据命令类型分发到相应的处理函数
// 参数:
//   - cmd: 解析后的 SQL 命令
//
// 返回:
//   - error: 执行错误信息
func (e *Executor) Execute(cmd *common.SQLCommand) error {
	if cmd == nil {
		return fmt.Errorf("SQL 命令不能为空")
	}

	if e.db == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	// SQL注入验证
	validator := security.NewSQLInjectionValidator()
	if err := validator.ValidateSQL(cmd.RawSQL); err != nil {
		return fmt.Errorf("安全验证失败: %w", err)
	}

	// 记录执行开始时间
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		if duration > common.SlowQueryThreshold {
			fmt.Printf("⏱️  执行耗时: %v\n", duration)
		}
	}()

	switch cmd.Type {
	case common.CommandShow:
		// 根据 ShowType 进一步分发
		switch strings.ToUpper(cmd.ShowType) {
		case "TABLES":
			return e.showTables()
		case "DATABASES":
			return e.showDatabases()
		case "COLUMNS":
			return e.showColumns(cmd.Table)
		default:
			return fmt.Errorf("不支持的 SHOW 命令类型: %s", cmd.ShowType)
		}
	case common.CommandDescribe:
		return e.describe(cmd.Table)
	case common.CommandSelect:
		return e.selectData(cmd)
	case common.CommandInsert:
		return e.insertData(cmd)
	case common.CommandUpdate:
		return e.updateData(cmd)
	case common.CommandDelete:
		return e.deleteData(cmd)
	case common.CommandCreate:
		return e.createTable(cmd)
	case common.CommandDrop:
		return e.dropTable(cmd)
	default:
		return fmt.Errorf("不支持的命令类型: %s", cmd.Type)
	}
}

// showTables 显示所有表
// 通过飞书 API 获取多维表格中的所有表
// 返回:
//   - error: 执行错误信息
func (e *Executor) showTables() error {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	// 调用飞书 API 获取表列表
	tables, err := e.getTableList(ctx)
	if err != nil {
		return fmt.Errorf("获取表列表失败: %w", err)
	}

	// 显示表格头部
	fmt.Println("📋 数据表列表:")
	fmt.Println("+------------------+")
	fmt.Println("| Tables_in_base   |")
	fmt.Println("+------------------+")

	// 显示表列表
	if len(tables) == 0 {
		fmt.Println("|   <无数据表>     |")
	} else {
		for _, table := range tables {
			// 处理中文字符的显示宽度
			displayName := table.Name
			if common.GetDisplayWidth(displayName) > 16 {
				// 截断过长的表名
				displayName = common.TruncateString(displayName, 13) + "..."
			}
			fmt.Printf("| %-16s |\n", common.PadString(displayName, 16))
		}
	}

	fmt.Println("+------------------+")
	fmt.Printf("\n共 %d 个数据表\n", len(tables))

	return nil
}

// showDatabases 显示数据库列表
// 在飞书多维表格环境中，每个 App 相当于一个数据库
// 返回:
//   - error: 执行错误信息
func (e *Executor) showDatabases() error {
	fmt.Println("🗄️  数据库列表:")
	fmt.Println("+--------------------+")
	fmt.Println("| Database           |")
	fmt.Println("+--------------------+")
	fmt.Printf("| %-18s |\n", "feishu_base")
	fmt.Println("+--------------------+")
	fmt.Println("\n💡 在飞书多维表格中，每个应用相当于一个数据库")

	return nil
}

// showColumns 显示表的列信息
// 参数:
//   - tableName: 表名
//
// 返回:
//   - error: 执行错误信息
func (e *Executor) showColumns(tableName string) error {
	if tableName == "" {
		return fmt.Errorf("表名不能为空")
	}

	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	// 获取表 ID
	tableID, err := e.getTableID(ctx, tableName)
	if err != nil {
		return err
	}

	// 获取字段列表
	fields, err := e.getFieldsList(ctx, tableID)
	if err != nil {
		return err
	}

	// 显示表头
	fmt.Printf("📋 表 '%s' 的字段信息:\n", tableName)
	fmt.Println("+-------------+-------------+------+-----+---------+-------+")
	fmt.Println("| Field       | Type        | Null | Key | Default | Extra |")
	fmt.Println("+-------------+-------------+------+-----+---------+-------+")

	// 显示字段信息
	if len(fields) == 0 {
		fmt.Println("|   <无字段>   |             |      |     |         |       |")
	} else {
		for _, field := range fields {
			fieldType := getFieldTypeString(field.Type)
			nullable := "YES"
			key := ""
			if field.IsPrimary {
				key = "PRI"
			}
			defaultVal := "NULL"
			extra := ""

			// 处理字段名显示宽度
			fieldName := field.FieldName
			if common.GetDisplayWidth(fieldName) > 11 {
				fieldName = common.TruncateString(fieldName, 8) + "..."
			}

			fmt.Printf("| %-11s | %-11s | %-4s | %-3s | %-7s | %-5s |\n",
				common.PadString(fieldName, 11), fieldType, nullable, key, defaultVal, extra)
		}
	}

	fmt.Println("+-------------+-------------+------+-----+---------+-------+")
	fmt.Printf("\n共 %d 个字段\n", len(fields))

	return nil
}

// getTableList 获取表列表
// 参数:
//   - ctx: 上下文
//
// 返回:
//   - []basesql.Table: 表列表
//   - error: 错误信息
func (e *Executor) getTableList(ctx context.Context) ([]basesql.Table, error) {
	apiReq := &basesql.APIRequest{
		Method: "GET",
		Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables", e.appToken),
	}

	resp, err := e.client.DoRequest(ctx, apiReq)
	if err != nil {
		return nil, fmt.Errorf("API 请求失败: %w", err)
	}

	var apiResp basesql.ListTablesAPIResponse
	if err := json.Unmarshal(resp.Body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查API调用是否成功
	if apiResp.Code != 0 || apiResp.Data == nil {
		return nil, fmt.Errorf("API调用失败: code=%d, msg=%s", apiResp.Code, apiResp.Msg)
	}

	// 转换指针切片为值切片
	tables := make([]basesql.Table, len(apiResp.Data.Items))
	for i, item := range apiResp.Data.Items {
		tables[i] = *item
	}
	return tables, nil
}

// getTableID 根据表名获取表 ID
// 参数:
//   - ctx: 上下文
//   - tableName: 表名
//
// 返回:
//   - string: 表 ID
//   - error: 错误信息
func (e *Executor) getTableID(ctx context.Context, tableName string) (string, error) {
	tables, err := e.getTableList(ctx)
	if err != nil {
		return "", err
	}

	for _, table := range tables {
		if table.Name == tableName {
			return table.TableID, nil
		}
	}

	return "", fmt.Errorf("表 '%s' 不存在", tableName)
}

// getFieldsList 获取字段列表
// 参数:
//   - ctx: 上下文
//   - tableID: 表 ID
//
// 返回:
//   - []basesql.Field: 字段列表
//   - error: 错误信息
func (e *Executor) getFieldsList(ctx context.Context, tableID string) ([]basesql.Field, error) {
	apiReq := &basesql.APIRequest{
		Method: "GET",
		Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/fields", e.appToken, tableID),
	}

	resp, err := e.client.DoRequest(ctx, apiReq)
	if err != nil {
		return nil, fmt.Errorf("获取字段列表失败: %w", err)
	}

	var apiResp basesql.ListFieldsAPIResponse
	if err := json.Unmarshal(resp.Body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析字段列表响应失败: %w", err)
	}

	// 检查API调用是否成功
	if apiResp.Code != 0 {
		return nil, fmt.Errorf("API调用失败: code=%d, msg=%s", apiResp.Code, apiResp.Msg)
	}

	// 检查Data字段是否为nil
	if apiResp.Data == nil {
		return []basesql.Field{}, nil
	}

	// 检查fieldsResp.Items是否为nil或空
	if apiResp.Data.Items == nil {
		return []basesql.Field{}, nil
	}

	if len(apiResp.Data.Items) == 0 {
		return []basesql.Field{}, nil
	}

	// 转换指针切片为值切片
	fields := make([]basesql.Field, len(apiResp.Data.Items))
	for i, item := range apiResp.Data.Items {
		if item == nil {
			continue
		}
		fields[i] = *item
	}
	return fields, nil
}

// getRecords 获取记录列表（支持分页获取所有数据）
// 参数:
//   - ctx: 上下文
//   - tableID: 表 ID
//
// 返回:
//   - []basesql.Record: 记录列表
//   - error: 错误信息
func (e *Executor) getRecords(ctx context.Context, tableID string) ([]basesql.Record, error) {
	return e.getRecordsWithLimit(ctx, tableID, -1) // -1 表示无限制
}

// getRecordsWithLimit 获取记录列表（支持分页获取数据，可指定限制）
// 参数:
//   - ctx: 上下文
//   - tableID: 表 ID
//   - limit: 记录数量限制，-1表示无限制
//
// 返回:
//   - []basesql.Record: 记录列表
//   - error: 错误信息
func (e *Executor) getRecordsWithLimit(ctx context.Context, tableID string, limit int) ([]basesql.Record, error) {
	var allRecords []basesql.Record
	pageToken := ""
	pageNum := 1

	for {
		// 构建查询参数
		queryParams := fmt.Sprintf("?page_size=500")
		if pageToken != "" {
			queryParams += fmt.Sprintf("&page_token=%s", pageToken)
		}

		apiReq := &basesql.APIRequest{
			Method: "GET",
			Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records%s", e.appToken, tableID, queryParams),
		}

		// 显示进度提示
		if pageNum == 1 {
			fmt.Printf("正在获取数据...")
		} else {
			fmt.Printf("\r正在获取数据... 第 %d 页", pageNum)
		}

		resp, err := e.client.DoRequest(ctx, apiReq)
		if err != nil {
			fmt.Println() // 换行
			return nil, fmt.Errorf("API 请求失败: %w", err)
		}

		var apiResp basesql.ListRecordsAPIResponse
		if err := json.Unmarshal(resp.Body, &apiResp); err != nil {
			fmt.Println() // 换行
			return nil, fmt.Errorf("解析记录响应失败: %w", err)
		}

		// 检查API调用是否成功
		if apiResp.Code != 0 || apiResp.Data == nil {
			fmt.Println() // 换行
			return nil, fmt.Errorf("API调用失败: code=%d, msg=%s", apiResp.Code, apiResp.Msg)
		}

		// 转换指针切片为值切片并添加到总记录中
		for _, item := range apiResp.Data.Items {
			allRecords = append(allRecords, *item)
			// 如果设置了限制且已达到限制，停止获取
			if limit > 0 && len(allRecords) >= limit {
				break
			}
		}

		// 如果设置了限制且已达到限制，停止获取
		if limit > 0 && len(allRecords) >= limit {
			break
		}

		// 检查是否还有更多数据
		if !apiResp.Data.HasMore {
			break
		}

		// 更新分页标记
		pageToken = apiResp.Data.PageToken
		pageNum++
	}

	// 清除进度提示
	fmt.Printf("\r数据获取完成，共 %d 条记录\n", len(allRecords))
	return allRecords, nil
}

// renderResultTable 渲染查询结果表格（原有的API方式，保留向后兼容）
// 参数:
//   - fields: 字段列表
//   - records: 记录列表
//
// 返回:
//   - error: 渲染错误信息
func (e *Executor) renderResultTable(fields []basesql.Field, records []basesql.Record) error {
	if len(fields) == 0 {
		fmt.Println("📭 表中没有字段")
		return nil
	}

	// 构建字段名列表
	fieldNames := make([]string, 0, len(fields))
	for _, field := range fields {
		fieldNames = append(fieldNames, field.FieldName)
	}

	// 计算列宽
	colWidths := e.calculateColumnWidths(fieldNames, records)

	// 渲染表格
	e.printTableHeader(fieldNames, colWidths)
	e.printTableRows(fieldNames, records, colWidths)
	e.printTableFooter(fieldNames, colWidths)

	fmt.Printf("\n📊 查询返回 %d 行数据\n", len(records))
	return nil
}

// renderGormResultTable 渲染GORM查询结果表格（新的GORM方式）
// 参数:
//   - columns: 列名列表
//   - records: 记录列表（map格式）
//
// 返回:
//   - error: 渲染错误信息
func (e *Executor) renderGormResultTable(columns []string, records []map[string]interface{}) error {
	if len(columns) == 0 {
		fmt.Println("📭 表中没有字段")
		return nil
	}

	// 计算列宽
	colWidths := e.calculateGormColumnWidths(columns, records)

	// 渲染表格
	e.printGormTableHeader(columns, colWidths)
	e.printGormTableRows(columns, records, colWidths)
	e.printGormTableFooter(columns, colWidths)

	fmt.Printf("\n📊 查询返回 %d 行数据\n", len(records))
	return nil
}

// calculateColumnWidths 计算列宽
// 参数:
//   - fieldNames: 字段名列表
//   - records: 记录列表
//
// 返回:
//   - map[string]int: 列宽映射
func (e *Executor) calculateColumnWidths(fieldNames []string, records []basesql.Record) map[string]int {
	colWidths := make(map[string]int)

	for _, fieldName := range fieldNames {
		// 字段名的显示宽度
		colWidths[fieldName] = common.GetDisplayWidth(fieldName)

		// 遍历所有记录，找到最大宽度
		for _, record := range records {
			if value, exists := record.Fields[fieldName]; exists && value != nil {
				valStr := common.FormatValue(value)
				displayWidth := common.GetDisplayWidth(valStr)
				if displayWidth > colWidths[fieldName] {
					colWidths[fieldName] = displayWidth
				}
			}
		}

		// 设置最小和最大宽度
		if colWidths[fieldName] < 8 {
			colWidths[fieldName] = 8
		} else if colWidths[fieldName] > 30 {
			colWidths[fieldName] = 30 // 限制最大宽度
		}
	}

	return colWidths
}

// printTableHeader 打印表格头部
// 参数:
//   - fieldNames: 字段名列表
//   - colWidths: 列宽映射
func (e *Executor) printTableHeader(fieldNames []string, colWidths map[string]int) {
	// 打印顶部边框
	fmt.Print("+")
	for _, fieldName := range fieldNames {
		fmt.Printf("%s+", strings.Repeat("-", colWidths[fieldName]+2))
	}
	fmt.Println()

	// 打印字段名
	fmt.Print("|")
	for _, fieldName := range fieldNames {
		displayName := fieldName
		if common.GetDisplayWidth(displayName) > colWidths[fieldName] {
			displayName = common.TruncateString(displayName, colWidths[fieldName]-3) + "..."
		}
		fmt.Printf(" %s |", common.PadString(displayName, colWidths[fieldName]))
	}
	fmt.Println()

	// 打印分隔线
	fmt.Print("+")
	for _, fieldName := range fieldNames {
		fmt.Printf("%s+", strings.Repeat("-", colWidths[fieldName]+2))
	}
	fmt.Println()
}

// printTableRows 打印表格数据行
// 参数:
//   - fieldNames: 字段名列表
//   - records: 记录列表
//   - colWidths: 列宽映射
func (e *Executor) printTableRows(fieldNames []string, records []basesql.Record, colWidths map[string]int) {
	for _, record := range records {
		fmt.Print("|")
		for _, fieldName := range fieldNames {
			val := ""
			if value, exists := record.Fields[fieldName]; exists && value != nil {
				val = common.FormatValue(value)
				// 截断过长的值
				if common.GetDisplayWidth(val) > colWidths[fieldName] {
					val = common.TruncateString(val, colWidths[fieldName]-3) + "..."
				}
			}
			fmt.Printf(" %s |", common.PadString(val, colWidths[fieldName]))
		}
		fmt.Println()
	}
}

// printTableFooter 打印表格底部
// 参数:
//   - fieldNames: 字段名列表
//   - colWidths: 列宽映射
func (e *Executor) printTableFooter(fieldNames []string, colWidths map[string]int) {
	fmt.Print("+")
	for _, fieldName := range fieldNames {
		fmt.Printf("%s+", strings.Repeat("-", colWidths[fieldName]+2))
	}
	fmt.Println()
}

// calculateGormColumnWidths 计算GORM查询结果的列宽
// 参数:
//   - columns: 列名列表
//   - records: 记录列表（map格式）
//
// 返回:
//   - map[string]int: 列宽映射
func (e *Executor) calculateGormColumnWidths(columns []string, records []map[string]interface{}) map[string]int {
	colWidths := make(map[string]int)

	for _, column := range columns {
		// 列名的显示宽度
		colWidths[column] = common.GetDisplayWidth(column)

		// 遍历所有记录，找到最大宽度
		for _, record := range records {
			if value, exists := record[column]; exists && value != nil {
				valStr := common.FormatValue(value)
				displayWidth := common.GetDisplayWidth(valStr)
				if displayWidth > colWidths[column] {
					colWidths[column] = displayWidth
				}
			}
		}

		// 设置最小和最大宽度
		if colWidths[column] < 8 {
			colWidths[column] = 8
		} else if colWidths[column] > 30 {
			colWidths[column] = 30 // 限制最大宽度
		}
	}

	return colWidths
}

// printGormTableHeader 打印GORM查询结果的表格头部
// 参数:
//   - columns: 列名列表
//   - colWidths: 列宽映射
func (e *Executor) printGormTableHeader(columns []string, colWidths map[string]int) {
	// 打印顶部边框
	fmt.Print("+")
	for _, column := range columns {
		fmt.Printf("%s+", strings.Repeat("-", colWidths[column]+2))
	}
	fmt.Println()

	// 打印列名
	fmt.Print("|")
	for _, column := range columns {
		displayName := column
		if common.GetDisplayWidth(displayName) > colWidths[column] {
			displayName = common.TruncateString(displayName, colWidths[column]-3) + "..."
		}
		fmt.Printf(" %s |", common.PadString(displayName, colWidths[column]))
	}
	fmt.Println()

	// 打印分隔线
	fmt.Print("+")
	for _, column := range columns {
		fmt.Printf("%s+", strings.Repeat("-", colWidths[column]+2))
	}
	fmt.Println()
}

// printGormTableRows 打印GORM查询结果的表格数据行
// 参数:
//   - columns: 列名列表
//   - records: 记录列表（map格式）
//   - colWidths: 列宽映射
func (e *Executor) printGormTableRows(columns []string, records []map[string]interface{}, colWidths map[string]int) {
	for _, record := range records {
		fmt.Print("|")
		for _, column := range columns {
			val := ""
			if value, exists := record[column]; exists && value != nil {
				val = common.FormatValue(value)
				// 截断过长的值
				if common.GetDisplayWidth(val) > colWidths[column] {
					val = common.TruncateString(val, colWidths[column]-3) + "..."
				}
			}
			fmt.Printf(" %s |", common.PadString(val, colWidths[column]))
		}
		fmt.Println()
	}
}

// printGormTableFooter 打印GORM查询结果的表格底部
// 参数:
//   - columns: 列名列表
//   - colWidths: 列宽映射
func (e *Executor) printGormTableFooter(columns []string, colWidths map[string]int) {
	fmt.Print("+")
	for _, column := range columns {
		fmt.Printf("%s+", strings.Repeat("-", colWidths[column]+2))
	}
	fmt.Println()
}

// getStringValue 安全地从 map 中获取字符串值
// 参数:
//   - m: 数据映射
//   - key: 键名
//
// 返回:
//   - string: 字符串值
func getStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok && val != nil {
		return fmt.Sprintf("%v", val)
	}
	return ""
}

// getFieldTypeString 将飞书字段类型转换为 SQL 类型字符串
func getFieldTypeString(fieldType basesql.FieldType) string {
	switch fieldType {
	case basesql.FieldTypeText:
		return "text"
	case basesql.FieldTypeNumber:
		return "number"
	case basesql.FieldTypeSingleSelect:
		return "select"
	case basesql.FieldTypeMultiSelect:
		return "multiselect"
	case basesql.FieldTypeDate:
		return "date"
	case basesql.FieldTypeCheckbox:
		return "checkbox"
	case basesql.FieldTypeUser:
		return "user"
	case basesql.FieldTypePhone:
		return "phone"
	case basesql.FieldTypeURL:
		return "url"
	case basesql.FieldTypeAttachment:
		return "attachment"
	case basesql.FieldTypeBarcode:
		return "barcode"
	case basesql.FieldTypeProgress:
		return "progress"
	case basesql.FieldTypeCurrency:
		return "currency"
	case basesql.FieldTypeRating:
		return "rating"
	case basesql.FieldTypeFormula:
		return "formula"
	case basesql.FieldTypeLookup:
		return "lookup"
	case basesql.FieldTypeCreatedTime:
		return "created_time"
	case basesql.FieldTypeModifiedTime:
		return "modified_time"
	case basesql.FieldTypeCreatedUser:
		return "created_user"
	case basesql.FieldTypeModifiedUser:
		return "modified_user"
	case basesql.FieldTypeAutoNumber:
		return "auto_number"
	default:
		return "unknown"
	}
}

// describe 描述表结构
func (e *Executor) describe(tableName string) error {
	return e.showColumns(tableName)
}

// selectData 查询数据
// 直接使用飞书API执行查询，完全绕过GORM回调系统
// 支持聚合函数如 COUNT(*), SUM(field), AVG(field) 等
// 参数:
//   - cmd: SQL 命令对象
//
// 返回:
//   - error: 执行错误信息
func (e *Executor) selectData(cmd *common.SQLCommand) error {
	if cmd.Table == "" {
		return fmt.Errorf("表名不能为空")
	}

	fmt.Printf("执行查询: %s\n", cmd.RawSQL)

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	// 获取表 ID
	tableID, err := e.getTableID(ctx, cmd.Table)
	if err != nil {
		return fmt.Errorf("获取表ID失败: %w", err)
	}

	// 获取字段列表
	fields, err := e.getFieldsList(ctx, tableID)
	if err != nil {
		return fmt.Errorf("获取字段列表失败: %w", err)
	}

	// 获取记录列表（考虑LIMIT限制）
	var records []basesql.Record
	if cmd.Limit > 0 {
		records, err = e.getRecordsWithLimit(ctx, tableID, cmd.Limit)
	} else {
		records, err = e.getRecords(ctx, tableID)
	}
	if err != nil {
		return fmt.Errorf("获取记录失败: %w", err)
	}

	// 如果是聚合查询，处理聚合函数
	if cmd.IsAggregate {
		return e.handleAggregateQuery(cmd, fields, records)
	}

	// 应用WHERE条件过滤记录
	filteredRecords := e.filterRecords(records, fields, cmd.Condition)

	// 如果没有结果，显示空表
	if len(filteredRecords) == 0 {
		fmt.Printf("📭 查询结果为空\n")
		return nil
	}

	// 渲染查询结果表格
	return e.renderResultTable(fields, filteredRecords)
}

// insertData 插入数据
// 重构后使用GORM执行插入，消除与GORM driver的代码重复
// 参数:
//   - cmd: SQL 命令对象
//
// 返回:
//   - error: 执行错误信息
func (e *Executor) insertData(cmd *common.SQLCommand) error {
	if cmd.Table == "" {
		return fmt.Errorf("表名不能为空")
	}

	fmt.Printf("📝 执行插入: %s\n", cmd.RawSQL)

	// 使用GORM的原生SQL执行，通过rawCallback处理
	// 这样可以复用GORM driver中的所有插入逻辑，避免代码重复
	result := e.db.Exec(cmd.RawSQL)
	if result.Error != nil {
		return fmt.Errorf("插入执行失败: %w", result.Error)
	}

	fmt.Printf("✅ 成功插入 %d 条记录\n", result.RowsAffected)
	return nil
}

// updateData 更新数据
// 重构后使用GORM执行更新，消除与GORM driver的代码重复
// 参数:
//   - cmd: SQL 命令对象
//
// 返回:
//   - error: 执行错误信息
func (e *Executor) updateData(cmd *common.SQLCommand) error {
	if cmd.Table == "" {
		return fmt.Errorf("表名不能为空")
	}

	fmt.Printf("🔄 执行更新: %s\n", cmd.RawSQL)

	// 检查是否有 WHERE 条件
	if cmd.Where == "" {
		fmt.Println("⚠️  警告: 没有 WHERE 条件，将更新所有记录！")
	}

	// 使用GORM的原生SQL执行，通过rawCallback处理
	// 这样可以复用GORM driver中的所有更新逻辑，避免代码重复
	result := e.db.Exec(cmd.RawSQL)
	if result.Error != nil {
		return fmt.Errorf("更新执行失败: %w", result.Error)
	}

	fmt.Printf("✅ 更新成功，影响 %d 行\n", result.RowsAffected)
	return nil
}

// deleteData 删除数据
// 重构后使用GORM执行删除，消除与GORM driver的代码重复
// 参数:
//   - cmd: SQL 命令对象
//
// 返回:
//   - error: 执行错误信息
func (e *Executor) deleteData(cmd *common.SQLCommand) error {
	if cmd.Table == "" {
		return fmt.Errorf("表名不能为空")
	}

	fmt.Printf("🗑️  执行删除: %s\n", cmd.RawSQL)

	// 检查是否有 WHERE 条件
	if cmd.Where == "" {
		fmt.Println("⚠️  警告: 没有 WHERE 条件，将删除所有数据！")
		fmt.Print("确认要继续吗？(y/N): ")
		// 这里可以添加用户确认逻辑
	}

	// 使用GORM的原生SQL执行，通过rawCallback处理
	// 这样可以复用GORM driver中的所有删除逻辑，避免代码重复
	result := e.db.Exec(cmd.RawSQL)
	if result.Error != nil {
		return fmt.Errorf("删除执行失败: %w", result.Error)
	}

	fmt.Printf("✅ 删除成功，影响 %d 行\n", result.RowsAffected)
	return nil
}

// createTable 创建表
// 参数:
//   - cmd: SQL 命令对象
//
// 返回:
//   - error: 执行错误信息
func (e *Executor) createTable(cmd *common.SQLCommand) error {
	if cmd.Table == "" {
		return fmt.Errorf("表名不能为空")
	}

	fmt.Printf("🏗️  执行创建表: %s\n", cmd.RawSQL)

	// 执行原生 SQL 创建表
	result := e.db.Exec(cmd.RawSQL)
	if result.Error != nil {
		return fmt.Errorf("创建表执行失败: %w", result.Error)
	}

	fmt.Printf("✅ 表 '%s' 创建成功\n", cmd.Table)
	return nil
}

// dropTable 删除表
// 参数:
//   - cmd: SQL 命令对象
//
// 返回:
//   - error: 执行错误信息
func (e *Executor) dropTable(cmd *common.SQLCommand) error {
	if cmd.Table == "" {
		return fmt.Errorf("表名不能为空")
	}

	fmt.Printf("🗑️  执行删除表: %s\n", cmd.RawSQL)
	fmt.Printf("⚠️  警告: 即将删除表 '%s' 及其所有数据！\n", cmd.Table)
	fmt.Print("确认要继续吗？(y/N): ")
	// 这里可以添加用户确认逻辑

	// 执行原生 SQL 删除表
	result := e.db.Exec(cmd.RawSQL)
	if result.Error != nil {
		return fmt.Errorf("删除表执行失败: %w", result.Error)
	}

	fmt.Printf("✅ 表 '%s' 删除成功\n", cmd.Table)
	return nil
}

// handleAggregateQuery 处理聚合查询
// 参数:
//   - cmd: SQL 命令对象
//   - fields: 字段列表
//   - records: 记录列表
//
// 返回:
//   - error: 执行错误信息
func (e *Executor) handleAggregateQuery(cmd *common.SQLCommand, fields []basesql.Field, records []basesql.Record) error {
	// 首先应用WHERE条件过滤记录
	filteredRecords := e.filterRecords(records, fields, cmd.Condition)

	var result interface{}
	var err error

	// 根据聚合函数类型计算结果
	switch cmd.AggregateFunction {
	case "COUNT":
		result = len(filteredRecords)
	case "SUM":
		result, err = e.calculateSum(filteredRecords, fields, cmd.AggregateField)
	case "AVG":
		result, err = e.calculateAvg(filteredRecords, fields, cmd.AggregateField)
	case "MIN":
		result, err = e.calculateMin(filteredRecords, fields, cmd.AggregateField)
	case "MAX":
		result, err = e.calculateMax(filteredRecords, fields, cmd.AggregateField)
	default:
		return fmt.Errorf("不支持的聚合函数: %s", cmd.AggregateFunction)
	}

	if err != nil {
		return fmt.Errorf("聚合计算失败: %w", err)
	}

	// 显示聚合结果
	fmt.Printf("+%s+\n", strings.Repeat("-", 20))
	fmt.Printf("| %-18s |\n", cmd.Fields[0])
	fmt.Printf("+%s+\n", strings.Repeat("-", 20))
	fmt.Printf("| %-18v |\n", result)
	fmt.Printf("+%s+\n", strings.Repeat("-", 20))
	fmt.Printf("\n📊 聚合查询返回 1 行数据\n")

	return nil
}

// filterRecords 根据WHERE条件过滤记录
// 参数:
//   - records: 原始记录列表
//   - fields: 字段列表
//   - conditions: WHERE条件
//
// 返回:
//   - []basesql.Record: 过滤后的记录列表
func (e *Executor) filterRecords(records []basesql.Record, fields []basesql.Field, conditions map[string]interface{}) []basesql.Record {
	if len(conditions) == 0 {
		return records
	}

	// 创建字段名到字段ID的映射
	fieldNameToID := make(map[string]string)
	for _, field := range fields {
		fieldNameToID[field.FieldName] = field.FieldID
	}

	var filtered []basesql.Record

	for _, record := range records {
		match := true

		for fieldName, expectedValue := range conditions {
			// 跳过操作符标记
			if strings.HasPrefix(fieldName, "_operator_") {
				continue
			}

			// 尝试使用字段名直接获取值
			actualValue := record.Fields[fieldName]

			// 如果使用字段名获取不到值，尝试使用字段ID
			if actualValue == nil {
				if fieldID, exists := fieldNameToID[fieldName]; exists {
					actualValue = record.Fields[fieldID]
				}
			}

			// 检查操作符
			operatorKey := "_operator_" + fieldName
			operator, hasOperator := conditions[operatorKey]

			if hasOperator && operator == "LIKE" {
				// LIKE操作
				if !e.matchLike(actualValue, expectedValue) {
					match = false
					break
				}
			} else {
				// 等值比较
				if !e.matchEqual(actualValue, expectedValue) {
					match = false
					break
				}
			}
		}

		if match {
			filtered = append(filtered, record)
		}
	}

	return filtered
}

// getFieldIDByName 根据字段名获取字段ID
func (e *Executor) getFieldIDByName(fields []basesql.Field, fieldName string) string {
	for _, field := range fields {
		if field.FieldName == fieldName {
			return field.FieldID
		}
	}
	return ""
}

// matchLike 执行LIKE匹配
func (e *Executor) matchLike(actualValue, expectedValue interface{}) bool {
	actualStr := fmt.Sprintf("%v", actualValue)
	expectedStr := fmt.Sprintf("%v", expectedValue)

	// 简单的LIKE实现，支持%通配符
	if strings.Contains(expectedStr, "%") {
		// 手动构建正则表达式模式
		pattern := ""
		for _, char := range expectedStr {
			if char == '%' {
				pattern += ".*"
			} else {
				// 转义正则表达式特殊字符
				charStr := string(char)
				if strings.ContainsAny(charStr, ".+*?^${}()|[]\\") {
					pattern += "\\" + charStr
				} else {
					pattern += charStr
				}
			}
		}

		// 确保完全匹配（从开始到结束）
		pattern = "^" + pattern + "$"

		matched, err := regexp.MatchString(pattern, actualStr)
		if err != nil {
			return false
		}

		return matched
	}

	// 如果没有通配符，检查是否包含
	return strings.Contains(actualStr, expectedStr)
}

// matchEqual 执行等值匹配
func (e *Executor) matchEqual(actualValue, expectedValue interface{}) bool {
	// 处理 nil 值
	if actualValue == nil {
		return expectedValue == nil || fmt.Sprintf("%v", expectedValue) == "" || fmt.Sprintf("%v", expectedValue) == "<nil>"
	}
	if expectedValue == nil {
		return actualValue == nil || fmt.Sprintf("%v", actualValue) == "" || fmt.Sprintf("%v", actualValue) == "<nil>"
	}

	// 转换为字符串进行比较
	actualStr := fmt.Sprintf("%v", actualValue)
	expectedStr := fmt.Sprintf("%v", expectedValue)

	// 处理空字符串和 "<nil>" 的情况
	if actualStr == "<nil>" {
		actualStr = ""
	}
	if expectedStr == "<nil>" {
		expectedStr = ""
	}

	return actualStr == expectedStr
}

// calculateSum 计算SUM聚合
func (e *Executor) calculateSum(records []basesql.Record, fields []basesql.Field, fieldName string) (float64, error) {
	if fieldName == "*" {
		return 0, fmt.Errorf("SUM函数不支持*参数")
	}

	fieldID := e.getFieldIDByName(fields, fieldName)
	if fieldID == "" {
		return 0, fmt.Errorf("字段 %s 不存在", fieldName)
	}

	var sum float64
	for _, record := range records {
		value := record.Fields[fieldID]
		if numValue, err := e.convertToNumber(value); err == nil {
			sum += numValue
		}
	}

	return sum, nil
}

// calculateAvg 计算AVG聚合
func (e *Executor) calculateAvg(records []basesql.Record, fields []basesql.Field, fieldName string) (float64, error) {
	sum, err := e.calculateSum(records, fields, fieldName)
	if err != nil {
		return 0, err
	}

	if len(records) == 0 {
		return 0, nil
	}

	return sum / float64(len(records)), nil
}

// calculateMin 计算MIN聚合
func (e *Executor) calculateMin(records []basesql.Record, fields []basesql.Field, fieldName string) (interface{}, error) {
	if fieldName == "*" {
		return nil, fmt.Errorf("MIN函数不支持*参数")
	}

	fieldID := e.getFieldIDByName(fields, fieldName)
	if fieldID == "" {
		return nil, fmt.Errorf("字段 %s 不存在", fieldName)
	}

	if len(records) == 0 {
		return nil, nil
	}

	var min interface{}
	for i, record := range records {
		value := record.Fields[fieldID]
		if i == 0 || e.compareValues(value, min) < 0 {
			min = value
		}
	}

	return min, nil
}

// calculateMax 计算MAX聚合
func (e *Executor) calculateMax(records []basesql.Record, fields []basesql.Field, fieldName string) (interface{}, error) {
	if fieldName == "*" {
		return nil, fmt.Errorf("MAX函数不支持*参数")
	}

	fieldID := e.getFieldIDByName(fields, fieldName)
	if fieldID == "" {
		return nil, fmt.Errorf("字段 %s 不存在", fieldName)
	}

	if len(records) == 0 {
		return nil, nil
	}

	var max interface{}
	for i, record := range records {
		value := record.Fields[fieldID]
		if i == 0 || e.compareValues(value, max) > 0 {
			max = value
		}
	}

	return max, nil
}

// convertToNumber 将值转换为数字
func (e *Executor) convertToNumber(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("无法转换为数字: %v", value)
	}
}

// compareValues 比较两个值
func (e *Executor) compareValues(a, b interface{}) int {
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)

	// 尝试数字比较
	if aNum, aErr := strconv.ParseFloat(aStr, 64); aErr == nil {
		if bNum, bErr := strconv.ParseFloat(bStr, 64); bErr == nil {
			if aNum < bNum {
				return -1
			} else if aNum > bNum {
				return 1
			}
			return 0
		}
	}

	// 字符串比较
	return strings.Compare(aStr, bStr)
}
