package configschema

import (
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
)

// ImpliedType returns the cty.Type that would result from decoding a
// configuration block using the receiving block schema.
//
// ImpliedType always returns a result, even if the given schema is
// inconsistent.
func (b *Block) ImpliedType() cty.Type {
	if b == nil {
		return cty.EmptyObject
	}

	atys := make(map[string]cty.Type)

	for _, attrS := range b.Attributes {
		name := attrS.Name
		atys[name] = WrapType(attrS.Type)
	}

	for _, blockS := range b.BlockTypes {
		name := blockS.TypeName
		if _, exists := atys[name]; exists {
			panic("invalid schema, blocks and attributes cannot have the same name")
		}

		childType := WrapBlock(blockS.Block).ImpliedType()

		switch blockS.Nesting {
		case tfprotov5.SchemaNestedBlockNestingModeSingle, tfprotov5.SchemaNestedBlockNestingModeGroup:
			atys[name] = childType
		case tfprotov5.SchemaNestedBlockNestingModeList:
			// We prefer to use a list where possible, since it makes our
			// implied type more complete, but if there are any
			// dynamically-typed attributes inside we must use a tuple
			// instead, which means our type _constraint_ must be
			// cty.DynamicPseudoType to allow the tuple type to be decided
			// separately for each value.
			if childType.HasDynamicTypes() {
				atys[name] = cty.DynamicPseudoType
			} else {
				atys[name] = cty.List(childType)
			}
		case tfprotov5.SchemaNestedBlockNestingModeSet:
			if childType.HasDynamicTypes() {
				panic("can't use cty.DynamicPseudoType inside a block type with tfprotov5.SchemaNestedBlockNestingModeSet")
			}
			atys[name] = cty.Set(childType)
		case tfprotov5.SchemaNestedBlockNestingModeMap:
			// We prefer to use a map where possible, since it makes our
			// implied type more complete, but if there are any
			// dynamically-typed attributes inside we must use an object
			// instead, which means our type _constraint_ must be
			// cty.DynamicPseudoType to allow the tuple type to be decided
			// separately for each value.
			if childType.HasDynamicTypes() {
				atys[name] = cty.DynamicPseudoType
			} else {
				atys[name] = cty.Map(childType)
			}
		default:
			panic("invalid nesting type")
		}
	}

	return cty.Object(atys)
}
