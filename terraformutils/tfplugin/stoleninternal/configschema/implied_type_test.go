package configschema

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestBlockImpliedType(t *testing.T) {
	tests := map[string]struct {
		Schema *tfprotov5.SchemaBlock
		Want   cty.Type
	}{
		"nil": {
			nil,
			cty.EmptyObject,
		},
		"empty": {
			&tfprotov5.SchemaBlock{},
			cty.EmptyObject,
		},
		"attributes": {
			&tfprotov5.SchemaBlock{
				Attributes: []*tfprotov5.SchemaAttribute{
					{
						Name:     "optional",
						Type:     tftypes.String,
						Optional: true,
					},
					{
						Name:     "required",
						Type:     tftypes.Number,
						Required: true,
					},
					{
						Name:     "computed",
						Type:     tftypes.List{ElementType: tftypes.Bool},
						Computed: true,
					},
					{
						Name:     "optional_computed",
						Type:     tftypes.Map{ElementType: tftypes.Bool},
						Optional: true,
					},
				},
			},
			cty.Object(map[string]cty.Type{
				"optional":          cty.String,
				"required":          cty.Number,
				"computed":          cty.List(cty.Bool),
				"optional_computed": cty.Map(cty.Bool),
			}),
		},
		"blocks": {
			&tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "single",
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeSingle,
						Block: &tfprotov5.SchemaBlock{
							Attributes: []*tfprotov5.SchemaAttribute{
								{
									Name:     "foo",
									Type:     tftypes.DynamicPseudoType,
									Required: true,
								},
							},
						},
					},
					{
						TypeName: "list",
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeList,
					},
					{
						TypeName: "set",
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeSet,
					},
					{
						TypeName: "map",
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeMap,
					},
				},
			},
			cty.Object(map[string]cty.Type{
				"single": cty.Object(map[string]cty.Type{
					"foo": cty.DynamicPseudoType,
				}),
				"list": cty.List(cty.EmptyObject),
				"set":  cty.Set(cty.EmptyObject),
				"map":  cty.Map(cty.EmptyObject),
			}),
		},
		"deep block nesting": {
			&tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "single",
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeSingle,
						Block: &tfprotov5.SchemaBlock{
							BlockTypes: []*tfprotov5.SchemaNestedBlock{
								{
									TypeName: "list",
									Nesting:  tfprotov5.SchemaNestedBlockNestingModeList,
									Block: &tfprotov5.SchemaBlock{
										BlockTypes: []*tfprotov5.SchemaNestedBlock{
											{
												TypeName: "set",
												Nesting:  tfprotov5.SchemaNestedBlockNestingModeSet,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			cty.Object(map[string]cty.Type{
				"single": cty.Object(map[string]cty.Type{
					"list": cty.List(cty.Object(map[string]cty.Type{
						"set": cty.Set(cty.EmptyObject),
					})),
				}),
			}),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := WrapBlock(test.Schema).ImpliedType()
			if !got.Equals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}
