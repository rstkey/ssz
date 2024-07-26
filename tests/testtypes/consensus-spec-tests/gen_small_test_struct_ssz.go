// Code generated by github.com/karalabe/ssz. DO NOT EDIT.

package consensus_spec_tests

import "github.com/karalabe/ssz"

// SizeSSZ returns the total size of the static ssz object.
func (obj *SmallTestStruct) SizeSSZ() uint32 {
	return 2 + 2
}

// DefineSSZ defines how an object is encoded/decoded.
func (obj *SmallTestStruct) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineUint16(codec, &obj.A) // Field  (0) - A - 2 bytes
	ssz.DefineUint16(codec, &obj.B) // Field  (1) - B - 2 bytes
}