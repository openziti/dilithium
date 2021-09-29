package dilithium

type nilInstrument struct{}

func NewNilInstrument() Instrument {
	return &nilInstrument{}
}

func (self *nilInstrument) NewInstance(_ string) InstrumentInstance {
	return &NilInstrumentInstance{}
}

type NilInstrumentInstance struct{}

func (n NilInstrumentInstance) Closed(Adapter) {}

func (n NilInstrumentInstance) WireMessageTx(wm *WireMessage) {}

func (n NilInstrumentInstance) WireMessageRetx(wm *WireMessage) {}

func (n NilInstrumentInstance) WireMessageRx(wm *WireMessage) {}

func (n NilInstrumentInstance) ReadError(err error) {}

func (n NilInstrumentInstance) WriteError(err error) {}

func (n NilInstrumentInstance) UnexpectedMessageType(mt messageType) {}

func (n NilInstrumentInstance) TxAck(wm *WireMessage) {}

func (n NilInstrumentInstance) RxAck(wm *WireMessage) {}

func (n NilInstrumentInstance) TxKeepalive(wm *WireMessage) {}

func (n NilInstrumentInstance) RxKeepalive(wm *WireMessage) {}

func (n NilInstrumentInstance) TxPortalCapacityChanged(capacity int) {}

func (n NilInstrumentInstance) TxPortalSzChanged(capacity int) {}

func (n NilInstrumentInstance) TxPortalRxSzChanged(sz int) {}

func (n NilInstrumentInstance) NewRetxMs(retxMs int) {}

func (n NilInstrumentInstance) NewRetxScale(retxScale float64) {}

func (n NilInstrumentInstance) DuplicateAck(ack int32) {}

func (n NilInstrumentInstance) RxPortalSzChanged(capacity int) {}

func (n NilInstrumentInstance) DuplicateRx(wm *WireMessage) {}

func (n NilInstrumentInstance) Allocate(id string) {}

func (n NilInstrumentInstance) Shutdown() {}
