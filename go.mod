module github.com/philip-dale/game-server

go 1.12

require (
	game-server/games v0.0.0
	game-server/messages v0.0.0
	game-server/users v0.0.0
	github.com/gorilla/websocket v1.4.2
	golang.org/x/net v0.0.0-20210220033124-5f55cee0dc0d // indirect
)

replace game-server/users => ./users

replace game-server/games => ./games

replace game-server/messages => ./messages
