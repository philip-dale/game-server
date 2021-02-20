package games

import (
	"game-server/messages"

	"github.com/gorilla/websocket"
)

const (
	BuzzGame = iota
)

// add other game status' here
var BuzzerGame BuzzGameInfo

type Game interface {
	AddUser(data messages.InitMessage, conn *websocket.Conn) int32
	EnableUser(uid int32) error
	DisableUser(uid int32) error
	RemoveUser(uid int32) error
	ProcessMessage(message []byte)
	GetSendChannel(uid int32) chan interface{}
}
