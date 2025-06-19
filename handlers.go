package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jung-kurt/gofpdf"
)

var userSettings = make(map[int64]*UserSettings)

func handleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	msg := update.Message
	chatID := msg.Chat.ID

	if _, exists := userSettings[chatID]; !exists {
		userSettings[chatID] = DefaultSettings()
	}

	s := userSettings[chatID]

	switch {
	case msg.IsCommand():
		handleCommand(bot, msg)
	case msg.Photo != nil:
		handleImage(bot, msg)
	case msg.Text != "":
		if s.Stage != "" {
			handleStageInput(bot, msg)
		} else {
			handleSettingsResponse(bot, msg)
		}
	default:
		bot.Send(tgbotapi.NewMessage(chatID, tr(chatID, "send_image")))
	}
}

func handleCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	switch msg.Command() {
	case "start":
		msg := tgbotapi.NewMessage(chatID, tr(chatID, "start"))
		labelHelp := "/help"
		labelSettings := "/settings"
		labelAbout := "/about"
		if userSettings[chatID].Language == "Английский" {
			labelHelp = "Help"
			labelSettings = "Settings"
			labelAbout = "About"
		}
		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton(labelHelp),
				tgbotapi.NewKeyboardButton(labelSettings),
				tgbotapi.NewKeyboardButton(labelAbout),
			),
		)
		bot.Send(msg)
	case "help":
		bot.Send(tgbotapi.NewMessage(chatID, tr(chatID, "help")))
	case "settings":
		showSettings(bot, msg)
	case "about":
		bot.Send(tgbotapi.NewMessage(chatID, tr(chatID, "about")))
	default:
		bot.Send(tgbotapi.NewMessage(chatID, tr(chatID, "unknown_command")))
	}
}

func showSettings(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	settings := userSettings[chatID]

	reply := tgbotapi.NewMessage(chatID, tr(chatID, "settings_menu")+
		"\n1. "+tr(chatID, "language")+": "+settings.Language+
		"\n2. "+tr(chatID, "format")+": "+settings.Format+
		"\n3. "+tr(chatID, "model")+": "+settings.Model+
		"\n\n"+tr(chatID, "settings_instruction"))
	reply.ReplyMarkup = settingsKeyboard(settings.Language)
	bot.Send(reply)
}

func handleStageInput(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	settings := userSettings[chatID]
	text := msg.Text

	switch settings.Stage {
	case "language":
		if text == "Русский" || text == "Russian" {
			settings.Language = "Русский"
		} else {
			settings.Language = "Английский"
		}
		settings.Stage = ""
		bot.Send(tgbotapi.NewMessage(chatID, tr(chatID, "language_set")+": "+settings.Language))
	case "format":
		settings.Format = text
		settings.Stage = ""
		bot.Send(tgbotapi.NewMessage(chatID, tr(chatID, "format_set")+": "+settings.Format))
	case "model":
		settings.Model = text
		settings.Stage = ""
		bot.Send(tgbotapi.NewMessage(chatID, tr(chatID, "model_set")+": "+settings.Model))
	}
	showSettings(bot, msg)
}

func handleSettingsResponse(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	text := msg.Text
	s := userSettings[chatID]

	switch text {
	case getLabel(s.Language, "change_lang"):
		s.Stage = "language"
		req := tgbotapi.NewMessage(chatID, tr(chatID, "language")+":")
		req.ReplyMarkup = langKeyboard()
		bot.Send(req)
	case getLabel(s.Language, "change_format"):
		s.Stage = "format"
		req := tgbotapi.NewMessage(chatID, tr(chatID, "format")+":")
		req.ReplyMarkup = formatKeyboard(s.Language)
		bot.Send(req)
	case getLabel(s.Language, "change_model"):
		s.Stage = "model"
		req := tgbotapi.NewMessage(chatID, tr(chatID, "model")+":")
		req.ReplyMarkup = modelKeyboard(s.Language)
		bot.Send(req)
	default:
		bot.Send(tgbotapi.NewMessage(chatID, tr(chatID, "unknown_command")))
	}
}

func tr(chatID int64, key string) string {
	lang := userSettings[chatID].Language

	rus := map[string]string{
		"start":                "Привет! Я помогу тебе распознать рукописный текст. Отправь фото!",
		"help":                 "Команды: /start, /help, /settings, /about",
		"about":                "🤖 Я использую нейросеть для распознавания рукописного текста. Разработчик: Mikudayo Team",
		"unknown_command":      "Неизвестная команда. Напиши /help.",
		"send_image":           "Пожалуйста, отправь изображение с рукописным текстом.",
		"settings_menu":        "⚙️ Настройки:",
		"settings_instruction": "Выбери, что хочешь изменить:",
		"language":             "Язык интерфейса",
		"format":               "Формат ответа",
		"model":                "Модель",
		"language_set":         "Язык интерфейса изменён на",
		"format_set":           "Формат ответа установлен",
		"model_set":            "Выбрана модель",
		"error_image":          "Не удалось получить изображение.",
		"error_download":       "Ошибка загрузки изображения.",
		"error_save":           "Ошибка сохранения изображения.",
		"error_ocr":            "Ошибка при распознавании текста",
		"error_config":         "Ошибка конфигурации: IAM_TOKEN или FOLDER_ID не установлены.",
		"pdf_not_supported":    "PDF пока не поддерживается.",
		"timing_header":        "Время выполнения:",
		"ocr_time":             "OCR",
		"gpt_time":             "DeepSeek",
		"total_time":           "Общее",
		"seconds":              "сек",
		"ocr_result":           "Распознанный текст",
		"gpt_result":           "Восстановленный текст",
	}
	en := map[string]string{
		"start":                "Hello! I will help you recognize handwritten text. Just send a photo!",
		"help":                 "Commands: /start, /help, /settings, /about",
		"about":                "🤖 I use a neural net to recognize handwritten text. Developer: Mikudayo Team",
		"unknown_command":      "Unknown command. Type /help.",
		"send_image":           "Please send an image with handwritten text.",
		"settings_menu":        "⚙️ Settings:",
		"settings_instruction": "Choose what you'd like to change:",
		"language":             "Interface language",
		"format":               "Response format",
		"model":                "Model",
		"language_set":         "Language set to",
		"format_set":           "Response format set to",
		"model_set":            "Model set to",
		"error_image":          "Failed to retrieve image.",
		"error_download":       "Error downloading image.",
		"error_save":           "Error saving image.",
		"error_ocr":            "Error recognizing text",
		"error_config":         "Configuration error: IAM_TOKEN or FOLDER_ID not set.",
		"pdf_not_supported":    "PDF is not supported yet.",
		"timing_header":        "Execution time:",
		"ocr_time":             "OCR",
		"gpt_time":             "DeepSeek",
		"total_time":           "Total",
		"seconds":              "sec",
		"ocr_result":           "Recognized text",
		"gpt_result":           "Restored text",
	}

	if lang == "Английский" {
		return en[key]
	}
	return rus[key]
}

func handleImage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	photo := msg.Photo[len(msg.Photo)-1]
	fileURL, err := bot.GetFileDirectURL(photo.FileID)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, tr(chatID, "error_image")))
		return
	}

	resp, err := http.Get(fileURL)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, tr(chatID, "error_download")))
		return
	}
	defer resp.Body.Close()

	tmpPath := fmt.Sprintf("photo_%d.jpg", chatID)
	out, err := os.Create(tmpPath)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, tr(chatID, "error_save")))
		return
	}
	defer os.Remove(tmpPath)
	defer out.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, tr(chatID, "error_save")))
		return
	}

	iamToken := os.Getenv("IAM_TOKEN")
	folderID := os.Getenv("FOLDER_ID")
	mistralAPIKey := os.Getenv("MISTRAL_API_KEY")
	if iamToken == "" || folderID == "" || mistralAPIKey == "" {
		bot.Send(tgbotapi.NewMessage(chatID, tr(chatID, "error_config")))
		return
	}

	ocrText, gptText, _, err := ProcessImage(tmpPath, folderID, iamToken, mistralAPIKey)
	responseMsg := ""
	if ocrText != "" {
		// responseMsg += fmt.Sprintf("%s:\n%s\n\n", tr(chatID, "ocr_result"), ocrText)
	}
	if gptText != "" {
		responseMsg += fmt.Sprintf("%s", gptText)
		strings.ReplaceAll(responseMsg, "слишком неразборчиво 9905148", "Текст слишком неразборчивый, попробуйте сфотографировать получше и повторите попытку.")
	}
	if err != nil {
		responseMsg += fmt.Sprintf("\n\n%s: %v", tr(chatID, "error_ocr"), err)
	}

	format := userSettings[chatID].Format
	switch format {
	case "TXT-файл":
		if gptText != "" {
			file := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{
				Name:  "result.txt",
				Bytes: []byte(gptText),
			})
			bot.Send(file)
		}
	case "PDF-файл":
		if gptText != "" {
			// Создаем PDF
			pdf := gofpdf.New("P", "mm", "A4", "")
			pdf.AddPage()
			pdf.AddUTF8Font("DejaVu", "", "DejaVuSans.ttf")
			pdf.SetFont("DejaVu", "", 12)
			pdf.MultiCell(190, 5, gptText, "", "", false)

			// Конвертируем PDF в байты
			var buf bytes.Buffer
			err := pdf.Output(&buf)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при создании PDF"))
				// return err
			}

			// Отправляем PDF
			file := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{
				Name:  "result.pdf",
				Bytes: buf.Bytes(),
			})
			_, err = bot.Send(file)
			// return err
		}
	default:
		if responseMsg != "" {
			bot.Send(tgbotapi.NewMessage(chatID, responseMsg))
		}
	}
}
