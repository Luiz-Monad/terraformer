package configschema

import (
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
)

// EmptyValue returns the "empty value" for the recieving block, which for
// a block type is a non-null object where all of the attribute values are
// the empty values of the block's attributes and nested block types.
//
// In other words, it returns the value that would be returned if an empty
// block were decoded against the recieving schema, assuming that no required
// attribute or block constraints were honored.
func (b *Block) EmptyValue() cty.Value {
	vals := make(map[string]cty.Value)
	for _, attrS := range b.Attributes {
		name := attrS.Name
		vals[name] = WrapAttribute(attrS).EmptyValue()
	}
	for _, blockS := range b.BlockTypes {
		name := blockS.TypeName
		vals[name] = WrapNestedBlock(blockS).EmptyValue()
	}
	return cty.ObjectVal(vals)
}

// EmptyValue returns the "empty value" for the receiving attribute, which is
// the value that would be returned if there were no definition of the attribute
// at all, ignoring any required constraint.
func (a *Attribute) EmptyValue() cty.Value {
	return cty.NullVal(WrapType(a.Type))
}

// EmptyValue returns the "empty value" for when there are zero nested blocks
// present of the receiving type.
func (b *NestedBlock) EmptyValue() cty.Value {
	impliedType := WrapBlock(b.Block).ImpliedType()
	switch b.Nesting {
	case tfprotov5.SchemaNestedBlockNestingModeSingle:
		return cty.NullVal(impliedType)
	case tfprotov5.SchemaNestedBlockNestingModeGroup:
		return WrapBlock(b.Block).EmptyValue()
	case tfprotov5.SchemaNestedBlockNestingModeList:
		if ty := impliedType; ty.HasDynamicTypes() {
			return cty.EmptyTupleVal
		} else {
			return cty.ListValEmpty(ty)
		}
	case tfprotov5.SchemaNestedBlockNestingModeMap:
		if ty := impliedType; ty.HasDynamicTypes() {
			return cty.EmptyObjectVal
		} else {
			return cty.MapValEmpty(ty)
		}
	case tfprotov5.SchemaNestedBlockNestingModeSet:
		return cty.SetValEmpty(impliedType)
	default:
		// Should never get here because the above is intended to be exhaustive,
		// but we'll be robust and return a result nonetheless.
		return cty.NullVal(cty.DynamicPseudoType)
	}
}
