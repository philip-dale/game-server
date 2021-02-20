package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"game-server/games"
	"game-server/messages"

	"github.com/gorilla/websocket"
)

var serverId string

func main() {

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	serverId = strconv.Itoa(r1.Intn(999999)) // change to random number later
	log.Println("Game Server Id = " + serverId)

	adminId := strconv.Itoa(r1.Intn(999999))
	games.BuzzerGame.Init(adminId)
	log.Println("Admin Server Id = " + adminId)

	http.HandleFunc("/ws", wsHandler)

	fmt.Printf("Starting server at port 50000\n")
	if err := http.ListenAndServe(":50000", nil); err != nil {
		log.Fatal(err)
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	uid, gameStatus, err := readInit(conn)
	if err != nil {
		errorMessage := messages.CreateErrorMessage(err.Error())
		sendJSON(conn, &errorMessage)
		log.Println(err)
		conn.Close()
		return
	}
	connectedMessage := messages.CreateConnectedMessage(uid)
	sendJSON(conn, &connectedMessage)

	userManager(conn, uid, gameStatus)
}

func sendJSON(conn *websocket.Conn, v interface{}) {
	if err := conn.WriteJSON(v); err != nil {
		log.Println(err)
		conn.Close()
		return
	}
}

func userManager(conn *websocket.Conn, uid int32, gameStatus games.Game) {
	connectionClosed := false
	conn.SetCloseHandler(func(code int, text string) error {
		connectionClosed = true
		return gameStatus.DisableUser(uid)
	})

	readChan := make(chan []byte, 1)
	go func() {
		readMessage(conn, readChan)
	}()
	for {
		select {
		case message := <-readChan:
			gameStatus.ProcessMessage(message)
		case status := <-gameStatus.GetSendChannel(uid):
			if connectionClosed {
				return
			}
			sendJSON(conn, &status)
		}
	}
}

func readInit(conn *websocket.Conn) (int32, games.Game, error) {
	// Wait 2 seconds for the init message otherwise close the connection
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	initMessage := messages.InitMessage{}
	err := conn.ReadJSON(&initMessage)
	conn.SetReadDeadline(time.Time{}) // no more timeout
	if err != nil {
		return -1, nil, err
	}
	if initMessage.MessageType != messages.InitMessageType {
		return -1, nil, errors.New("Incorrect MessageType")
	}

	if initMessage.ServerId != serverId {
		return -1, nil, errors.New("Incorrect ServerId")
	}

	uid := int32(0)

	var gameStatus games.Game

	// add support for other games here
	switch initMessage.GameType {
	case games.BuzzGame:
		gameStatus = &games.BuzzerGame
	default:
		return -1, nil, errors.New("Unknown Game")
	}

	uid = gameStatus.AddUser(initMessage, conn)

	return uid, gameStatus, nil
}

func readMessage(conn *websocket.Conn, message chan []byte) {
	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			conn.Close()
			return
		}
		message <- p
	}
}