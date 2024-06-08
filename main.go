package main

import (
	"log"
	"os"
	"strings"
	"time"

	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const layout = "02-01-2006 15:04"

type Partido struct {
	Cancha  string
	DiaHora time.Time
	Precio  string
}

type Jugador struct {
	Nombre string
	Pago   bool
	Plata  int
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

	//
	partido.Precio = "10000"

	for update := range updates {
		manejo_update(bot, update, &lista, &partido)
	}
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
	} else if update.Message.ReplyToMessage != nil && strings.Contains(update.Message.ReplyToMessage.Text, "¿Qué día y a qué hora?") {
		manejo_fecha(bot, update, partido, lista)
	}
}

//-----------------

func buscar_jugador(lista []Jugador, nombre string) *Jugador {
	for _, jugador := range lista {
		if jugador.Nombre == nombre {
			return &jugador
		}
	}
	return nil
}

func boolToStr(b bool) string {
	if b {
		return "Sí"
	}
	return "No"
}

func calcularPrecioPorJugadorSinApuesta(precio string, lista []Jugador) int {
	precioInt, _ := strconv.Atoi(precio)
	if len(lista) == 0 {
		return 0
	}
	return precioInt / len(lista)
}

//-----------

func manejo_comandos(bot *tgbotapi.BotAPI, update tgbotapi.Update, lista *[]Jugador, partido *Partido) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
	cmd := strings.ToLower(update.Message.Command())

	switch update.Message.Command() {
	case "sumo":
		*lista = append(*lista, Jugador{Nombre: update.Message.From.FirstName, Pago: false})
		//
		for i := 0; i < 9; i++ {
			*lista = append(*lista, Jugador{Nombre: "Juan", Pago: false})
		}
		//
		msg.Text = "Jugadores que suman al partido por ahora: " + imprimir_nombres(*lista)

	case "bajar":
		*lista = bajar_jugador(*lista, update.Message.From.FirstName)
		msg.Text = "Se borró de la lista de jugadores a " + update.Message.From.FirstName
	case "jugadores":
		msg.Text = "Los jugadores que van al partido por ahora son: " + imprimir_nombres(*lista)

		//--------------------------
	case "jugador":
		nombreJugador := update.Message.CommandArguments()
		jugador := buscar_jugador(*lista, nombreJugador)
		if jugador != nil {
			msg.Text = "Jugador: " + jugador.Nombre + "\nPago: " + boolToStr(jugador.Pago)
		} else {
			msg.Text = "No se encontró al jugador " + nombreJugador
		}

	//--------------------------

	case "crearpartido":
		msg.Text = "¿Qué tipo de cancha querés?"
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Fútbol 5", "cancha_futbol5"),
				tgbotapi.NewInlineKeyboardButtonData("Fútbol 7", "cancha_futbol7"),
				tgbotapi.NewInlineKeyboardButtonData("Fútbol 8", "cancha_futbol8"),
			),
		)
		//-----------------------------
	case "precio":
		msg.Text = "Este es el precio de la cancha: " + partido.Precio
		//---------------------------
	case "partido":
		if partido.Cancha == "" {
			msg.Text = "Todavía no hay un partido creado"
		} else {
			msg.Text = "Cancha: " + partido.Cancha + "\nPrecio: " + partido.Precio + "$\nDía y hora: " + partido.DiaHora.Format(layout) + "\nJugadores: " + imprimir_nombres(*lista)
		}
	case "estado":
		msg.Text = "Estoy funcionando"

	case "partido_terminado":
		valor_por_persona := calcularPrecioPorJugadorSinApuesta(partido.Precio, *lista)
		for i := range *lista {
			(*lista)[i].Plata = valor_por_persona
		}
		msg.Text = "El partido ha terminado. El costo por persona es:" + strconv.Itoa(valor_por_persona) + " Listas: " + imprimir_nombres(*lista)

	default:
		//-----------------
		if strings.HasPrefix(cmd, "jugador:") {
			nombreJugador := strings.TrimPrefix(cmd, "jugador:")
			jugador := buscar_jugador(*lista, nombreJugador)
			if jugador != nil {
				msg.Text = "Jugador: " + jugador.Nombre + "\nPago: xxx " + strconv.Itoa(jugador.Plata)
			} else {
				msg.Text = "No se encontró al jugador " + nombreJugador
			} //--------------
		} else {
			msg.Text = "No entiendo ese comando"
		}
	}

	if _, err := bot.Send(msg); err != nil {
		log.Panic(err)
	}
}

func manejo_fecha(bot *tgbotapi.BotAPI, update tgbotapi.Update, partido *Partido, lista *[]Jugador) {
	diaHora, err := time.Parse(layout, update.Message.Text)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Formato de fecha y hora incorrecto. Por favor, usa el formato DD-MM-YYYY HH:MM.")
		bot.Send(msg)
		return
	}
	partido.DiaHora = diaHora
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Partido creado:\nCancha: "+partido.Cancha+"\nPrecio: "+partido.Precio+"$\nDía y hora: "+partido.DiaHora.Format(layout)+"\nJugadores: "+imprimir_nombres(*lista))
	bot.Send(msg)
}

func manejo_callback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, lista *[]Jugador, partido *Partido) {
	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "")

	canchaElegida := ""
	jugadoresRequeridos := 0
	precio := ""

	switch callback.Data {
	case "cancha_futbol5":
		canchaElegida = "Fútbol 5"
		jugadoresRequeridos = 10
		precio = "10000"
	case "cancha_futbol7":
		canchaElegida = "Fútbol 7"
		jugadoresRequeridos = 14
		precio = "14000"
	case "cancha_futbol8":
		canchaElegida = "Fútbol 8"
		jugadoresRequeridos = 16
		precio = "16000"
	}

	if len(*lista) < jugadoresRequeridos {
		msg.Text = "No hay suficientes jugadores para crear el partido de " + canchaElegida
	} else {
		partido.Cancha = canchaElegida
		partido.Precio = precio
		msg.Text = "Has elegido " + partido.Cancha + ". ¿Qué día y a qué hora? (formato: DD-MM-YYYY HH:MM)"
		msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}
	}

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
