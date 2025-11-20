package models

import (
	"sync"

	"github.com/dh1tw/gosamplerate"
	"github.com/gorilla/websocket"
)

type Room struct {
	ID           string
	FromLanguage string
	ToLanguage   string

	Clients    map[*Client]bool
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan []byte

	ClientAudioBuffers sync.Map

	TranslationWS   *websocket.Conn
	TranslationMux  sync.Mutex
	TranslationLock sync.Mutex
	ReconnectLock   sync.Mutex
	IsReconnecting  bool
	ShouldStopTrans bool

	Src                  *gosamplerate.Src
	SrcMu                sync.Mutex
	TranslationQueueLock sync.Mutex
}

type RoomManager struct {
	Rooms map[string]*Room
	Mu    sync.RWMutex
}

func NewRoomManager() *RoomManager {
	return &RoomManager{Rooms: make(map[string]*Room)}
}

func (rm *RoomManager) GetRoom(id, fromLang, toLang string) *Room {
	rm.Mu.Lock()
	defer rm.Mu.Unlock()
	if room, ok := rm.Rooms[id]; ok {
		return room
	}
	room := &Room{
		ID:           id,
		FromLanguage: fromLang,
		ToLanguage:   toLang,
		Clients:      make(map[*Client]bool),
		Register:     make(chan *Client),
		Unregister:   make(chan *Client),
		Broadcast:    make(chan []byte),
	}
	rm.Rooms[id] = room
	return room
}
