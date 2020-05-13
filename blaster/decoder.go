package blaster

import (
	"encoding/gob"
	"github.com/pkg/errors"
)

func decode(mh cmsg, dec *gob.Decoder) (*cmsgPair, error) {
	switch mh.Mt {
	case Hello:
		return &cmsgPair{h: mh}, nil

	default:
		return nil, errors.Errorf("unknown Mt [%d]", mh.Mt)
	}
}
