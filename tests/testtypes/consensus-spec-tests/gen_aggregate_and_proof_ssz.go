// Code generated by github.com/karalabe/ssz. DO NOT EDIT.

package consensus_spec_tests

import "github.com/karalabe/ssz"

// SizeSSZ returns either the static size of the object if fixed == true, or
// the total size otherwise.
func (obj *AggregateAndProof) SizeSSZ(fixed bool) uint32 {
	var size = uint32(8 + 4 + 96)
	if fixed {
		return size
	}
	size += ssz.SizeDynamicObject(obj.Aggregate)

	return size
}

// DefineSSZ defines how an object is encoded/decoded.
func (obj *AggregateAndProof) DefineSSZ(codec *ssz.Codec) {
	// Define the static data (fields and dynamic offsets)
	ssz.DefineUint64(codec, &obj.Index)                  // Field  (0) -          Index -  8 bytes
	ssz.DefineDynamicObjectOffset(codec, &obj.Aggregate) // Offset (1) -      Aggregate -  4 bytes
	ssz.DefineStaticBytes(codec, obj.SelectionProof[:])  // Field  (2) - SelectionProof - 96 bytes

	// Define the dynamic data (fields)
	ssz.DefineDynamicObjectContent(codec, &obj.Aggregate) // Field  (1) -      Aggregate - ? bytes
}
