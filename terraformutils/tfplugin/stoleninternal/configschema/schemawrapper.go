package configschema

import (
	"fmt"

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
	if b == nil {
		return nil
	}
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

type Diagnostics struct {
	diags []*tfprotov5.Diagnostic
}

func WrapDiagnostics(ds []*tfprotov5.Diagnostic) *Diagnostics {
	return &Diagnostics{ds}
}

// HasError returns true if the collection has an error severity Diagnostic.
func (d Diagnostics) HasError() bool {
	for _, diag := range d.diags {
		if diag.Severity == tfprotov5.DiagnosticSeverityError {
			return true
		}
	}
	return false
}

func (d Diagnostics) ToError() error {
	var errs []error
	for _, diag := range d.diags {
		if diag.Severity == tfprotov5.DiagnosticSeverityError {
			errs = append(errs, fmt.Errorf("error %s on %v: %s", diag.Summary, diag.Attribute, diag.Detail))
		}
	}
	if len(d.diags) == 1 {
		return fmt.Errorf("%s", errs[0])
	}
	msg := "encountered errors:"
	for _, err := range errs {
		msg = fmt.Sprintf("%s\n%s", msg, err)
	}
	return fmt.Errorf(msg)
}
