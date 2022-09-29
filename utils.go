
package hoom

import (
	crand "crypto/rand"
	"encoding/binary"
	"io"
)

func RandUint64()(v uint64){
	var buf [8]byte
	_, err := io.ReadFull(crand.Reader, buf[:])
	if err != nil {
		panic(err)
	}
	v = binary.BigEndian.Uint64(buf[:])
	return
}
