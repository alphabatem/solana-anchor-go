package main

import (
	"errors"
	"fmt"

	. "github.com/gagliardetto/utilz"
)

var conf = &Config{}

type Config struct {
	Encoding            EncoderName
	TypeID              TypeIDName
	Debug               bool
	DstDir              string
	Package             string
	ModPath             string
	RemoveAccountSuffix bool
}

func GetConfig() *Config {
	return conf
}

// Validate validates
func (cfg *Config) Validate() error {
	if cfg == nil {
		return errors.New("cfg is nil")
	}
	if !isValidEncoder(cfg.Encoding) {
		return fmt.Errorf("Encoder kind is not valid: %q", cfg.Encoding)
	}
	if !isValidTypeIDName(cfg.TypeID) {
		return fmt.Errorf("TypeID kind is not valid: %q", cfg.TypeID)
	}
	return nil
}

// isValidEncoder only accepts Borsh: the generator now emits hand-rolled Borsh codegen against
// fluxrpc/solana-go/binary, which has no Bin/CompactU16-style alternate encoding modes, and the
// "bin"/"compact-u16" modes were never fully correct even under the old gagliardetto/binary
// backend (complex-enum codegen only ever round-tripped under Borsh).
func isValidEncoder(enc EncoderName) bool {
	return SliceContains(
		[]string{
			string(EncodingBorsh),
		},
		string(enc),
	)
}

type TypeIDName string

const (
	TypeIDUvarint32 TypeIDName = "uvarint32"
	TypeIDUint32    TypeIDName = "uint32"
	TypeIDUint8     TypeIDName = "uint8"
	TypeIDAnchor    TypeIDName = "anchor"
	TypeIDNoType    TypeIDName = "notype"
)

// isValidTypeIDName only accepts Anchor: it's the only mode this generator's own example output
// ever actually used (Uvarint32/Uint32/Uint8/NoType were unimplemented `// TODO` stubs already).
func isValidTypeIDName(typeID TypeIDName) bool {
	return SliceContains(
		[]string{
			string(TypeIDAnchor),
		},
		string(typeID),
	)
}

type TypeIDNameSlice []TypeIDName

func (slice TypeIDNameSlice) Has(v TypeIDName) bool {
	for _, enc := range slice {
		if v == enc {
			return true
		}
	}
	return false
}
func (name TypeIDName) On(
	candidates TypeIDNameSlice,
	fn func(),
) TypeIDName {
	if candidates.Has(GetConfig().TypeID) {
		fn()
	}
	return name
}

type EncoderName string

const (
	// EncodingBin and EncodingCompactU16 are not currently accepted by isValidEncoder — the
	// generator only emits Borsh codegen — but the constants and dispatch machinery are kept
	// so unreachable-mode branches elsewhere still compile.
	EncodingBin   EncoderName = "bin"
	EncodingBorsh EncoderName = "borsh"
	// https://docs.solana.com/developing/programming-model/transactions#compact-array-format
	EncodingCompactU16 EncoderName = "compact-u16"
)

type EncoderNameSlice []EncoderName

func (slice EncoderNameSlice) Has(v EncoderName) bool {
	for _, enc := range slice {
		if v == enc {
			return true
		}
	}
	return false
}
func (name EncoderName) On(
	anyEncoding EncoderNameSlice,
	fn func(),
) EncoderName {
	if anyEncoding.Has(GetConfig().Encoding) {
		fn()
	}
	return name
}

func (name EncoderName) OnEncodingBin(fn func()) EncoderName {
	if GetConfig().Encoding == EncodingBin {
		fn()
	}
	return name
}

func (name EncoderName) OnEncodingBorsh(fn func()) EncoderName {
	if GetConfig().Encoding == EncodingBorsh {
		fn()
	}
	return name
}

func (name EncoderName) OnEncodingCompactU16(fn func()) EncoderName {
	if GetConfig().Encoding == EncodingCompactU16 {
		fn()
	}
	return name
}
