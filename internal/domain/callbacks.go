package domain

// ProgressFunc - callback прогресса скачивания
type ProgressFunc func(downloaded int64)

// LogFunc - callback для отправки логов
type LogFunc func(msg string)
