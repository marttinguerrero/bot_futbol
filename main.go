package main

import (
	"log"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const layout = "02-01-2006 15:04"

type Partido struct {
	Id        int
	Cancha    string
	DiaHora   time.Time
	Precio    string
	Ubicacion string
	Creado    bool
	Paso      int
}

type Jugador struct {
	Nombre string
	Pago   bool
}

func main() {
	run()
}

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

	updates := bot.GetUpdatesChan(u)
	lista := []Jugador{}
	partido := Partido{}

	for update := range updates {
		hora_actual := obtener_numeros(time.Now())
		hora_partido := obtener_numeros(partido.DiaHora)

		if partido.Creado && hora_partido < hora_actual {
			eliminarPartido(bot, update.Message.Chat.ID, &partido)
			partido = Partido{}
			continue
		}
		manejo_update(bot, update, &lista, &partido)
	}

}

func eliminarPartido(bot *tgbotapi.BotAPI, chatID int64, partido *Partido) {
	partido.Creado = false
	partido.Paso = 0
	msg := tgbotapi.NewMessage(chatID, "El partido ha finalizado. ¡Es hora de crear otro partido!")
	bot.Send(msg)
}

func manejo_update(bot *tgbotapi.BotAPI, update tgbotapi.Update, lista *[]Jugador, partido *Partido) {
	if update.Message != nil {
		manejo_mensaje(bot, update, lista, partido)
	} else if update.CallbackQuery != nil {
		manejo_callback(bot, update.CallbackQuery, lista, partido)
	}
}

func manejo_mensaje(bot *tgbotapi.BotAPI, update tgbotapi.Update, lista *[]Jugador, partido *Partido) {
	if update.Message.IsCommand() {
		manejo_comandos(bot, update, lista, partido)
	} else if update.Message.ReplyToMessage != nil {
		switch partido.Paso {
		case 1:
			manejo_ubicacion(bot, update, partido, lista)
		case 2:
			manejo_fecha(bot, update, partido, lista)
		}
	}
}

func manejo_comandos(bot *tgbotapi.BotAPI, update tgbotapi.Update, lista *[]Jugador, partido *Partido) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
	switch update.Message.Command() {
	case "sumo":
		if !partido.Creado {
			msg.Text = "Primero debes crear un partido con el comando /crearpartido"
		} else {
			*lista = append(*lista, Jugador{Nombre: update.Message.From.FirstName, Pago: false})
			msg.Text = "Jugadores que suman al partido por ahora: " + imprimir_nombres(*lista)
		}
	case "sumoa":
		if !partido.Creado {
			msg.Text = "Primero debes crear un partido con el comando /crearpartido"
		} else {
			parts := strings.SplitN(update.Message.Text, " ", 2)
			if len(parts) == 2 {
				nombreAmigo := parts[1]
				*lista = append(*lista, Jugador{Nombre: nombreAmigo, Pago: false})
				msg.Text = "Se agregó a " + nombreAmigo + " a la lista de jugadores.\nJugadores que suman al partido por ahora: " + imprimir_nombres(*lista)
			} else {
				msg.Text = "Por favor, proporciona un nombre después del comando /sumoa."
			}
		}
	case "bajar":
		*lista = bajar_jugador(*lista, update.Message.From.FirstName)
		msg.Text = "Se borró de la lista de jugadores a " + update.Message.From.FirstName
	case "bajoa":
		if !partido.Creado {
			msg.Text = "Primero debes crear un partido con el comando /crearpartido"
		} else {
			parts := strings.SplitN(update.Message.Text, " ", 2)
			if len(parts) == 2 {
				nombre_amigo := parts[1]
				*lista = bajar_jugador(*lista, nombre_amigo)
				msg.Text = "Se borró de la lista de jugadores a " + nombre_amigo
			} else {
				msg.Text = "Por favor, proporciona un nombre después del comando /bajoa."
			}
		}
	case "jugadores":
		if !partido.Creado {
			msg.Text = "Todavía no hay un partido creado"
		} else {
			msg.Text = "Los jugadores que van al partido por ahora son: " + imprimir_nombres(*lista)
		}
	case "crearpartido":
		if partido.Creado {
			msg.Text = "Ya hay un partido creado. No puedes crear otro partido."
		} else {
			partido.Paso = 1
			msg.Text = "¿Dónde querés jugar?"
			msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}
		}
	case "partido":
		if !partido.Creado {
			msg.Text = "Todavía no hay un partido creado"
		} else {
			msg.Text = "Cancha: " + partido.Cancha + "\nPrecio: " + partido.Precio + "$\nDía y hora: " + partido.DiaHora.Format(layout) + "\nUbicación: " + partido.Ubicacion + "\nJugadores: " + imprimir_nombres(*lista)
		}
	case "estado":
		msg.Text = "Estoy funcionando"
	default:
		msg.Text = "No entiendo ese comando"
	}

	if _, err := bot.Send(msg); err != nil {
		log.Panic(err)
	}
}

func manejo_ubicacion(bot *tgbotapi.BotAPI, update tgbotapi.Update, partido *Partido, lista *[]Jugador) {
	partido.Ubicacion = update.Message.Text
	partido.Paso = 2
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "¿Qué día y a qué hora? (formato: DD-MM-YYYY HH:MM)")
	msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}
	bot.Send(msg)
}
func manejo_fecha(bot *tgbotapi.BotAPI, update tgbotapi.Update, partido *Partido, lista *[]Jugador) {
	diaHora, err := time.Parse(layout, update.Message.Text)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Formato de fecha y hora incorrecto. Por favor, usa el formato DD-MM-YYYY HH:MM.")
		bot.Send(msg)
		return
	}
	partido.DiaHora = diaHora
	partido.Paso = 3
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "¿Qué tipo de cancha querés?")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Fútbol 5", "cancha_futbol5"),
			tgbotapi.NewInlineKeyboardButtonData("Fútbol 7", "cancha_futbol7"),
			tgbotapi.NewInlineKeyboardButtonData("Fútbol 8", "cancha_futbol8"),
		),
	)
	bot.Send(msg)
}

func manejo_callback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, lista *[]Jugador, partido *Partido) {
	if partido.Paso != 3 {
		return
	}

	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "")

	switch callback.Data {
	case "cancha_futbol5":
		partido.Cancha = "Fútbol 5"
		partido.Precio = "10000"
	case "cancha_futbol7":
		partido.Cancha = "Fútbol 7"
		partido.Precio = "14000"
	case "cancha_futbol8":
		partido.Cancha = "Fútbol 8"
		partido.Precio = "16000"
	}

	partido.Creado = true
	msg.Text = "Partido creado:\nCancha: " + partido.Cancha + "\nPrecio: " + partido.Precio + "$\nDía y hora: " + partido.DiaHora.Format(layout) + "\nUbicación: " + partido.Ubicacion + "\nJugadores: " + imprimir_nombres(*lista)

	if _, err := bot.Send(msg); err != nil {
		log.Panic(err)
	}

	callbackResponse := tgbotapi.NewCallback(callback.ID, callback.Data)
	if _, err := bot.Request(callbackResponse); err != nil {
		log.Panic(err)
	}
}

func bajar_jugador(lista []Jugador, nombre string) []Jugador {
	index := -1
	for i, v := range lista {
		if v.Nombre == nombre {
			index = i
			break
		}
	}
	if index == -1 {
		return lista
	}
	return append(lista[:index], lista[index+1:]...)
}

func imprimir_nombres(lista []Jugador) string {
	nombres := []string{}
	for _, jugador := range lista {
		nombres = append(nombres, jugador.Nombre)
	}
	return strings.Join(nombres, ", ")
}

func obtener_numeros(hora time.Time) string {
	cadenaHora := hora.Format("02-01-2006 15:04:05")

	cadenaSoloNumeros := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, cadenaHora)

	return cadenaSoloNumeros
}
