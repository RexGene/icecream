package connecter

import (
	"github.com/RexGene/common/threadpool"
	"github.com/RexGene/icecream/manager/databackupmanager"
	"github.com/RexGene/icecream/manager/datasendmanager"
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

type Connecter struct {
	socket *socket.Socket

	conn              *net.UDPConn
	isRunning         bool
	dataSendManager   *datasendmanager.DataSendManager
	dataBackupManager *databackupmanager.DataBackupManager
	socketmanager     *socketmanager.SocketManager
}

func (self *Connecter) SendMessage(msg proto.Message) {
	converter.SendMessage(self.socket, msg)
}

func New(conn *net.UDPConn, addr *net.UDPAddr) *Connecter {
	dataBackupManager := databackupmanager.New()
	dataSendManager := datasendmanager.New()
	socketmanager := socketmanager.New()
	socketmanager.SetDataBackupManager(dataBackupManager)
	dataSendManager.Init(conn, dataBackupManager, socketmanager)

	sk := socket.New()
	sk.SetSender(dataSendManager)
	sk.Addr = addr
	return &Connecter{
		socket:            sk,
		dataSendManager:   dataSendManager,
		conn:              conn,
		socketmanager:     socketmanager,
		dataBackupManager: dataBackupManager,
	}
}

func (self *Connecter) listen() {
	for self.isRunning {
		buffer := converter.MakeBuffer(READ_BUFFER_SIZE)
		readLen, targetAddr, err := self.conn.ReadFromUDP(buffer)
		if err == nil {
			if readLen >= ICHEAD_SIZE {
				task := func() {
					converter.HandlePacket(
						self.dataSendManager,
						self.socketmanager,
						self.dataBackupManager,
						targetAddr, buffer, self.socket)
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

func (self *Connecter) connect() {
	buffer := self.dataBackupManager.MakeBuffer(ICHEAD_SIZE)
	converter.SendData(self.socket, buffer, protocol.START_FLAG)
}

func (self *Connecter) Start() {
	go self.dataSendManager.ExecuteForSocket(self.socket)
	go self.dataBackupManager.Execute()
	go self.listen()

	self.isRunning = true
	self.connect()
}

func (self *Connecter) Stop() {
	self.dataSendManager.Stop()
	self.dataBackupManager.Stop()
	self.isRunning = false
}

func (self *Connecter) Close() {
	self.Stop()
	self.conn.Close()
}