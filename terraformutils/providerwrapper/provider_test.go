package providerwrapper //nolint

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestIgnoredAttributes(t *testing.T) {
	attributes := []*tfprotov5.SchemaAttribute{
		{
			Name:     "computed_attribute",
			Type:     tftypes.Number,
			Computed: true,
		},
		{
			Name:     "required_attribute",
			Type:     tftypes.String,
			Required: true,
		},
	}

	testCases := map[string]struct {
		block                []*tfprotov5.SchemaNestedBlock
		ignoredAttributes    []string
		notIgnoredAttributes []string
	}{
		"nesting_set": {[]*tfprotov5.SchemaNestedBlock{
			{
				TypeName: "attribute_one",
				Block: &tfprotov5.SchemaBlock{
					Attributes: attributes,
				},
				Nesting: tfprotov5.SchemaNestedBlockNestingModeSet,
			},
		}, []string{"nesting_set.attribute_one.computed_attribute"},
			[]string{"nesting_set.attribute_one.required_attribute"}},
		"nesting_list": {[]*tfprotov5.SchemaNestedBlock{
			{
				TypeName: "attribute_one",
				Block: &tfprotov5.SchemaBlock{
					Attributes: []*tfprotov5.SchemaAttribute{},
					BlockTypes: []*tfprotov5.SchemaNestedBlock{
						{
							TypeName: "attribute_two_nested",
							Nesting:  tfprotov5.SchemaNestedBlockNestingModeList,
							Block: &tfprotov5.SchemaBlock{
								Attributes: attributes,
							},
						},
					},
				},
				Nesting: tfprotov5.SchemaNestedBlockNestingModeList,
			},
		}, []string{"nesting_list.0.attribute_one.0.attribute_two_nested.computed_attribute"},
			[]string{"nesting_list.0.attribute_one.0.attribute_two_nested.required_attribute"}},
	}

	for key, tc := range testCases {
		t.Run(key, func(t *testing.T) {
			provider := ProviderWrapper{}
			readOnlyAttributes := provider.readObjBlocks(tc.block, []string{}, key)
			for _, attr := range tc.ignoredAttributes {
				if ignored := isAttributeIgnored(attr, readOnlyAttributes); !ignored {
					t.Errorf("attribute \"%s\" was not ignored. Pattern list: %s", attr, readOnlyAttributes)
				}
			}

			for _, attr := range tc.notIgnoredAttributes {
				if ignored := isAttributeIgnored(attr, readOnlyAttributes); ignored {
					t.Errorf("attribute \"%s\" was ignored. Pattern list: %s", attr, readOnlyAttributes)
				}
			}
		})
	}
}

func isAttributeIgnored(name string, patterns []string) bool {
	ignored := false
	for _, pattern := range patterns {
		if match, _ := regexp.MatchString(pattern, name); match {
			ignored = true
			break
		}
	}
	return ignored
}
