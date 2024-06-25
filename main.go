package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
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
	Alarmas   []int64
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

	//partido := Partido{}

	go verificarYEliminarPartidoVencido(bot, &partido)

	go programarAlarma(bot, &partido)

	//go verificarYEnviarAlarmas(bot, &partido)

	for update := range updates {
		if update.Message != nil {
			partido.ChatID = update.Message.Chat.ID
		}
		manejo_update(bot, update, &lista, &partido)
	}
}

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

// ------------------------------
func programarAlarma(bot *tgbotapi.BotAPI, partido *Partido) {

	for {

		hora_actual := obtener_numeros_reales(time.Now())
		hora_partido := obtener_numeros_reales(partido.DiaHora)

		for _, alarma := range partido.Alarmas {

			hora_alarma := int64(alarma) * 100

			if hora_actual+(hora_alarma) == hora_partido {

				chatID := partido.ChatID
				msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("¡Atención! El partido comienza en %d horas.", alarma))
				bot.Send(msg)
			}
		}
		time.Sleep(1 * time.Minute)
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

	case "equipoOscuro":
		if !partido.Creado {
			msg.Text = "Primero debes crear un partido con el comando /crearpartido"
		} else {
			partes := strings.SplitN(update.Message.Text, ":", 2)
			if len(partes) == 2 {
				jugadoresOscuro := strings.Split(partes[1], ",")
				cantidadJugadores := len(jugadoresOscuro)
				equipoOscuro := []Jugador{}
				equipoClaro := []Jugador{}

				mensajeError := validarCantidadJugadores(partido.Cancha, cantidadJugadores)
				if mensajeError != "" {
					msg.Text = mensajeError
					break
				}

				validado, jugadoresNoAnotados := validarJugadoresAnotados(*lista, jugadoresOscuro)
				if !validado {
					msg.Text = fmt.Sprintf("Los siguientes jugadores no están anotados para el partido: %s", strings.Join(jugadoresNoAnotados, ", "))
					break
				}

				for _, nombre := range jugadoresOscuro {
					nombre = strings.TrimSpace(nombre)
					for _, jugador := range *lista {
						if jugador.Nombre == nombre {
							equipoOscuro = append(equipoOscuro, jugador)
							break
						}
					}
				}

				for _, jugador := range *lista {
					encontrado := false
					for _, nombre := range jugadoresOscuro {
						if strings.TrimSpace(nombre) == jugador.Nombre {
							encontrado = true
							break
						}
					}
					if !encontrado {
						equipoClaro = append(equipoClaro, jugador)
					}
				}

				partido.Equipos.Oscuro = equipoOscuro
				partido.Equipos.Claro = equipoClaro

				msg.Text = "Equipo Oscuro: " + imprimir_nombres(partido.Equipos.Oscuro) + "\nEquipo Claro: " + imprimir_nombres(partido.Equipos.Claro)
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
				cantidadJugadores := len(jugadoresClaro)
				equipoOscuro := []Jugador{}
				equipoClaro := []Jugador{}

				mensajeError := validarCantidadJugadores(partido.Cancha, cantidadJugadores)
				if mensajeError != "" {
					msg.Text = mensajeError
					break
				}

				validado, jugadoresNoAnotados := validarJugadoresAnotados(*lista, jugadoresClaro)
				if !validado {
					msg.Text = fmt.Sprintf("Los siguientes jugadores no están anotados para el partido: %s", strings.Join(jugadoresNoAnotados, ", "))
					break
				}

				for _, nombre := range jugadoresClaro {
					nombre = strings.TrimSpace(nombre)
					for _, jugador := range *lista {
						if jugador.Nombre == nombre {
							equipoClaro = append(equipoClaro, jugador)
							break
						}
					}
				}

				for _, jugador := range *lista {
					encontrado := false
					for _, nombre := range jugadoresClaro {
						if strings.TrimSpace(nombre) == jugador.Nombre {
							encontrado = true
							break
						}
					}
					if !encontrado {
						equipoOscuro = append(equipoOscuro, jugador)
					}
				}

				partido.Equipos.Oscuro = equipoOscuro
				partido.Equipos.Claro = equipoClaro

				msg.Text = "Equipo Claro: " + imprimir_nombres(partido.Equipos.Claro) + "\nEquipo Oscuro: " + imprimir_nombres(partido.Equipos.Oscuro)
			} else {
				msg.Text = "Por favor, proporciona los nombres de los jugadores después del comando /equipoClaro separados por comas."
			}
		}

	case "ponerAlarma":
		if !partido.Creado {
			msg.Text = "Primero debes crear un partido con el comando /crearpartido"
		} else {
			parts := strings.SplitN(update.Message.Text, ":", 2)
			if len(parts) == 2 {
				horaAlarmaStr := parts[1]
				horaAlarma, err := strconv.ParseInt(horaAlarmaStr, 10, 64)
				if err != nil {
					msg.Text = "Error al parsear la hora de la alarma."
					break
				}
				partido.Alarmas = append(partido.Alarmas, horaAlarma)
				msg.Text = fmt.Sprintf("Alarma configurada para %d horas antes del partido.", horaAlarma)

			} else {
				msg.Text = "Por favor, proporciona la hora de la alarma después del comando /ponerAlarma."
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

func obtener_numeros_reales(hora time.Time) int64 {
	cadenaHora := hora.Format("02-01-2006 15:04:05")

	cadenaSoloNumeros := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, cadenaHora)

	numero, err := strconv.ParseInt(cadenaSoloNumeros, 10, 64)
	if err != nil {
		fmt.Println("Error convirtiendo a int64:", err)
		return 0 // O algún valor por defecto
	}

	numeroStr := strconv.FormatInt(numero, 10)
	if len(numeroStr) > 6 {
		numeroStr = numeroStr[len(numeroStr)-6:]
	}

	if len(numeroStr) > 4 {
		numeroStr = numeroStr[:4]
	}

	resultado, err := strconv.ParseInt(numeroStr, 10, 64)
	if err != nil {
		fmt.Println("Error convirtiendo a int64:", err)
		return 0 // O algún valor por defecto
	}

	return resultado
}
