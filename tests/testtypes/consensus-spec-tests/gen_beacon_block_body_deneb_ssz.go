// Code generated by github.com/karalabe/ssz. DO NOT EDIT.

package consensus_spec_tests

import "github.com/karalabe/ssz"

// Cached static size computed on package init.
var staticSizeCacheBeaconBlockBodyDeneb = 96 + (*Eth1Data)(nil).SizeSSZ() + 32 + 4 + 4 + 4 + 4 + 4 + (*SyncAggregate)(nil).SizeSSZ() + 4 + 4 + 4

// SizeSSZ returns either the static size of the object if fixed == true, or
// the total size otherwise.
func (obj *BeaconBlockBodyDeneb) SizeSSZ(fixed bool) uint32 {
	var size = uint32(staticSizeCacheBeaconBlockBodyDeneb)
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfStaticObjects(obj.ProposerSlashings)
	size += ssz.SizeSliceOfDynamicObjects(obj.AttesterSlashings)
	size += ssz.SizeSliceOfDynamicObjects(obj.Attestations)
	size += ssz.SizeSliceOfStaticObjects(obj.Deposits)
	size += ssz.SizeSliceOfStaticObjects(obj.VoluntaryExits)
	size += ssz.SizeDynamicObject(obj.ExecutionPayload)
	size += ssz.SizeSliceOfStaticObjects(obj.BlsToExecutionChanges)
	size += ssz.SizeSliceOfStaticBytes(obj.BlobKzgCommitments)

	return size
}

// DefineSSZ defines how an object is encoded/decoded.
func (obj *BeaconBlockBodyDeneb) DefineSSZ(codec *ssz.Codec) {
	// Define the static data (fields and dynamic offsets)
	ssz.DefineStaticBytes(codec, obj.RandaoReveal[:])                       // Field  ( 0) -          RandaoReveal - 96 bytes
	ssz.DefineStaticObject(codec, &obj.Eth1Data)                            // Field  ( 1) -              Eth1Data -  ? bytes (Eth1Data)
	ssz.DefineStaticBytes(codec, obj.Graffiti[:])                           // Field  ( 2) -              Graffiti - 32 bytes
	ssz.DefineSliceOfStaticObjectsOffset(codec, &obj.ProposerSlashings)     // Offset ( 3) -     ProposerSlashings -  4 bytes
	ssz.DefineSliceOfDynamicObjectsOffset(codec, &obj.AttesterSlashings)    // Offset ( 4) -     AttesterSlashings -  4 bytes
	ssz.DefineSliceOfDynamicObjectsOffset(codec, &obj.Attestations)         // Offset ( 5) -          Attestations -  4 bytes
	ssz.DefineSliceOfStaticObjectsOffset(codec, &obj.Deposits)              // Offset ( 6) -              Deposits -  4 bytes
	ssz.DefineSliceOfStaticObjectsOffset(codec, &obj.VoluntaryExits)        // Offset ( 7) -        VoluntaryExits -  4 bytes
	ssz.DefineStaticObject(codec, &obj.SyncAggregate)                       // Field  ( 8) -         SyncAggregate -  ? bytes (SyncAggregate)
	ssz.DefineDynamicObjectOffset(codec, &obj.ExecutionPayload)             // Offset ( 9) -      ExecutionPayload -  4 bytes
	ssz.DefineSliceOfStaticObjectsOffset(codec, &obj.BlsToExecutionChanges) // Offset (10) - BlsToExecutionChanges -  4 bytes
	ssz.DefineSliceOfStaticBytesOffset(codec, &obj.BlobKzgCommitments)      // Offset (11) -    BlobKzgCommitments -  4 bytes

	// Define the dynamic data (fields)
	ssz.DefineSliceOfStaticObjectsContent(codec, &obj.ProposerSlashings, 16)     // Field  ( 3) -     ProposerSlashings - ? bytes
	ssz.DefineSliceOfDynamicObjectsContent(codec, &obj.AttesterSlashings, 2)     // Field  ( 4) -     AttesterSlashings - ? bytes
	ssz.DefineSliceOfDynamicObjectsContent(codec, &obj.Attestations, 128)        // Field  ( 5) -          Attestations - ? bytes
	ssz.DefineSliceOfStaticObjectsContent(codec, &obj.Deposits, 16)              // Field  ( 6) -              Deposits - ? bytes
	ssz.DefineSliceOfStaticObjectsContent(codec, &obj.VoluntaryExits, 16)        // Field  ( 7) -        VoluntaryExits - ? bytes
	ssz.DefineDynamicObjectContent(codec, &obj.ExecutionPayload)                 // Field  ( 9) -      ExecutionPayload - ? bytes
	ssz.DefineSliceOfStaticObjectsContent(codec, &obj.BlsToExecutionChanges, 16) // Field  (10) - BlsToExecutionChanges - ? bytes
	ssz.DefineSliceOfStaticBytesContent(codec, &obj.BlobKzgCommitments, 4096)    // Field  (11) -    BlobKzgCommitments - ? bytes
}
