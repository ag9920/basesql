package basesql

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
)

type Migrator struct {
	*Dialector
	migrator.Migrator
}

// AutoMigrate 自动迁移表结构
func (m Migrator) AutoMigrate(values ...interface{}) error {
	for _, value := range values {
		if err := m.RunAutoMigrate(value); err != nil {
			return err
		}
	}
	return nil
}

// RunAutoMigrate 执行自动迁移
func (m Migrator) RunAutoMigrate(value interface{}) error {
	if err := m.CreateTable(value); err != nil {
		return err
	}

	if err := m.UpdateColumns(value); err != nil {
		return err
	}

	return nil
}

// CreateTable 创建表
func (m Migrator) CreateTable(values ...interface{}) error {
	for _, value := range values {
		tx := m.DB.Session(&gorm.Session{})
		if err := m.createTable(value, tx); err != nil {
			return err
		}
	}
	return nil
}

// createTable 创建单个表
func (m Migrator) createTable(value interface{}, tx *gorm.DB) error {
	schemaValue := tx.Statement.Schema
	if schemaValue == nil {
		var err error
		if schemaValue, err = schema.Parse(value, &sync.Map{}, tx.NamingStrategy); err != nil {
			return err
		}
	}

	// 检查表是否已存在
	if m.HasTable(schemaValue.Table) {
		return nil
	}

	// 创建字段列表
	var fields []*CreateFieldRequest
	for _, field := range schemaValue.Fields {
		// 跳过自增字段，但保留主键和唯一字段
		if field.AutoIncrement {
			continue
		}

		fields = append(fields, &CreateFieldRequest{
			FieldName:   field.DBName,
			Type:        m.getFieldType(field),
			Description: field.Comment,
			Property:    map[string]interface{}{},
		})
	}

	// 调用 API 创建表
	apiReq := &APIRequest{
		Method: "POST",
		Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables", m.Dialector.Config.AppToken),
		Body: &CreateTableRequest{
			Table: &TableRequest{
				Name:            schemaValue.Table,
				DefaultViewName: "默认视图",
				Fields:          fields,
			},
		},
	}

	_, err := m.Dialector.Client.DoRequest(context.Background(), apiReq)
	return err
}

// HasTable 检查表是否存在
func (m Migrator) HasTable(table interface{}) bool {
	var tableName string
	if v, ok := table.(string); ok {
		tableName = v
	} else {
		// 获取表名
		if stmt := m.DB.Statement; stmt.Schema != nil {
			tableName = stmt.Schema.Table
		} else {
			// 解析模型获取表名
			var err error
			schemaValue, err := schema.Parse(table, &sync.Map{}, m.DB.NamingStrategy)
			if err != nil {
				return false
			}
			tableName = schemaValue.Table
		}
	}

	// 调用 API 获取表列表
	apiReq := &APIRequest{
		Method: "GET",
		Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables", m.Dialector.Config.AppToken),
	}

	resp, err := m.Dialector.Client.DoRequest(context.Background(), apiReq)
	if err != nil {
		return false
	}

	var apiResp ListTablesAPIResponse
	if err := json.Unmarshal(resp.Body, &apiResp); err != nil {
		return false
	}

	// 检查API调用是否成功
	if apiResp.Code != 0 || apiResp.Data == nil {
		return false
	}

	// 检查表名是否存在
	for _, table := range apiResp.Data.Items {
		if table.Name == tableName {
			return true
		}
	}

	return false
}

// getTableID 通过表名获取表 ID
func (m Migrator) getTableID(tableName string) (string, error) {
	// 调用 API 获取表列表
	apiReq := &APIRequest{
		Method: "GET",
		Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables", m.Dialector.Config.AppToken),
	}

	resp, err := m.Dialector.Client.DoRequest(context.Background(), apiReq)
	if err != nil {
		return "", err
	}

	var apiResp ListTablesAPIResponse
	if err := json.Unmarshal(resp.Body, &apiResp); err != nil {
		return "", err
	}

	// 检查API调用是否成功
	if apiResp.Code != 0 || apiResp.Data == nil {
		return "", fmt.Errorf("API调用失败: code=%d, msg=%s", apiResp.Code, apiResp.Msg)
	}

	// 查找表名对应的 ID
	for _, table := range apiResp.Data.Items {
		if table.Name == tableName {
			return table.TableID, nil
		}
	}

	return "", fmt.Errorf("table %s not found", tableName)
}

// DropTable 删除表
func (m Migrator) DropTable(values ...interface{}) error {
	for _, value := range values {
		if err := m.dropTable(value); err != nil {
			return err
		}
	}
	return nil
}

// dropTable 删除单个表
func (m Migrator) dropTable(value interface{}) error {
	tableName := m.DB.NamingStrategy.TableName(value.(string))

	// 先获取表 ID
	tableID, err := m.getTableID(tableName)
	if err != nil {
		return err
	}

	// 调用 API 删除表
	apiReq := &APIRequest{
		Method: "DELETE",
		Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables/%s", m.Dialector.Config.AppToken, tableID),
	}

	_, err = m.Dialector.Client.DoRequest(context.Background(), apiReq)
	return err
}

// UpdateColumns 更新列
func (m Migrator) UpdateColumns(value interface{}) error {
	schemaValue := m.DB.Statement.Schema
	if schemaValue == nil {
		var err error
		if schemaValue, err = schema.Parse(value, &sync.Map{}, m.DB.NamingStrategy); err != nil {
			return err
		}
	}

	// 检查表是否存在，如果不存在则跳过更新
	if !m.HasTable(schemaValue.Table) {
		return nil
	}

	// 获取现有字段
	existingFields, err := m.getTableFields(schemaValue.Table)
	if err != nil {
		return err
	}

	// 更新字段
	for _, field := range schemaValue.Fields {
		// 跳过自增字段，但保留主键和唯一字段
		if field.AutoIncrement {
			continue
		}

		// 检查字段是否存在
		if existingField, ok := existingFields[field.DBName]; ok {
			// 跳过主字段的更新
			if existingField.IsPrimary {
				continue
			}

			// 检查字段类型是否需要更新
			expectedType := m.getFieldType(field)
			if existingField.Type != expectedType {
				// 更新字段
				if err := m.updateField(schemaValue.Table, existingField.FieldID, field); err != nil {
					return err
				}
			}
		} else {
			// 创建字段
			if err := m.addField(schemaValue.Table, field); err != nil {
				return err
			}
		}
	}

	return nil
}

// getTableFields 获取表的所有字段
func (m Migrator) getTableFields(tableName string) (map[string]*Field, error) {
	// 先获取表 ID
	tableID, err := m.getTableID(tableName)
	if err != nil {
		return nil, err
	}

	// 调用 API 获取字段列表
	apiReq := &APIRequest{
		Method: "GET",
		Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/fields", m.Dialector.Config.AppToken, tableID),
	}

	resp, err := m.Dialector.Client.DoRequest(context.Background(), apiReq)
	if err != nil {
		return nil, err
	}

	var listResp ListFieldsResponse
	if err := json.Unmarshal(resp.Body, &listResp); err != nil {
		return nil, err
	}

	// 构建字段映射
	fields := make(map[string]*Field)
	for _, field := range listResp.Items {
		fields[field.FieldName] = field
	}

	return fields, nil
}

// updateField 更新字段
func (m Migrator) updateField(tableName, fieldID string, field *schema.Field) error {
	// 先获取表 ID
	tableID, err := m.getTableID(tableName)
	if err != nil {
		return err
	}

	// 获取字段类型
	fieldType := m.getFieldType(field)

	// 调用 API 更新字段
	apiReq := &APIRequest{
		Method: "PUT",
		Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/fields/%s", m.Dialector.Config.AppToken, tableID, fieldID),
		Body: map[string]interface{}{
			"field_name":  field.DBName,
			"type":        int(fieldType),
			"ui_type":     m.getUIType(fieldType),
			"description": field.Comment,
			"property":    map[string]interface{}{},
		},
	}

	_, err = m.Dialector.Client.DoRequest(context.Background(), apiReq)
	return err
}

// addField 添加字段
func (m Migrator) addField(tableName string, field *schema.Field) error {
	// 先获取表 ID
	tableID, err := m.getTableID(tableName)
	if err != nil {
		return err
	}

	// 调用 API 创建字段
	apiReq := &APIRequest{
		Method: "POST",
		Path:   fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/fields", m.Dialector.Config.AppToken, tableID),
		Body: &CreateFieldRequest{
			FieldName:   field.DBName,
			Type:        m.getFieldType(field),
			Description: field.Comment,
			Property:    map[string]interface{}{},
		},
	}

	_, err = m.Dialector.Client.DoRequest(context.Background(), apiReq)
	return err
}

// getFieldType 获取字段类型
func (m Migrator) getFieldType(field *schema.Field) FieldType {
	switch field.DataType {
	case schema.Bool:
		return FieldTypeCheckbox
	case schema.Int, schema.Uint:
		return FieldTypeNumber
	case schema.Float:
		return FieldTypeNumber
	case schema.String:
		return FieldTypeText
	case schema.Time:
		return FieldTypeDate
	case schema.Bytes:
		return FieldTypeText
	default:
		return FieldTypeText
	}
}

// getUIType 获取字段的 UI 类型
func (m Migrator) getUIType(fieldType FieldType) string {
	switch fieldType {
	case FieldTypeText:
		return "Text"
	case FieldTypeNumber:
		return "Number"
	case FieldTypeCheckbox:
		return "Checkbox"
	case FieldTypeDate:
		return "DateTime"
	case FieldTypeSingleSelect:
		return "SingleSelect"
	case FieldTypeMultiSelect:
		return "MultiSelect"
	case FieldTypeUser:
		return "User"
	case FieldTypePhone:
		return "Phone"
	case FieldTypeURL:
		return "Url"
	case FieldTypeAttachment:
		return "Attachment"
	case FieldTypeBarcode:
		return "Barcode"
	case FieldTypeProgress:
		return "Progress"
	case FieldTypeCurrency:
		return "Currency"
	case FieldTypeRating:
		return "Rating"
	case FieldTypeFormula:
		return "Formula"
	case FieldTypeLookup:
		return "Lookup"
	case FieldTypeCreatedTime:
		return "CreatedTime"
	case FieldTypeModifiedTime:
		return "ModifiedTime"
	case FieldTypeCreatedUser:
		return "CreatedUser"
	case FieldTypeModifiedUser:
		return "ModifiedUser"
	case FieldTypeAutoNumber:
		return "AutoNumber"
	default:
		return "Text"
	}
}
