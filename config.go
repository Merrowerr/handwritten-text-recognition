// config.go с поддержкой пользовательских настроек и кнопок
package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type UserSettings struct {
	Language string // Язык интерфейса
	Format   string // Формат вывода
	Model    string // Модель
	Stage    string // Временное поле для отслеживания выбора
}

func DefaultSettings() *UserSettings {
	return &UserSettings{
		Language: "Русский",
		Format:   "Простой текст",
		Model:    "Базовая (быстрая)",
		Stage:    "",
	}
}

func settingsKeyboard(lang string) tgbotapi.ReplyKeyboardMarkup {
	row1 := tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton(getLabel(lang, "change_lang")),
		tgbotapi.NewKeyboardButton(getLabel(lang, "change_format")),
	)
	row2 := tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton(getLabel(lang, "change_model")),
	)
	return tgbotapi.NewReplyKeyboard(row1, row2)
}

func langKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Русский"),
			tgbotapi.NewKeyboardButton("Английский"),
		),
	)
}

func formatKeyboard(lang string) tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(getLabel(lang, "plain_text")),
			tgbotapi.NewKeyboardButton("TXT-файл"),
			tgbotapi.NewKeyboardButton("PDF-файл"),
		),
	)
}

func modelKeyboard(lang string) tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(getLabel(lang, "model_basic")),
			tgbotapi.NewKeyboardButton(getLabel(lang, "model_improved")),
		),
	)
}

func getLabel(lang, key string) string {
	en := map[string]string{
		"change_lang":    "Change Language",
		"change_format":  "Change Format",
		"change_model":   "Change Model",
		"plain_text":     "Plain Text",
		"model_basic":    "Basic (fast)",
		"model_improved": "Improved (accurate)",
	}
	ru := map[string]string{
		"change_lang":    "Язык интерфейса",
		"change_format":  "Формат ответа",
		"change_model":   "Выбор модели",
		"plain_text":     "Простой текст",
		"model_basic":    "Базовая (быстрая)",
		"model_improved": "Улучшенная (точная)",
	}

	if lang == "Английский" {
		return en[key]
	}
	return ru[key]
}
