// Code generated by github.com/karalabe/ssz. DO NOT EDIT.

package consensus_spec_tests

import "github.com/karalabe/ssz"

// SizeSSZ returns either the static size of the object if fixed == true, or
// the total size otherwise.
func (obj *BeaconBlock) SizeSSZ(fixed bool) uint32 {
	var size = uint32(8 + 8 + 32 + 32 + 4)
	if fixed {
		return size
	}
	size += ssz.SizeDynamicObject(obj.Body)

	return size
}

// DefineSSZ defines how an object is encoded/decoded.
func (obj *BeaconBlock) DefineSSZ(codec *ssz.Codec) {
	// Define the static data (fields and dynamic offsets)
	ssz.DefineUint64(codec, &obj.Slot)              // Field  (0) -          Slot -  8 bytes
	ssz.DefineUint64(codec, &obj.ProposerIndex)     // Field  (1) - ProposerIndex -  8 bytes
	ssz.DefineStaticBytes(codec, obj.ParentRoot[:]) // Field  (2) -    ParentRoot - 32 bytes
	ssz.DefineStaticBytes(codec, obj.StateRoot[:])  // Field  (3) -     StateRoot - 32 bytes
	ssz.DefineDynamicObjectOffset(codec, &obj.Body) // Offset (4) -          Body -  4 bytes

	// Define the dynamic data (fields)
	ssz.DefineDynamicObjectContent(codec, &obj.Body) // Field  (4) -          Body - ? bytes
}
