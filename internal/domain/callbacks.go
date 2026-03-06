package domain

// ProgressFunc - callback прогресса скачивания
// total может быть -1 если размер неизвестен
type ProgressFunc func(downloaded, total int64)

// LogFunc - callback для отправки логов
type LogFunc func(msg string)
