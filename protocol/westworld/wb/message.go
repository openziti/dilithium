package wb

import (
	"encoding/binary"
	"fmt"
	"github.com/michaelquigley/dilithium/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
)

type Type uint8

const (
	HELLO Type = iota
	DATA
	ACK
	CLOSE
)

type WireMessage struct {
	Sequence int32
	Type     Type
	Ack      int32
	Data     []byte
	buffer   []byte
	len      int
	pool     *BufferPool
	trace    []*stackFrame
	freed    bool
	refs     int32
	lock     *sync.Mutex
}

func NewHello(sequence int32, pool *BufferPool) (*WireMessage, error) {
	wm := &WireMessage{
		Sequence: sequence,
		Type:     HELLO,
		Ack:      -1,
		Data:     nil,
		buffer:   pool.Get().([]byte),
		pool:     pool,
		lock:     new(sync.Mutex),
	}
	runtime.SetFinalizer(wm, finalizer)
	return wm.encode()
}

func NewHelloAck(sequence, ack int32, pool *BufferPool) (*WireMessage, error) {
	wm := &WireMessage{
		Sequence: sequence,
		Type:     HELLO,
		Ack:      ack,
		Data:     nil,
		buffer:   pool.Get().([]byte),
		pool:     pool,
		lock:     new(sync.Mutex),
	}
	runtime.SetFinalizer(wm, finalizer)
	return wm.encode()
}

func NewData(sequence int32, data []byte, pool *BufferPool) (*WireMessage, error) {
	wm := &WireMessage{
		Sequence: sequence,
		Type:     DATA,
		Ack:      -1,
		Data:     data,
		buffer:   pool.Get().([]byte),
		pool:     pool,
		lock:     new(sync.Mutex),
	}
	//runtime.SetFinalizer(wm, finalizer)
	return wm.encode()
}

func NewAck(forSequence int32, pool *BufferPool) (*WireMessage, error) {
	wm := &WireMessage{
		Sequence: -1,
		Type:     ACK,
		Ack:      forSequence,
		Data:     nil,
		buffer:   pool.Get().([]byte),
		pool:     pool,
		lock:     new(sync.Mutex),
	}
	return wm.encode()
}

func (self *WireMessage) ToBuffer() []byte {
	self.recordStackFrame("toBuffer", 2)
	return self.buffer[:self.len]
}

func (self *WireMessage) encode() (*WireMessage, error) {
	buffer := util.NewByteWriter(self.buffer)
	if err := binary.Write(buffer, binary.LittleEndian, self.Sequence); err != nil {
		return nil, err
	}
	if err := binary.Write(buffer, binary.LittleEndian, uint8(self.Type)); err != nil {
		return nil, err
	}
	if err := binary.Write(buffer, binary.LittleEndian, self.Ack); err != nil {
		return nil, err
	}
	if err := binary.Write(buffer, binary.LittleEndian, int32(len(self.Data))); err != nil {
		return nil, err
	}
	if self.Data != nil && len(self.Data) > 0 {
		n, err := buffer.Write(self.Data)
		if err != nil {
			return nil, err
		}
		if n != len(self.Data) {
			return nil, errors.New("short write")
		}
	}
	self.Data = self.buffer[13 : 13+len(self.Data)]
	self.len = buffer.Len()
	return self, nil
}

func FromBuffer(buffer []byte, pool *BufferPool) (*WireMessage, error) {
	wm := &WireMessage{buffer: buffer, pool: pool}
	if sequence, err := util.ReadInt32(buffer[:4]); err == nil {
		wm.Sequence = sequence
	} else {
		return nil, err
	}
	if type_, err := util.ReadUint8(buffer[4:5]); err == nil {
		wm.Type = Type(type_)
	} else {
		return nil, err
	}
	if ack, err := util.ReadInt32(buffer[5:9]); err == nil {
		wm.Ack = ack
	} else {
		return nil, err
	}
	if len, err := util.ReadInt32(buffer[9:13]); err == nil {
		wm.Data = buffer[13 : 13+len]
		wm.len = 13 + int(len)
	} else {
		return nil, err
	}
	return wm, nil
}

func ReadWireMessage(conn *net.UDPConn, pool *BufferPool) (wm *WireMessage, peer *net.UDPAddr, err error) {
	buffer := pool.Get().([]byte)

	_, peer, err = conn.ReadFromUDP(buffer)
	if err != nil {
		return nil, peer, errors.Wrap(err, "read from peer")
	}

	wm, err = FromBuffer(buffer, pool)
	if err != nil {
		return nil, peer, errors.Wrap(err, "from data")
	}

	return wm, peer, err
}

func (self *WireMessage) WriteMessage(conn *net.UDPConn, peer *net.UDPAddr) error {
	self.recordStackFrame("write", 2)
	//self.DumpTrace()
	n, err := conn.WriteToUDP(self.buffer[:self.len], peer)
	if err != nil {
		return errors.Wrap(err, "write to peer")
	}
	if n != self.len {
		return errors.New("short write to peer")
	}

	return nil
}

func (self *WireMessage) RewriteAck(ack int32) error {
	self.recordStackFrame("rewriteAck", 2)
	buffer := util.NewByteWriter(self.buffer[5:9])
	if err := binary.Write(buffer, binary.LittleEndian, ack); err != nil {
		return err
	}
	self.Ack = ack
	return nil
}

func (self *WireMessage) Touch() {
	self.recordStackFrame("access", 2)
	if self.freed {
		logrus.Errorf("touch after free! [%v]", self.trace)
	}
}

func (self *WireMessage) Ref() {
	atomic.AddInt32(&self.refs, 1)
}

func (self *WireMessage) Unref() {
	atomic.AddInt32(&self.refs, -1)
	refs := atomic.LoadInt32(&self.refs)
	if refs == 0 {
		self.free()
	}
}

func (self *WireMessage) free() {
	self.recordStackFrame("free", 3)
	if !self.freed && self.pool != nil && self.buffer != nil {
		self.pool.Put(self.buffer)
		self.buffer = nil
		self.len = 0
		self.freed = true
		runtime.GC()
	} else {
		logrus.Errorf("double-free! [%v]", self.trace)
	}
}

func (self *WireMessage) DumpTrace() {
	fmt.Printf("trace = #%d {\n", self.Sequence)
	for i, frame := range self.trace {
		fmt.Printf("\t[%d]: %s (%s)\n\t\t%s:%d\n", i, frame.f, frame.op, frame.file, frame.line)
	}
	fmt.Println("}")
}

func finalizer(wm *WireMessage) {
	if !wm.freed {
		logrus.Errorf("not-freed, finalizing [%d] [%p]", wm.Sequence, wm)
	}
}

func (self *WireMessage) recordStackFrame(op string, deep int) {
	pc, file, line, ok := runtime.Caller(deep)
	if !ok {
		self.trace = append(self.trace, &stackFrame{op, "?", 0, "?"})
		return
	}

	fn := runtime.FuncForPC(pc)
	self.trace = append(self.trace, &stackFrame{op, file, line, fn.Name()})
}

type stackFrame struct {
	op   string
	file string
	line int
	f    string
}
