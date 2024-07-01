// ssz: Go Simple Serialize (SSZ) codec library
// Copyright 2024 ssz Authors
// SPDX-License-Identifier: BSD-3-Clause

package consensus_spec_tests

import "github.com/karalabe/ssz"

type Withdrawal struct {
	Index     uint64
	Validator uint64
	Address   Address
	Amount    uint64
}

func (w *Withdrawal) StaticSSZ() bool { return true }
func (w *Withdrawal) SizeSSZ() uint32 { return 44 }

func (w *Withdrawal) EncodeSSZ(enc *ssz.Encoder) {
	ssz.EncodeUint64(enc, w.Index)           // Field (0) - Index          -  8 bytes
	ssz.EncodeUint64(enc, w.Validator)       // Field (1) - ValidatorIndex -  8 bytes
	ssz.EncodeStaticBytes(enc, w.Address[:]) // Field (2) - Address        - 20 bytes
	ssz.EncodeUint64(enc, w.Amount)          // Field (3) - Amount         -  8 bytes
}

func (w *Withdrawal) DecodeSSZ(dec *ssz.Decoder) {
	ssz.DecodeUint64(dec, &w.Index)          // Field (0) - Index          -  8 bytes
	ssz.DecodeUint64(dec, &w.Validator)      // Field (1) - ValidatorIndex -  8 bytes
	ssz.DecodeStaticBytes(dec, w.Address[:]) // Field (2) - Address        - 20 bytes
	ssz.DecodeUint64(dec, &w.Amount)         // Field (3) - Amount         -  8 bytes
}
