package configschema

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestBlockEmptyValue(t *testing.T) {
	tests := []struct {
		Schema *tfprotov5.SchemaBlock
		Want   cty.Value
	}{
		{
			&tfprotov5.SchemaBlock{},
			cty.EmptyObjectVal,
		},
		{
			&tfprotov5.SchemaBlock{
				Attributes: []*tfprotov5.SchemaAttribute{
					{
						Name:     "str",
						Type:     tftypes.String,
						Required: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"str": cty.NullVal(cty.String),
			}),
		},
		{
			&tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "single",
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeSingle,
						Block: &tfprotov5.SchemaBlock{
							Attributes: []*tfprotov5.SchemaAttribute{
								{
									Name:     "str",
									Type:     tftypes.String,
									Required: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"single": cty.NullVal(cty.Object(map[string]cty.Type{
					"str": cty.String,
				})),
			}),
		},
		{
			&tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "group",
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeGroup,
						Block: &tfprotov5.SchemaBlock{
							Attributes: []*tfprotov5.SchemaAttribute{
								{
									Name:     "str",
									Type:     tftypes.String,
									Required: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"group": cty.ObjectVal(map[string]cty.Value{
					"str": cty.NullVal(cty.String),
				}),
			}),
		},
		{
			&tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "list",
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeList,
						Block: &tfprotov5.SchemaBlock{
							Attributes: []*tfprotov5.SchemaAttribute{
								{
									Name:     "str",
									Type:     tftypes.String,
									Required: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"list": cty.ListValEmpty(cty.Object(map[string]cty.Type{
					"str": cty.String,
				})),
			}),
		},
		{
			&tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "list_dynamic",
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeList,
						Block: &tfprotov5.SchemaBlock{
							Attributes: []*tfprotov5.SchemaAttribute{
								{
									Name:     "str",
									Type:     tftypes.DynamicPseudoType,
									Required: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"list_dynamic": cty.EmptyTupleVal,
			}),
		},
		{
			&tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "map",
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeMap,
						Block: &tfprotov5.SchemaBlock{
							Attributes: []*tfprotov5.SchemaAttribute{
								{
									Name:     "str",
									Type:     tftypes.String,
									Required: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapValEmpty(cty.Object(map[string]cty.Type{
					"str": cty.String,
				})),
			}),
		},
		{
			&tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "map_dynamic",
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeMap,
						Block: &tfprotov5.SchemaBlock{
							Attributes: []*tfprotov5.SchemaAttribute{
								{
									Name:     "str",
									Type:     tftypes.DynamicPseudoType,
									Required: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"map_dynamic": cty.EmptyObjectVal,
			}),
		},
		{
			&tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "set",
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeSet,
						Block: &tfprotov5.SchemaBlock{
							Attributes: []*tfprotov5.SchemaAttribute{
								{
									Name:     "str",
									Type:     tftypes.String,
									Required: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"str": cty.String,
				})),
			}),
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%#v", test.Schema), func(t *testing.T) {
			got := WrapBlock(test.Schema).EmptyValue()
			if !test.Want.RawEquals(got) {
				t.Errorf("wrong result\nschema: %#v\ngot: %#v\nwant: %#v", test.Schema, got, test.Want)
			}

			// The empty value must always conform to the implied type of
			// the schema.
			wantTy := WrapBlock(test.Schema).ImpliedType()
			gotTy := got.Type()
			if errs := gotTy.TestConformance(wantTy); len(errs) > 0 {
				t.Errorf("empty value has incorrect type\ngot: %#v\nwant: %#v\nerrors: %#v", gotTy, wantTy, errs)
			}
		})
	}
}
