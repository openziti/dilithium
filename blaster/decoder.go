package blaster

import (
	"encoding/gob"
	"github.com/pkg/errors"
)

func decode(mh cmsg, dec *gob.Decoder) (*cmsgPair, error) {
	switch mh.Mt {
	case Hello:
		h := chello{}
		if err := dec.Decode(&h); err != nil {
			return nil, errors.Wrap(err, "decode hello")
		}
		return &cmsgPair{h: mh, p: h}, nil

	default:
		return nil, errors.Errorf("unknown Mt [%d]", mh.Mt)
	}
}
