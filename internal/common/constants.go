package common

import "time"

// API 相关常量
const (
	// DefaultBaseURL 默认飞书 API 基础 URL
	DefaultBaseURL = "https://open.feishu.cn"

	// DefaultTimeout 默认请求超时时间
	DefaultTimeout = 30 * time.Second

	// DefaultMaxRetries 默认最大重试次数
	DefaultMaxRetries = 3

	// DefaultRetryInterval 默认重试间隔
	DefaultRetryInterval = 1 * time.Second

	// DefaultRateLimitQPS 默认每秒请求限制（飞书 API 限制）
	DefaultRateLimitQPS = 50

	// DefaultBatchSize 默认批量操作大小
	DefaultBatchSize = 100

	// MaxPageSize 最大页面大小（飞书 API 限制）
	MaxPageSize = 500

	// MaxBatchRecords 最大批量记录数（飞书 API 限制）
	MaxBatchRecords = 500

	// MaxBatchSize 最大批量大小（别名，保持兼容性）
	MaxBatchSize = MaxBatchRecords

	// MaxFieldNameLength 最大字段名长度
	MaxFieldNameLength = 100

	// MaxTableNameLength 最大表名长度
	MaxTableNameLength = 100
)

// 连接池相关常量
const (
	// DefaultMaxConnections 默认最大连接数
	DefaultMaxConnections = 100

	// DefaultConnectionTimeout 默认连接超时时间
	DefaultConnectionTimeout = 30 * time.Second

	// DefaultKeepAlive 默认保持连接时间
	DefaultKeepAlive = 30 * time.Second
)

// 时间戳相关常量
const (
	// MillisecondThreshold 毫秒时间戳阈值
	MillisecondThreshold = 1000000000000

	// MillisecondsPerSecond 每秒毫秒数
	MillisecondsPerSecond = 1000

	// NanosecondsPerMillisecond 每毫秒纳秒数
	NanosecondsPerMillisecond = 1000000
)

// HTTP 状态码相关常量
const (
	// HTTPStatusOKMin HTTP 成功状态码最小值
	HTTPStatusOKMin = 200

	// HTTPStatusClientErrorMin HTTP 客户端错误状态码最小值
	HTTPStatusClientErrorMin = 400

	// HTTPStatusClientErrorMax HTTP 客户端错误状态码最大值
	HTTPStatusClientErrorMax = 500

	// HTTPStatusServerErrorMin HTTP 服务器错误状态码最小值
	HTTPStatusServerErrorMin = 500

	// HTTPStatusTooManyRequests HTTP 请求过多状态码
	HTTPStatusTooManyRequests = 429
)

// 性能监控相关常量
const (
	// SlowQueryThreshold 慢查询阈值
	SlowQueryThreshold = 100 * time.Millisecond
)

// 字段类型常量
const (
	// SystemFieldCreatedTime 创建时间字段类型
	SystemFieldCreatedTime = 1001

	// SystemFieldModifiedTime 最后更新时间字段类型
	SystemFieldModifiedTime = 1002

	// SystemFieldCreatedUser 创建人字段类型
	SystemFieldCreatedUser = 1003

	// SystemFieldModifiedUser 修改人字段类型
	SystemFieldModifiedUser = 1004

	// SystemFieldAutoNumber 自动编号字段类型
	SystemFieldAutoNumber = 1005
)

// 错误消息常量
const (
	// ErrMsgInvalidConfig 无效配置错误消息
	ErrMsgInvalidConfig = "配置无效"

	// ErrMsgConnectionFailed 连接失败错误消息
	ErrMsgConnectionFailed = "连接失败"

	// ErrMsgAuthFailed 认证失败错误消息
	ErrMsgAuthFailed = "认证失败"

	// ErrMsgPermissionDenied 权限不足错误消息
	ErrMsgPermissionDenied = "权限不足"

	// ErrMsgRateLimitExceeded 请求频率超限错误消息
	ErrMsgRateLimitExceeded = "请求频率超限"
)
