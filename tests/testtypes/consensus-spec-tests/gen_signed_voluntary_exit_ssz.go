// Code generated by github.com/karalabe/ssz. DO NOT EDIT.

package consensus_spec_tests

import "github.com/karalabe/ssz"

// Cached static size computed on package init.
var staticSizeCacheSignedVoluntaryExit = (*VoluntaryExit)(nil).SizeSSZ() + 96

// SizeSSZ returns the total size of the static ssz object.
func (obj *SignedVoluntaryExit) SizeSSZ() uint32 {
	return staticSizeCacheSignedVoluntaryExit
}

// DefineSSZ defines how an object is encoded/decoded.
func (obj *SignedVoluntaryExit) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineStaticObject(codec, &obj.Exit)       // Field  (0) -      Exit -  ? bytes (VoluntaryExit)
	ssz.DefineStaticBytes(codec, obj.Signature[:]) // Field  (1) - Signature - 96 bytes
}
