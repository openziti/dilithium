package blaster

import (
	"encoding/gob"
	"github.com/pkg/errors"
)

func decode(mh cmsg, dec *gob.Decoder) (*cmsgPair, error) {
	switch mh.mt {
	case Hello:
		return &cmsgPair{h: mh}, nil

	default:
		return nil, errors.Errorf("unknown mt [%d]", mh.mt)
	}
}
