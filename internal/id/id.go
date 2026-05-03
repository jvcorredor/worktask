// Package id mints the short hexadecimal identifiers used to name task files.
//
// Each call to [New] returns a fresh 8-character lowercase hex string drawn
// from crypto/rand. The identifier is the stable handle for a task across
// renames, edits, and the open/closed transition.
package id

import (
	"crypto/rand"
	"encoding/hex"
)

// New returns a freshly minted 8-character lowercase hex task identifier.
// It draws four bytes from crypto/rand and panics if the system entropy
// source fails, since the CLI cannot continue without a usable identifier.
func New() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
