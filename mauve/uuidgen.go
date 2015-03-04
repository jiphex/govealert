package mauve

import (
	"encoding/base32"
	"math/rand"
	"time"
)

// It doesn't really matter what this does, it'd be fine if it
// return actual UUIDs, but doing that is slightly more difficult
// than just making up a random number and converting it to base32
func RandomID() string {
	s := rand.NewSource(time.Now().UTC().UnixNano()) // don't really need a good random source
	bytes := make([]byte, 10)                        // 10 bytes because that means no padding (=).
	for i := range bytes {
		bytes[i] = byte(rand.New(s).Intn(256))
	}
	enc := base32.StdEncoding.EncodeToString(bytes)
	return enc
}
