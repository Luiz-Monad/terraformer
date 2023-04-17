package configschema

import (
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func UnwrapType(t cty.Type) tftypes.Type {
	b, err := t.MarshalJSON()
	if err != nil {
		return tftypes.Object{}
	}
	tftype, err := tftypes.ParseJSONType(b)
	if err != nil {
		return tftypes.Object{}
	}
	return tftype
}

func WrapType(t tftypes.Type) cty.Type {
	b, err := t.MarshalJSON()
	if err != nil {
		return cty.NilType
	}
	var ctype cty.Type
	err = ctype.UnmarshalJSON(b)
	if err != nil {
		return cty.NilType
	}
	return ctype
}

type Block struct {
	tfprotov5.SchemaBlock
}

func WrapBlock(b *tfprotov5.SchemaBlock) *Block {
	return &Block{*b}
}

type Attribute struct {
	tfprotov5.SchemaAttribute
}

func WrapAttribute(a *tfprotov5.SchemaAttribute) *Attribute {
	return &Attribute{*a}
}

type NestedBlock struct {
	tfprotov5.SchemaNestedBlock
}

func WrapNestedBlock(b *tfprotov5.SchemaNestedBlock) *Block {
	return &Block{*b.Block}
}
