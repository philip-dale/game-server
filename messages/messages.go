package messages

type BaseMessage struct {
	MessageType int32
}

type InitMessage struct {
	MessageType int32
	UserId      int32
	ServerId    string
	PlayerName  string
	GameType    int32
}

type ErrorMessage struct {
	MessageType int32
	MessageText string
}

type ConnectedMessage struct {
	MessageType int32
	UserId      int32
}

type GameStatusMessage struct {
	MessageType int32
	Status      interface{}
}

type PlayerStatusMessage struct {
	MessageType int32
	Status      interface{}
}

type BuzzActionMessage struct {
	MessageType int32
	UserId      int32
	Buzzing     bool
}

type AdminMessage struct {
	MessageType int32
	AdminId     string
	Command     int32
}

type KickPlayerMessage struct {
	MessageType int32
	AdminId     string
	UserId      int32
}

type QuitMessage struct {
	MessageType int32
	UserId      int32
}

const (
	InitMessageType         = iota
	ConnectedMessageType    = iota
	ErrorMessageType        = iota
	GameStatusMessageType   = iota
	PlayerStatusMessageType = iota
	BuzzActionMessageType   = iota
	AdminMessageType        = iota
	QuitMessageType         = iota
	KickPlayerMessageType   = iota
)

func CreateErrorMessage(text string) ErrorMessage {
	return ErrorMessage{MessageType: ErrorMessageType, MessageText: text}
}

func CreateConnectedMessage(userId int32) ConnectedMessage {
	return ConnectedMessage{MessageType: ConnectedMessageType, UserId: userId}
}

func CreateGameStatusMessage(status interface{}) GameStatusMessage {
	return GameStatusMessage{MessageType: GameStatusMessageType, Status: status}
}

func CreatePlayerStatusMessage(status interface{}) PlayerStatusMessage {
	return PlayerStatusMessage{MessageType: PlayerStatusMessageType, Status: status}
}
