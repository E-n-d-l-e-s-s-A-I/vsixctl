# vsixctl

Быстрый и надёжный асинхронный CLI-менеджер расширений для VS Code

[![CI](https://github.com/E-n-d-l-e-s-s-A-I/vsixctl/actions/workflows/ci.yml/badge.svg)](https://github.com/E-n-d-l-e-s-s-A-I/vsixctl/actions/workflows/ci.yml)
[![Codecov](https://codecov.io/gh/E-n-d-l-e-s-s-A-I/vsixctl/branch/master/graph/badge.svg)](https://codecov.io/gh/E-n-d-l-e-s-s-A-I/vsixctl)
[![Release](https://img.shields.io/github/v/release/E-n-d-l-e-s-s-A-I/vsixctl)](https://github.com/E-n-d-l-e-s-s-A-I/vsixctl/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/E-n-d-l-e-s-s-A-I/vsixctl)](https://goreportcard.com/report/github.com/E-n-d-l-e-s-s-A-I/vsixctl)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE) 

[English](README.en.md)

## Возможности

- Поиск расширений в VS Code Marketplace
- Асинхронная установка, обновление и удаление 
- Установка конкретных версий расширений

![demo](assets/demo.gif)

## Установка

### Linux (WSL)

```sh
curl -sSL https://raw.githubusercontent.com/E-n-d-l-e-s-s-A-I/vsixctl/master/install.sh | sh
```

### Windows

Скачайте zip-архив для вашей платформы со страницы [Releases](https://github.com/E-n-d-l-e-s-s-A-I/vsixctl/releases), распакуйте и добавьте директорию с `vsixctl.exe` в переменную окружения `PATH`.

### Из исходников

Требуется [Go](https://go.dev/dl/) 1.25+.

```sh
go install github.com/E-n-d-l-e-s-s-A-I/vsixctl@latest
```

## Использование

```sh
# Полнотекстовый поиск расширений
vsixctl search go

# Поиск по id
vsixctl search golang.go --type id

# Поиск по названию
vsixctl search go --type name

# Установка расширений
vsixctl install golang.go ms-python.python

# Установка конкретной версии
vsixctl install golang.go@0.44.0

# Список установленных расширений
vsixctl ls

# Доступные версии расширения
vsixctl versions golang.go

# Обновление всех расширений
vsixctl update

# Обновление конкретных расширений
vsixctl update golang.go esbenp.prettier-vscode

# Удаление расширения
vsixctl rm golang.go
```

## Конфигурация

Конфиг файл создаётся автоматически при первом запуске со значениями по умолчанию. Путь к конфигу `~/.config/vsixctl/config.json`.

```json
{
  "logLevel": "warn",
  "extensionsPath": "~/.vscode/extensions",
  "platform": "linux-x64",
  "parallelism": 3,
  "sourceIdleTimeout": "2s",
  "queryTimeout": "7s",
  "queryRetries": 2,
  "progressBarStyle": "pacman"
}
```

| Поле                | Тип    | По умолчанию | Описание                                                                                                                                                       |
|---------------------|--------|--------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `logLevel`          | string | `"warn"`     | Уровень логирования: `debug`, `info`, `warn`, `error`                                                                                                          |
| `extensionsPath`    | string | —            | Путь к директории расширений VS Code. Определяется автоматически при первом запуске                                                                            |
| `platform`          | string | —            | Платформа: `linux-x64`, `linux-arm64`, `darwin-x64`, `darwin-arm64`, `win32-x64`, `win32-arm64`. Определяется автоматически при первом запуске                 |
| `parallelism`       | int    | `3`          | Количество параллельных загрузок                                                                                                                               |
| `sourceIdleTimeout` | string | `"2s"`       | Таймаут бездействия источника скачивания. Если источник перестал отдавать данные в течение этого таймаута, произойдет переключение на дополнительные источники |
| `queryTimeout`      | string | `"7s"`       | Таймаут запросов к marketplace, за исключением запроса на скачивание расширения                                                                                |
| `queryRetries`      | int    | `2`          | Количество повторных попыток при неудачном запросе метаданных                                                                                                  |
| `progressBarStyle`  | string | `"pacman"`   | Стиль прогресс-баров. На данный момент доступен только `pacman`                                                                                                |
