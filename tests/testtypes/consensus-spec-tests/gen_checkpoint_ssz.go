// Code generated by github.com/karalabe/ssz. DO NOT EDIT.

package consensus_spec_tests

import "github.com/karalabe/ssz"

// SizeSSZ returns the total size of the static ssz object.
func (obj *Checkpoint) SizeSSZ() uint32 {
	return 8 + 32
}

// DefineSSZ defines how an object is encoded/decoded.
func (obj *Checkpoint) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineUint64(codec, &obj.Epoch)     // Field  (0) - Epoch -  8 bytes
	ssz.DefineStaticBytes(codec, &obj.Root) // Field  (1) -  Root - 32 bytes
}
