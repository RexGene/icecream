package socket

import (
	"github.com/RexGene/icecream/manager/datasendmanager"
	"github.com/RexGene/icecream/protocol"
	"net"
	"sync"
	"time"
)

type recvBackupData struct {
	data []byte
	size uint
}

type Socket struct {
	sync.RWMutex

	SrcSeq uint16
	DstSeq uint16

	Token uint32
	Addr  *net.UDPAddr

	state           int
	lastControlTime int64
	sender          *datasendmanager.DataSendManager
	recvMap         map[uint16]*recvBackupData
}

func New() *Socket {
	return &Socket{
		recvMap: make(map[uint16]*recvBackupData),
	}
}

func (self *Socket) EachBackupPacket(seqId uint16, handlePacket func([]byte, uint)) uint16 {
	data := self.recvMap[seqId]
	for data != nil {
		delete(self.recvMap, seqId)
		if handlePacket != nil {
			handlePacket(data.data, data.size)
		}

		seqId++
		data = self.recvMap[seqId]
	}

	return seqId
}

func (self *Socket) InsertBackupList(seqId uint16, data []byte, size uint) {
	backupData := &recvBackupData{
		data: data,
		size: size,
	}

	self.recvMap[seqId] = backupData
	return
}

func (self *Socket) SetSender(sender *datasendmanager.DataSendManager) {
	self.sender = sender
}

func (self *Socket) Format(head *protocol.ICHead, addr *net.UDPAddr,
	sender *datasendmanager.DataSendManager) {
	self.Token = head.Token
	self.Addr = addr
	self.sender = sender
}

func (self *Socket) GetLastUpdateTime() int64 {
	return self.lastControlTime
}

func (self *Socket) GetState() int {
	return self.state
}

func (self *Socket) SetState(state int) {
	self.state = state
}

func (self *Socket) SendData(data []byte, size uint, isNeedBackup bool) {
	self.sender.SendData(self, data, size, isNeedBackup)
}

func (self *Socket) GetSrcSeq() uint16 {
	return self.SrcSeq
}

func (self *Socket) GetDstSeq() uint16 {
	return self.DstSeq
}

func (self *Socket) GetToken() uint32 {
	return self.Token
}

func (self *Socket) GetAddr() *net.UDPAddr {
	return self.Addr
}

func (self *Socket) SetToken(token uint32) {
	self.Token = token
}

func (self *Socket) IncDstSeq() {
	self.lastControlTime = time.Now().Unix()
	self.DstSeq++
}

func (self *Socket) AddDstSeq(value uint16) {
	self.DstSeq += value
}

func (self *Socket) IncSrcSeq() {
	self.SrcSeq++
}

func (self *Socket) SetSrcSeq(seq uint16) {
	self.SrcSeq = seq
}

func (self *Socket) SetDstSeq(seq uint16) {
	self.lastControlTime = time.Now().Unix()
	self.DstSeq = seq
}
