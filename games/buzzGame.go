package games

import (
	"encoding/json"
	"errors"
	"game-server/messages"
	"game-server/users"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type BuzzStatus struct {
	Buzzing   bool
	LockedOut bool
	Score     int32
}

type BuzzInfo struct {
	UserInfo   users.UserInfo
	BuzzStatus BuzzStatus
	sendChan   chan interface{}
	conn       *websocket.Conn
}

type BuzzGameInfo struct {
	playerBuzzing   bool
	buzzingPlayerId int32
	BuzzStatus      []BuzzInfo
	lock            sync.Mutex
	adminId         string
	randSource      rand.Source
	randGen         *rand.Rand
	randSetup       bool
}

func (b *BuzzGameInfo) Init(adminId string) {
	b.adminId = adminId
	b.buzzingPlayerId = -1
	b.playerBuzzing = false
}

func (b *BuzzGameInfo) findIndex(uid int32) (int, error) {
	for i, v := range b.BuzzStatus {
		if v.UserInfo.UserId == uid {
			return i, nil
		}
	}
	return 0, errors.New("User Id Not Found")
}

func (b *BuzzGameInfo) AddUser(data messages.InitMessage, conn *websocket.Conn) int32 {
	b.lock.Lock()

	_, err := b.findIndex(data.UserId)
	if err == nil {
		b.lock.Unlock()
		b.EnableUser(data, conn)
		return data.UserId
	}
	// else add a new user

	if b.randSetup == false {
		b.randSource = rand.NewSource(time.Now().UnixNano())
		b.randGen = rand.New(b.randSource)
	}

	uid := b.randGen.Int31()
	b.BuzzStatus = append(b.BuzzStatus, BuzzInfo{
		UserInfo: users.UserInfo{PlayerName: data.PlayerName, Active: true, UserId: uid},
		BuzzStatus: BuzzStatus{
			Buzzing:   false,
			LockedOut: false,
			Score:     0,
		},
		sendChan: make(chan interface{}, 4),
		conn:     conn,
	})
	b.lock.Unlock()
	b.sendGameUpdate()
	return uid
}

func (b *BuzzGameInfo) EnableUser(data messages.InitMessage, conn *websocket.Conn) error {
	b.lock.Lock()

	i, err := b.findIndex(data.UserId)
	if err != nil {
		b.lock.Unlock()
		return err
	}
	b.BuzzStatus[i].UserInfo.PlayerName = data.PlayerName
	b.BuzzStatus[i].UserInfo.Active = true
	b.BuzzStatus[i].BuzzStatus.Buzzing = false
	b.BuzzStatus[i].BuzzStatus.LockedOut = false
	b.BuzzStatus[i].sendChan = make(chan interface{}, 4)
	b.BuzzStatus[i].conn = conn

	b.lock.Unlock()
	b.sendGameUpdate()
	return nil
}

func (b *BuzzGameInfo) DisableUser(uid int32) error {
	b.lock.Lock()

	i, err := b.findIndex(uid)
	if err != nil {
		b.lock.Unlock()
		return err
	}

	b.BuzzStatus[i].UserInfo.Active = false
	b.lock.Unlock()
	b.sendGameUpdate()
	log.Println("Disabled User " + b.BuzzStatus[i].UserInfo.PlayerName)
	return nil
}

func (b *BuzzGameInfo) RemoveUser(uid int32) error {
	b.lock.Lock()
	i, err := b.findIndex(uid)
	if err != nil {
		b.lock.Unlock()
		return err
	}
	b.BuzzStatus[i].conn.Close()
	b.BuzzStatus = append(b.BuzzStatus[:i], b.BuzzStatus[i+1:]...)

	b.lock.Unlock()
	b.sendGameUpdate()
	return nil
}

func (b *BuzzGameInfo) ProcessMessage(message []byte) {

	var m messages.BaseMessage
	if err := json.Unmarshal(message, &m); err != nil {
		return
	}

	switch m.MessageType {
	case messages.BuzzActionMessageType:
		b.processAction(message)
	case messages.AdminMessageType:
		b.processAdmin(message)
	case messages.QuitMessageType:
		b.processQuit(message)
	case messages.KickPlayerMessageType:
		b.processKick(message)
	}
}

func (b *BuzzGameInfo) processAction(message []byte) {
	var action messages.BuzzActionMessage
	if err := json.Unmarshal(message, &action); err != nil {
		return
	}
	b.lock.Lock()
	if action.Buzzing {
		if !b.playerBuzzing {
			i, err := b.findIndex(action.UserId)
			if err != nil {
				b.lock.Unlock()
				return
			}
			if !b.BuzzStatus[i].BuzzStatus.LockedOut {
				b.playerBuzzing = true
				b.buzzingPlayerId = action.UserId
				b.BuzzStatus[i].BuzzStatus.Buzzing = true
			}
		}
	}
	b.lock.Unlock()
	b.sendGameUpdate()
}

func (b *BuzzGameInfo) processQuit(message []byte) {
	var m messages.QuitMessage
	if err := json.Unmarshal(message, &m); err != nil {
		return
	}
	b.RemoveUser(m.UserId)
}

func (b *BuzzGameInfo) processKick(message []byte) {
	var kick messages.KickPlayerMessage
	if err := json.Unmarshal(message, &kick); err != nil {
		return
	}

	if kick.AdminId != b.adminId {
		return
	}

	b.RemoveUser(kick.UserId)
}

func (b *BuzzGameInfo) processAdmin(message []byte) {
	var admin messages.AdminMessage
	if err := json.Unmarshal(message, &admin); err != nil {
		return
	}

	if admin.AdminId != b.adminId {
		return
	}

	b.lock.Lock()
	switch admin.Command {
	case 0:
		if b.playerBuzzing {
			i, err := b.findIndex(b.buzzingPlayerId)
			if err != nil {
				b.lock.Unlock()
				return
			}
			b.BuzzStatus[i].BuzzStatus.Score += 1
			b.clearBuzz(i)
			b.nextRound()
		}

	case 1:
		if b.playerBuzzing {
			i, err := b.findIndex(b.buzzingPlayerId)
			if err != nil {
				b.lock.Unlock()
				return
			}
			b.BuzzStatus[i].BuzzStatus.LockedOut = true
			b.clearBuzz(i)
		}
	case 2:
		b.nextRound()
	case 3:
		b.reset()
	case 4:
		i, err := b.findIndex(b.buzzingPlayerId)
		if err != nil {
			b.lock.Unlock()
			return
		}
		b.clearBuzz(i)
	}

	b.lock.Unlock()
	b.sendGameUpdate()
}

func (b *BuzzGameInfo) nextRound() {
	for i := range b.BuzzStatus {
		b.BuzzStatus[i].BuzzStatus.Buzzing = false
		b.BuzzStatus[i].BuzzStatus.LockedOut = false
	}
	b.playerBuzzing = false
	b.buzzingPlayerId = -1
}

func (b *BuzzGameInfo) clearBuzz(index int) {
	b.BuzzStatus[index].BuzzStatus.Buzzing = false
	b.playerBuzzing = false
	b.buzzingPlayerId = -1
}

func (b *BuzzGameInfo) reset() {
	for i := range b.BuzzStatus {
		b.BuzzStatus[i].BuzzStatus.Buzzing = false
		b.BuzzStatus[i].BuzzStatus.LockedOut = false
		b.BuzzStatus[i].BuzzStatus.Score = 0
	}
	b.playerBuzzing = false
	b.buzzingPlayerId = -1
}

func (b *BuzzGameInfo) sendGameUpdate() {
	for i := range b.BuzzStatus {
		b.BuzzStatus[i].sendChan <- messages.CreateGameStatusMessage(b.BuzzStatus)
		b.BuzzStatus[i].sendChan <- messages.CreatePlayerStatusMessage(b.BuzzStatus[i].BuzzStatus)
	}
}

func (b *BuzzGameInfo) sendPlayerUpdate(uid int32) {

	b.lock.Lock()
	i, err := b.findIndex(uid)

	if err != nil {
		b.lock.Unlock()
		return
	}

	b.BuzzStatus[i].sendChan <- messages.CreatePlayerStatusMessage(b.BuzzStatus[i].BuzzStatus)
	b.lock.Unlock()
}

func (b *BuzzGameInfo) GetSendChannel(uid int32) chan interface{} {
	b.lock.Lock()
	i, err := b.findIndex(uid)
	b.lock.Unlock()
	if err != nil {
		return nil
	}
	return b.BuzzStatus[i].sendChan
}
