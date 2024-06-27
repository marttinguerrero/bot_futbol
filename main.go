package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const layout = "02-01-2006 15:04"

type Equipos struct {
	Oscuro []Jugador
	Claro  []Jugador
}

type Partido struct {
	ChatID    int64
	Cancha    string
	DiaHora   time.Time
	Precio    string
	Ubicacion string
	Creado    bool
	Paso      int
	Equipos   Equipos
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

	go verificarYEliminarPartidoVencido(bot, &partido)

	for update := range updates {
		if update.Message != nil {
			partido.ChatID = update.Message.Chat.ID
		}
		manejo_update(bot, update, &lista, &partido)
	}
}

func crearEquiposAlAzar(lista []Jugador) (Equipos, string) {
	cantidadJugadores := len(lista)
	var cantidadPorEquipo int

	switch cantidadJugadores {
	case 10:
		cantidadPorEquipo = 5
	case 14:
		cantidadPorEquipo = 7
	case 16:
		cantidadPorEquipo = 8
	default:
		return Equipos{}, "La cantidad de jugadores no es válida. Debe ser exactamente 10, 14 o 16 jugadores."
	}
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(cantidadJugadores, func(i, j int) { lista[i], lista[j] = lista[j], lista[i] })

	equipoOscuro := lista[:cantidadPorEquipo]
	equipoClaro := lista[cantidadPorEquipo : cantidadPorEquipo*2]

	return Equipos{Oscuro: equipoOscuro, Claro: equipoClaro}, ""
}
func asignarEquipos(partido *Partido, lista *[]Jugador, jugadoresInput []string, equipoPrincipal *[]Jugador, equipoOpuesto *[]Jugador, nombreEquipoPrincipal, nombreEquipoOpuesto string) string {
	cantidadJugadores := len(jugadoresInput)

	mensajeError := validarCantidadJugadores(partido.Cancha, cantidadJugadores)
	if mensajeError != "" {
		return mensajeError
	}

	validado, jugadoresNoAnotados := validarJugadoresAnotados(*lista, jugadoresInput)
	if !validado {
		return fmt.Sprintf("Los siguientes jugadores no están anotados para el partido: %s", strings.Join(jugadoresNoAnotados, ", "))
	}

	*equipoPrincipal = []Jugador{}
	*equipoOpuesto = []Jugador{}

	for _, nombre := range jugadoresInput {
		nombre = strings.TrimSpace(nombre)
		for _, jugador := range *lista {
			if jugador.Nombre == nombre {
				*equipoPrincipal = append(*equipoPrincipal, jugador)
				break
			}
		}
	}

	for _, jugador := range *lista {
		encontrado := false
		for _, nombre := range jugadoresInput {
			if strings.TrimSpace(nombre) == jugador.Nombre {
				encontrado = true
				break
			}
		}
		if !encontrado {
			*equipoOpuesto = append(*equipoOpuesto, jugador)
		}
	}

	return fmt.Sprintf("%s: %s\n%s: %s", nombreEquipoPrincipal, imprimir_nombres(*equipoPrincipal), nombreEquipoOpuesto, imprimir_nombres(*equipoOpuesto))
}

// func asignarEquipos(lista []Jugador, jugadoresOscuro []string) (Equipos, error) {
// 	equipoOscuro := []Jugador{}
// 	equipoClaro := []Jugador{}
// 	nombresOscuro := make(map[string]bool)

// 	for _, nombre := range jugadoresOscuro {
// 		nombresOscuro[strings.TrimSpace(nombre)] = true
// 	}

// 	for _, jugador := range lista {
// 		if nombresOscuro[jugador.Nombre] {
// 			equipoOscuro = append(equipoOscuro, jugador)
// 		} else {
// 			equipoClaro = append(equipoClaro, jugador)
// 		}
// 	}

// 	if len(equipoOscuro) != 5 || len(equipoClaro) != 5 {
// 		return Equipos{}, fmt.Errorf("los equipos deben tener exactamente 5 jugadores cada uno")
// 	}

// 	return Equipos{Oscuro: equipoOscuro, Claro: equipoClaro}, nil
// }

func validarCantidadJugadores(cancha string, cantidad int) (mensajeError string) {
	switch cancha {
	case "Fútbol 5":
		if cantidad != 5 {
			mensajeError = "Para Fútbol 5 se necesitan exactamente 5 jugadores en cada equipo."
		}
	case "Fútbol 7":
		if cantidad != 7 {
			mensajeError = "Para Fútbol 7 se necesitan exactamente 7 jugadores en cada equipo."
		}
	case "Fútbol 8":
		if cantidad != 8 {
			mensajeError = "Para Fútbol 8 se necesitan exactamente 8 jugadores en cada equipo."
		}

	default:
		mensajeError = "No se reconoce el tipo de cancha especificado."
	}
	return mensajeError
}

func validarJugadoresAnotados(lista []Jugador, nombres []string) (bool, []string) {
	jugadoresAnotados := make(map[string]bool)
	for _, jugador := range lista {
		jugadoresAnotados[jugador.Nombre] = true
	}

	jugadoresNoAnotados := []string{}
	for _, nombre := range nombres {
		nombre = strings.TrimSpace(nombre)
		if !jugadoresAnotados[nombre] {
			jugadoresNoAnotados = append(jugadoresNoAnotados, nombre)
		}
	}

	return len(jugadoresNoAnotados) == 0, jugadoresNoAnotados
}

func verificarYEliminarPartidoVencido(bot *tgbotapi.BotAPI, partido *Partido) {
	for {
		hora_actual := obtener_numeros(time.Now())
		hora_partido := obtener_numeros(partido.DiaHora)
		if partido.Creado && hora_partido < hora_actual {
			chatID := partido.ChatID
			fmt.Println("Verificación: Eliminando partido vencido")
			eliminar_partido(bot, chatID, partido)
		}
	}
}

func eliminar_partido(bot *tgbotapi.BotAPI, chatID int64, partido *Partido) {
	partido.Creado = false
	partido.Paso = 0

	msg := tgbotapi.NewMessage(chatID, "El partido ha finalizado, /crearpartido para crear el siguiente")
	_, err := bot.Send(msg)
	if err != nil {
		fmt.Printf("Error al enviar mensaje: %v\n", err)
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
	} else if update.Message.ReplyToMessage != nil {
		switch partido.Paso {
		case 1:
			manejo_ubicacion(bot, update, partido)
		case 2:
			manejo_fecha(bot, update, partido)
		}
	}
}

func jugadorEnLista(lista []Jugador, nombre string) bool {
	for _, jugador := range lista {
		if jugador.Nombre == nombre {
			return true
		}
	}
	return false
}

func chequear_sumar(partido *Partido, lista *[]Jugador) bool {
	if partido.Cancha == "Fútbol 5" {
		if len(*lista) >= 10 {
			return false
		}

	} else if partido.Cancha == "Fútbol 7" {
		if len(*lista) >= 14 {
			return false
		}
	} else if partido.Cancha == "Fútbol 8" {
		if len(*lista) >= 16 {
			return false
		}
	}
	return true

}
func imprimir_jugadores(partido *Partido, lista *[]Jugador) string {
	var resultado string
	if partido.Cancha == "Fútbol 5" {
		for i := 1; i <= 10; i++ {
			if i <= len(*lista) {
				resultado += fmt.Sprintf("%d. %s\n", i, (*lista)[i-1].Nombre)
			} else {
				resultado += fmt.Sprintf("%d. --------\n", i)
			}
		}

		return resultado
	}
	if partido.Cancha == "Fútbol 7" {
		for i := 1; i <= 14; i++ {
			if i <= len(*lista) {
				resultado += fmt.Sprintf("%d. %s\n", i, (*lista)[i-1].Nombre)
			} else {
				resultado += fmt.Sprintf("%d. --------\n", i)
			}
		}

		return resultado
	}
	if partido.Cancha == "Fútbol 8" {
		for i := 1; i <= 16; i++ {
			if i <= len(*lista) {
				resultado += fmt.Sprintf("%d. %s\n", i, (*lista)[i-1].Nombre)
			} else {
				resultado += fmt.Sprintf("%d. --------\n", i)
			}
		}

		return resultado
	}
	return resultado
}

func manejo_comandos(bot *tgbotapi.BotAPI, update tgbotapi.Update, lista *[]Jugador, partido *Partido) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
	switch update.Message.Command() {
	case "sumo":
		if !partido.Creado {
			msg.Text = "Primero debes crear un partido con el comando /crearpartido"
		} else {
			if !chequear_sumar(partido, lista) {
				msg.Text = "Se llego a la cantidad maxima de jugadores para el partido."
				break
			}
			if !jugadorEnLista(*lista, update.Message.From.FirstName) {
				*lista = append(*lista, Jugador{Nombre: update.Message.From.FirstName, Pago: false})
				msg.Text = imprimir_jugadores(partido, lista)
			} else {
				msg.Text = "El jugador ya fue agregado a la lista "
			}

		}

	case "sumoa":
		if !partido.Creado {
			msg.Text = "Primero debes crear un partido con el comando /crearpartido"
		} else {
			if !chequear_sumar(partido, lista) {
				msg.Text = "Se llego a la cantidad maxima de jugadores para el partido."
				break
			}
			parts := strings.SplitN(update.Message.Text, " ", 2)
			if len(parts) == 2 {
				nombreAmigo := parts[1]
				if !jugadorEnLista(*lista, nombreAmigo) {
					*lista = append(*lista, Jugador{Nombre: nombreAmigo, Pago: false})
					msg.Text = imprimir_jugadores(partido, lista)

				} else {
					msg.Text = "El jugador ya fue agregado a la lista "
				}
			} else {
				msg.Text = "Por favor, proporciona un nombre después del comando /sumoa."
			}
		}
	case "bajar":
		*lista = bajar_jugador(*lista, update.Message.From.FirstName)
		msg.Text = imprimir_jugadores(partido, lista)
	case "bajoa":
		if !partido.Creado {
			msg.Text = "Primero debes crear un partido con el comando /crearpartido"
		} else {
			parts := strings.SplitN(update.Message.Text, " ", 2)
			if len(parts) == 2 {
				nombre_amigo := parts[1]
				*lista = bajar_jugador(*lista, nombre_amigo)
				msg.Text = imprimir_jugadores(partido, lista)
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
	case "equipoOscuro":
		if !partido.Creado {
			msg.Text = "Primero debes crear un partido con el comando /crearpartido"
		} else {
			partes := strings.SplitN(update.Message.Text, ":", 2)
			if len(partes) == 2 {
				jugadoresOscuro := strings.Split(partes[1], ",")
				resultado := asignarEquipos(partido, lista, jugadoresOscuro, &partido.Equipos.Oscuro, &partido.Equipos.Claro, "Equipo Oscuro", "Equipo Claro")
				msg.Text = resultado
			} else {
				msg.Text = "Por favor, proporciona los nombres de los jugadores después del comando /equipoOscuro separados por comas."
			}
		}

	case "equipoClaro":
		if !partido.Creado {
			msg.Text = "Primero debes crear un partido con el comando /crearpartido"
		} else {
			partes := strings.SplitN(update.Message.Text, ":", 2)
			if len(partes) == 2 {
				jugadoresClaro := strings.Split(partes[1], ",")
				resultado := asignarEquipos(partido, lista, jugadoresClaro, &partido.Equipos.Claro, &partido.Equipos.Oscuro, "Equipo Claro", "Equipo Oscuro")
				msg.Text = resultado
			} else {
				msg.Text = "Por favor, proporciona los nombres de los jugadores después del comando /equipoClaro separados por comas."
			}
		}

	case "reiniciarEquipos":
		if !partido.Creado {
			msg.Text = "Primero debes crear un partido con el comando /crearpartido"
		} else {
			partido.Equipos.Oscuro = nil
			partido.Equipos.Claro = nil
			msg.Text = "Se han reiniciado los equipos. Puedes volver a asignarlos con los comandos /equipoOscuro y /equipoClaro."
		}

	case "verEquipos":
		msg.Text = "Equipo Oscuro: " + imprimir_nombres(partido.Equipos.Oscuro) + "\nEquipo Claro: " + imprimir_nombres(partido.Equipos.Claro)

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
	case "crearEquiposAlAzar":
		cantidadJugadores := len(*lista)
		if cantidadJugadores != 10 && cantidadJugadores != 14 && cantidadJugadores != 16 {
			msg.Text = "La cantidad de jugadores no es válida. Debe ser exactamente 10, 14 o 16 jugadores."
			break
		}

		equipos, mensajeError := crearEquiposAlAzar(*lista)
		if mensajeError != "" {
			msg.Text = mensajeError
		} else {
			partido.Equipos = equipos
			msg.Text = "Equipos creados al azar:\n" +
				"Equipo Oscuro: " + imprimir_nombres(equipos.Oscuro) + "\n" +
				"Equipo Claro: " + imprimir_nombres(equipos.Claro)
		}
	case "ayuda":
		msg.Text = "ℹ️ *Comandos disponibles:*\n\n" +
			"🙋 /sumo - Añadir jugador al partido\n" +
			"🙋‍♂️ /sumoa [nombre] - Añadir amigo al partido\n" +
			"🚶 /bajar - Quitar jugador del partido\n" +
			"🚶‍♂️ /bajoa [nombre] - Quitar amigo del partido\n" +
			"👥 /jugadores - Ver jugadores del partido\n" +
			"⚫ /equipoOscuro:[jugador1,jugador2,...] - Asignar equipo oscuro\n" +
			"⚪ /equipoClaro:[jugador1,jugador2,...] - Asignar equipo claro\n" +
			"🔄 /reiniciarEquipos - Reiniciar asignación de equipos\n" +
			"👀 /verEquipos - Ver equipos asignados\n" +
			"🎮 /crearpartido - Crear un nuevo partido\n" +
			"ℹ️ /partido - Ver detalles del partido\n" +
			"🔍 /estado - Estado del bot\n" +
			"🎲 /crearEquiposAlAzar - Crear equipos al azar\n\n" +
			"."
		msg.ParseMode = "markdown"
	default:
		msg.Text = "No entiendo ese comando"
	}

	if _, err := bot.Send(msg); err != nil {
		log.Panic(err)
	}
}

func manejo_ubicacion(bot *tgbotapi.BotAPI, update tgbotapi.Update, partido *Partido) {
	partido.Ubicacion = update.Message.Text
	partido.Paso = 2
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "¿Qué día y a qué hora? (formato: DD-MM-YYYY HH:MM)")
	msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}
	bot.Send(msg)
}
func manejo_fecha(bot *tgbotapi.BotAPI, update tgbotapi.Update, partido *Partido) {
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
