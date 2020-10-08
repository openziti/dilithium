package westworld3

import "sync"

type pool struct {
	id    string
	bufSz uint32
	store *sync.Pool
	ii    InstrumentInstance
}

func newPool(id string, bufSz uint32, ii InstrumentInstance) *pool {
	p := &pool{
		id:    id,
		bufSz: bufSz,
		store: new(sync.Pool),
		ii:    ii,
	}
	p.store.New = p.allocate
	return p
}

func (self *pool) get() *buffer {
	buf := self.store.Get().(*buffer)
	buf.ref()
	return buf
}

func (self *pool) put(buffer *buffer) {
	self.store.Put(buffer)
}

func (self *pool) allocate() interface{} {
	if self.ii != nil {
		self.ii.allocate(self.id)
	}
	return newBuffer(self)
}
