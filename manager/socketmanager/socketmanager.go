package socketmanager

import (
	"github.com/RexGene/icecream/icinterface"
	"github.com/RexGene/icecream/manager/databackupmanager"
	"log"
	"math/rand"
	"sync"
	"time"
)

var instance *SocketManager

type SocketManager struct {
	sync.RWMutex
	dataMap           map[uint32]icinterface.ISocket
	dataBackupManager *databackupmanager.DataBackupManager
	stopEvent         chan bool
}

func GetInstance() *SocketManager {
	if instance == nil {
		instance = New()
	}

	return instance
}

func (self *SocketManager) SetDataBackupManager(dataBackupManager *databackupmanager.DataBackupManager) {
	self.dataBackupManager = dataBackupManager
}

func (self *SocketManager) CheckAndRemoveTimeoutSocket() {
	for {
		select {
		case <-time.After(time.Second * 2):
			now := time.Now().Unix()
			for token, socket := range self.dataMap {
				diff := now - socket.GetLastUpdateTime()
				state := socket.GetState()
				if state == icinterface.SYN_STATE {
					if diff > 2 {
						self.removeSocket(token)
					}
				} else if state == icinterface.SYN_NORMAL {
					if diff > 60 {
						self.removeSocket(token)
					}
				}
			}
		case <-self.stopEvent:
			return
		}
	}

}

func (self *SocketManager) removeSocket(token uint32) {
	self.Lock()
	defer self.Unlock()

	log.Println("remove socket:", token)
	delete(self.dataMap, token)
	if self.dataBackupManager != nil {
		self.dataBackupManager.SendCmd(token, 0, nil, databackupmanager.REMOVE)
	}
}

func (self *SocketManager) GetSocket(token uint32) icinterface.ISocket {
	self.RLock()
	defer self.RUnlock()

	log.Println("get socket:", self.dataMap)
	socket, _ := self.dataMap[token]
	return socket
}

func (self *SocketManager) AddSocketByToken(socket icinterface.ISocket, token uint32) {
	self.Lock()
	defer self.Unlock()

	socket.SetState(icinterface.SYN_NORMAL)
	socket.SetToken(token)
	self.dataMap[token] = socket
}

func (self *SocketManager) AddSocket(socket icinterface.ISocket) bool {
	self.Lock()
	defer self.Unlock()

	token, isSuccess := self.MakeToken()
	if !isSuccess {
		log.Println("[-]token is empty")
		return false
	}

	socket.SetState(icinterface.SYN_NORMAL)
	socket.SetToken(token)
	self.dataMap[token] = socket
	return true
}

func (self *SocketManager) MakeToken() (uint32, bool) {
	count := 10
	for i := 0; i < count; i++ {
		token := rand.Uint32()
		if self.dataMap[token] == nil {
			return token, true
		}
	}

	return 0, false
}

func (self *SocketManager) Stop() {
	self.stopEvent <- true
}

func New() *SocketManager {
	return &SocketManager{
		dataMap:   make(map[uint32]icinterface.ISocket),
		stopEvent: make(chan bool, 1),
	}
}
