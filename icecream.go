package icecream

import (
	"github.com/RexGene/common/threadpool"
	"github.com/RexGene/icecream/icinterface"
	"github.com/RexGene/icecream/manager/connectormanager"
	"github.com/RexGene/icecream/manager/databackupmanager"
	"github.com/RexGene/icecream/manager/datasendmanager"
	"github.com/RexGene/icecream/manager/handlermanager"
	"github.com/RexGene/icecream/manager/protocolmanager"
	"github.com/RexGene/icecream/manager/socketmanager"
	"github.com/RexGene/icecream/net/connector"
	"github.com/RexGene/icecream/net/converter"
	"github.com/golang/protobuf/proto"
	"log"
	"net"
)

const (
	MAX_CONNECT_COUNT = 1024
	READ_BUFFER_SIZE  = 65535
	ICHEAD_SIZE       = 16
)

type IceCream struct {
	udpAddr          *net.UDPAddr
	conn             *net.UDPConn
	dataSendManager  *datasendmanager.DataSendManager
	dataBacupManager *databackupmanager.DataBackupManager
	socketmanager    *socketmanager.SocketManager
	protocolManager  *protocolmanager.ProtocolManager
	handlerManager   *handlermanager.HandlerManager
	isRunning        bool
}

func New() (*IceCream, error) {
	iceCream := &IceCream{
		isRunning: false,
	}

	iceCream.init()

	return iceCream, nil
}

func (self *IceCream) SendMessage(socket icinterface.ISocket, id int, msg proto.Message) {
	converter.SendMessage(socket, id, msg)
}

func (self *IceCream) RegistProtocol(id uint32, makeFunc func() proto.Message) {
	self.protocolManager.RegistProtocol(id, makeFunc)
}

func (self *IceCream) RegistHandler(id uint32, handleFunc func(icinterface.ISocket, proto.Message)) {
	self.handlerManager.RegistHandler(id, handleFunc)
}

func (self *IceCream) Connect(serverName string, addr string) (*connector.Connector, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	udpConn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, err
	}

	connector := connector.New(udpConn, udpAddr)
	connector.Start()

	connectormanager.GetInstance().Insert(serverName, connector)

	return connector, nil
}

func (self *IceCream) listen() {
	for self.isRunning {
		buffer := converter.MakeBuffer(READ_BUFFER_SIZE)
		readLen, targetAddr, err := self.conn.ReadFromUDP(buffer)
		log.Println("[?] after read:", readLen)
		if err == nil {
			if readLen >= ICHEAD_SIZE {
				task := func() {
					if converter.HandlePacket(
						datasendmanager.GetInstance(),
						socketmanager.GetInstance(),
						databackupmanager.GetInstance(),
						handlermanager.GetInstance(),
						targetAddr, buffer, uint(readLen), nil) {
						converter.FreeBuffer(buffer)
					}
				}

				threadpool.GetInstance().Start(task)
			} else {
                if readLen != 1 {
                    log.Println("[!] data len too short:", readLen)
                } else {
                    // keep live
                }
			}
		} else {
			log.Println("[!]", err)
		}
	}
}

func (self *IceCream) init() {
}

func (self *IceCream) Start(addr string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp4", udpAddr)
	if err != nil {
		return err
	}

	self.udpAddr = udpAddr
	self.conn = conn
	self.isRunning = true

	dataBacupManager := databackupmanager.GetInstance()

	dataSendManager := datasendmanager.GetInstance()
	socketmanager := socketmanager.GetInstance()
	socketmanager.SetDataBackupManager(dataBacupManager)

	dataSendManager.Init(conn, dataBacupManager, socketmanager)
	databackupmanager.GetInstance().SetSender(datasendmanager.GetInstance())

	self.dataSendManager = dataSendManager
	self.dataBacupManager = dataBacupManager
	self.socketmanager = socketmanager
	self.protocolManager = protocolmanager.GetInstance()
	self.handlerManager = handlermanager.GetInstance()

	go dataSendManager.Execute()
	go dataBacupManager.Execute()
	go self.handlerManager.Execute()
	go socketmanager.CheckAndRemoveTimeoutSocket()
	go self.listen()

	return nil
}

func (self *IceCream) Stop() error {
	self.isRunning = false
	self.dataSendManager.Stop()
	self.dataBacupManager.Stop()
	self.socketmanager.Stop()
	self.handlerManager.Stop()
	self.conn.Close()

	return nil

}
