![Obligatory xkcd](http://imgs.xkcd.com/comics/standards.png)

# Simple Serialize (SSZ)... v15

[![API Reference](https://pkg.go.dev/badge/github.com/karalabe/ssz)](https://pkg.go.dev/github.com/karalabe/ssz?tab=doc) ![Build Status](https://github.com/karalabe/ssz/actions/workflows/tests.yml/badge.svg)

Package `ssz` provides a zero-allocation, opinionated toolkit for working with Ethereum's [Simple Serialize (SSZ)](https://github.com/ethereum/consensus-specs/blob/dev/ssz/simple-serialize.md) format through Go. The primary focus is on code maintainability, only secondarily striving towards raw performance.

***Please note, this repository is a work in progress. The API is unstable and breaking changes will regularly be made. Hashing is not yet implemented. Do not depend on this in publicly available modules.***

## Goals and objectives

- **Elegant API surface:** Binary protocols are low level constructs and writing encoders/decoders entails boilerplate and fumbling with details. Code generators can do a good job in achieving performance, but with a too low level API, the generated code becomes impossible to humanly maintain. That isn't an issue, until you throw something at the generator it cannot understand (e.g. multiplexed types), at which point you'll be in deep pain. By defining an API that is elegant from a dev perspective, we can create maintainable code for the special snowflake types, yet still generate it for the rest of boring types.
- **Reduced redundancies:** The API aims to make the common case easy and the less common case possible. Redundancies in user encoding/decoding code are deliberately avoided to remove subtle bugs (even at a slight hit on performance). If the user's types require some asymmetry, explicit encoding and decoding code paths are still supported.
- **Support existing types:** Serialization libraries often assume the user is going to define a completely new, isolated type-set for all the things they want to encode. That is simply not the case, and to introduce a new encoding library into a pre-existing codebase, it must play nicely with the existing types. That means common Go typing and aliasing patterns should be supported without annotating everything with new methods.
- **Performant, as meaningful:** Encoding/decoding code should be performant, even if we're giving up some of it to cater for the above goals. Language constructs that are known to be slow (e.g. reflection) should be avoided, and code should have performance similar to low level generated ones, including 0 needing allocations. That said, a meaningful application of the library will *do* something with the encoded data, which will almost certainly take more time than generating/parsing a binary blob.

## Expectations

Whilst we aim to be a become the SSZ encoder of `go-ethereum` - and more generally, a go-to encoder for all Go applications requiring to work with Ethereum data blobs - there is no guarantee that this outcome will occur. At the present moment, this package is still in the design and experimentation phase and is not ready for a formal proposal.

There are several possible outcomes from this experiment:

- We determine the effort required to implement all current and future SSZ features are not worth it, abandoning this package.
- All the needed features are shipped, but the package is rejected in favor of some other design that is considered superior.
- The API design of this package get merged into some other existing library and this work gets abandoned in its favor.
- The package turns out simple enough, performant enough and popular enough to be accepted into `go-ethereum` beyond a test.
- Some other unforeseen outcome of the infinite possibilities.

## Design

### Responsibilities

The `ssz` package splits the responsibility between user code and library code in the way pictured below:

![Scope](./docs/scope.svg)

- Users are responsible for creating Go structs, which are mapped one-to-one to the SSZ container type.
- The library is responsible for creating all other SSZ types from the fields of the user-defined structs.
- Some SSZ types require specific types to be used due to robustness and performance reasons.
- SSZ unions are not implemented as they are an unused (and disliked) feature in Ethereum.

### Weird stuff

The [Simple Serialize spec](https://github.com/ethereum/consensus-specs/blob/dev/ssz/simple-serialize.md) has schema definitions for mapping SSZ data to [JSON](https://github.com/ethereum/consensus-specs/blob/dev/ssz/simple-serialize.md#json-mapping). We believe in separation of concerns. This library does not concern itself with encoding/decoding from formats other than SSZ.

## How to use

First up, you need to add the package to your project:

```go
go get github.com/karalabe/ssz
```

### Static types

Some data types in Ethereum will only contain a handful of statically sized fields. One such example would be a `Withdrawal` as seen below.

```go
type Address [20]byte

type Withdrawal struct {
    Index     uint64
    Validator uint64
    Address   Address
    Amount    uint64
}
```

In order to encode/decode such a (standalone) object via SSZ, it needs to implement the `ssz.StaticObject` interface:

```go
type StaticObject interface {
	// SizeSSZ returns the total size of an SSZ object.
	SizeSSZ() uint32

	// DefineSSZ defines how an object would be encoded/decoded.
	DefineSSZ(codec *Codec)
}
```

- The `SizeSSZ` seems self-explanatory. It returns the total size of the final SSZ, and for static types such as a `Withdrawal`, you need to calculate this by hand (or by a code generator, more on that later).
- The `DefineSSZ` is more involved. It expects you to define what fields, in what order and with what types are going to be encoded. Essentially, it's the serialization format.

```go
func (w *Withdrawal) SizeSSZ() uint32 { return 44 }

func (w *Withdrawal) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineUint64(codec, &w.Index)          // Field (0) - Index          -  8 bytes
	ssz.DefineUint64(codec, &w.Validator)      // Field (1) - ValidatorIndex -  8 bytes
	ssz.DefineStaticBytes(codec, w.Address[:]) // Field (2) - Address        - 20 bytes
	ssz.DefineUint64(codec, &w.Amount)         // Field (3) - Amount         -  8 bytes
}
```

- The `DefineXYZ` methods should feel self-explanatory. They spill out what fields to encode in what order and into what types. The interesting tidbit is the addressing of the fields. Since this code is used for *both* encoding and decoding, it needs to be able to instantiate any `nil` fields during decoding, so pointers are needed.
- Another interesting part is that we haven't defined an encoder/decoder for `Address`, rather just sliced it into `[]byte`. It is common in Go world that byte slices or arrays are aliased into various types, so instead of requiring the user to annotate all those tiny utility types, they can just use them directly.

To encode the above `Witness` into an SSZ stream, use either `ssz.EncodeToStream` or `ssz.EncodeToBytes`. The former will write into a stream directly, whilst the latter will write into a bytes buffer directly. In both cases you need to supply the output location to avoid GC allocations in the library.

```go
func main() {
	out := new(bytes.Buffer)
	if err := ssz.EncodeToStream(out, new(Withdrawal)); err != nil {
		panic(err)
	}
	fmt.Printf("ssz: %#x\n", blob)
}
```

To decode an SSZ blob, use `ssz.DecodeFromStream` and `ssz.DecodeFromBytes` with the same disclaimers about allocations. Note, decoding requires knowing the *size* of the SSZ blob in advance. Unfortunately, this is a limitation of the SSZ format.

### Dynamic types

Most data types in Ethereum will contain a cool mix of static and dynamic data fields. Encoding those is much more interesting, yet still proudly simple. One such a data type would be an `ExecutionPayload` as seen below:

```go
type Hash      [32]byte
type LogsBLoom [256]byte

type ExecutionPayload struct {
	ParentHash    Hash
	FeeRecipient  Address
	StateRoot     Hash
	ReceiptsRoot  Hash
	LogsBloom     LogsBLoom
	PrevRandao    Hash
	BlockNumber   uint64
	GasLimit      uint64
	GasUsed       uint64
	Timestamp     uint64
	ExtraData     []byte
	BaseFeePerGas *uint256.Int
	BlockHash     Hash
	Transactions  [][]byte
	Withdrawals   []*Withdrawal
}
```

Do note, we've reused the previously defined `Address` and `Withdrawal` types. You'll need those too to make this part of the code work. The `uint256.Int` type is from the `github.com/holiman/uint256` package.

In order to encode/decode such a (standalone) object via SSZ, it needs to implement the `ssz.DynamicObject` interface:

```go
type DynamicObject interface {
	// SizeSSZ returns either the static size of the object if fixed == true, or
	// the total size otherwise.
	SizeSSZ(fixed bool) uint32

	// DefineSSZ defines how an object would be encoded/decoded.
	DefineSSZ(codec *Codec)
}
```

If you look at it more closely, you'll notice that it's almost the same as `ssz.StaticObject`, except the type of `SizeSSZ` is different, here taking an extra boolean argument. The method name/type clash is deliberate: it guarantees compile time that dynamic objects cannot end up in static ssz slots and vice versa.

```go
func (e *ExecutionPayload) SizeSSZ(fixed bool) uint32 {
	// Start out with the static size
	size := uint32(512)
	if fixed {
		return size
	}
	// Append all the dynamic sizes
	size += ssz.SizeDynamicBytes(e.ExtraData)           // Field (10) - ExtraData    - max 32 bytes (not enforced)
	size += ssz.SizeSliceOfDynamicBytes(e.Transactions) // Field (13) - Transactions - max 1048576 items, 1073741824 bytes each (not enforced)
	size += ssz.SizeSliceOfStaticObjects(e.Withdrawals) // Field (14) - Withdrawals  - max 16 items, 44 bytes each (not enforced)

	return size
}
```

Opposed to the static `Withdrawal` from the previous section, `ExecutionPayload` has both static and dynamic fields, so we can't just return a pre-computed literal number.

- First up, we will still need to know the static size of the object to avoid costly runtime calculations over and over. Just for reference, that would be the size of all the static fields in the object + 4 bytes for each dynamic field (offset encoding). Feel free to verify the number `512` above.
  - If the caller requested only the static size via the `fixed` parameter, return early.
- If the caller, however, requested the total size of the object, we need to iterate over all the dynamic fields and accumulate all their sizes too.
  - For all the usual Go suspects like slices and arrays of bytes; 2D sliced and arrays of bytes (i.e. `ExtraData` and `Transactions` above), there are helper methods available in the `ssz` package. 
  - For types implementing `ssz.StaticObject / ssz.DynamicObject` (e.g. one item of `Withdrawals` above), there are again helper methods available to use them as single objects, static array of objects, of dynamic slice of objects.

The codec itself is very similar to the static example before:

```go
func (e *ExecutionPayload) DefineSSZ(codec *ssz.Codec) {
	// Define the static data (fields and dynamic offsets)
	ssz.DefineStaticBytes(codec, e.ParentHash[:])               // Field  ( 0) - ParentHash    -  32 bytes
	ssz.DefineStaticBytes(codec, e.FeeRecipient[:])             // Field  ( 1) - FeeRecipient  -  20 bytes
	ssz.DefineStaticBytes(codec, e.StateRoot[:])                // Field  ( 2) - StateRoot     -  32 bytes
	ssz.DefineStaticBytes(codec, e.ReceiptsRoot[:])             // Field  ( 3) - ReceiptsRoot  -  32 bytes
	ssz.DefineStaticBytes(codec, e.LogsBloom[:])                // Field  ( 4) - LogsBloom     - 256 bytes
	ssz.DefineStaticBytes(codec, e.PrevRandao[:])               // Field  ( 5) - PrevRandao    -  32 bytes
	ssz.DefineUint64(codec, &e.BlockNumber)                     // Field  ( 6) - BlockNumber   -   8 bytes
	ssz.DefineUint64(codec, &e.GasLimit)                        // Field  ( 7) - GasLimit      -   8 bytes
	ssz.DefineUint64(codec, &e.GasUsed)                         // Field  ( 8) - GasUsed       -   8 bytes
	ssz.DefineUint64(codec, &e.Timestamp)                       // Field  ( 9) - Timestamp     -   8 bytes
	ssz.DefineDynamicBytesOffset(codec, &e.ExtraData)           // Offset (10) - ExtraData     -   4 bytes
	ssz.DefineUint256(codec, &e.BaseFeePerGas)                  // Field  (11) - BaseFeePerGas -  32 bytes
	ssz.DefineStaticBytes(codec, e.BlockHash[:])                // Field  (12) - BlockHash     -  32 bytes
	ssz.DefineSliceOfDynamicBytesOffset(codec, &e.Transactions) // Offset (13) - Transactions  -   4 bytes
	ssz.DefineSliceOfStaticObjectsOffset(codec, &e.Withdrawals) // Offset (14) - Withdrawals   -   4 bytes

	// Define the dynamic data (fields)
	ssz.DefineDynamicBytesContent(codec, &e.ExtraData, 32)                                 // Field (10) - ExtraData
	ssz.DefineSliceOfDynamicBytesContent(codec, &e.Transactions, 1_048_576, 1_073_741_824) // Field (13) - Transactions
	ssz.DefineSliceOfStaticObjectsContent(codec, &e.Withdrawals, 16)                       // Field (14) - Withdrawals
}
```

Most of the `DefineXYZ` methods are similar as before. However, you might spot two distinct sets of method calls, `DefineXYZOffset` and `DefineXYZContent`. You'll need to use these for dynamic fields:
  - When SSZ encodes a dynamic object, it encodes it in two steps.
    - A 4-byte offset pointing to the dynamic data is written into the static SSZ area.
    - The dynamic object's actual encoding are written into the dynamic SSZ area.
  - Encoding libraries can take two routes to handle this scenario:
    - Explicitly require the user to give one command to write the object offset, followed by another command later to write the object content. This is fast, but leaks out encoding detail into user code.
    - Require only one command from the user, under the hood writing the object offset immediately, and stashing the object itself away for later serialization when the dynamic area is reached. This keeps the offset notion hidden from users, but entails a GC hit to the encoder.
  - This package was decided to be allocation free, thus the user is needs to be aware that they need to define the dynamic offset first and the dynamic content later. It's a tradeoff to achieve 50-100% speed increase.
  - You might also note that dynamic fields also pass in size limits that the decoder can enforce.

To encode the above `ExecutionPayload` do just as we have done with the static `Witness` object.

### Asymmetric types

For types defined in perfect isolation - dedicated for SSZ - it's easy to define the fields with the perfect types, and perfect sizes, and perfect everything. Generating or writing an elegant encoder for those, is easy.

In reality, often you'll need to encode/decode types which already exist in a codebase, which might not map so cleanly onto the SSZ defined structure spec you want (e.g. you have one union type of `ExecutionPayload` that contains all the Bellatrix, Capella, Deneb, etc fork fields together) and you want to encode/decode them differently based on the context.

Most SSZ libraries will not permit you to do such a thing. Reflection based libraries *cannot* infer the context in which they should switch encoders and can neither can they represent multiple encodings at the same time. Generator based libraries again have no meaningful way to specify optional fields based on different constraints and contexts. 

The only way to handle such scenarios is to write the encoders by hand, and furthermore, encoding might be dependent on what's in the struct, whilst decoding might be dependent on what's it contained within. Completely asymmetric, so our unified *codec definition* approach from the previous sections cannot work.

For these scenarios, this package has support for asymmetric encoders/decoders, where the caller can independently implement the two paths with their unique quirks.

To avoid having a real-world example's complexity overshadow the point we're trying to make here, we'll just convert the previously demoed `Withdrawal` encoding/decoding from the unified `codec` version to a separate `encoder` and `decoder` version.

```go
func (w *Withdrawal) DefineSSZ(codec *ssz.Codec) {
	codec.DefineEncoder(func(enc *ssz.Encoder) {
		ssz.EncodeUint64(enc, w.Index)           // Field (0) - Index          -  8 bytes
		ssz.EncodeUint64(enc, w.Validator)       // Field (1) - ValidatorIndex -  8 bytes
		ssz.EncodeStaticBytes(enc, w.Address[:]) // Field (2) - Address        - 20 bytes
		ssz.EncodeUint64(enc, w.Amount)          // Field (3) - Amount         -  8 bytes
	})
	codec.DefineDecoder(func(dec *ssz.Decoder) {
		ssz.DecodeUint64(dec, &w.Index)          // Field (0) - Index          -  8 bytes
		ssz.DecodeUint64(dec, &w.Validator)      // Field (1) - ValidatorIndex -  8 bytes
		ssz.DecodeStaticBytes(dec, w.Address[:]) // Field (2) - Address        - 20 bytes
		ssz.DecodeUint64(dec, &w.Amount)         // Field (3) - Amount         -  8 bytes
	})
}
```

- As you can see, we piggie-back on the already existing `ssz.Object`'s `DefineSSZ` method, and do *not* require implementing new functions. This is good because we want to be able to seamlessly use unified or split encoders without having to tell everyone about it.
- Whereas previously we had a bunch of `DefineXYZ` method to enumerate the fields for the unified encoding/decoding, here we replaced them with separate definitions for the encoder and decoder via `codec.DefineEncoder` and `codec.DefineDecoder`.
- The implementation of the encoder and decoder follows the exact same pattern and naming conventions as with the `codec` but instead of operating on a `ssz.Codec` object, we're operating on an `ssz.Encoder`/`ssz.Decoder` objects; and instead of calling methods named `ssz.DefineXYZ`, we're calling methods named `ssz.EncodeXYZ` and `ssz.DecodeXYZ`.
- Perhaps note, the `EncodeXYZ` methods do not take pointers to everything anymore, since they do not require the ability to instantiate the field during operation.

Encoding the above `Witness` into an SSZ stream, you use the same thing as before. Everything is seamless.

### Checked types

If your types are using strongly typed arrays (e.g. `[32]byte`, and not `[]byte`) for static lists, the above codes work just fine. However, some types might want to use `[]byte` as the field type, but have it still *behave* as if it was `[32]byte`. This poses an issue, because if the decoder only sees `[]byte`, it cannot figure out how much data you want to decode into it. For those scenarios, we have *checked methods*.

The previous `Withdrawal` is a good example. Let's replace the `type Address [20]byte` alias, with a plain `[]byte` slice (not a `[20]byte` array, rather an opaque `[]byte` slice).

```go
type Withdrawal struct {
    Index     uint64
    Validator uint64
    Address   []byte
    Amount    uint64
}
```

The code for the `SizeSSZ` remains the same. The code for `DefineSSZ` changes ever so slightly:

```go
func (w *Withdrawal) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineUint64(codec, &w.Index)                   // Field (0) - Index          -  8 bytes
	ssz.DefineUint64(codec, &w.Validator)               // Field (1) - ValidatorIndex -  8 bytes
	ssz.DefineCheckedStaticBytes(codec, &w.Address, 20) // Field (2) - Address        - 20 bytes
	ssz.DefineUint64(codec, &w.Amount)                  // Field (3) - Amount         -  8 bytes
}
```

Notably, the `ssz.DefineStaticBytes` call from our old code (which got given a `[20]byte` array), is replaced with `ssz.DefineCheckedStaticBytes`. The latter method operates on an opaque `[]byte` slice, so if we want it to behave like a static sized list, we need to tell it how large it's needed to be. This will result in a runtime check to ensure that the size is correct before decoding.

Note, *checked methods* entail a runtime cost. When decoding such opaque slices, we can't blindly fill the fields with data, rather we need to ensure that they are allocated and that they are of the correct size.  Ideally only use *checked methods* for prototyping or for pre-existing types where you just have to run with whatever you have and can't change the field to an array.

## Generated encoders

More often than not, the Go structs that you'd like to serialize to/from SSZ are simple data containers. Without some particular quirk you'd like to explicitly support, there's little reason to spend precious time counting the bits and digging through a long list of encoder methods to call.

For those scenarios, the library also supports generating the encoding/decoding code via a Go command:

```go
go run github.com/karalabe/ssz/cmd/sszgen --help
```

### Inferred field sizes

Let's go back to our very simple `Withdrawal` type from way back.

```go
type Withdrawal struct {
    Index     uint64
    Validator uint64
    Address   [20]byte
    Amount    uint64
}
```

This seems like a fairly simple thing that we should be able to automatically generate a codec for. Let's try:

```
go run github.com/karalabe/ssz/cmd/sszgen --type Withdrawal
```

Calling the generator on this type will produce the following (very nice I might say) code:

```go
// Code generated by github.com/karalabe/ssz. DO NOT EDIT.

package main

import "github.com/karalabe/ssz"

// SizeSSZ returns the total size of the static ssz object.
func (obj *Withdrawal) SizeSSZ() uint32 {
	return 8 + 8 + 20 + 8
}

// DefineSSZ defines how an object is encoded/decoded.
func (obj *Withdrawal) DefineSSZ(codec *ssz.Codec) {
	ssz.DefineUint64(codec, &obj.Index)          // Field  (0) -     Index -  8 bytes
	ssz.DefineUint64(codec, &obj.Validator)      // Field  (1) - Validator -  8 bytes
	ssz.DefineStaticBytes(codec, obj.Address[:]) // Field  (2) -   Address - 20 bytes
	ssz.DefineUint64(codec, &obj.Amount)         // Field  (3) -    Amount -  8 bytes
}
```

It has everything we would have written ourselves: `SizeSSZ` and `DefineSSZ`... and it also has a lot of useful comments we for sure wouldn't have written outselves. Generator for the win!

Ok, but this was too easy. All the fields of the `Withdrawal` object were primitive types of known lengths, so there's no heavy lifting involved at all. Lets take a look at a juicier example.

### Explicit field sizes

For our complex test, lets pick our dynamic `ExecutionPayload` type from before, but lets make it as hard as it gets and remove all size information from the Go types (e.g. instead of using `[32]byte`, we can make it extra hard by using `[]byte` only).

Now, obviously, if we were to write serialization code by hand, we'd take advantage of our knowledge of what each of these fields is semantically, so we could provide the necessary sizes for a decoder to use. If we want to, however, generate the serialization code, we need to share all that "insider-knowledge" with the code generator somehow.

The standard way in Go world is through struct tags. Specifically in the context of this library, it will be through the `ssz-size` and `ssz-max` tags. These follow the convention set previously by other Go SSZ libraries;

- `ssz-size` can be used to declare a field having a static size
- `ssz-max` can be used to declare a field having a dynamic size with a size cap.
- Both tags support multiple dimensions via comma-separation and omitting via `?`

```go
type ExecutionPayload struct {
	ParentHash    []byte        `ssz-size:"32"`
	FeeRecipient  []byte        `ssz-size:"32"`
	StateRoot     []byte        `ssz-size:"20"`
	ReceiptsRoot  []byte        `ssz-size:"32"`
	LogsBloom     []byte        `ssz-size:"256"`
	PrevRandao    []byte        `ssz-size:"32"`
	BlockNumber   uint64
	GasLimit      uint64
	GasUsed       uint64
	Timestamp     uint64
	ExtraData     []byte        `ssz-max:"32"`
	BaseFeePerGas *uint256.Int
	BlockHash     []byte        `ssz-size:"32"`
	Transactions  [][]byte      `ssz-max:"1048576,1073741824"`
	Withdrawals   []*Withdrawal `ssz-max:"16"`
}
```

Calling the generator as before, just with the `ExecutionPayload` yields in the below, much more interesting code:

```go
// Code generated by github.com/karalabe/ssz. DO NOT EDIT.

package main

import "github.com/karalabe/ssz"

// SizeSSZ returns either the static size of the object if fixed == true, or
// the total size otherwise.
func (obj *ExecutionPayload) SizeSSZ(fixed bool) uint32 {
	var size = uint32(32 + 32 + 20 + 32 + 256 + 32 + 8 + 8 + 8 + 8 + 4 + 32 + 32 + 4 + 4)
	if fixed {
		return size
	}
	size += ssz.SizeDynamicBytes(obj.ExtraData)
	size += ssz.SizeSliceOfDynamicBytes(obj.Transactions)
	size += ssz.SizeSliceOfStaticObjects(obj.Withdrawals)

	return size
}

// DefineSSZ defines how an object is encoded/decoded.
func (obj *ExecutionPayload) DefineSSZ(codec *ssz.Codec) {
	// Define the static data (fields and dynamic offsets)
	ssz.DefineCheckedStaticBytes(codec, &obj.ParentHash, 32)      // Field  ( 0) -    ParentHash -  32 bytes
	ssz.DefineCheckedStaticBytes(codec, &obj.FeeRecipient, 32)    // Field  ( 1) -  FeeRecipient -  32 bytes
	ssz.DefineCheckedStaticBytes(codec, &obj.StateRoot, 20)       // Field  ( 2) -     StateRoot -  20 bytes
	ssz.DefineCheckedStaticBytes(codec, &obj.ReceiptsRoot, 32)    // Field  ( 3) -  ReceiptsRoot -  32 bytes
	ssz.DefineCheckedStaticBytes(codec, &obj.LogsBloom, 256)      // Field  ( 4) -     LogsBloom - 256 bytes
	ssz.DefineCheckedStaticBytes(codec, &obj.PrevRandao, 32)      // Field  ( 5) -    PrevRandao -  32 bytes
	ssz.DefineUint64(codec, &obj.BlockNumber)                     // Field  ( 6) -   BlockNumber -   8 bytes
	ssz.DefineUint64(codec, &obj.GasLimit)                        // Field  ( 7) -      GasLimit -   8 bytes
	ssz.DefineUint64(codec, &obj.GasUsed)                         // Field  ( 8) -       GasUsed -   8 bytes
	ssz.DefineUint64(codec, &obj.Timestamp)                       // Field  ( 9) -     Timestamp -   8 bytes
	ssz.DefineDynamicBytesOffset(codec, &obj.ExtraData)           // Offset (10) -     ExtraData -   4 bytes
	ssz.DefineUint256(codec, &obj.BaseFeePerGas)                  // Field  (11) - BaseFeePerGas -  32 bytes
	ssz.DefineCheckedStaticBytes(codec, &obj.BlockHash, 32)       // Field  (12) -     BlockHash -  32 bytes
	ssz.DefineSliceOfDynamicBytesOffset(codec, &obj.Transactions) // Offset (13) -  Transactions -   4 bytes
	ssz.DefineSliceOfStaticObjectsOffset(codec, &obj.Withdrawals) // Offset (14) -   Withdrawals -   4 bytes

	// Define the dynamic data (fields)
	ssz.DefineDynamicBytesContent(codec, &obj.ExtraData, 32)                            // Field  (10) -     ExtraData - ? bytes
	ssz.DefineSliceOfDynamicBytesContent(codec, &obj.Transactions, 1048576, 1073741824) // Field  (13) -  Transactions - ? bytes
	ssz.DefineSliceOfStaticObjectsContent(codec, &obj.Withdrawals, 16)                  // Field  (14) -   Withdrawals - ? bytes
}
```

Points of interests to note:

- The generator realized that this type contains dynamic fields (either through `ssz-max` tags or via embedded dynamic objects), so it generated an implementation for `ssz.DynamicObject` (vs. `ssz.StaticObject` in the previous section).
- The generator took into consideration all the size `ssz-size` and `ssz-max` fields to generate serialization calls with different based types and runtime size checks.
  - *Note, it is less performant to have runtime size checks like this, so if you know the size of a field, arrays are always preferable vs dynamic lists.*  

### Cross-validated field sizes

We've seen that the size of a field can either be deduced automatically, or it can be provided to the generator explicitly. But what happens if we provide an ssz struct tag for a field of known size?

```go
type Withdrawal struct {
    Index     uint64   `ssz-size:"8"`
    Validator uint64   `ssz-size:"8"`
    Address   [20]byte `ssz-size:"32"` // Deliberately wrong tag size
    Amount    uint64   `ssz-size:"8"`
}
```

```go
go run github.com/karalabe/ssz/cmd/sszgen --type Withdrawal

failed to validate field Withdrawal.Address: array of byte basic type tag conflict: field is 20 bytes, tag wants [32] bytes
```

The code generator will take into consideration the information in both the field's Go type and the struct tag, and will cross validate them against each other. If there's a size conflict, it will abort the code generation.

This functionality can be very helpful in detecting refactor issues, where the user changes the type of a field, which would result in a different encoding. By having the field tagged with an `ssz-size`, such an error would be detected.

As such, we'd recommend *always* tagging all SSZ encoded fields with their sizes. It results in both safer code and self-documenting code.

### Go generate

Perhaps just a mention, anyone using the code generator should call it from a `go:generate` compile instruction. It is much simpler and once added to the code, it can always be called via running `go generate`.

### Multi-type ordering

When generating code for multiple types at once (with one call or many), there's one ordering issue you need to be aware of.

When the code generator finds a field that is a struct of some sort, it needs to decide if it's a static or a dynamic type. To do that, it relies on checking if the type implements the `ssz.StaticObject` or `ssz.DynamicObject` interface. If if doesn't implement either, the generator will error.

This means, however, that if you have a type that's embedded in another type (e.g. in our examples above, `Withdrawal` was embedded inside `ExecutionPayload` in a slice), you need to generate the code for the inner type first, and then the outer type. This ensures that when the outer type is resolving the interface of the inner one, that is already generated and available.

## Quick reference

The table below is a summary of the methods available for `SizeSSZ` and `DefineSSZ`:

- The *Size API* is to be used to implement the `SizeSSZ` method's dynamic parts.
- The *Symmetric API* is to be used if the encoding/decoding doesn't require specialised logic.
- The *Asymmetric API* is to be used if encoding or decoding requires special casing.

*If some type you need is missing, please open an issue, so it can be added.*

|            Type             |                                              Size API                                               |                                                                                                               Symmetric API                                                                                                               |                                                                                                            Asymmetric Encoding                                                                                                            |                                                                                                            Asymmetric Decoding                                                                                                            |
|:---------------------------:|:---------------------------------------------------------------------------------------------------:|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------:|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------:|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------:|
|           `bool`            |                                              `1 byte`                                               |                                                                                   [`DefineBool`](https://pkg.go.dev/github.com/karalabe/ssz#DefineBool)                                                                                   |                                                                                   [`EncodeBool`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeBool)                                                                                   |                                                                                   [`DecodeBool`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeBool)                                                                                   |
|          `uint64`           |                                              `8 bytes`                                              |                                                                                 [`DefineUint64`](https://pkg.go.dev/github.com/karalabe/ssz#DefineUint64)                                                                                 |                                                                                 [`EncodeUint64`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeUint64)                                                                                 |                                                                                 [`DecodeUint64`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeUint64)                                                                                 |
| `[N]byte` as `bitvector[N]` |                                              `N bytes`                                              |                                                                            [`DefineArrayOfBits`](https://pkg.go.dev/github.com/karalabe/ssz#DefineArrayOfBits)                                                                            |                                                                            [`EncodeArrayOfBits`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeArrayOfBits)                                                                            |                                                                            [`DecodeArrayOfBits`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeArrayOfBits)                                                                            |
|     `bitfield.Bitlist`²     |           [`SizeSliceOfBits`](https://pkg.go.dev/github.com/karalabe/ssz#SizeSliceOfBits)           |                     [`DefineSliceOfBitsOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DefineSliceOfBitsOffset) [`DefineSliceOfBitsContent`](https://pkg.go.dev/github.com/karalabe/ssz#DefineSliceOfBitsContent)                     |                     [`EncodeSliceOfBitsOffset`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeSliceOfBitsOffset) [`EncodeSliceOfBitsContent`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeSliceOfBitsContent)                     |                     [`DecodeSliceOfBitsOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeSliceOfBitsOffset) [`DecodeSliceOfBitsContent`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeSliceOfBitsContent)                     |
|         `[N]uint64`         |                                            `N * 8 bytes`                                            |                                                                         [`DefineArrayOfUint64s`](https://pkg.go.dev/github.com/karalabe/ssz#DefineArrayOfUint64s)                                                                         |                                                                         [`EncodeArrayOfUint64s`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeArrayOfUint64s)                                                                         |                                                                         [`DecodeArrayOfUint64s`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeArrayOfUint64s)                                                                         |
|         `[]uint64`          |        [`SizeSliceOfUint64s`](https://pkg.go.dev/github.com/karalabe/ssz#SizeSliceOfUint64s)        |               [`DefineSliceOfUint64sOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DefineSliceOfUint64sOffset) [`DefineSliceOfUint64sContent`](https://pkg.go.dev/github.com/karalabe/ssz#DefineSliceOfUint64sContent)               |               [`EncodeSliceOfUint64sOffset`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeSliceOfUint64sOffset) [`EncodeSliceOfUint64sContent`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeSliceOfUint64sContent)               |               [`DecodeSliceOfUint64sOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeSliceOfUint64sOffset) [`DecodeSliceOfUint64sContent`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeSliceOfUint64sContent)               |
|       `*uint256.Int`¹       |                                             `32 bytes`                                              |                                                                                [`DefineUint256`](https://pkg.go.dev/github.com/karalabe/ssz#DefineUint256)                                                                                |                                                                                [`EncodeUint256`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeUint256)                                                                                |                                                                                [`DecodeUint256`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeUint256)                                                                                |
|          `[N]byte`          |                                              `N bytes`                                              |                                                                            [`DefineStaticBytes`](https://pkg.go.dev/github.com/karalabe/ssz#DefineStaticBytes)                                                                            |                                                                            [`EncodeStaticBytes`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeStaticBytes)                                                                            |                                                                            [`DecodeStaticBytes`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeStaticBytes)                                                                            |
|    `[N]byte` in `[]byte`    |                                              `N bytes`                                              |                                                                     [`DefineCheckedStaticBytes`](https://pkg.go.dev/github.com/karalabe/ssz#DefineCheckedStaticBytes)                                                                     |                                                                     [`EncodeCheckedStaticBytes`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeCheckedStaticBytes)                                                                     |                                                                     [`DecodeCheckedStaticBytes`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeCheckedStaticBytes)                                                                     |
|          `[]byte`           |          [`SizeDynamicBytes`](https://pkg.go.dev/github.com/karalabe/ssz#SizeDynamicBytes)          |                   [`DefineDynamicBytesOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DefineDynamicBytesOffset) [`DefineDynamicBytesContent`](https://pkg.go.dev/github.com/karalabe/ssz#DefineDynamicBytesContent)                   |                   [`EncodeDynamicBytesOffset`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeDynamicBytesOffset) [`EncodeDynamicBytesContent`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeDynamicBytesContent)                   |                   [`DecodeDynamicBytesOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeDynamicBytesOffset) [`DecodeDynamicBytesContent`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeDynamicBytesContent)                   |
|        `[M][N]byte`         |                                            `M * N bytes`                                            |                                                                     [`DefineArrayOfStaticBytes`](https://pkg.go.dev/github.com/karalabe/ssz#DefineArrayOfStaticBytes)                                                                     |                                                                     [`EncodeArrayOfStaticBytes`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeArrayOfStaticBytes)                                                                     |                                                                     [`DecodeArrayOfStaticBytes`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeArrayOfStaticBytes)                                                                     |
| `[M][N]byte` in `[][N]byte` |                                            `M * N bytes`                                            |                                                              [`DefineCheckedArrayOfStaticBytes`](https://pkg.go.dev/github.com/karalabe/ssz#DefineCheckedArrayOfStaticBytes)                                                              |                                                              [`EncodeCheckedArrayOfStaticBytes`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeCheckedArrayOfStaticBytes)                                                              |                                                              [`DecodeCheckedArrayOfStaticBytes`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeCheckedArrayOfStaticBytes)                                                              |
|         `[][N]byte`         |    [`SizeSliceOfStaticBytes`](https://pkg.go.dev/github.com/karalabe/ssz#SizeSliceOfStaticBytes)    |       [`DefineSliceOfStaticBytesOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DefineSliceOfStaticBytesOffset) [`DefineSliceOfStaticBytesContent`](https://pkg.go.dev/github.com/karalabe/ssz#DefineSliceOfStaticBytesContent)       |       [`EncodeSliceOfStaticBytesOffset`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeSliceOfStaticBytesOffset) [`EncodeSliceOfStaticBytesContent`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeSliceOfStaticBytesContent)       |       [`DecodeSliceOfStaticBytesOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeSliceOfStaticBytesOffset) [`DecodeSliceOfStaticBytesContent`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeSliceOfStaticBytesContent)       |
|         `[][]byte`          |   [`SizeSliceOfDynamicBytes`](https://pkg.go.dev/github.com/karalabe/ssz#SizeSliceOfDynamicBytes)   |     [`DefineSliceOfDynamicBytesOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DefineSliceOfDynamicBytesOffset) [`DefineSliceOfDynamicBytesContent`](https://pkg.go.dev/github.com/karalabe/ssz#DefineSliceOfDynamicBytesContent)     |     [`EncodeSliceOfDynamicBytesOffset`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeSliceOfDynamicBytesOffset) [`EncodeSliceOfDynamicBytesContent`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeSliceOfDynamicBytesContent)     |     [`DecodeSliceOfDynamicBytesOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeSliceOfDynamicBytesOffset) [`DecodeSliceOfDynamicBytesContent`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeSliceOfDynamicBytesContent)     |
|     `ssz.StaticObject`      |                                       `Object(nil).SizeSSZ()`                                       |                                                                           [`DefineStaticObject`](https://pkg.go.dev/github.com/karalabe/ssz#DefineStaticObject)                                                                           |                                                                           [`EncodeStaticObject`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeStaticObject)                                                                           |                                                                           [`DecodeStaticObject`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeStaticObject)                                                                           |
|    `[]ssz.StaticObject`     |  [`SizeSliceOfStaticObjects`](https://pkg.go.dev/github.com/karalabe/ssz#SizeSliceOfStaticObjects)  |   [`DefineSliceOfStaticObjectsOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DefineSliceOfStaticObjectsOffset) [`DefineSliceOfStaticObjectsContent`](https://pkg.go.dev/github.com/karalabe/ssz#DefineSliceOfStaticObjectsContent)   |   [`EncodeSliceOfStaticObjectsOffset`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeSliceOfStaticObjectsOffset) [`EncodeSliceOfStaticObjectsContent`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeSliceOfStaticObjectsContent)   |   [`DecodeSliceOfStaticObjectsOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeSliceOfStaticObjectsOffset) [`DecodeSliceOfStaticObjectsContent`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeSliceOfStaticObjectsContent)   |
|     `ssz.DynamicObject`     |         [`SizeDynamicObject`](https://pkg.go.dev/github.com/karalabe/ssz#SizeDynamicObject)         |                   [`DefineDynamicBytesOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DefineDynamicBytesOffset) [`DefineDynamicBytesContent`](https://pkg.go.dev/github.com/karalabe/ssz#DefineDynamicBytesContent)                   |                   [`EncodeDynamicBytesOffset`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeDynamicBytesOffset) [`EncodeDynamicBytesContent`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeDynamicBytesContent)                   |                   [`DecodeDynamicBytesOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeDynamicBytesOffset) [`DecodeDynamicBytesContent`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeDynamicBytesContent)                   |
|    `[]ssz.DynamicObject`    | [`SizeSliceOfDynamicObjects`](https://pkg.go.dev/github.com/karalabe/ssz#SizeSliceOfDynamicObjects) | [`DefineSliceOfDynamicObjectsOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DefineSliceOfDynamicObjectsOffset) [`DefineSliceOfDynamicObjectsContent`](https://pkg.go.dev/github.com/karalabe/ssz#DefineSliceOfDynamicObjectsContent) | [`EncodeSliceOfDynamicObjectsOffset`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeSliceOfDynamicObjectsOffset) [`EncodeSliceOfDynamicObjectsContent`](https://pkg.go.dev/github.com/karalabe/ssz#EncodeSliceOfDynamicObjectsContent) | [`DecodeSliceOfDynamicObjectsOffset`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeSliceOfDynamicObjectsOffset) [`DecodeSliceOfDynamicObjectsContent`](https://pkg.go.dev/github.com/karalabe/ssz#DecodeSliceOfDynamicObjectsContent) |

*¹Type is from `github.com/holiman/uint256`.* \
*²Type is from `github.com/prysmaticlabs/go-bitfield`*.

## Performance

The goal of this package is to be close in performance to low level generated encoders, without sacrificing maintainability. It should, however, be significantly faster than runtime reflection encoders.

The package includes a set of benchmarks for handling the beacon spec types and datasets. You can run them with `go test ./tests --run=NONE --bench=.`.

The below numbers were achieved on a MaxBook Pro M2 Max:

|        Beacon Type         |   Benchmark   |    Speed     |  Throughout   |
|:--------------------------:|:-------------:|:------------:|:-------------:|
|     AggregateAndProof      | encode-stream | 64.36 ns/op  | 5236.49 MB/s  |
|                            | encode-buffer | 75.35 ns/op  | 4472.25 MB/s  |
|                            | decode-stream | 176.9 ns/op  | 1904.66 MB/s  |
|                            | decode-buffer | 107.0 ns/op  | 3150.75 MB/s  |
|        Attestation         | encode-stream | 47.98 ns/op  | 4772.33 MB/s  |
|                            | encode-buffer | 51.68 ns/op  | 4430.80 MB/s  |
|                            | decode-stream | 110.0 ns/op  | 2082.66 MB/s  |
|                            | decode-buffer | 73.45 ns/op  | 3117.73 MB/s  |
|      AttestationData       | encode-stream | 36.43 ns/op  | 3513.67 MB/s  |
|                            | encode-buffer | 38.31 ns/op  | 3341.30 MB/s  |
|                            | decode-stream | 76.72 ns/op  | 1668.49 MB/s  |
|                            | decode-buffer | 49.36 ns/op  | 2593.11 MB/s  |
|      AttesterSlashing      | encode-stream | 126.1 ns/op  | 4377.66 MB/s  |
|                            | encode-buffer | 114.4 ns/op  | 4826.32 MB/s  |
|                            | decode-stream | 297.7 ns/op  | 1854.35 MB/s  |
|                            | decode-buffer | 173.7 ns/op  | 3178.51 MB/s  |
|        BeaconBlock         | encode-stream | 762.1 ns/op  | 9168.59 MB/s  |
|                            | encode-buffer | 837.1 ns/op  | 8346.78 MB/s  |
|                            | decode-stream |  1979 ns/op  | 3531.23 MB/s  |
|                            | decode-buffer |  1071 ns/op  | 6525.61 MB/s  |
|      BeaconBlockBody       | encode-stream |  1323 ns/op  | 12948.71 MB/s |
|                            | encode-buffer |  1572 ns/op  | 10895.84 MB/s |
|                            | decode-stream |  3467 ns/op  | 4939.50 MB/s  |
|                            | decode-buffer |  1841 ns/op  | 9303.13 MB/s  |
|     BeaconBlockHeader      | encode-stream | 26.46 ns/op  | 4232.26 MB/s  |
|                            | encode-buffer | 28.89 ns/op  | 3877.38 MB/s  |
|                            | decode-stream | 56.66 ns/op  | 1976.86 MB/s  |
|                            | decode-buffer | 37.28 ns/op  | 3004.39 MB/s  |
|        BeaconState         | encode-stream | 176132 ns/op | 15271.05 MB/s |
|                            | encode-buffer | 205609 ns/op | 13081.71 MB/s |
|                            | decode-stream | 532203 ns/op | 5053.93 MB/s  |
|                            | decode-buffer | 209053 ns/op | 12866.19 MB/s |
|    BLSToExecutionChange    | encode-stream | 20.04 ns/op  | 3792.75 MB/s  |
|                            | encode-buffer | 23.30 ns/op  | 3261.26 MB/s  |
|                            | decode-stream | 43.76 ns/op  | 1736.79 MB/s  |
|                            | decode-buffer | 30.60 ns/op  | 2483.93 MB/s  |
|         Checkpoint         | encode-stream | 17.78 ns/op  | 2249.76 MB/s  |
|                            | encode-buffer | 20.09 ns/op  | 1990.79 MB/s  |
|                            | decode-stream | 36.44 ns/op  | 1097.80 MB/s  |
|                            | decode-buffer | 28.33 ns/op  | 1411.95 MB/s  |
|          Deposit           | encode-stream | 84.76 ns/op  | 14629.76 MB/s |
|                            | encode-buffer | 104.2 ns/op  | 11895.43 MB/s |
|                            | decode-stream | 255.3 ns/op  | 4856.27 MB/s  |
|                            | decode-buffer | 113.5 ns/op  | 10928.48 MB/s |
|        DepositData         | encode-stream | 23.03 ns/op  | 7988.72 MB/s  |
|                            | encode-buffer | 29.45 ns/op  | 6247.34 MB/s  |
|                            | decode-stream | 51.96 ns/op  | 3541.28 MB/s  |
|                            | decode-buffer | 45.85 ns/op  | 4012.88 MB/s  |
|       DepositMessage       | encode-stream | 22.25 ns/op  | 3954.66 MB/s  |
|                            | encode-buffer | 30.44 ns/op  | 2890.78 MB/s  |
|                            | decode-stream | 44.73 ns/op  | 1967.53 MB/s  |
|                            | decode-buffer | 31.59 ns/op  | 2786.09 MB/s  |
|         Eth1Block          | encode-stream | 21.31 ns/op  | 2252.78 MB/s  |
|                            | encode-buffer | 23.07 ns/op  | 2080.28 MB/s  |
|                            | decode-stream | 43.79 ns/op  | 1096.10 MB/s  |
|                            | decode-buffer | 30.72 ns/op  | 1562.55 MB/s  |
|          Eth1Data          | encode-stream | 20.66 ns/op  | 3484.33 MB/s  |
|                            | encode-buffer | 25.65 ns/op  | 2807.36 MB/s  |
|                            | decode-stream | 42.75 ns/op  | 1684.35 MB/s  |
|                            | decode-buffer | 31.00 ns/op  | 2322.56 MB/s  |
|      ExecutionPayload      | encode-stream | 121.0 ns/op  | 49701.64 MB/s |
|                            | encode-buffer | 211.9 ns/op  | 28381.10 MB/s |
|                            | decode-stream | 396.2 ns/op  | 15180.96 MB/s |
|                            | decode-buffer | 262.1 ns/op  | 22951.19 MB/s |
|   ExecutionPayloadHeader   | encode-stream | 67.48 ns/op  | 8832.36 MB/s  |
|                            | encode-buffer | 78.91 ns/op  | 7553.25 MB/s  |
|                            | decode-stream | 158.2 ns/op  | 3768.06 MB/s  |
|                            | decode-buffer | 94.97 ns/op  | 6275.76 MB/s  |
|            Fork            | encode-stream | 20.47 ns/op  |  781.47 MB/s  |
|                            | encode-buffer | 23.31 ns/op  |  686.52 MB/s  |
|                            | decode-stream | 42.49 ns/op  |  376.59 MB/s  |
|                            | decode-buffer | 31.47 ns/op  |  508.46 MB/s  |
|      HistoricalBatch       | encode-stream | 32476 ns/op  | 16144.11 MB/s |
|                            | encode-buffer | 46433 ns/op  | 11291.17 MB/s |
|                            | decode-stream | 111437 ns/op | 4704.78 MB/s  |
|                            | decode-buffer | 38557 ns/op  | 13597.77 MB/s |
|     HistoricalSummary      | encode-stream | 16.74 ns/op  | 3823.29 MB/s  |
|                            | encode-buffer | 20.66 ns/op  | 3098.03 MB/s  |
|                            | decode-stream | 35.51 ns/op  | 1802.29 MB/s  |
|                            | decode-buffer | 27.03 ns/op  | 2368.03 MB/s  |
|     IndexedAttestation     | encode-stream | 51.46 ns/op  | 4741.58 MB/s  |
|                            | encode-buffer | 53.81 ns/op  | 4534.74 MB/s  |
|                            | decode-stream | 114.8 ns/op  | 2125.90 MB/s  |
|                            | decode-buffer | 72.70 ns/op  | 3356.35 MB/s  |
|     PendingAttestation     | encode-stream | 53.93 ns/op  | 2762.89 MB/s  |
|                            | encode-buffer | 54.11 ns/op  | 2753.43 MB/s  |
|                            | decode-stream | 116.0 ns/op  | 1284.27 MB/s  |
|                            | decode-buffer | 75.02 ns/op  | 1986.16 MB/s  |
|      ProposerSlashing      | encode-stream | 55.02 ns/op  | 7561.01 MB/s  |
|                            | encode-buffer | 61.17 ns/op  | 6800.62 MB/s  |
|                            | decode-stream | 117.6 ns/op  | 3538.03 MB/s  |
|                            | decode-buffer | 77.87 ns/op  | 5341.99 MB/s  |
|  SignedBeaconBlockHeader   | encode-stream | 30.84 ns/op  | 6745.16 MB/s  |
|                            | encode-buffer | 35.03 ns/op  | 5937.03 MB/s  |
|                            | decode-stream | 67.64 ns/op  | 3074.88 MB/s  |
|                            | decode-buffer | 45.91 ns/op  | 4530.83 MB/s  |
| SignedBLSToExecutionChange | encode-stream | 25.88 ns/op  | 6647.02 MB/s  |
|                            | encode-buffer | 34.28 ns/op  | 5017.71 MB/s  |
|                            | decode-stream | 62.50 ns/op  | 2752.11 MB/s  |
|                            | decode-buffer | 47.10 ns/op  | 3651.46 MB/s  |
|    SignedVoluntaryExit     | encode-stream | 25.04 ns/op  | 4471.99 MB/s  |
|                            | encode-buffer | 26.43 ns/op  | 4237.30 MB/s  |
|                            | decode-stream | 47.22 ns/op  | 2371.77 MB/s  |
|                            | decode-buffer | 35.13 ns/op  | 3188.19 MB/s  |
|       SyncAggregate        | encode-stream | 16.79 ns/op  | 9529.59 MB/s  |
|                            | encode-buffer | 21.54 ns/op  | 7427.30 MB/s  |
|                            | decode-stream | 36.61 ns/op  | 4369.90 MB/s  |
|                            | decode-buffer | 29.30 ns/op  | 5460.15 MB/s  |
|       SyncCommittee        | encode-stream |  1028 ns/op  | 23956.16 MB/s |
|                            | encode-buffer |  1332 ns/op  | 18493.17 MB/s |
|                            | decode-stream |  3242 ns/op  | 7594.97 MB/s  |
|                            | decode-buffer |  1290 ns/op  | 19081.26 MB/s |
|         Validator          | encode-stream | 37.17 ns/op  | 3255.24 MB/s  |
|                            | encode-buffer | 38.51 ns/op  | 3142.10 MB/s  |
|                            | decode-stream | 80.98 ns/op  | 1494.23 MB/s  |
|                            | decode-buffer | 49.85 ns/op  | 2427.16 MB/s  |
|       VoluntaryExit        | encode-stream | 18.98 ns/op  |  843.05 MB/s  |
|                            | encode-buffer | 22.92 ns/op  |  698.01 MB/s  |
|                            | decode-stream | 36.31 ns/op  |  440.65 MB/s  |
|                            | decode-buffer | 27.40 ns/op  |  583.86 MB/s  |
|         Withdrawal         | encode-stream | 27.58 ns/op  | 1595.26 MB/s  |
|                            | encode-buffer | 31.53 ns/op  | 1395.58 MB/s  |
|                            | decode-stream | 51.02 ns/op  |  862.45 MB/s  |
|                            | decode-buffer | 38.39 ns/op  | 1146.25 MB/s  |
