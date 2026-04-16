// =================================================================================
// Monitor Types - 监控相关类型定义（手动维护）
// =================================================================================

package service

// EventQuery 事件查询条件
type EventQuery struct {
	TargetName string `json:"targetName"` // 目标名称
	TableName  string `json:"tableName"`  // 表名
	EventType  string `json:"eventType"`  // 事件类型
	Success    *bool  `json:"success"`    // 是否成功
	StartTime  int64  `json:"startTime"`  // 开始时间
	EndTime    int64  `json:"endTime"`    // 结束时间
	Page       int    `json:"page"`       // 页码
	PageSize   int    `json:"pageSize"`   // 每页数量
}

// ErrorQuery 错误查询条件
type ErrorQuery struct {
	Level      string `json:"level"`      // 错误级别
	TargetName string `json:"targetName"` // 目标名称
	TableName  string `json:"tableName"`  // 表名
	Keyword    string `json:"keyword"`    // 关键词搜索
	StartTime  int64  `json:"startTime"`  // 开始时间
	EndTime    int64  `json:"endTime"`    // 结束时间
	Page       int    `json:"page"`       // 页码
	PageSize   int    `json:"pageSize"`   // 每页数量
}
