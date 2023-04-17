package configschema

import (
	"fmt"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/go-cty/cty/convert"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
)

// CoerceValue attempts to force the given value to conform to the type
// implied by the receiever.
//
// This is useful in situations where a configuration must be derived from
// an already-decoded value. It is always better to decode directly from
// configuration where possible since then source location information is
// still available to produce diagnostics, but in special situations this
// function allows a compatible result to be obtained even if the
// configuration objects are not available.
//
// If the given value cannot be converted to conform to the receiving schema
// then an error is returned describing one of possibly many problems. This
// error may be a cty.PathError indicating a position within the nested
// data structure where the problem applies.
func (b *Block) CoerceValue(in cty.Value) (cty.Value, error) {
	var path cty.Path
	return b.coerceValue(in, path)
}

func (b *Block) coerceValue(in cty.Value, path cty.Path) (cty.Value, error) {
	switch {
	case in.IsNull():
		return cty.NullVal(b.ImpliedType()), nil
	case !in.IsKnown():
		return cty.UnknownVal(b.ImpliedType()), nil
	}

	ty := in.Type()
	if !ty.IsObjectType() {
		return cty.UnknownVal(b.ImpliedType()), path.NewErrorf("an object is required")
	}

	for name := range ty.AttributeTypes() {
		definedA := false
		for _, attrS := range b.Attributes {
			if attrS.Name == name {
				definedA = true
			}
		}
		if definedA {
			continue
		}
		definedB := false
		for _, blockS := range b.BlockTypes {
			if blockS.TypeName == name {
				definedB = true
			}
		}
		if definedB {
			continue
		}
		return cty.UnknownVal(b.ImpliedType()), path.NewErrorf("unexpected attribute %q", name)
	}

	attrs := make(map[string]cty.Value)

	for _, attrS := range b.Attributes {
		name := attrS.Name
		var val cty.Value
		switch {
		case ty.HasAttribute(name):
			val = in.GetAttr(name)
		case attrS.Computed || attrS.Optional:
			val = cty.NullVal(WrapType(attrS.Type))
		default:
			return cty.UnknownVal(b.ImpliedType()), path.NewErrorf("attribute %q is required", name)
		}

		val, err := WrapAttribute(attrS).coerceValue(val, append(path, cty.GetAttrStep{Name: name}))
		if err != nil {
			return cty.UnknownVal(b.ImpliedType()), err
		}

		attrs[name] = val
	}
	for _, blockS := range b.BlockTypes {
		typeName := blockS.TypeName
		impliedType := WrapNestedBlock(blockS).ImpliedType()
		switch blockS.Nesting {

		case tfprotov5.SchemaNestedBlockNestingModeSingle, tfprotov5.SchemaNestedBlockNestingModeGroup:
			switch {
			case ty.HasAttribute(typeName):
				var err error
				val := in.GetAttr(typeName)
				attrs[typeName], err = WrapNestedBlock(blockS).coerceValue(val, append(path, cty.GetAttrStep{Name: typeName}))
				if err != nil {
					return cty.UnknownVal(b.ImpliedType()), err
				}
			default:
				attrs[typeName] = WrapNestedBlock(blockS).EmptyValue()
			}

		case tfprotov5.SchemaNestedBlockNestingModeList:
			switch {
			case ty.HasAttribute(typeName):
				coll := in.GetAttr(typeName)

				switch {
				case coll.IsNull():
					attrs[typeName] = cty.NullVal(cty.List(impliedType))
					continue
				case !coll.IsKnown():
					attrs[typeName] = cty.UnknownVal(cty.List(impliedType))
					continue
				}

				if !coll.CanIterateElements() {
					return cty.UnknownVal(b.ImpliedType()), path.NewErrorf("must be a list")
				}
				l := coll.LengthInt()

				if l == 0 {
					attrs[typeName] = cty.ListValEmpty(impliedType)
					continue
				}
				elems := make([]cty.Value, 0, l)
				{
					path = append(path, cty.GetAttrStep{Name: typeName})
					for it := coll.ElementIterator(); it.Next(); {
						var err error
						idx, val := it.Element()
						val, err = WrapNestedBlock(blockS).coerceValue(val, append(path, cty.IndexStep{Key: idx}))
						if err != nil {
							return cty.UnknownVal(b.ImpliedType()), err
						}
						elems = append(elems, val)
					}
				}
				attrs[typeName] = cty.ListVal(elems)
			default:
				attrs[typeName] = cty.ListValEmpty(impliedType)
			}

		case tfprotov5.SchemaNestedBlockNestingModeSet:
			switch {
			case ty.HasAttribute(typeName):
				coll := in.GetAttr(typeName)

				switch {
				case coll.IsNull():
					attrs[typeName] = cty.NullVal(cty.Set(impliedType))
					continue
				case !coll.IsKnown():
					attrs[typeName] = cty.UnknownVal(cty.Set(impliedType))
					continue
				}

				if !coll.CanIterateElements() {
					return cty.UnknownVal(b.ImpliedType()), path.NewErrorf("must be a set")
				}
				l := coll.LengthInt()

				if l == 0 {
					attrs[typeName] = cty.SetValEmpty(impliedType)
					continue
				}
				elems := make([]cty.Value, 0, l)
				{
					path = append(path, cty.GetAttrStep{Name: typeName})
					for it := coll.ElementIterator(); it.Next(); {
						var err error
						idx, val := it.Element()
						val, err = WrapNestedBlock(blockS).coerceValue(val, append(path, cty.IndexStep{Key: idx}))
						if err != nil {
							return cty.UnknownVal(b.ImpliedType()), err
						}
						elems = append(elems, val)
					}
				}
				attrs[typeName] = cty.SetVal(elems)
			default:
				attrs[typeName] = cty.SetValEmpty(impliedType)
			}

		case tfprotov5.SchemaNestedBlockNestingModeMap:
			switch {
			case ty.HasAttribute(typeName):
				coll := in.GetAttr(typeName)

				switch {
				case coll.IsNull():
					attrs[typeName] = cty.NullVal(cty.Map(impliedType))
					continue
				case !coll.IsKnown():
					attrs[typeName] = cty.UnknownVal(cty.Map(impliedType))
					continue
				}

				if !coll.CanIterateElements() {
					return cty.UnknownVal(b.ImpliedType()), path.NewErrorf("must be a map")
				}
				l := coll.LengthInt()
				if l == 0 {
					attrs[typeName] = cty.MapValEmpty(impliedType)
					continue
				}
				elems := make(map[string]cty.Value)
				{
					path = append(path, cty.GetAttrStep{Name: typeName})
					for it := coll.ElementIterator(); it.Next(); {
						var err error
						key, val := it.Element()
						if key.Type() != cty.String || key.IsNull() || !key.IsKnown() {
							return cty.UnknownVal(b.ImpliedType()), path.NewErrorf("must be a map")
						}
						val, err = WrapNestedBlock(blockS).coerceValue(val, append(path, cty.IndexStep{Key: key}))
						if err != nil {
							return cty.UnknownVal(b.ImpliedType()), err
						}
						elems[key.AsString()] = val
					}
				}

				// If the attribute values here contain any DynamicPseudoTypes,
				// the concrete type must be an object.
				useObject := false
				switch {
				case coll.Type().IsObjectType():
					useObject = true
				default:
					// It's possible that we were given a map, and need to coerce it to an object
					ety := coll.Type().ElementType()
					for _, v := range elems {
						if !v.Type().Equals(ety) {
							useObject = true
							break
						}
					}
				}

				if useObject {
					attrs[typeName] = cty.ObjectVal(elems)
				} else {
					attrs[typeName] = cty.MapVal(elems)
				}
			default:
				attrs[typeName] = cty.MapValEmpty(impliedType)
			}

		default:
			// should never happen because above is exhaustive
			panic(fmt.Errorf("unsupported nesting mode %#v", blockS.Nesting))
		}
	}

	return cty.ObjectVal(attrs), nil
}

func (a *Attribute) coerceValue(in cty.Value, path cty.Path) (cty.Value, error) {
	val, err := convert.Convert(in, WrapType(a.Type))
	if err != nil {
		return cty.UnknownVal(WrapType(a.Type)), path.NewError(err)
	}
	return val, nil
}
