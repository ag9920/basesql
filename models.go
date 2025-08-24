package basesql

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ag9920/basesql/internal/common"
)

// FieldType 飞书多维表格字段类型枚举
// 定义了飞书多维表格支持的所有字段类型，每个类型对应一个唯一的数字标识
type FieldType int

// 飞书多维表格支持的字段类型常量定义
// 这些常量与飞书 API 文档中的字段类型编号保持一致
const (
	FieldTypeText         FieldType = 1    // 多行文本字段
	FieldTypeNumber       FieldType = 2    // 数字字段
	FieldTypeSingleSelect FieldType = 3    // 单选字段
	FieldTypeMultiSelect  FieldType = 4    // 多选字段
	FieldTypeDate         FieldType = 5    // 日期字段
	FieldTypeCheckbox     FieldType = 7    // 复选框字段
	FieldTypeUser         FieldType = 11   // 人员字段
	FieldTypePhone        FieldType = 13   // 电话号码字段
	FieldTypeURL          FieldType = 15   // 超链接字段
	FieldTypeAttachment   FieldType = 17   // 附件字段
	FieldTypeBarcode      FieldType = 18   // 条码字段
	FieldTypeProgress     FieldType = 19   // 进度字段
	FieldTypeCurrency     FieldType = 20   // 货币字段
	FieldTypeRating       FieldType = 21   // 评分字段
	FieldTypeFormula      FieldType = 22   // 公式字段
	FieldTypeLookup       FieldType = 23   // 查找引用字段
	FieldTypeCreatedTime  FieldType = 1001 // 创建时间字段（系统字段）
	FieldTypeModifiedTime FieldType = 1002 // 最后更新时间字段（系统字段）
	FieldTypeCreatedUser  FieldType = 1003 // 创建人字段（系统字段）
	FieldTypeModifiedUser FieldType = 1004 // 修改人字段（系统字段）
	FieldTypeAutoNumber   FieldType = 1005 // 自动编号字段（系统字段）
)

// FieldTypeMapping 字段类型到字符串的映射表
// 用于 JSON 序列化时将字段类型枚举转换为对应的字符串表示
var FieldTypeMapping = map[FieldType]string{
	FieldTypeText:         "1",
	FieldTypeNumber:       "2",
	FieldTypeSingleSelect: "3",
	FieldTypeMultiSelect:  "4",
	FieldTypeDate:         "5",
	FieldTypeCheckbox:     "7",
	FieldTypeUser:         "11",
	FieldTypePhone:        "13",
	FieldTypeURL:          "15",
	FieldTypeAttachment:   "17",
	FieldTypeBarcode:      "18",
	FieldTypeProgress:     "19",
	FieldTypeCurrency:     "20",
	FieldTypeRating:       "21",
	FieldTypeFormula:      "22",
	FieldTypeLookup:       "23",
	FieldTypeCreatedTime:  "1001",
	FieldTypeModifiedTime: "1002",
	FieldTypeCreatedUser:  "1003",
	FieldTypeModifiedUser: "1004",
	FieldTypeAutoNumber:   "1005",
}

// IsValidFieldType 检查字段类型是否有效
// 参数:
//   - fieldType: 要检查的字段类型
//
// 返回:
//   - bool: 如果字段类型有效返回 true，否则返回 false
func IsValidFieldType(fieldType FieldType) bool {
	_, exists := FieldTypeMapping[fieldType]
	return exists
}

// GetFieldTypeName 获取字段类型的可读名称
// 参数:
//   - fieldType: 字段类型枚举值
//
// 返回:
//   - string: 字段类型的可读名称
func GetFieldTypeName(fieldType FieldType) string {
	switch fieldType {
	case FieldTypeText:
		return "多行文本"
	case FieldTypeNumber:
		return "数字"
	case FieldTypeSingleSelect:
		return "单选"
	case FieldTypeMultiSelect:
		return "多选"
	case FieldTypeDate:
		return "日期"
	case FieldTypeCheckbox:
		return "复选框"
	case FieldTypeUser:
		return "人员"
	case FieldTypePhone:
		return "电话号码"
	case FieldTypeURL:
		return "超链接"
	case FieldTypeAttachment:
		return "附件"
	case FieldTypeBarcode:
		return "条码"
	case FieldTypeProgress:
		return "进度"
	case FieldTypeCurrency:
		return "货币"
	case FieldTypeRating:
		return "评分"
	case FieldTypeFormula:
		return "公式"
	case FieldTypeLookup:
		return "查找引用"
	case FieldTypeCreatedTime:
		return "创建时间"
	case FieldTypeModifiedTime:
		return "最后更新时间"
	case FieldTypeCreatedUser:
		return "创建人"
	case FieldTypeModifiedUser:
		return "修改人"
	case FieldTypeAutoNumber:
		return "自动编号"
	default:
		return "未知类型"
	}
}

// Table 飞书多维表格的数据表结构
// 表示一个完整的数据表，包含表的基本信息和所有字段定义
type Table struct {
	TableID  string   `json:"table_id"` // 表的唯一标识符
	Name     string   `json:"name"`     // 表名称
	Revision int64    `json:"revision"` // 表的版本号，用于并发控制
	Fields   []*Field `json:"fields"`   // 表中的所有字段列表
}

// Validate 验证表结构的有效性
// 返回:
//   - error: 如果表结构无效，返回相应的错误信息
func (t *Table) Validate() error {
	if t.TableID == "" {
		return ErrInvalidConfig("表 ID 不能为空")
	}
	if t.Name == "" {
		return ErrInvalidConfig("表名称 不能为空")
	}
	if len(t.Fields) == 0 {
		return ErrInvalidConfig("表必须至少包含一个字段")
	}

	// 验证所有字段
	for i, field := range t.Fields {
		if err := field.Validate(); err != nil {
			return fmt.Errorf("字段 %d 验证失败: %w", i, err)
		}
	}

	return nil
}

// GetFieldByName 根据字段名称获取字段
// 参数:
//   - fieldName: 字段名称
//
// 返回:
//   - *Field: 找到的字段，如果未找到返回 nil
func (t *Table) GetFieldByName(fieldName string) *Field {
	for _, field := range t.Fields {
		if field.FieldName == fieldName {
			return field
		}
	}
	return nil
}

// GetFieldByID 根据字段 ID 获取字段
// 参数:
//   - fieldID: 字段 ID
//
// 返回:
//   - *Field: 找到的字段，如果未找到返回 nil
func (t *Table) GetFieldByID(fieldID string) *Field {
	for _, field := range t.Fields {
		if field.FieldID == fieldID {
			return field
		}
	}
	return nil
}

// Field 飞书多维表格的字段结构
// 表示表中的一个字段，包含字段的所有属性和配置信息
type Field struct {
	FieldID     string                 `json:"field_id"`    // 字段的唯一标识符
	FieldName   string                 `json:"field_name"`  // 字段名称
	Type        FieldType              `json:"type"`        // 字段类型
	Property    map[string]interface{} `json:"property"`    // 字段属性配置（如选项列表、格式设置等）
	Description string                 `json:"description"` // 字段描述
	IsPrimary   bool                   `json:"is_primary"`  // 是否为主键字段
}

// Validate 验证字段结构的有效性
// 返回:
//   - error: 如果字段结构无效，返回相应的错误信息
func (f *Field) Validate() error {
	if f.FieldID == "" {
		return ErrInvalidConfig("字段 ID 不能为空")
	}
	if f.FieldName == "" {
		return ErrInvalidConfig("字段名称 不能为空")
	}
	if !IsValidFieldType(f.Type) {
		return ErrInvalidConfig(fmt.Sprintf("无效的字段类型: %d", f.Type))
	}

	// 验证字段名称格式（不能包含特殊字符）
	if !common.IsValidIdentifier(f.FieldName) {
		return ErrInvalidConfig("字段名称 格式无效")
	}

	return nil
}

// IsSystemField 判断是否为系统字段
// 返回:
//   - bool: 如果是系统字段返回 true，否则返回 false
func (f *Field) IsSystemField() bool {
	return f.Type >= FieldTypeCreatedTime && f.Type <= FieldTypeAutoNumber
}

// IsReadOnly 判断字段是否为只读
// 返回:
//   - bool: 如果字段只读返回 true，否则返回 false
func (f *Field) IsReadOnly() bool {
	// 系统字段和公式字段通常是只读的
	return f.IsSystemField() || f.Type == FieldTypeFormula || f.Type == FieldTypeLookup
}

// MarshalJSON 自定义 JSON 序列化
func (f *Field) MarshalJSON() ([]byte, error) {
	type Alias Field
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  FieldTypeMapping[f.Type],
		Alias: (*Alias)(f),
	})
}

// UnmarshalJSON 自定义 JSON 反序列化
func (f *Field) UnmarshalJSON(data []byte) error {
	type Alias Field
	aux := &struct {
		Type int `json:"type"`
		*Alias
	}{
		Alias: (*Alias)(f),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// 直接将数字类型转换为 FieldType
	f.Type = FieldType(aux.Type)
	return nil
}

// Record 飞书多维表格的记录结构
// 表示表中的一条数据记录，包含记录的所有字段值和元数据信息
type Record struct {
	RecordID       string                 `json:"record_id"`        // 记录的唯一标识符
	Fields         map[string]interface{} `json:"fields"`           // 记录的字段值映射，key 为字段名，value 为字段值
	CreatedTime    int64                  `json:"created_time"`     // 记录创建时间（毫秒时间戳）
	LastModified   int64                  `json:"last_modified"`    // 记录最后修改时间（毫秒时间戳）
	CreatedBy      *User                  `json:"created_by"`       // 记录创建者信息
	LastModifiedBy *User                  `json:"last_modified_by"` // 记录最后修改者信息
}

// Validate 验证记录结构的有效性
// 返回:
//   - error: 如果记录结构无效，返回相应的错误信息
func (r *Record) Validate() error {
	if r.RecordID == "" {
		return ErrInvalidConfig("记录 ID 不能为空")
	}
	if r.Fields == nil {
		return ErrInvalidConfig("记录字段不能为 nil")
	}
	return nil
}

// GetFieldValue 获取指定字段的值
// 参数:
//   - fieldName: 字段名称
//
// 返回:
//   - interface{}: 字段值，如果字段不存在返回 nil
//   - bool: 字段是否存在
func (r *Record) GetFieldValue(fieldName string) (interface{}, bool) {
	value, exists := r.Fields[fieldName]
	return value, exists
}

// SetFieldValue 设置指定字段的值
// 参数:
//   - fieldName: 字段名称
//   - value: 字段值
func (r *Record) SetFieldValue(fieldName string, value interface{}) {
	if r.Fields == nil {
		r.Fields = make(map[string]interface{})
	}
	r.Fields[fieldName] = value
}

// GetCreatedTime 获取记录创建时间
// 返回:
//   - time.Time: 创建时间
func (r *Record) GetCreatedTime() time.Time {
	if r.CreatedTime == 0 {
		return time.Time{}
	}
	return time.Unix(r.CreatedTime/1000, 0)
}

// GetLastModified 获取记录最后修改时间
// 返回:
//   - time.Time: 最后修改时间
func (r *Record) GetLastModified() time.Time {
	if r.LastModified == 0 {
		return time.Time{}
	}
	return time.Unix(r.LastModified/1000, 0)
}

// User 飞书用户信息结构
// 表示飞书系统中的用户基本信息
type User struct {
	ID     string `json:"id"`      // 用户的唯一标识符
	Name   string `json:"name"`    // 用户的中文名称
	EnName string `json:"en_name"` // 用户的英文名称
	Email  string `json:"email"`   // 用户的邮箱地址
}

// Validate 验证用户信息的有效性
// 返回:
//   - error: 如果用户信息无效，返回相应的错误信息
func (u *User) Validate() error {
	if u.ID == "" {
		return ErrInvalidConfig("用户 ID 不能为空")
	}
	if u.Name == "" && u.EnName == "" {
		return ErrInvalidConfig("用户名称不能为空")
	}
	return nil
}

// GetDisplayName 获取用户的显示名称
// 优先返回中文名称，如果中文名称为空则返回英文名称
// 返回:
//   - string: 用户的显示名称
func (u *User) GetDisplayName() string {
	if u.Name != "" {
		return u.Name
	}
	return u.EnName
}

// Option 飞书多维表格的选项结构
// 用于单选、多选等字段类型的选项定义
type Option struct {
	ID    string `json:"id"`    // 选项的唯一标识符
	Name  string `json:"name"`  // 选项名称
	Color int    `json:"color"` // 选项颜色代码
}

// Validate 验证选项结构的有效性
// 返回:
//   - error: 如果选项结构无效，返回相应的错误信息
func (o *Option) Validate() error {
	if o.ID == "" {
		return ErrInvalidConfig("选项 ID 不能为空")
	}
	if o.Name == "" {
		return ErrInvalidConfig("选项名称 不能为空")
	}
	return nil
}

// CreateRecordRequest 创建记录的请求结构
// 用于向飞书多维表格创建新记录
type CreateRecordRequest struct {
	Fields map[string]interface{} `json:"fields"` // 记录的字段值映射
}

// Validate 验证创建记录请求的有效性
// 返回:
//   - error: 如果请求结构无效，返回相应的错误信息
func (req *CreateRecordRequest) Validate() error {
	if req.Fields == nil || len(req.Fields) == 0 {
		return ErrInvalidConfig("创建记录时字段不能为空")
	}
	return nil
}

// CreateRecordAPIResponse 飞书API创建记录的完整响应结构
// 包含飞书API标准的code、data、msg字段
type CreateRecordAPIResponse struct {
	Code int                   `json:"code"` // 响应码
	Msg  string                `json:"msg"`  // 响应消息
	Data *CreateRecordResponse `json:"data"` // 实际数据
}

// CreateRecordResponse 创建记录的响应结构
// 包含创建操作的结果和新创建的记录信息
type CreateRecordResponse struct {
	Record *Record `json:"record"` // 创建的记录信息
}

// ListRecordsRequest 列表记录的请求结构
// 用于查询飞书多维表格中的记录列表
type ListRecordsRequest struct {
	ViewID           string         `json:"view_id,omitempty"`             // 视图 ID，指定查询的视图
	Filter           *FilterRequest `json:"filter,omitempty"`              // 过滤条件
	Sort             []string       `json:"sort,omitempty"`                // 排序条件
	FieldNames       []string       `json:"field_names,omitempty"`         // 指定返回的字段名列表
	TextFieldAsArray bool           `json:"text_field_as_array,omitempty"` // 文本字段是否以数组形式返回
	UserIDType       string         `json:"user_id_type,omitempty"`        // 用户 ID 类型
	DisplayFormula   bool           `json:"display_formula,omitempty"`     // 是否显示公式
	AutomaticFields  bool           `json:"automatic_fields,omitempty"`    // 是否包含自动字段
	PageToken        string         `json:"page_token,omitempty"`          // 分页标记
	PageSize         int            `json:"page_size,omitempty"`           // 每页记录数，最大 500
}

// Validate 验证列表记录请求的有效性
// 返回:
//   - error: 如果请求结构无效，返回相应的错误信息
func (req *ListRecordsRequest) Validate() error {
	if req.PageSize < 0 {
		return ErrInvalidConfig("页面大小 不能为负数")
	}
	if req.PageSize > common.MaxPageSize {
		return ErrInvalidConfig(fmt.Sprintf("页面大小 不能超过 %d", common.MaxPageSize))
	}
	return nil
}

// ListRecordsAPIResponse 飞书API列表记录的完整响应结构
// 包含飞书API标准的code、data、msg字段
type ListRecordsAPIResponse struct {
	Code int                  `json:"code"` // 响应码
	Msg  string               `json:"msg"`  // 响应消息
	Data *ListRecordsResponse `json:"data"` // 实际数据
}

// ListRecordsResponse 列表记录的响应结构
// 包含查询到的记录列表和分页信息
type ListRecordsResponse struct {
	HasMore   bool      `json:"has_more"`   // 是否还有更多数据
	PageToken string    `json:"page_token"` // 下一页的分页标记
	Total     int       `json:"total"`      // 总记录数
	Items     []*Record `json:"items"`      // 记录列表
}

// IsSuccess 判断列表记录操作是否成功
// 返回:
//   - bool: 操作是否成功
func (resp *ListRecordsResponse) IsSuccess() bool {
	return resp.Items != nil
}

// GetRecords 获取记录列表
// 返回:
//   - []*Record: 记录列表，如果没有数据返回空切片
func (resp *ListRecordsResponse) GetRecords() []*Record {
	if resp.Items == nil {
		return []*Record{}
	}
	return resp.Items
}

// UpdateRecordRequest 更新记录的请求结构
// 用于更新飞书多维表格中的现有记录
type UpdateRecordRequest struct {
	Fields map[string]interface{} `json:"fields"` // 要更新的字段值映射
}

// Validate 验证更新记录请求的有效性
// 返回:
//   - error: 如果请求结构无效，返回相应的错误信息
func (req *UpdateRecordRequest) Validate() error {
	if req.Fields == nil || len(req.Fields) == 0 {
		return ErrInvalidConfig("更新记录时字段不能为空")
	}
	return nil
}

// UpdateRecordResponse 更新记录的响应结构
// 包含更新操作的结果和更新后的记录信息
type UpdateRecordResponse struct {
	Record *Record `json:"record"` // 更新后的记录信息
}

// IsSuccess 判断更新记录操作是否成功
// 返回:
//   - bool: 操作是否成功
func (resp *UpdateRecordResponse) IsSuccess() bool {
	return resp.Record != nil
}

// GetRecord 获取更新后的记录
// 返回:
//   - *Record: 更新后的记录，如果操作失败返回 nil
func (resp *UpdateRecordResponse) GetRecord() *Record {
	return resp.Record
}

// BatchCreateRecordsRequest 批量创建记录的请求结构
// 用于一次性创建多条记录，提高操作效率
type BatchCreateRecordsRequest struct {
	Records []*CreateRecordRequest `json:"records"` // 要创建的记录列表
}

// Validate 验证批量创建记录请求的有效性
// 返回:
//   - error: 如果请求结构无效，返回相应的错误信息
func (req *BatchCreateRecordsRequest) Validate() error {
	if len(req.Records) == 0 {
		return ErrInvalidConfig("批量创建记录时记录列表 不能为空")
	}
	if len(req.Records) > common.MaxBatchSize {
		return ErrInvalidConfig(fmt.Sprintf("批量创建记录 数量不能超过 %d", common.MaxBatchSize))
	}

	// 验证每个记录
	for i, record := range req.Records {
		if err := record.Validate(); err != nil {
			return fmt.Errorf("记录 %d 验证失败: %w", i, err)
		}
	}
	return nil
}

// BatchCreateRecordsResponse 批量创建记录的响应结构
// 包含批量创建操作的结果和新创建的记录列表
type BatchCreateRecordsResponse struct {
	Records []*Record `json:"records"` // 创建的记录列表
}

// IsSuccess 判断批量创建记录操作是否成功
// 返回:
//   - bool: 操作是否成功
func (resp *BatchCreateRecordsResponse) IsSuccess() bool {
	return resp.Records != nil && len(resp.Records) > 0
}

// GetRecords 获取创建的记录列表
// 返回:
//   - []*Record: 创建的记录列表，如果操作失败返回空切片
func (resp *BatchCreateRecordsResponse) GetRecords() []*Record {
	if resp.Records == nil {
		return []*Record{}
	}
	return resp.Records
}

// GetRecordCount 获取成功创建的记录数量
// 返回:
//   - int: 成功创建的记录数量
func (resp *BatchCreateRecordsResponse) GetRecordCount() int {
	return len(resp.GetRecords())
}

// BatchUpdateRecordsRequest 批量更新记录的请求结构
// 用于一次性更新多条记录，提高操作效率
type BatchUpdateRecordsRequest struct {
	Records []*BatchUpdateRecord `json:"records"` // 要更新的记录列表
}

// Validate 验证批量更新记录请求的有效性
// 返回:
//   - error: 如果请求结构无效，返回相应的错误信息
func (req *BatchUpdateRecordsRequest) Validate() error {
	if len(req.Records) == 0 {
		return ErrInvalidConfig("批量更新记录时记录列表 不能为空")
	}
	if len(req.Records) > common.MaxBatchSize {
		return ErrInvalidConfig(fmt.Sprintf("批量更新记录 数量不能超过 %d", common.MaxBatchSize))
	}

	// 验证每个记录
	for i, record := range req.Records {
		if err := record.Validate(); err != nil {
			return fmt.Errorf("记录 %d 验证失败: %w", i, err)
		}
	}
	return nil
}

// BatchUpdateRecord 批量更新记录项
// 表示单个要更新的记录及其新的字段值
type BatchUpdateRecord struct {
	RecordID string                 `json:"record_id"` // 要更新的记录 ID
	Fields   map[string]interface{} `json:"fields"`    // 要更新的字段值映射
}

// Validate 验证批量更新记录项的有效性
// 返回:
//   - error: 如果记录项无效，返回相应的错误信息
func (r *BatchUpdateRecord) Validate() error {
	if r.RecordID == "" {
		return ErrInvalidConfig("记录 ID 不能为空")
	}
	if r.Fields == nil || len(r.Fields) == 0 {
		return ErrInvalidConfig("更新字段不能为空")
	}
	return nil
}

// BatchUpdateRecordsResponse 批量更新记录的响应结构
// 包含批量更新操作的结果和更新后的记录列表
type BatchUpdateRecordsResponse struct {
	Records []*Record `json:"records"` // 更新后的记录列表
}

// IsSuccess 判断批量更新记录操作是否成功
// 返回:
//   - bool: 操作是否成功
func (resp *BatchUpdateRecordsResponse) IsSuccess() bool {
	return resp.Records != nil && len(resp.Records) > 0
}

// GetRecords 获取更新后的记录列表
// 返回:
//   - []*Record: 更新后的记录列表，如果操作失败返回空切片
func (resp *BatchUpdateRecordsResponse) GetRecords() []*Record {
	if resp.Records == nil {
		return []*Record{}
	}
	return resp.Records
}

// GetRecordCount 获取成功更新的记录数量
// 返回:
//   - int: 成功更新的记录数量
func (resp *BatchUpdateRecordsResponse) GetRecordCount() int {
	return len(resp.GetRecords())
}

// FilterCondition 过滤条件结构
// 用于定义记录查询时的过滤条件
type FilterCondition struct {
	FieldName string        `json:"field_name"` // 字段名称
	Operator  string        `json:"operator"`   // 操作符（如 is、contains、isEmpty 等）
	Value     []interface{} `json:"value"`      // 过滤值列表，支持不同类型
}

// Validate 验证过滤条件的有效性
// 返回:
//   - error: 如果过滤条件无效，返回相应的错误信息
func (fc *FilterCondition) Validate() error {
	if fc.FieldName == "" {
		return ErrInvalidConfig("过滤条件的字段名 不能为空")
	}
	if fc.Operator == "" {
		return ErrInvalidConfig("过滤条件的操作符 不能为空")
	}

	// 验证操作符是否有效
	validOperators := []string{
		"is", "isNot", "contains", "doesNotContain",
		"isEmpty", "isNotEmpty", "isGreater", "isGreaterEqual",
		"isLess", "isLessEqual", "like", "in", "notIn",
	}
	validOperator := false
	for _, op := range validOperators {
		if fc.Operator == op {
			validOperator = true
			break
		}
	}
	if !validOperator {
		return ErrInvalidConfig(fmt.Sprintf("无效的过滤操作符: %s", fc.Operator))
	}

	// 某些操作符需要值
	needsValue := []string{"is", "isNot", "contains", "doesNotContain", "isGreater", "isGreaterEqual", "isLess", "isLessEqual", "like", "in", "notIn"}
	for _, op := range needsValue {
		if fc.Operator == op && len(fc.Value) == 0 {
			return ErrInvalidConfig(fmt.Sprintf("操作符 %s 需要提供过滤值", fc.Operator))
		}
	}

	return nil
}

// FilterRequest 过滤请求结构
// 用于组合多个过滤条件进行复杂查询
type FilterRequest struct {
	Conjunction string             `json:"conjunction"` // 连接符（and 或 or）
	Conditions  []*FilterCondition `json:"conditions"`  // 过滤条件列表
}

// Validate 验证过滤请求的有效性
// 返回:
//   - error: 如果过滤请求无效，返回相应的错误信息
func (fr *FilterRequest) Validate() error {
	if len(fr.Conditions) == 0 {
		return ErrInvalidConfig("过滤请求必须包含至少一个条件")
	}

	// 验证连接符
	if fr.Conjunction != "" && fr.Conjunction != "and" && fr.Conjunction != "or" {
		return ErrInvalidConfig("连接符只能是 'and' 或 'or'")
	}

	// 验证每个过滤条件
	for i, condition := range fr.Conditions {
		if err := condition.Validate(); err != nil {
			return fmt.Errorf("过滤条件 %d 验证失败: %w", i, err)
		}
	}

	return nil
}

// BatchDeleteRecordsRequest 批量删除记录的请求结构
// 用于一次性删除多条记录
type BatchDeleteRecordsRequest struct {
	Records []string `json:"records"` // 要删除的记录 ID 列表
}

// Validate 验证批量删除记录请求的有效性
// 返回:
//   - error: 如果请求结构无效，返回相应的错误信息
func (req *BatchDeleteRecordsRequest) Validate() error {
	if len(req.Records) == 0 {
		return ErrInvalidConfig("批量删除记录时记录 ID 列表 不能为空")
	}
	if len(req.Records) > common.MaxBatchSize {
		return ErrInvalidConfig(fmt.Sprintf("批量删除记录 数量不能超过 %d", common.MaxBatchSize))
	}

	// 验证每个记录 ID
	for i, recordID := range req.Records {
		if recordID == "" {
			return ErrInvalidConfig(fmt.Sprintf("记录 ID %d 不能为空", i))
		}
	}

	return nil
}

// BatchDeleteRecordsResponse 批量删除记录的响应结构
// 包含批量删除操作的结果
type BatchDeleteRecordsResponse struct {
	Records []string `json:"records"` // 成功删除的记录 ID 列表
}

// IsSuccess 判断批量删除记录操作是否成功
// 返回:
//   - bool: 操作是否成功
func (resp *BatchDeleteRecordsResponse) IsSuccess() bool {
	return resp.Records != nil && len(resp.Records) > 0
}

// GetDeletedRecords 获取成功删除的记录 ID 列表
// 返回:
//   - []string: 成功删除的记录 ID 列表
func (resp *BatchDeleteRecordsResponse) GetDeletedRecords() []string {
	if resp.Records == nil {
		return []string{}
	}
	return resp.Records
}

// GetDeletedCount 获取成功删除的记录数量
// 返回:
//   - int: 成功删除的记录数量
func (resp *BatchDeleteRecordsResponse) GetDeletedCount() int {
	return len(resp.GetDeletedRecords())
}

// CreateFieldRequest 创建字段的请求结构
// 用于在飞书多维表格中创建新字段
type CreateFieldRequest struct {
	FieldName   string                 `json:"field_name"`            // 字段名称
	Type        FieldType              `json:"type"`                  // 字段类型
	Property    map[string]interface{} `json:"property,omitempty"`    // 字段属性配置
	Description string                 `json:"description,omitempty"` // 字段描述
}

// Validate 验证创建字段请求的有效性
// 返回:
//   - error: 如果请求结构无效，返回相应的错误信息
func (req *CreateFieldRequest) Validate() error {
	if req.FieldName == "" {
		return ErrInvalidConfig("字段名称 不能为空")
	}
	if !IsValidFieldType(req.Type) {
		return ErrInvalidConfig(fmt.Sprintf("无效的字段类型: %d", req.Type))
	}

	// 验证字段名称格式
	if !common.IsValidIdentifier(req.FieldName) {
		return ErrInvalidConfig("字段名称 格式无效")
	}

	// 字段名称长度限制
	if len(req.FieldName) > common.MaxFieldNameLength {
		return ErrInvalidConfig(fmt.Sprintf("字段名称 长度不能超过 %d 个字符", common.MaxFieldNameLength))
	}

	return nil
}

// MarshalJSON 自定义 JSON 序列化
// 将 FieldType 枚举转换为整数类型以符合 API 要求
func (r *CreateFieldRequest) MarshalJSON() ([]byte, error) {
	type Alias CreateFieldRequest
	return json.Marshal(&struct {
		Type int `json:"type"`
		*Alias
	}{
		Type:  int(r.Type),
		Alias: (*Alias)(r),
	})
}

// CreateFieldResponse 创建字段的响应结构
// 包含创建操作的结果和新创建的字段信息
type CreateFieldResponse struct {
	Field *Field `json:"field"` // 创建的字段信息
}

// IsSuccess 判断创建字段操作是否成功
// 返回:
//   - bool: 操作是否成功
func (resp *CreateFieldResponse) IsSuccess() bool {
	return resp.Field != nil
}

// GetField 获取创建的字段
// 返回:
//   - *Field: 创建的字段，如果操作失败返回 nil
func (resp *CreateFieldResponse) GetField() *Field {
	return resp.Field
}

// ListFieldsResponse 查询字段的响应结构
// 包含表中所有字段的列表
type ListFieldsResponse struct {
	Items []*Field `json:"items"` // 字段列表
}

// ListFieldsAPIResponse 飞书API查询字段的完整响应结构
// 包含飞书API标准的code、data、msg字段
type ListFieldsAPIResponse struct {
	Code int                 `json:"code"` // 响应码
	Msg  string              `json:"msg"`  // 响应消息
	Data *ListFieldsResponse `json:"data"` // 实际数据
}

// IsSuccess 判断查询字段操作是否成功
// 返回:
//   - bool: 操作是否成功
func (resp *ListFieldsResponse) IsSuccess() bool {
	return resp.Items != nil
}

// GetFields 获取字段列表
// 返回:
//   - []*Field: 字段列表，如果操作失败返回空切片
func (resp *ListFieldsResponse) GetFields() []*Field {
	if resp.Items == nil {
		return []*Field{}
	}
	return resp.Items
}

// GetFieldCount 获取字段数量
// 返回:
//   - int: 字段数量
func (resp *ListFieldsResponse) GetFieldCount() int {
	return len(resp.GetFields())
}

// GetFieldByName 根据字段名称获取字段
// 参数:
//   - fieldName: 字段名称
//
// 返回:
//   - *Field: 找到的字段，如果未找到返回 nil
func (resp *ListFieldsResponse) GetFieldByName(fieldName string) *Field {
	for _, field := range resp.GetFields() {
		if field.FieldName == fieldName {
			return field
		}
	}
	return nil
}

// GetFieldByID 根据字段 ID 获取字段
// 参数:
//   - fieldID: 字段 ID
//
// 返回:
//   - *Field: 找到的字段，如果未找到返回 nil
func (resp *ListFieldsResponse) GetFieldByID(fieldID string) *Field {
	for _, field := range resp.GetFields() {
		if field.FieldID == fieldID {
			return field
		}
	}
	return nil
}

// CreateTableRequest 创建表的请求结构
// 用于在飞书多维表格中创建新的数据表
type CreateTableRequest struct {
	Table *TableRequest `json:"table"` // 表的详细信息
}

// Validate 验证创建表请求的有效性
// 返回:
//   - error: 如果请求结构无效，返回相应的错误信息
func (req *CreateTableRequest) Validate() error {
	if req.Table == nil {
		return ErrInvalidConfig("表信息不能为空")
	}
	return req.Table.Validate()
}

// TableRequest 表请求结构
// 包含创建表所需的所有信息
type TableRequest struct {
	Name            string                `json:"name"`                        // 表名称
	DefaultViewName string                `json:"default_view_name,omitempty"` // 默认视图名称
	Fields          []*CreateFieldRequest `json:"fields"`                      // 字段列表
}

// Validate 验证表请求的有效性
// 返回:
//   - error: 如果表请求无效，返回相应的错误信息
func (req *TableRequest) Validate() error {
	if req.Name == "" {
		return ErrInvalidConfig("表名称 不能为空")
	}

	// 表名称长度限制
	if len(req.Name) > common.MaxTableNameLength {
		return ErrInvalidConfig(fmt.Sprintf("表名称 长度不能超过 %d 个字符", common.MaxTableNameLength))
	}

	// 验证表名称格式
	if !common.IsValidIdentifier(req.Name) {
		return ErrInvalidConfig("表名称 格式无效")
	}

	if len(req.Fields) == 0 {
		return ErrInvalidConfig("表必须至少包含一个字段")
	}

	// 验证每个字段
	for i, field := range req.Fields {
		if err := field.Validate(); err != nil {
			return fmt.Errorf("字段 %d 验证失败: %w", i, err)
		}
	}

	// 检查字段名称是否重复
	fieldNames := make(map[string]bool)
	for i, field := range req.Fields {
		if fieldNames[field.FieldName] {
			return ErrInvalidConfig(fmt.Sprintf("字段名称 '%s' 在位置 %d 重复", field.FieldName, i))
		}
		fieldNames[field.FieldName] = true
	}

	return nil
}

// CreateDefaultViewRequest 创建默认视图的请求结构
// 用于为新创建的表设置默认视图
type CreateDefaultViewRequest struct {
	Name     string `json:"name"`      // 视图名称
	ViewType string `json:"view_type"` // 视图类型（如 grid、kanban 等）
}

// Validate 验证创建默认视图请求的有效性
// 返回:
//   - error: 如果请求结构无效，返回相应的错误信息
func (req *CreateDefaultViewRequest) Validate() error {
	if req.Name == "" {
		return ErrInvalidConfig("视图名称 不能为空")
	}
	if req.ViewType == "" {
		return ErrInvalidConfig("视图类型 不能为空")
	}

	// 验证视图类型
	validViewTypes := []string{"grid", "kanban", "gallery", "gantt", "form"}
	validType := false
	for _, vt := range validViewTypes {
		if req.ViewType == vt {
			validType = true
			break
		}
	}
	if !validType {
		return ErrInvalidConfig(fmt.Sprintf("无效的视图类型: %s", req.ViewType))
	}

	return nil
}

// CreateTableResponse 创建数据表的响应结构
// 包含创建操作的结果和新创建的表信息
type CreateTableResponse struct {
	TableID       string `json:"table_id"`        // 创建的表 ID
	DefaultViewID string `json:"default_view_id"` // 默认视图 ID
}

// IsSuccess 判断创建表操作是否成功
// 返回:
//   - bool: 操作是否成功
func (resp *CreateTableResponse) IsSuccess() bool {
	return resp.TableID != ""
}

// GetTableID 获取创建的表 ID
// 返回:
//   - string: 表 ID
func (resp *CreateTableResponse) GetTableID() string {
	return resp.TableID
}

// GetDefaultViewID 获取默认视图 ID
// 返回:
//   - string: 默认视图 ID
func (resp *CreateTableResponse) GetDefaultViewID() string {
	return resp.DefaultViewID
}

// ListTablesAPIResponse 飞书API查询数据表的完整响应结构
// 包含飞书API标准的code、data、msg字段
type ListTablesAPIResponse struct {
	Code int                 `json:"code"` // 响应码
	Msg  string              `json:"msg"`  // 响应消息
	Data *ListTablesResponse `json:"data"` // 实际数据
}

// ListTablesResponse 查询数据表的响应结构
// 包含多维表格中所有表的列表
type ListTablesResponse struct {
	HasMore   bool     `json:"has_more"`   // 是否还有更多数据
	PageToken string   `json:"page_token"` // 下一页的分页标记
	Total     int      `json:"total"`      // 总记录数
	Items     []*Table `json:"items"`      // 表列表
}

// IsSuccess 判断查询表操作是否成功
// 返回:
//   - bool: 操作是否成功
func (resp *ListTablesResponse) IsSuccess() bool {
	return resp.Items != nil
}

// GetTables 获取表列表
// 返回:
//   - []*Table: 表列表，如果操作失败返回空切片
func (resp *ListTablesResponse) GetTables() []*Table {
	if resp.Items == nil {
		return []*Table{}
	}
	return resp.Items
}

// GetTableCount 获取表数量
// 返回:
//   - int: 表数量
func (resp *ListTablesResponse) GetTableCount() int {
	return len(resp.GetTables())
}

// GetTableByName 根据表名称获取表
// 参数:
//   - tableName: 表名称
//
// 返回:
//   - *Table: 找到的表，如果未找到返回 nil
func (resp *ListTablesResponse) GetTableByName(tableName string) *Table {
	for _, table := range resp.GetTables() {
		if table.Name == tableName {
			return table
		}
	}
	return nil
}

// GetTableByID 根据表 ID 获取表
// 参数:
//   - tableID: 表 ID
//
// 返回:
//   - *Table: 找到的表，如果未找到返回 nil
func (resp *ListTablesResponse) GetTableByID(tableID string) *Table {
	for _, table := range resp.GetTables() {
		if table.TableID == tableID {
			return table
		}
	}
	return nil
}

// ConvertToGoValue 将飞书多维表格字段值转换为 Go 语言标准类型
// 该方法提供了类型安全的转换，支持多种数据类型的智能转换
// 参数:
//   - value: 来自飞书 API 的原始字段值
//
// 返回:
//   - interface{}: 转换后的 Go 语言类型值，转换失败时返回对应类型的零值
func (f *Field) ConvertToGoValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch f.Type {
	case FieldTypeText, FieldTypePhone, FieldTypeURL, FieldTypeBarcode:
		return f.convertToString(value)

	case FieldTypeNumber, FieldTypeCurrency, FieldTypeProgress, FieldTypeRating:
		return f.convertToFloat64(value)

	case FieldTypeCheckbox:
		return f.convertToBool(value)

	case FieldTypeDate, FieldTypeCreatedTime, FieldTypeModifiedTime:
		return f.convertToTime(value)

	case FieldTypeSingleSelect:
		return f.convertToSingleSelect(value)

	case FieldTypeMultiSelect:
		return f.convertToMultiSelect(value)

	case FieldTypeUser, FieldTypeCreatedUser, FieldTypeModifiedUser:
		return f.convertToUserList(value)

	case FieldTypeAttachment:
		return f.convertToAttachmentList(value)

	default:
		return value
	}
}

// convertToString 将值转换为字符串类型
func (f *Field) convertToString(value interface{}) string {
	if str, ok := value.(string); ok {
		return str
	}

	// 处理飞书API返回的文本字段格式：[{"text": "内容", "type": "text"}]
	if valueList, ok := value.([]interface{}); ok && len(valueList) > 0 {
		if textObj, ok := valueList[0].(map[string]interface{}); ok {
			if text, exists := textObj["text"]; exists {
				if str, ok := text.(string); ok {
					return str
				}
			}
		}
	}

	// 处理单个文本对象格式：{"text": "内容", "type": "text"}
	if textObj, ok := value.(map[string]interface{}); ok {
		if text, exists := textObj["text"]; exists {
			if str, ok := text.(string); ok {
				return str
			}
		}
	}

	return fmt.Sprintf("%v", value)
}

// convertToFloat64 将值转换为 float64 类型
func (f *Field) convertToFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		if num, err := strconv.ParseFloat(v, 64); err == nil {
			return num
		}
	}
	return 0.0
}

// convertToBool 将值转换为布尔类型
func (f *Field) convertToBool(value interface{}) bool {
	if b, ok := value.(bool); ok {
		return b
	}
	// 支持字符串转布尔
	if str, ok := value.(string); ok {
		switch strings.ToLower(str) {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		}
	}
	return false
}

// convertToTime 将值转换为时间类型
func (f *Field) convertToTime(value interface{}) time.Time {
	switch v := value.(type) {
	case int64:
		// 飞书时间戳是毫秒级别
		return time.Unix(v/1000, (v%1000)*1000000)
	case float64:
		// 处理浮点数时间戳
		return time.Unix(int64(v)/1000, int64(v*1000000)%1000000000)
	case string:
		// 尝试多种时间格式解析
		formats := []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, v); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

// convertToSingleSelect 将值转换为单选选项
func (f *Field) convertToSingleSelect(value interface{}) string {
	if option, ok := value.(map[string]interface{}); ok {
		if text, exists := option["text"]; exists {
			if str, ok := text.(string); ok {
				return str
			}
		}
		// 兼容旧版本 API
		if name, exists := option["name"]; exists {
			if str, ok := name.(string); ok {
				return str
			}
		}
	}
	// 如果直接是字符串，也支持
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}

// convertToMultiSelect 将值转换为多选选项列表
func (f *Field) convertToMultiSelect(value interface{}) []string {
	var result []string

	if options, ok := value.([]interface{}); ok {
		for _, opt := range options {
			if option, ok := opt.(map[string]interface{}); ok {
				if text, exists := option["text"]; exists {
					if str, ok := text.(string); ok {
						result = append(result, str)
					}
				} else if name, exists := option["name"]; exists {
					// 兼容旧版本 API
					if str, ok := name.(string); ok {
						result = append(result, str)
					}
				}
			} else if str, ok := opt.(string); ok {
				// 支持直接字符串数组
				result = append(result, str)
			}
		}
	}
	return result
}

// convertToUserList 将值转换为用户 ID 列表
func (f *Field) convertToUserList(value interface{}) []string {
	var result []string

	if users, ok := value.([]interface{}); ok {
		for _, u := range users {
			if user, ok := u.(map[string]interface{}); ok {
				if id, exists := user["id"]; exists {
					if str, ok := id.(string); ok {
						result = append(result, str)
					}
				}
			} else if str, ok := u.(string); ok {
				// 支持直接字符串数组
				result = append(result, str)
			}
		}
	}
	return result
}

// GetStringFromMap 安全地从 map 中获取字符串值
func GetStringFromMap(m map[string]interface{}, key string) string {
	return common.GetStringValue(m, key)
}

// convertToAttachmentList 将值转换为附件 URL 列表
func (f *Field) convertToAttachmentList(value interface{}) []string {
	var result []string

	if attachments, ok := value.([]interface{}); ok {
		for _, att := range attachments {
			if attachment, ok := att.(map[string]interface{}); ok {
				if url, exists := attachment["url"]; exists {
					if str, ok := url.(string); ok {
						result = append(result, str)
					}
				} else if token, exists := attachment["token"]; exists {
					// 兼容 token 字段
					if str, ok := token.(string); ok {
						result = append(result, str)
					}
				}
			}
		}
	}
	return result
}

// ConvertFromGoValue 将 Go 语言标准类型转换为飞书多维表格字段值
// 该方法将 Go 类型转换为飞书 API 期望的格式，确保数据能正确提交到飞书
// 参数:
//   - value: Go 语言类型的值
//
// 返回:
//   - interface{}: 转换后的飞书 API 格式值，转换失败时返回 nil 或对应类型的零值
func (f *Field) ConvertFromGoValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch f.Type {
	case FieldTypeText, FieldTypePhone, FieldTypeURL, FieldTypeBarcode:
		return f.convertFromString(value)

	case FieldTypeNumber, FieldTypeCurrency, FieldTypeProgress, FieldTypeRating:
		return f.convertFromNumber(value)

	case FieldTypeCheckbox:
		return f.convertFromBool(value)

	case FieldTypeDate, FieldTypeCreatedTime, FieldTypeModifiedTime:
		return f.convertFromTime(value)

	case FieldTypeSingleSelect:
		return f.convertFromSingleSelect(value)

	case FieldTypeMultiSelect:
		return f.convertFromMultiSelect(value)

	case FieldTypeUser, FieldTypeCreatedUser, FieldTypeModifiedUser:
		return f.convertFromUserList(value)

	case FieldTypeAttachment:
		return f.convertFromAttachmentList(value)

	default:
		return value
	}
}

// convertFromString 将值转换为字符串格式
func (f *Field) convertFromString(value interface{}) string {
	if str, ok := value.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", value)
}

// convertFromNumber 将值转换为数字格式
func (f *Field) convertFromNumber(value interface{}) float64 {
	switch v := value.(type) {
	case int:
		return float64(v)
	case int8:
		return float64(v)
	case int16:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case uint:
		return float64(v)
	case uint8:
		return float64(v)
	case uint16:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return v
	case string:
		if num, err := strconv.ParseFloat(v, 64); err == nil {
			return num
		}
	}
	return 0.0
}

// convertFromBool 将值转换为布尔格式
func (f *Field) convertFromBool(value interface{}) bool {
	if b, ok := value.(bool); ok {
		return b
	}
	// 支持字符串和数字转布尔
	switch v := value.(type) {
	case string:
		switch strings.ToLower(v) {
		case "true", "1", "yes", "on":
			return true
		}
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%v", v) != "0"
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%v", v) != "0"
	case float32, float64:
		return fmt.Sprintf("%v", v) != "0"
	}
	return false
}

// convertFromTime 将值转换为时间格式
func (f *Field) convertFromTime(value interface{}) interface{} {
	switch v := value.(type) {
	case time.Time:
		// 如果是零值时间，返回 nil 而不是当前时间
		if v.IsZero() {
			return nil
		}
		// 转换为毫秒时间戳
		return v.Unix()*1000 + int64(v.Nanosecond()/1000000)
	case *time.Time:
		if v == nil || v.IsZero() {
			return nil
		}
		return v.Unix()*1000 + int64(v.Nanosecond()/1000000)
	case string:
		// 尝试解析字符串时间
		formats := []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, v); err == nil {
				return t.Unix()*1000 + int64(t.Nanosecond()/1000000)
			}
		}
	case int64:
		// 假设是时间戳（秒或毫秒）
		if v > common.MillisecondThreshold {
			// 毫秒时间戳
			return v
		} else {
			// 秒时间戳
			return v * 1000
		}
	}
	return nil
}

// convertFromSingleSelect 将值转换为单选格式
func (f *Field) convertFromSingleSelect(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		if v == "" {
			return nil
		}
		return map[string]interface{}{
			"text": v,
		}
	case map[string]interface{}:
		// 如果已经是正确格式，直接返回
		if _, exists := v["text"]; exists {
			return v
		}
		// 尝试从其他字段获取文本
		if name, exists := v["name"]; exists {
			return map[string]interface{}{
				"text": name,
			}
		}
	}
	return nil
}

// convertFromMultiSelect 将值转换为多选格式
func (f *Field) convertFromMultiSelect(value interface{}) interface{} {
	switch v := value.(type) {
	case []string:
		if len(v) == 0 {
			return []map[string]interface{}{}
		}
		var options []map[string]interface{}
		for _, str := range v {
			if str != "" {
				options = append(options, map[string]interface{}{
					"text": str,
				})
			}
		}
		return options
	case []interface{}:
		var options []map[string]interface{}
		for _, item := range v {
			if str, ok := item.(string); ok && str != "" {
				options = append(options, map[string]interface{}{
					"text": str,
				})
			} else if opt, ok := item.(map[string]interface{}); ok {
				if _, exists := opt["text"]; exists {
					options = append(options, opt)
				}
			}
		}
		return options
	}
	return []map[string]interface{}{}
}

// convertFromUserList 将值转换为用户列表格式
func (f *Field) convertFromUserList(value interface{}) interface{} {
	switch v := value.(type) {
	case []string:
		if len(v) == 0 {
			return []map[string]interface{}{}
		}
		var users []map[string]interface{}
		for _, userID := range v {
			if userID != "" {
				users = append(users, map[string]interface{}{
					"id": userID,
				})
			}
		}
		return users
	case []interface{}:
		var users []map[string]interface{}
		for _, item := range v {
			if str, ok := item.(string); ok && str != "" {
				users = append(users, map[string]interface{}{
					"id": str,
				})
			} else if user, ok := item.(map[string]interface{}); ok {
				if _, exists := user["id"]; exists {
					users = append(users, user)
				}
			}
		}
		return users
	case string:
		// 单个用户 ID
		if v != "" {
			return []map[string]interface{}{
				{"id": v},
			}
		}
	}
	return []map[string]interface{}{}
}

// convertFromAttachmentList 将值转换为附件列表格式
func (f *Field) convertFromAttachmentList(value interface{}) interface{} {
	switch v := value.(type) {
	case []string:
		if len(v) == 0 {
			return []map[string]interface{}{}
		}
		var attachments []map[string]interface{}
		for _, url := range v {
			if url != "" {
				attachments = append(attachments, map[string]interface{}{
					"url": url,
				})
			}
		}
		return attachments
	case []interface{}:
		var attachments []map[string]interface{}
		for _, item := range v {
			if str, ok := item.(string); ok && str != "" {
				attachments = append(attachments, map[string]interface{}{
					"url": str,
				})
			} else if att, ok := item.(map[string]interface{}); ok {
				if _, exists := att["url"]; exists {
					attachments = append(attachments, att)
				} else if _, exists := att["token"]; exists {
					attachments = append(attachments, att)
				}
			}
		}
		return attachments
	}
	return []map[string]interface{}{}
}
