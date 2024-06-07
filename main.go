package main

import (
	"log"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const layout = "02-01-2006 15:04"

func run() {
	apiToken := os.Getenv("TELEGRAM_APITOKEN")
	if apiToken == "" {
		log.Fatal("TELEGRAM_APITOKEN environment variable is required")
	}

	bot, err := tgbotapi.NewBotAPI(apiToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	var lista []string
	var partido struct {
		Cancha  string
		DiaHora time.Time
		Precio  string
	}

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			if update.Message.IsCommand() {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

				switch update.Message.Command() {
				case "sumo":
					lista = sumar_jugador(lista, update)
					msg.Text = "Jugadores que suman al partido por ahora: " + strings.Join(lista, ", ")
				case "bajar":
					lista = bajar_jugador(lista, update.Message.From.FirstName)
					msg.Text = "Se borró de la lista de jugadores a " + update.Message.From.FirstName
				case "jugadores":
					msg.Text = "Los jugadores que van al partido por ahora son: " + strings.Join(lista, ", ")

				case "crearpartido":
					msg.Text = "¿Qué tipo de cancha querés?"
					msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
						tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData("Fútbol 5", "cancha_futbol5"),
							tgbotapi.NewInlineKeyboardButtonData("Fútbol 7", "cancha_futbol7"),
							tgbotapi.NewInlineKeyboardButtonData("Fútbol 8", "cancha_futbol8"),
						),
					)

				case "partido":
					if partido.Cancha == "" {
						msg.Text = "Todavía no hay un partido creado"
					} else {
						msg.Text = "Cancha: " + partido.Cancha + "\nPrecio: " + partido.Precio + "$\nDía y hora: " + partido.DiaHora.Format(layout) + "\nJugadores: " + strings.Join(lista, ", ")
					}
				case "estado":
					msg.Text = "Estoy funcionando"
				default:
					msg.Text = "No entiendo ese comando"
				}

				if _, err := bot.Send(msg); err != nil {
					log.Panic(err)
				}
			} else if update.Message.ReplyToMessage != nil && strings.Contains(update.Message.ReplyToMessage.Text, "¿Qué día y a qué hora?") {
				diaHora, err := time.Parse(layout, update.Message.Text)
				if err != nil {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Formato de fecha y hora incorrecto. Por favor, usa el formato DD-MM-YYYY HH:MM.")
					bot.Send(msg)
					continue
				}
				partido.DiaHora = diaHora

				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Partido creado:\nCancha: "+partido.Cancha+"\nPrecio: "+partido.Precio+"$\nDía y hora: "+partido.DiaHora.Format(layout)+"\nJugadores: "+strings.Join(lista, ", "))
				bot.Send(msg)
			}
		} else if update.CallbackQuery != nil {
			callback := update.CallbackQuery
			msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "")

			switch callback.Data {
			case "cancha_futbol5":
				partido.Cancha = "Fútbol 5"
			case "cancha_futbol7":
				partido.Cancha = "Fútbol 7"
			case "cancha_futbol8":
				partido.Cancha = "Fútbol 8"
			}

			msg.Text = "Has elegido " + partido.Cancha + ". ¿Qué día y a qué hora? (formato: DD-MM-YYYY HH:MM)"
			if partido.Cancha == "Fútbol 5" {
				partido.Precio = "10000"
			} else if partido.Cancha == "Fútbol 7" {
				partido.Precio = "14000"
			} else if partido.Cancha == "Fútbol 8" {
				partido.Precio = "16000"
			}

			msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}
			if _, err := bot.Send(msg); err != nil {
				log.Panic(err)
			}

			callbackResponse := tgbotapi.NewCallback(callback.ID, callback.Data)
			if _, err := bot.Request(callbackResponse); err != nil {
				log.Panic(err)
			}
		}
	}
}

func bajar_jugador(lista []string, element string) []string {
	index := -1
	for i, v := range lista {
		if v == element {
			index = i
			break
		}
	}

	if index == -1 {
		return lista
	}

	return append(lista[:index], lista[index+1:]...)
}

func sumar_jugador(lista []string, update tgbotapi.Update) []string {
	lista = append(lista, update.Message.From.FirstName)
	return lista
}

func main() {
	run()
}
