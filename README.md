# vsixctl

Быстрый и надёжный асинхронный CLI-менеджер расширений для VS Code

[![CI](https://github.com/E-n-d-l-e-s-s-A-I/vsixctl/actions/workflows/ci.yml/badge.svg)](https://github.com/E-n-d-l-e-s-s-A-I/vsixctl/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/E-n-d-l-e-s-s-A-I/vsixctl)](https://github.com/E-n-d-l-e-s-s-A-I/vsixctl/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/E-n-d-l-e-s-s-A-I/vsixctl)](https://goreportcard.com/report/github.com/E-n-d-l-e-s-s-A-I/vsixctl)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE) 

[English](README.en.md)

## Возможности

- Поиск расширений в VS Code Marketplace
- Асинхронная установка, обновление и удаление 
- Установка конкретных версий расширений

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
