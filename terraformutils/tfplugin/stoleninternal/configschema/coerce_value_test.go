package configschema

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestCoerceValue(t *testing.T) {
	tests := map[string]struct {
		Schema    *tfprotov5.SchemaBlock
		Input     cty.Value
		WantValue cty.Value
		WantErr   string
	}{
		"empty schema and value": {
			&tfprotov5.SchemaBlock{},
			cty.EmptyObjectVal,
			cty.EmptyObjectVal,
			``,
		},
		"attribute present": {
			&tfprotov5.SchemaBlock{
				Attributes: []*tfprotov5.SchemaAttribute{
					{
						Name:     "foo",
						Type:     tftypes.String,
						Optional: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.True,
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("true"),
			}),
			``,
		},
		"single block present": {
			&tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "foo",
						Block:    &tfprotov5.SchemaBlock{},
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeSingle,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.EmptyObjectVal,
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.EmptyObjectVal,
			}),
			``,
		},
		"single block wrong type": {
			&tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "foo",
						Block:    &tfprotov5.SchemaBlock{},
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeSingle,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.True,
			}),
			cty.DynamicVal,
			`.foo: an object is required`,
		},
		"list block with one item": {
			&tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "foo",
						Block:    &tfprotov5.SchemaBlock{},
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeList,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{cty.EmptyObjectVal}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{cty.EmptyObjectVal}),
			}),
			``,
		},
		"set block with one item": {
			&tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "foo",
						Block:    &tfprotov5.SchemaBlock{},
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeSet,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{cty.EmptyObjectVal}), // can implicitly convert to set
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetVal([]cty.Value{cty.EmptyObjectVal}),
			}),
			``,
		},
		"map block with one item": {
			&tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "foo",
						Block:    &tfprotov5.SchemaBlock{},
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeMap,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.MapVal(map[string]cty.Value{"foo": cty.EmptyObjectVal}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.MapVal(map[string]cty.Value{"foo": cty.EmptyObjectVal}),
			}),
			``,
		},
		"list block with one item having an attribute": {
			&tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "foo",
						Block: &tfprotov5.SchemaBlock{
							Attributes: []*tfprotov5.SchemaAttribute{
								{
									Name:     "bar",
									Type:     tftypes.String,
									Required: true,
								},
							},
						},
						Nesting: tfprotov5.SchemaNestedBlockNestingModeMap,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"bar": cty.StringVal("hello"),
				})}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"bar": cty.StringVal("hello"),
				})}),
			}),
			``,
		},
		"list block with one item having a missing attribute": {
			&tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "foo",
						Block: &tfprotov5.SchemaBlock{
							Attributes: []*tfprotov5.SchemaAttribute{
								{
									Name:     "bar",
									Type:     tftypes.String,
									Required: true,
								},
							},
						},
						Nesting: tfprotov5.SchemaNestedBlockNestingModeMap,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{cty.EmptyObjectVal}),
			}),
			cty.DynamicVal,
			`.foo[0]: attribute "bar" is required`,
		},
		"list block with one item having an extraneous attribute": {
			&tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "foo",
						Block:    &tfprotov5.SchemaBlock{},
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeList,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"bar": cty.StringVal("hello"),
				})}),
			}),
			cty.DynamicVal,
			`.foo[0]: unexpected attribute "bar"`,
		},
		"missing optional attribute": {
			&tfprotov5.SchemaBlock{
				Attributes: []*tfprotov5.SchemaAttribute{
					{
						Name:     "foo",
						Type:     tftypes.String,
						Optional: true,
					},
				},
			},
			cty.EmptyObjectVal,
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.String),
			}),
			``,
		},
		"missing optional single block": {
			&tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "foo",
						Block:    &tfprotov5.SchemaBlock{},
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeSingle,
					},
				},
			},
			cty.EmptyObjectVal,
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.EmptyObject),
			}),
			``,
		},
		"missing optional list block": {
			&tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "foo",
						Block:    &tfprotov5.SchemaBlock{},
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeList,
					},
				},
			},
			cty.EmptyObjectVal,
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListValEmpty(cty.EmptyObject),
			}),
			``,
		},
		"missing optional set block": {
			&tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "foo",
						Block:    &tfprotov5.SchemaBlock{},
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeSet,
					},
				},
			},
			cty.EmptyObjectVal,
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.SetValEmpty(cty.EmptyObject),
			}),
			``,
		},
		"missing optional map block": {
			&tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "foo",
						Block:    &tfprotov5.SchemaBlock{},
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeMap,
					},
				},
			},
			cty.EmptyObjectVal,
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.MapValEmpty(cty.EmptyObject),
			}),
			``,
		},
		"missing required attribute": {
			&tfprotov5.SchemaBlock{
				Attributes: []*tfprotov5.SchemaAttribute{
					{
						Name:     "foo",
						Type:     tftypes.String,
						Required: true,
					},
				},
			},
			cty.EmptyObjectVal,
			cty.DynamicVal,
			`attribute "foo" is required`,
		},
		"missing required single block": {
			&tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "foo",
						Block:    &tfprotov5.SchemaBlock{},
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeSingle,
						MinItems: 1,
						MaxItems: 1,
					},
				},
			},
			cty.EmptyObjectVal,
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.EmptyObject),
			}),
			``,
		},
		"unknown nested list": {
			&tfprotov5.SchemaBlock{
				Attributes: []*tfprotov5.SchemaAttribute{
					{
						Name:     "attr",
						Type:     tftypes.String,
						Required: true,
					},
				},
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "foo",
						Block:    &tfprotov5.SchemaBlock{},
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeList,
						MinItems: 2,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"attr": cty.StringVal("test"),
				"foo":  cty.UnknownVal(cty.EmptyObject),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"attr": cty.StringVal("test"),
				"foo":  cty.UnknownVal(cty.List(cty.EmptyObject)),
			}),
			"",
		},
		"unknowns in nested list": {
			&tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "foo",
						Block: &tfprotov5.SchemaBlock{
							Attributes: []*tfprotov5.SchemaAttribute{
								{
									Name:     "attr",
									Type:     tftypes.String,
									Required: true,
								},
							},
						},
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeList,
						MinItems: 2,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"attr": cty.UnknownVal(cty.String),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"attr": cty.UnknownVal(cty.String),
					}),
				}),
			}),
			"",
		},
		"unknown nested set": {
			&tfprotov5.SchemaBlock{
				Attributes: []*tfprotov5.SchemaAttribute{
					{
						Name:     "attr",
						Type:     tftypes.String,
						Required: true,
					},
				},
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "foo",
						Block:    &tfprotov5.SchemaBlock{},
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeSet,
						MinItems: 1,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"attr": cty.StringVal("test"),
				"foo":  cty.UnknownVal(cty.EmptyObject),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"attr": cty.StringVal("test"),
				"foo":  cty.UnknownVal(cty.Set(cty.EmptyObject)),
			}),
			"",
		},
		"unknown nested map": {
			&tfprotov5.SchemaBlock{
				Attributes: []*tfprotov5.SchemaAttribute{
					{
						Name:     "attr",
						Type:     tftypes.String,
						Required: true,
					},
				},
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "foo",
						Block:    &tfprotov5.SchemaBlock{},
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeMap,
						MinItems: 1,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"attr": cty.StringVal("test"),
				"foo":  cty.UnknownVal(cty.Map(cty.String)),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"attr": cty.StringVal("test"),
				"foo":  cty.UnknownVal(cty.Map(cty.EmptyObject)),
			}),
			"",
		},
		"extraneous attribute": {
			&tfprotov5.SchemaBlock{},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("bar"),
			}),
			cty.DynamicVal,
			`unexpected attribute "foo"`,
		},
		"wrong attribute type": {
			&tfprotov5.SchemaBlock{
				Attributes: []*tfprotov5.SchemaAttribute{
					{
						Name:     "foo",
						Type:     tftypes.Number,
						Required: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.False,
			}),
			cty.DynamicVal,
			`.foo: number required`,
		},
		"unset computed value": {
			&tfprotov5.SchemaBlock{
				Attributes: []*tfprotov5.SchemaAttribute{
					{
						Name:     "foo",
						Type:     tftypes.String,
						Optional: true,
						Computed: true,
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.String),
			}),
			``,
		},
		"dynamic value attributes": {
			&tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "foo",
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeMap,
						Block: &tfprotov5.SchemaBlock{
							Attributes: []*tfprotov5.SchemaAttribute{
								{
									Name:     "bar",
									Type:     tftypes.String,
									Optional: true,
									Computed: true,
								},
								{
									Name:     "baz",
									Type:     tftypes.DynamicPseudoType,
									Optional: true,
									Computed: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ObjectVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("beep"),
					}),
					"b": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("boop"),
						"baz": cty.NumberIntVal(8),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ObjectVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("beep"),
						"baz": cty.NullVal(cty.DynamicPseudoType),
					}),
					"b": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("boop"),
						"baz": cty.NumberIntVal(8),
					}),
				}),
			}),
			``,
		},
		"dynamic attributes in map": {
			// Convert a block represented as a map to an object if a
			// DynamicPseudoType causes the element types to mismatch.
			&tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "foo",
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeMap,
						Block: &tfprotov5.SchemaBlock{
							Attributes: []*tfprotov5.SchemaAttribute{
								{
									Name:     "bar",
									Type:     tftypes.String,
									Optional: true,
									Computed: true,
								},
								{
									Name:     "baz",
									Type:     tftypes.DynamicPseudoType,
									Optional: true,
									Computed: true,
								},
							},
						},
					},
				},
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("beep"),
					}),
					"b": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("boop"),
					}),
				}),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ObjectVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("beep"),
						"baz": cty.NullVal(cty.DynamicPseudoType),
					}),
					"b": cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("boop"),
						"baz": cty.NullVal(cty.DynamicPseudoType),
					}),
				}),
			}),
			``,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gotValue, gotErrObj := WrapBlock(test.Schema).CoerceValue(test.Input)

			if gotErrObj == nil {
				if test.WantErr != "" {
					t.Fatalf("coersion succeeded; want error: %q", test.WantErr)
				}
			} else {
				gotErr := tfdiagsFormatError(gotErrObj)
				if gotErr != test.WantErr {
					t.Fatalf("wrong error\ngot:  %#v\nwant: %s", gotErrObj, test.WantErr)
				}
				return
			}

			if !gotValue.RawEquals(test.WantValue) {
				t.Errorf("wrong result\ninput: %#v\ngot:   %#v\nwant:  %#v", test.Input, gotValue, test.WantValue)
			}

			// The coerced value must always conform to the implied type of
			// the schema.
			wantTy := WrapBlock(test.Schema).ImpliedType()
			gotTy := gotValue.Type()
			if errs := gotTy.TestConformance(wantTy); len(errs) > 0 {
				t.Errorf("empty value has incorrect type\ngot: %#v\nwant: %#v\nerrors: %#v", gotTy, wantTy, errs)
			}
		})
	}
}

// FormatError is a helper function to produce a user-friendly string
// representation of certain special error types that we might want to
// include in diagnostic messages.
//
// This currently has special behavior only for cty.PathError, where a
// non-empty path is rendered in a HCL-like syntax as context.
func tfdiagsFormatError(err error) string {
	perr, ok := err.(cty.PathError)
	if !ok || len(perr.Path) == 0 {
		return err.Error()
	}

	return fmt.Sprintf("%s: %s", tfdiagsFormatCtyPath(perr.Path), perr.Error())
}

// FormatCtyPath is a helper function to produce a user-friendly string
// representation of a cty.Path. The result uses a syntax similar to the
// HCL expression language in the hope of it being familiar to users.
func tfdiagsFormatCtyPath(path cty.Path) string {
	var buf bytes.Buffer
	for _, step := range path {
		switch ts := step.(type) {
		case cty.GetAttrStep:
			fmt.Fprintf(&buf, ".%s", ts.Name)
		case cty.IndexStep:
			buf.WriteByte('[')
			key := ts.Key
			keyTy := key.Type()
			switch {
			case key.IsNull():
				buf.WriteString("null")
			case !key.IsKnown():
				buf.WriteString("(not yet known)")
			case keyTy == cty.Number:
				bf := key.AsBigFloat()
				buf.WriteString(bf.Text('g', -1))
			case keyTy == cty.String:
				buf.WriteString(strconv.Quote(key.AsString()))
			default:
				buf.WriteString("...")
			}
			buf.WriteByte(']')
		}
	}
	return buf.String()
}
