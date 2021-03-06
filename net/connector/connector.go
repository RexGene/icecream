package connector

import (
	"github.com/RexGene/common/threadpool"
	"github.com/RexGene/icecream/icinterface"
	"github.com/RexGene/icecream/manager/databackupmanager"
	"github.com/RexGene/icecream/manager/datasendmanager"
	"github.com/RexGene/icecream/manager/handlermanager"
	"github.com/RexGene/icecream/manager/socketmanager"
	"github.com/RexGene/icecream/net/converter"
	"github.com/RexGene/icecream/net/socket"
	"github.com/RexGene/icecream/protocol"
	"github.com/golang/protobuf/proto"
	"log"
	"net"
)

const (
	READ_BUFFER_SIZE = 65535
	ICHEAD_SIZE      = 16
)

type Connector struct {
	socket *socket.Socket

	conn              *net.UDPConn
	isRunning         bool
	dataSendManager   *datasendmanager.DataSendManager
	dataBackupManager *databackupmanager.DataBackupManager
	socketmanager     *socketmanager.SocketManager
	handlerManager    *handlermanager.HandlerManager
}

func (self *Connector) SendMessage(id int, msg proto.Message) {
	converter.SendMessage(self.socket, id, msg)
}

func New(conn *net.UDPConn, addr *net.UDPAddr) *Connector {
	dataBackupManager := databackupmanager.New()
	dataSendManager := datasendmanager.New()
	socketmanager := socketmanager.New()
	handlerManager := handlermanager.New()
	socketmanager.SetDataBackupManager(dataBackupManager)
	dataSendManager.Init(conn, dataBackupManager, socketmanager)
	dataBackupManager.SetSender(dataSendManager)

	sk := socket.New()
	sk.SetSender(dataSendManager)
	sk.Addr = addr
	return &Connector{
		socket:            sk,
		dataSendManager:   dataSendManager,
		conn:              conn,
		socketmanager:     socketmanager,
		handlerManager:    handlerManager,
		dataBackupManager: dataBackupManager,
	}
}

func (self *Connector) listen() {
	for self.isRunning {
		buffer := converter.MakeBuffer(READ_BUFFER_SIZE)
		readLen, targetAddr, err := self.conn.ReadFromUDP(buffer)
		if err == nil {
			if readLen >= ICHEAD_SIZE {
				task := func() {
					if converter.HandlePacket(
						self.dataSendManager,
						self.socketmanager,
						self.dataBackupManager,
						self.handlerManager,
						targetAddr, buffer, uint(readLen), self.socket) {
						converter.FreeBuffer(buffer)
					}
				}

				threadpool.GetInstance().Start(task)
			} else {
				log.Println("[!]data len too short:", readLen)
			}
		} else {
			log.Println("[!]", err)
		}
	}
}

func (self *Connector) connect() {
	buffer := self.dataBackupManager.MakeBuffer(ICHEAD_SIZE)
	converter.SendData(self.socket, buffer, uint(len(buffer)), protocol.START_FLAG, 0)
}

func (self *Connector) Start() {
	go self.dataSendManager.ExecuteForSocket(self.socket)
	go self.dataBackupManager.Execute()
	go self.handlerManager.Execute()
	go self.listen()

	self.isRunning = true
	self.connect()
}

func (self *Connector) Stop() {
	self.dataSendManager.Stop()
	self.dataBackupManager.Stop()
	self.handlerManager.Stop()
	self.isRunning = false
}

func (self *Connector) Close() {
	self.Stop()
	self.conn.Close()
}

func (self *Connector) RegistHandler(id uint32, handleFunc func(icinterface.ISocket, proto.Message)) {
	self.handlerManager.RegistHandler(id, handleFunc)
}
