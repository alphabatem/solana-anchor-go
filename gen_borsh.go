package main

import (
	. "github.com/dave/jennifer/jen"
)

// genBorshFile builds "borsh.go": local replacements for the pieces of gagliardetto/binary that
// fluxrpc/solana-go/binary has no equivalent for — an 8-byte discriminator type, 128-bit integer
// types (raw little-endian byte layout, which is also the correct Borsh wire format), and
// float32/64 helpers (fluxrpc's Encoder/Decoder have no float support at all).
func genBorshFile(idl IDL) (*FileWrapper, error) {
	file := NewGoFile(idl.Metadata.Name, true)

	file.Add(
		Comment("TypeID is an 8-byte Anchor/Borsh discriminator.").Line(),
		Type().Id("TypeID").Index(Lit(8)).Byte().Line(),
	)
	file.Add(
		Func().Params(Id("id").Id("TypeID")).Id("Bytes").
			Params().
			Params(Index().Byte()).
			Block(Return(Id("id").Index(Op(":")))).
			Line(),
	)

	file.Add(
		Comment("Uint128 is the raw little-endian byte layout of a Borsh u128.").Line(),
		Type().Id("Uint128").Index(Lit(16)).Byte().Line(),
	)
	file.Add(
		Func().Params(Id("v").Id("Uint128")).Id("MarshalWithEncoder").
			Params(Id("encoder").Op("*").Qual(PkgDfuseBinary, "Encoder")).
			Params(Err().Error()).
			BlockFunc(func(body *Group) {
				body.Id("encoder").Dot("WriteBytes").Call(Id("v").Index(Op(":")))
				body.Return(Id("encoder").Dot("Err").Call())
			}).Line(),
	)
	file.Add(
		Func().Params(Id("v").Op("*").Id("Uint128")).Id("UnmarshalWithDecoder").
			Params(Id("decoder").Op("*").Qual(PkgDfuseBinary, "Decoder")).
			Params(Err().Error()).
			BlockFunc(func(body *Group) {
				body.Copy(Id("v").Index(Op(":")), Id("decoder").Dot("ReadBytes").Call(Lit(16)))
				body.Return(Id("decoder").Dot("Err").Call())
			}).Line(),
	)

	file.Add(
		Comment("Int128 is byte-identical to Uint128 — two's-complement is already correct as raw little-endian bytes.").Line(),
		Type().Id("Int128").Index(Lit(16)).Byte().Line(),
	)
	file.Add(
		Func().Params(Id("v").Id("Int128")).Id("MarshalWithEncoder").
			Params(Id("encoder").Op("*").Qual(PkgDfuseBinary, "Encoder")).
			Params(Err().Error()).
			BlockFunc(func(body *Group) {
				body.Id("encoder").Dot("WriteBytes").Call(Id("v").Index(Op(":")))
				body.Return(Id("encoder").Dot("Err").Call())
			}).Line(),
	)
	file.Add(
		Func().Params(Id("v").Op("*").Id("Int128")).Id("UnmarshalWithDecoder").
			Params(Id("decoder").Op("*").Qual(PkgDfuseBinary, "Decoder")).
			Params(Err().Error()).
			BlockFunc(func(body *Group) {
				body.Copy(Id("v").Index(Op(":")), Id("decoder").Dot("ReadBytes").Call(Lit(16)))
				body.Return(Id("decoder").Dot("Err").Call())
			}).Line(),
	)

	// fluxrpc/solana-go/binary has no float32/64 support at all: encode/decode via IEEE-754 bit
	// patterns over the existing uint32/64 primitives.
	file.Add(
		Func().Id("WriteFloat32").
			Params(Id("encoder").Op("*").Qual(PkgDfuseBinary, "Encoder"), Id("v").Float32()).
			Block(
				Id("encoder").Dot("WriteUint32").Call(Qual("math", "Float32bits").Call(Id("v"))),
			).Line(),
	)
	file.Add(
		Func().Id("ReadFloat32").
			Params(Id("decoder").Op("*").Qual(PkgDfuseBinary, "Decoder")).
			Params(Float32()).
			Block(
				Return(Qual("math", "Float32frombits").Call(Id("decoder").Dot("ReadUint32").Call())),
			).Line(),
	)
	file.Add(
		Func().Id("WriteFloat64").
			Params(Id("encoder").Op("*").Qual(PkgDfuseBinary, "Encoder"), Id("v").Float64()).
			Block(
				Id("encoder").Dot("WriteUint64").Call(Qual("math", "Float64bits").Call(Id("v"))),
			).Line(),
	)
	file.Add(
		Func().Id("ReadFloat64").
			Params(Id("decoder").Op("*").Qual(PkgDfuseBinary, "Decoder")).
			Params(Float64()).
			Block(
				Return(Qual("math", "Float64frombits").Call(Id("decoder").Dot("ReadUint64").Call())),
			).Line(),
	)

	return &FileWrapper{
		Name: "borsh",
		File: file,
	}, nil
}
