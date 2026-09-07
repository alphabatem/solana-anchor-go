package main

import (
	"fmt"

	. "github.com/dave/jennifer/jen"
	"github.com/davecgh/go-spew/spew"
	. "github.com/gagliardetto/utilz"
)

const (
	PkgSolanaGo       = "github.com/fluxrpc/solana-go"
	PkgRpc            = "github.com/fluxrpc/solana-go/rpc"
	PkgDfuseBinary    = "github.com/fluxrpc/solana-go/binary"
	PkgTreeout        = "github.com/gagliardetto/treeout"
	PkgGoFuzz         = "github.com/gagliardetto/gofuzz"
	PkgTestifyRequire = "github.com/stretchr/testify/require"
)

type FileWrapper struct {
	Name string
	File *File
}

func typeStringToType(ts IdlTypeAsString) *Statement {
	stat := newStatement()
	switch ts {
	case IdlTypeBool:
		stat.Bool()
	case IdlTypeU8:
		stat.Uint8()
	case IdlTypeI8:
		stat.Int8()
	case IdlTypeU16:
		stat.Uint16()
	case IdlTypeI16:
		stat.Int16()
	case IdlTypeU32:
		stat.Uint32()
	case IdlTypeI32:
		stat.Int32()
	case IdlTypeU64:
		stat.Uint64()
	case IdlTypeI64:
		stat.Int64()
	case IdlTypeU128:
		stat.Id("Uint128")
	case IdlTypeI128:
		stat.Id("Int128")
	case IdlTypeBytes:
		stat.Index().Byte()
	case IdlTypeString:
		stat.String()
	case IdlTypePubkey:
		stat.Qual(PkgSolanaGo, "PublicKey")
	case IdlTypeF32:
		stat.Float32()
	case IdlTypeF64:
		stat.Float64()

	// Custom:
	case IdlTypeUnixTimestamp:
		stat.Qual(PkgSolanaGo, "UnixTimeSeconds")
	case IdlTypeHash:
		stat.Qual(PkgSolanaGo, "Hash")
	case IdlTypeDuration:
		stat.Qual(PkgSolanaGo, "DurationSeconds")

	default:
		panic(Sf("unknown type string: %s", ts))
	}

	return stat
}

func genField(field IdlField, pointer bool) Code {
	st := newStatement()
	st.Id(ToCamel(field.Name)).
		Add(func() Code {
			if isComplexEnum(field.Type) {
				return Op("*")
			}
			if pointer {
				return Op("*")
			}
			return nil
		}()).
		Add(genTypeName(field.Type))
	return st
}

func genTypeName(idlTypeEnv IdlType) Code {
	st := newStatement()
	switch {
	case idlTypeEnv.IsString():
		{
			st.Add(typeStringToType(idlTypeEnv.GetString()))
		}
	case idlTypeEnv.IsIdlTypeOption():
		{
			opt := idlTypeEnv.GetIdlTypeOption()
			// TODO: optional = pointer?
			st.Add(genTypeName(opt.Option))
		}
	case idlTypeEnv.IsIdlTypeVec():
		{
			vec := idlTypeEnv.GetIdlTypeVec()
			st.Index().Add(genTypeName(vec.Vec))
		}
	case idlTypeEnv.IsIdlTypeDefined():
		{
			st.Add(Id(idlTypeEnv.GetIdlTypeDefined().Defined.Name))
		}
	case idlTypeEnv.IsArray():
		{
			arr := idlTypeEnv.GetArray()
			st.Index(Id(Itoa(arr.Num))).Add(genTypeName(arr.Elem))
		}
	default:
		panic(spew.Sdump(idlTypeEnv))
	}
	return st
}

func codeToString(code Code) string {
	return Sf("%#v", code)
}

// typeRegistryComplexEnum contains all types that are a complex enum (and thus implemented as an interface).
var typeRegistryComplexEnum = make(map[string]struct{})

func isComplexEnum(envel IdlType) bool {
	if envel.IsIdlTypeDefined() {
		_, ok := typeRegistryComplexEnum[envel.GetIdlTypeDefined().Defined.Name]
		return ok
	}
	return false
}

func addTypeNameIsComplexEnum(name string) {
	typeRegistryComplexEnum[name] = struct{}{}
}

func registerComplexEnums(idl *IDL, def IdlTypeDef) {
	switch def.Type.Kind {
	case IdlTypeDefTyKindEnum:
		enumTypeName := def.Name
		if !def.Type.Variants.IsSimpleEnum() {
			addTypeNameIsComplexEnum(enumTypeName)
		}
	}
}

func getElementFieldName(i int) string {
	return fmt.Sprintf("Elem_%d", i)
}

// isU8Element reports whether t is the scalar IDL type "u8" — array/vec-of-u8 gets a raw-bytes
// fast path instead of an element-by-element loop.
func isU8Element(t IdlType) bool {
	return t.IsString() && t.GetString() == IdlTypeU8
}

// genEncodePrimitive emits, into body, a statement that writes value (of scalar IDL type ts) to
// the local `encoder`. u128/i128 are handled by genEncodeValue directly (they're Defined-shaped,
// not primitive-shaped, from an encoding perspective) and never reach here.
func genEncodePrimitive(body *Group, ts IdlTypeAsString, value *Statement) {
	switch ts {
	case IdlTypeBool:
		body.Id("encoder").Dot("WriteBool").Call(value)
	case IdlTypeU8:
		body.Id("encoder").Dot("WriteUint8").Call(value)
	case IdlTypeI8:
		body.Id("encoder").Dot("WriteUint8").Call(Uint8().Call(value))
	case IdlTypeU16:
		body.Id("encoder").Dot("WriteUint16").Call(value)
	case IdlTypeI16:
		body.Id("encoder").Dot("WriteUint16").Call(Uint16().Call(value))
	case IdlTypeU32:
		body.Id("encoder").Dot("WriteUint32").Call(value)
	case IdlTypeI32:
		body.Id("encoder").Dot("WriteUint32").Call(Uint32().Call(value))
	case IdlTypeU64:
		body.Id("encoder").Dot("WriteUint64").Call(value)
	case IdlTypeI64:
		body.Id("encoder").Dot("WriteInt64").Call(value)
	case IdlTypeString:
		body.Id("encoder").Dot("WriteBorshString").Call(value)
	case IdlTypePubkey:
		body.Id("encoder").Dot("WritePublicKey").Call(value)
	case IdlTypeHash:
		body.Id("encoder").Dot("WriteHash").Call(value)
	case IdlTypeF32:
		body.Id("WriteFloat32").Call(Id("encoder"), value)
	case IdlTypeF64:
		body.Id("WriteFloat64").Call(Id("encoder"), value)
	case IdlTypeUnixTimestamp:
		body.Id("encoder").Dot("WriteInt64").Call(Int64().Call(value))
	case IdlTypeDuration:
		body.Id("encoder").Dot("WriteInt64").Call(Int64().Call(value))
	default:
		panic(Sf("genEncodePrimitive: unsupported type: %s", ts))
	}
}

// genDecodePrimitiveExpr returns the expression that reads a value of scalar IDL type ts from the
// local `decoder`. u128/i128/bytes are handled by genDecodeValue directly, never here.
func genDecodePrimitiveExpr(ts IdlTypeAsString) Code {
	switch ts {
	case IdlTypeBool:
		return Id("decoder").Dot("ReadBool").Call()
	case IdlTypeU8:
		return Id("decoder").Dot("ReadUint8").Call()
	case IdlTypeI8:
		return Int8().Call(Id("decoder").Dot("ReadUint8").Call())
	case IdlTypeU16:
		return Id("decoder").Dot("ReadUint16").Call()
	case IdlTypeI16:
		return Int16().Call(Id("decoder").Dot("ReadUint16").Call())
	case IdlTypeU32:
		return Id("decoder").Dot("ReadUint32").Call()
	case IdlTypeI32:
		return Int32().Call(Id("decoder").Dot("ReadUint32").Call())
	case IdlTypeU64:
		return Id("decoder").Dot("ReadUint64").Call()
	case IdlTypeI64:
		return Id("decoder").Dot("ReadInt64").Call()
	case IdlTypeString:
		return Id("decoder").Dot("ReadBorshString").Call()
	case IdlTypePubkey:
		return Id("decoder").Dot("ReadPublicKey").Call()
	case IdlTypeHash:
		return Id("decoder").Dot("ReadHash").Call()
	case IdlTypeF32:
		return Id("ReadFloat32").Call(Id("decoder"))
	case IdlTypeF64:
		return Id("ReadFloat64").Call(Id("decoder"))
	case IdlTypeUnixTimestamp:
		return Qual(PkgSolanaGo, "UnixTimeSeconds").Call(Id("decoder").Dot("ReadInt64").Call())
	case IdlTypeDuration:
		return Qual(PkgSolanaGo, "DurationSeconds").Call(Id("decoder").Dot("ReadInt64").Call())
	default:
		panic(Sf("genDecodePrimitiveExpr: unsupported type: %s", ts))
	}
}

// emitCheckedCall emits "if err := <call>; err != nil { return err }". Nested Marshal/Unmarshal
// calls can't just rely on the shared encoder/decoder's sticky .Err() state checked once at the
// end: a complex enum's "unknown variant" (or "unknown enum index") error is a genuine
// application-level error, not reflected in that sticky state, so it must be explicitly
// propagated here or it would otherwise be silently discarded.
func emitCheckedCall(body *Group, call *Statement) {
	body.If(Id("err").Op(":=").Add(call), Id("err").Op("!=").Nil()).Block(
		Return(Id("err")),
	)
}

// genEncodeValue emits, into body, the statement(s) that Borsh-encode value (a jennifer
// expression of IDL type t, e.g. Id("obj").Dot("Foo")) into the local `encoder`. Recurses for
// Option/Vec/Array/Defined. varPrefix must be unique to this top-level field (e.g. its exported
// Go name) so nested temp/loop variables can't collide with a sibling field's; each recursive
// call extends it with a distinguishing suffix so nested levels within the same field don't
// collide with each other either.
func genEncodeValue(body *Group, t IdlType, value *Statement, varPrefix string) {
	switch {
	case t.IsString():
		ts := t.GetString()
		switch ts {
		case IdlTypeU128, IdlTypeI128:
			emitCheckedCall(body, Add(value).Dot("MarshalWithEncoder").Call(Id("encoder")))
		case IdlTypeBytes:
			body.Id("encoder").Dot("WriteUint32").Call(Uint32().Call(Len(value)))
			body.Id("encoder").Dot("WriteBytes").Call(value)
		default:
			genEncodePrimitive(body, ts, value)
		}
	case t.IsIdlTypeOption():
		opt := t.GetIdlTypeOption()
		body.If(Add(value).Op("==").Nil()).Block(
			Id("encoder").Dot("WriteBool").Call(False()),
		).Else().BlockFunc(func(g *Group) {
			g.Id("encoder").Dot("WriteBool").Call(True())
			genEncodeValue(g, opt.Option, Parens(Op("*").Add(value)), varPrefix+"O")
		})
	case t.IsIdlTypeVec():
		vec := t.GetIdlTypeVec()
		body.Id("encoder").Dot("WriteUint32").Call(Uint32().Call(Len(value)))
		if isU8Element(vec.Vec) {
			body.Id("encoder").Dot("WriteBytes").Call(value)
		} else {
			loopVar := varPrefix + "Elem"
			body.For(List(Id("_"), Id(loopVar)).Op(":=").Range().Add(value)).BlockFunc(func(g *Group) {
				genEncodeValue(g, vec.Vec, Id(loopVar), varPrefix+"V")
			})
		}
	case t.IsArray():
		arr := t.GetArray()
		if isU8Element(arr.Elem) {
			body.Id("encoder").Dot("WriteBytes").Call(Add(value).Index(Op(":")))
		} else {
			loopVar := varPrefix + "Elem"
			body.For(List(Id("_"), Id(loopVar)).Op(":=").Range().Add(value)).BlockFunc(func(g *Group) {
				genEncodeValue(g, arr.Elem, Id(loopVar), varPrefix+"A")
			})
		}
	case t.IsIdlTypeDefined():
		emitCheckedCall(body, Add(value).Dot("MarshalWithEncoder").Call(Id("encoder")))
	default:
		panic(spew.Sdump(t))
	}
}

// genDecodeValue emits, into body, the statement(s) that Borsh-decode a value of IDL type t from
// the local `decoder`, assigning the result into the addressable lvalue dest (a jennifer
// expression whose Go type is exactly what genTypeName(t) would produce). See genEncodeValue for
// the varPrefix uniqueness contract.
func genDecodeValue(body *Group, t IdlType, dest *Statement, varPrefix string) {
	switch {
	case t.IsString():
		ts := t.GetString()
		switch ts {
		case IdlTypeU128, IdlTypeI128:
			emitCheckedCall(body, Parens(Op("&").Add(dest)).Dot("UnmarshalWithDecoder").Call(Id("decoder")))
		case IdlTypeBytes:
			lenVar := varPrefix + "N"
			body.Id(lenVar).Op(":=").Id("decoder").Dot("ReadUint32").Call()
			body.Add(dest).Op("=").Id("decoder").Dot("ReadBytesCopy").Call(Int().Call(Id(lenVar)))
		default:
			body.Add(dest).Op("=").Add(genDecodePrimitiveExpr(ts))
		}
	case t.IsIdlTypeOption():
		opt := t.GetIdlTypeOption()
		tmpVar := varPrefix + "Tmp"
		body.If(Id("decoder").Dot("ReadBool").Call()).BlockFunc(func(g *Group) {
			g.Var().Id(tmpVar).Add(genTypeName(opt.Option))
			genDecodeValue(g, opt.Option, Id(tmpVar), varPrefix+"O")
			g.Add(dest).Op("=").Op("&").Id(tmpVar)
		})
	case t.IsIdlTypeVec():
		vec := t.GetIdlTypeVec()
		lenVar := varPrefix + "N"
		body.Id(lenVar).Op(":=").Id("decoder").Dot("ReadUint32").Call()
		if isU8Element(vec.Vec) {
			body.If(Id(lenVar).Op(">").Lit(uint32(0))).Block(
				Add(dest).Op("=").Id("decoder").Dot("ReadBytesCopy").Call(Int().Call(Id(lenVar))),
			)
		} else {
			loopVar := varPrefix + "I"
			elemVar := varPrefix + "Elem"
			body.If(Id(lenVar).Op(">").Lit(uint32(0))).BlockFunc(func(g *Group) {
				g.Add(dest).Op("=").Make(genTypeName(t), Lit(0), Int().Call(Id(lenVar)))
				g.For(Id(loopVar).Op(":=").Lit(uint32(0)), Id(loopVar).Op("<").Id(lenVar), Id(loopVar).Op("++")).BlockFunc(func(g2 *Group) {
					g2.Var().Id(elemVar).Add(genTypeName(vec.Vec))
					genDecodeValue(g2, vec.Vec, Id(elemVar), varPrefix+"V")
					g2.Add(dest).Op("=").Append(Add(dest), Id(elemVar))
				})
			})
		}
	case t.IsArray():
		arr := t.GetArray()
		if isU8Element(arr.Elem) {
			body.Copy(Add(dest).Index(Op(":")), Id("decoder").Dot("ReadBytes").Call(Lit(arr.Num)))
		} else {
			loopVar := varPrefix + "I"
			elemVar := varPrefix + "Elem"
			body.For(Id(loopVar).Op(":=").Lit(0), Id(loopVar).Op("<").Lit(arr.Num), Id(loopVar).Op("++")).BlockFunc(func(g *Group) {
				g.Var().Id(elemVar).Add(genTypeName(arr.Elem))
				genDecodeValue(g, arr.Elem, Id(elemVar), varPrefix+"A")
				g.Add(dest).Index(Id(loopVar)).Op("=").Id(elemVar)
			})
		}
	case t.IsIdlTypeDefined():
		emitCheckedCall(body, Parens(Op("&").Add(dest)).Dot("UnmarshalWithDecoder").Call(Id("decoder")))
	default:
		panic(spew.Sdump(t))
	}
}

func genTypeDef(idl *IDL, withDiscriminator *[8]byte, def IdlTypeDef) Code {
	st := newStatement()
	switch def.Type.Kind {
	case IdlTypeDefTyKindStruct:
		code := Empty()
		code.Type().Id(def.Name).StructFunc(func(fieldsGroup *Group) {
			if def.Type.Fields == nil {
				emptyFields := []IdlField{}
				def.Type.Fields = (*IdlStructFieldSlice)(&emptyFields)
			}
			for fieldIndex, field := range *def.Type.Fields {
				for docIndex, doc := range field.Docs {
					if docIndex == 0 && fieldIndex > 0 {
						fieldsGroup.Line()
					}
					fieldsGroup.Comment(doc)
				}
				fieldsGroup.Add(genField(field, field.Type.IsIdlTypeOption()))
			}
		})
		st.Add(code.Line())

		{
			// generate encoder and decoder methods (for borsh):
			if GetConfig().Encoding == EncodingBorsh {
				code := Empty()
				exportedAccountName := ToCamel(def.Name)

				if withDiscriminator != nil {
					discriminatorName := exportedAccountName + "Discriminator"
					sighash := *withDiscriminator
					code.Var().Id(discriminatorName).Op("=").Id("TypeID").Op("{").ListFunc(func(byteGroup *Group) {
						for _, byteVal := range sighash[:] {
							byteGroup.Lit(int(byteVal))
						}
					}).Op("}")

					// Declare MarshalWithEncoder:
					code.Line().Line().Add(
						genMarshalWithEncoder_struct(
							idl,
							true,
							exportedAccountName,
							discriminatorName,
							*def.Type.Fields,
							false,
						))

					// Declare UnmarshalWithDecoder
					code.Line().Line().Add(
						genUnmarshalWithDecoder_struct(
							idl,
							true,
							exportedAccountName,
							discriminatorName,
							*def.Type.Fields,
							false,
						))
				} else {
					// Declare MarshalWithEncoder:
					code.Line().Line().Add(
						genMarshalWithEncoder_struct(
							idl,
							false,
							exportedAccountName,
							"",
							*def.Type.Fields,
							false,
						))

					// Declare UnmarshalWithDecoder
					code.Line().Line().Add(
						genUnmarshalWithDecoder_struct(
							idl,
							false,
							exportedAccountName,
							"",
							*def.Type.Fields,
							false,
						))
				}

				st.Add(code.Line().Line())
			}
		}

	case IdlTypeDefTyKindEnum:
		code := newStatement()
		enumTypeName := def.Name

		if def.Type.Variants.IsSimpleEnum() {
			code.Type().Id(enumTypeName).Uint8()
			code.Line().Const().Parens(DoGroup(func(gr *Group) {
				for variantIndex, variant := range *def.Type.Variants {

					for docIndex, doc := range variant.Docs {
						if docIndex == 0 {
							gr.Line()
						}
						gr.Comment(doc).Line()
					}

					gr.Id(formatSimpleEnumVariantName(variant.Name, enumTypeName)).Add(func() Code {
						if variantIndex == 0 {
							return Id(enumTypeName).Op("=").Iota()
						}
						return nil
					}()).Line()
				}
				// TODO: check for fields, etc.
			}))

			// Generate stringer for the uint8 enum values:
			code.Line().Line().Func().Params(Id("value").Id(enumTypeName)).Id("String").
				Params().
				Params(String()).
				BlockFunc(func(body *Group) {
					body.Switch(Id("value")).BlockFunc(func(switchBlock *Group) {
						for _, variant := range *def.Type.Variants {
							switchBlock.Case(Id(formatSimpleEnumVariantName(variant.Name, enumTypeName))).Line().Return(Lit(variant.Name))
						}
						switchBlock.Default().Line().Return(Lit(""))
					})

				})
			code.Line().Line()

			// Simple enums have no other type reflecting over them anymore (no generic
			// Encode/Decode), so they need their own explicit Marshal/Unmarshal too — the
			// discriminant is just the underlying uint8 value itself.
			if GetConfig().Encoding == EncodingBorsh {
				code.Func().Params(Id("obj").Id(enumTypeName)).Id("MarshalWithEncoder").
					Params(Id("encoder").Op("*").Qual(PkgDfuseBinary, "Encoder")).
					Params(Err().Error()).
					BlockFunc(func(body *Group) {
						body.Id("encoder").Dot("WriteUint8").Call(Uint8().Call(Id("obj")))
						body.Return(Id("encoder").Dot("Err").Call())
					})
				code.Line().Line()

				code.Func().Params(Id("obj").Op("*").Id(enumTypeName)).Id("UnmarshalWithDecoder").
					Params(Id("decoder").Op("*").Qual(PkgDfuseBinary, "Decoder")).
					Params(Err().Error()).
					BlockFunc(func(body *Group) {
						body.Op("*").Id("obj").Op("=").Id(enumTypeName).Call(Id("decoder").Dot("ReadUint8").Call())
						body.Return(Id("decoder").Dot("Err").Call())
					})
				code.Line().Line()
			}
			st.Add(code.Line())
		} else {
			addTypeNameIsComplexEnum(enumTypeName)
			interfaceTypeName := ToLowerCamel(enumTypeName)
			interfaceMethodName := formatInterfaceMethodName(enumTypeName)

			// Declare the wrapper struct of the enum type interface
			code.Type().Id(enumTypeName).Struct(
				Id("Value").Qual("", interfaceTypeName),
			).Line().Line()

			{
				// Declare Marshal/UnmarshalWithDecoder of the wrapper struct
				code.Line().Line().Add(
					genMarshalWithEncoder_enum(
						enumTypeName,
						def.Type.Variants,
					))

				code.Line().Line().Add(
					genUnmarshalWithDecoder_enum(
						enumTypeName,
						def.Type.Variants,
					))
				code.Line().Line()
			}

			// Declare the interface of the enum type:
			code.Type().Id(interfaceTypeName).Interface(
				Id(interfaceMethodName).Call(),
			).Line().Line()

			for _, variant := range *def.Type.Variants {
				// Name of the variant type if the enum is a complex enum (i.e. enum variants are inline structs):
				variantTypeNameComplex := formatComplexEnumVariantTypeName(enumTypeName, variant.Name)

				// Declare the enum variant types:
				if variant.IsUint8() {
					// TODO: make the name {variantTypeName}_{interface_name} ???
					code.Type().Id(variantTypeNameComplex).Uint8().Line().Line()
				} else {
					code.Type().Id(variantTypeNameComplex).StructFunc(
						func(structGroup *Group) {
							switch {
							case variant.Fields.IdlEnumFieldsNamed != nil:
								for _, variantField := range *variant.Fields.IdlEnumFieldsNamed {
									structGroup.Add(genField(variantField, variantField.Type.IsIdlTypeOption()))
								}
							default:
								for i, variantTupleItem := range *variant.Fields.IdlEnumFieldsTuple {
									variantField := IdlField{
										Name: getElementFieldName(i),
										Type: variantTupleItem,
									}
									structGroup.Add(genField(variantField, variantField.Type.IsIdlTypeOption()))
								}
							}
						},
					).Line().Line()
				}

				if variant.IsUint8() {
					// Declare MarshalWithEncoder
					code.Line().Line().Func().Params(Id("obj").Id(variantTypeNameComplex)).Id("MarshalWithEncoder").
						Params(
							ListFunc(func(params *Group) {
								// Parameters:
								params.Id("encoder").Op("*").Qual(PkgDfuseBinary, "Encoder")
							}),
						).
						Params(
							ListFunc(func(results *Group) {
								// Results:
								results.Err().Error()
							}),
						).
						BlockFunc(func(body *Group) {
							body.Return(Nil())
						})
					code.Line().Line()

					// Declare UnmarshalWithDecoder
					code.Func().Params(Id("obj").Op("*").Id(variantTypeNameComplex)).Id("UnmarshalWithDecoder").
						Params(
							ListFunc(func(params *Group) {
								// Parameters:
								params.Id("decoder").Op("*").Qual(PkgDfuseBinary, "Decoder")
							}),
						).
						Params(
							ListFunc(func(results *Group) {
								// Results:
								results.Err().Error()
							}),
						).
						BlockFunc(func(body *Group) {
							body.Return(Nil())
						})
					code.Line().Line()
				} else {
					if variant.Fields != nil && variant.Fields.IdlEnumFieldsNamed != nil {
						// Declare MarshalWithEncoder:
						code.Line().Line().Add(
							genMarshalWithEncoder_struct(
								idl,
								false,
								variantTypeNameComplex,
								"",
								*variant.Fields.IdlEnumFieldsNamed,
								false,
							))

						// Declare UnmarshalWithDecoder
						code.Line().Line().Add(
							genUnmarshalWithDecoder_struct(
								idl,
								false,
								variantTypeNameComplex,
								"",
								*variant.Fields.IdlEnumFieldsNamed,
								false,
							))
						code.Line().Line()
					}

					if variant.Fields != nil && variant.Fields.IdlEnumFieldsNamed == nil && variant.Fields.IdlEnumFieldsTuple != nil {
						idlEnumTypeFields := []IdlField{}
						for i, variantTupleItem := range *variant.Fields.IdlEnumFieldsTuple {
							idlEnumTypeFields = append(idlEnumTypeFields, IdlField{
								Name: getElementFieldName(i),
								Type: variantTupleItem,
							})
						}

						code.Line().Line().Add(
							genMarshalWithEncoder_struct(
								idl,
								false,
								variantTypeNameComplex,
								"",
								idlEnumTypeFields,
								false,
							))

						// Declare UnmarshalWithDecoder
						code.Line().Line().Add(
							genUnmarshalWithDecoder_struct(
								idl,
								false,
								variantTypeNameComplex,
								"",
								idlEnumTypeFields,
								false,
							))
						code.Line().Line()
					}
				}

				// Declare the method to implement the parent enum interface:
				code.Func().Params(Id("_").Id(variantTypeNameComplex)).Id(interfaceMethodName).Params().Block().Line().Line()
			}

			st.Add(code.Line().Line())
		}

	default:
		panic(Sf("not implemented: %s", spew.Sdump(def.Type.Kind)))
	}
	return st
}

func formatEnumContainerName(enumTypeName string) string {
	return ToLowerCamel(enumTypeName) + "Container"
}

func formatInterfaceMethodName(enumTypeName string) string {
	return "is" + ToCamel(enumTypeName)
}

func formatBuilderFuncName(insExportedName string) string {
	return "New" + insExportedName + "InstructionBuilder"
}

func formatInstructionTypeName(insExportedName string) string {
	return insExportedName + "Instruction"
}

func formatByteSliceName(insExportedName string) string {
	return ToCamel(ToLower(insExportedName)) + "Bytes"
}

func formatConstantName(insExportedName string) string {
	return "Constant" + ToCamel(ToLower(insExportedName))
}

// genMarshalWithEncoder_enum emits the wrapper type's MarshalWithEncoder: a discriminant byte
// (variant declaration order) followed by the active variant's own encoding.
func genMarshalWithEncoder_enum(
	receiverTypeName string,
	variants *IdlEnumVariantSlice,
) Code {
	code := Empty()
	{
		code.Func().Params(Id("obj").Id(receiverTypeName)).Id("MarshalWithEncoder").
			Params(
				ListFunc(func(params *Group) {
					// Parameters:
					params.Id("encoder").Op("*").Qual(PkgDfuseBinary, "Encoder")
				}),
			).
			Params(
				ListFunc(func(results *Group) {
					// Results:
					results.Err().Error()
				}),
			).BlockFunc(func(body *Group) {
			body.Switch(Id("v").Op(":=").Id("obj").Dot("Value").Op(".").Parens(Type())).
				BlockFunc(func(switchGroup *Group) {
					if variants != nil {
						for variantIndex, variant := range variants.GetEnumVariantTypeName() {
							switchGroup.Case(Id(formatComplexEnumVariantTypeName(receiverTypeName, variant))).
								BlockFunc(func(caseGroup *Group) {
									caseGroup.Id("encoder").Dot("WriteUint8").Call(Lit(variantIndex))
									emitCheckedCall(caseGroup, Id("v").Dot("MarshalWithEncoder").Call(Id("encoder")))
								})
						}
					}
					switchGroup.Default().BlockFunc(func(caseGroup *Group) {
						caseGroup.Return(Qual("fmt", "Errorf").Call(Lit(Sf("%%T: unknown enum variant for %s", receiverTypeName)), Id("obj").Dot("Value")))
					})
				})
			body.Return(Id("encoder").Dot("Err").Call())
		})
	}
	return code
}

// genUnmarshalWithDecoder_enum emits the wrapper type's UnmarshalWithDecoder: read the
// discriminant byte, then decode into the matching variant type and assign it.
func genUnmarshalWithDecoder_enum(
	receiverTypeName string,
	variants *IdlEnumVariantSlice,
) Code {
	code := Empty()
	{
		code.Func().Params(Id("obj").Op("*").Id(receiverTypeName)).Id("UnmarshalWithDecoder").
			Params(
				ListFunc(func(params *Group) {
					// Parameters:
					params.Id("decoder").Op("*").Qual(PkgDfuseBinary, "Decoder")
				}),
			).
			Params(
				ListFunc(func(results *Group) {
					// Results:
					results.Err().Error()
				}),
			).BlockFunc(func(body *Group) {
			body.Id("variantIndex").Op(":=").Id("decoder").Dot("ReadUint8").Call()
			body.If(Id("err").Op(":=").Id("decoder").Dot("Err").Call(), Id("err").Op("!=").Nil()).Block(
				Return(Id("err")),
			)
			body.Switch(Id("variantIndex")).
				BlockFunc(func(switchGroup *Group) {
					for variantIndex, variantName := range variants.GetEnumVariantTypeName() {
						switchGroup.Case(Lit(variantIndex)).
							BlockFunc(func(caseGroup *Group) {
								tmpName := "tmp"
								caseGroup.Id(tmpName).Op(":=").New(Id(formatComplexEnumVariantTypeName(receiverTypeName, variantName)))
								caseGroup.If(
									Id("err").Op(":=").Id(tmpName).Dot("UnmarshalWithDecoder").Call(Id("decoder")),
									Id("err").Op("!=").Nil(),
								).Block(
									Return(Id("err")),
								)
								caseGroup.Id("obj").Dot("Value").Op("=").Op("*").Id(tmpName)
							})
					}
					switchGroup.Default().
						BlockFunc(func(caseGroup *Group) {
							caseGroup.Return(Qual("fmt", "Errorf").Call(Lit("unknown enum index: %v"), Id("variantIndex")))
						})
				})
			body.Return(Id("decoder").Dot("Err").Call())
		})
	}
	return code
}

// genMarshalWithEncoder_struct emits a MarshalWithEncoder method that writes an optional
// discriminator followed by each field via genEncodeValue, using fluxrpc/solana-go/binary's
// sticky-error Encoder: every field write is a bare statement, checked once at the end.
// fieldIsRequiredPointer reports whether obj.Field's actual Go type is a pointer despite field's
// IDL type not being Option — i.e. no Borsh presence-flag byte, but genField still declared it
// with a leading "*" (either because it's a complex enum, which genField always pointer-izes, or
// because the caller declared every field as a pointer regardless of IDL type — instruction args
// do this for "was this arg set" nil-checking in Validate(), unrelated to wire representation).
func fieldIsRequiredPointer(t IdlType, alwaysPointerFields bool) bool {
	return !t.IsIdlTypeOption() && (isComplexEnum(t) || alwaysPointerFields)
}

// genMarshalWithEncoder_struct emits a MarshalWithEncoder method that writes an optional
// discriminator followed by each field via genEncodeValue, using fluxrpc/solana-go/binary's
// sticky-error Encoder: every field write is a bare statement, checked once at the end.
// alwaysPointerFields must be true when the caller declared every field as a Go pointer
// regardless of IDL type (instruction args), and false when pointer-ness already matches
// IsIdlTypeOption() (plain struct/account/event/enum-variant fields).
func genMarshalWithEncoder_struct(
	idl *IDL,
	withDiscriminator bool,
	receiverTypeName string,
	discriminatorName string,
	fields []IdlField,
	alwaysPointerFields bool,
) Code {
	code := Empty()
	{
		code.Func().Params(Id("obj").Id(receiverTypeName)).Id("MarshalWithEncoder").
			Params(
				ListFunc(func(params *Group) {
					// Parameters:
					params.Id("encoder").Op("*").Qual(PkgDfuseBinary, "Encoder")
				}),
			).
			Params(
				ListFunc(func(results *Group) {
					// Results:
					results.Err().Error()
				}),
			).
			BlockFunc(func(body *Group) {
				// Body:
				if withDiscriminator && discriminatorName != "" {
					body.Comment("Write account discriminator:")
					body.Id("encoder").Dot("WriteBytes").Call(Id(discriminatorName).Index(Op(":")))
				}

				for _, field := range fields {
					exportedArgName := ToCamel(field.Name)
					if field.Type.IsIdlTypeOption() {
						body.Commentf("Serialize `%s` param (optional):", exportedArgName)
					} else {
						body.Commentf("Serialize `%s` param:", exportedArgName)
					}
					dest := Id("obj").Dot(exportedArgName)
					if fieldIsRequiredPointer(field.Type, alwaysPointerFields) {
						dest = Parens(Op("*").Add(dest))
					}
					genEncodeValue(body, field.Type, dest, exportedArgName)
				}
				body.Return(Id("encoder").Dot("Err").Call())
			})
	}
	return code
}

// genUnmarshalWithDecoder_struct emits an UnmarshalWithDecoder method that reads-and-checks an
// optional discriminator, then decodes each field via genDecodeValue, checked once at the end.
// See genMarshalWithEncoder_struct for the alwaysPointerFields contract.
func genUnmarshalWithDecoder_struct(
	idl *IDL,
	withDiscriminator bool,
	receiverTypeName string,
	discriminatorName string,
	fields []IdlField,
	alwaysPointerFields bool,
) Code {
	code := Empty()
	{
		code.Func().Params(Id("obj").Op("*").Id(receiverTypeName)).Id("UnmarshalWithDecoder").
			Params(
				ListFunc(func(params *Group) {
					// Parameters:
					params.Id("decoder").Op("*").Qual(PkgDfuseBinary, "Decoder")
				}),
			).
			Params(
				ListFunc(func(results *Group) {
					// Results:
					results.Err().Error()
				}),
			).
			BlockFunc(func(body *Group) {
				// Body:
				if withDiscriminator && discriminatorName != "" {
					body.Comment("Read and check account discriminator:")
					body.BlockFunc(func(discReadBody *Group) {
						discReadBody.Var().Id("discriminator").Id("TypeID")
						discReadBody.Copy(Id("discriminator").Index(Op(":")), Id("decoder").Dot("ReadBytes").Call(Lit(8)))
						discReadBody.If(Id("err").Op(":=").Id("decoder").Dot("Err").Call(), Id("err").Op("!=").Nil()).Block(
							Return(Id("err")),
						)
						discReadBody.If(Id("discriminator").Op("!=").Id(discriminatorName)).Block(
							Return(
								Qual("fmt", "Errorf").Call(
									Line().Lit("wrong discriminator: wanted %s, got %s"),
									Line().Qual("fmt", "Sprint").Call(Id(discriminatorName).Index(Op(":"))),
									Line().Qual("fmt", "Sprint").Call(Id("discriminator").Index(Op(":"))),
								),
							),
						)
					})
				}

				for _, field := range fields {
					exportedArgName := ToCamel(field.Name)
					if field.Type.IsIdlTypeOption() {
						body.Commentf("Deserialize `%s` (optional):", exportedArgName)
					} else {
						body.Commentf("Deserialize `%s`:", exportedArgName)
					}
					if fieldIsRequiredPointer(field.Type, alwaysPointerFields) {
						tmpVar := exportedArgName + "Tmp"
						body.Var().Id(tmpVar).Add(genTypeName(field.Type))
						genDecodeValue(body, field.Type, Id(tmpVar), exportedArgName)
						body.Id("obj").Dot(exportedArgName).Op("=").Op("&").Id(tmpVar)
					} else {
						genDecodeValue(body, field.Type, Id("obj").Dot(exportedArgName), exportedArgName)
					}
				}
				body.Return(Id("decoder").Dot("Err").Call())
			})
	}
	return code
}

func formatComplexEnumVariantTypeName(enumTypeName string, variantName string) string {
	return ToCamel(Sf("%s_%s_Tuple", enumTypeName, variantName))
}

func formatSimpleEnumVariantName(variantName string, enumTypeName string) string {
	return ToCamel(Sf("%s_%s", enumTypeName, variantName))
}
