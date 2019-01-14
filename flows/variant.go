package flows

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	ipfix "github.com/CN-TU/go-ipfix"
)

type incompatibleVariantError string

func (i incompatibleVariantError) Error() string {
	return string(i)
}

// MakeIncompatibleVariantError returns a new error signifying an incompatible variant
func MakeIncompatibleVariantError(format string, a ...interface{}) error {
	return incompatibleVariantError(fmt.Sprintf(format, a...))
}

func getVariantSources(variants []maybeASTVariant) ([]astFragment, []int) {
	var sources []astFragment
	var max []int
	for _, variant := range variants {
		if variant.HasVariants() {
			sources = append(sources, variant.Source())
			max = append(max, len(variant.SubVariants()))
			subsources, submax := getVariantSources(variant.SubVariants())
			sources = append(sources, subsources...)
			max = append(max, submax...)
		}
	}
	return sources, max
}

func resolveASTVariantsRec(resolver TypeResolver, sources []astFragment, max []int, chosen map[astFragment]int, which int, args []maybeASTVariant) (maybeASTVariant, error) {
	if which == len(sources)-1 {
		ret := make([]maybeASTVariant, max[which])
		onlyIllegal := true
		for i := 0; i < max[which]; i++ {
			chosen[sources[which]] = i
			ies := make([]ipfix.InformationElement, len(args))
			for i := range ies {
				switch ie := args[i].GetSpecific(chosen).(type) {
				case *astVariant:
					panic("unresolved variant")
				case *singleVariant:
					ies[i] = ie.ie
				case *noVariant:
					continue
				default:
					panic("bad variant type")
				}
			}
			res, err := resolver(ies)
			if err != nil {
				if _, ok := err.(incompatibleVariantError); ok {
					ret[i] = &noVariant{}
					continue
				}
				return nil, err
			}
			ret[i] = toVariant(res, nil)
			onlyIllegal = false
		}
		if onlyIllegal {
			return &noVariant{}, nil
		}
		return &astVariant{
			ies:    ret,
			source: sources[which],
		}, nil
	}

	ret := make([]maybeASTVariant, max[which])
	onlyIllegal := true
	for i := 0; i < max[which]; i++ {
		chosen[sources[which]] = i
		v, err := resolveASTVariantsRec(resolver, sources, max, chosen, which+1, args)
		if err != nil {
			return nil, err
		}
		if _, ok := v.(*noVariant); !ok {
			onlyIllegal = false
		}
		ret[i] = v
	}
	if onlyIllegal {
		return &noVariant{}, nil
	}
	return &astVariant{
		ies:    ret,
		source: sources[which],
	}, nil
}

func resolveASTVariants(resolver TypeResolver, args ...maybeASTVariant) (maybeASTVariant, error) {
	sources, max := getVariantSources(args)
	if len(sources) > 0 {
		ret, err := resolveASTVariantsRec(resolver, sources, max, make(map[astFragment]int), 0, args)
		if err != nil {
			return nil, err
		}
		if _, ok := ret.(*noVariant); ok {
			return nil, errors.New("couldn't resolve types")
		}
		return ret, nil
	}

	ies := make([]ipfix.InformationElement, len(args))
	for i := range ies {
		switch ie := args[i].(type) {
		case *astVariant:
			panic("variant without source")
		case *singleVariant:
			ies[i] = ie.ie
		case *noVariant:
			return nil, fmt.Errorf("expected an argument type at pos %d", i)
		default:
			panic("bad variant type")
		}
	}
	res, err := resolver(ies)
	if err != nil {
		return nil, err
	}

	return &singleVariant{res}, nil
}

func toVariant(ie interface{}, fragment astFragment) maybeASTVariant {
	switch ie := ie.(type) {
	case []ipfix.InformationElement:
		ies := make([]maybeASTVariant, len(ie))
		for i := range ies {
			ies[i] = toVariant(ie[i], fragment)
		}
		return &astVariant{
			source: fragment,
			ies:    ies,
		}
	case ipfix.InformationElement:
		return &singleVariant{ie}
	case *singleVariant:
		return ie
	case maybeASTVariant:
		return ie
	}
	panic(fmt.Sprintf("toVariant used with unknown type %#v!", ie))
}

type maybeVariant interface {
	SpecificIE(map[int]int) ipfix.InformationElement
	IE() ipfix.InformationElement
}

type variant struct {
	source int
	ies    []maybeVariant
}

func (v *variant) SpecificIE(chooser map[int]int) ipfix.InformationElement {
	return v.ies[chooser[v.source]].SpecificIE(chooser)
}

func (v *variant) IE() ipfix.InformationElement {
	panic("IE called on variant")
}

func (v *variant) String() string {
	ies := make([]string, len(v.ies))
	for i := range ies {
		ies[i] = fmt.Sprint(v.ies[i])
	}
	return fmt.Sprintf("$%d={%s}", v.source, strings.Join(ies, ", "))
}

type maybeASTVariant interface {
	GetSpecific(map[astFragment]int) maybeASTVariant
	Source() astFragment
	SubVariants() []maybeASTVariant
	HasVariants() bool
	simplify() maybeVariant
}

type astVariant struct {
	source astFragment
	ies    []maybeASTVariant
}

func (v *astVariant) simplifyRec(sources []int, max []int, sourcesDedup map[int][]astFragment, chosen map[astFragment]int, which int) maybeVariant {
	if which == len(sources)-1 {
		var ret []maybeVariant
		for i := 0; i < max[which]; i++ {
			for _, source := range sourcesDedup[sources[which]] {
				chosen[source] = i
			}
			variant := v.GetSpecific(chosen)
			if ie, ok := variant.(*singleVariant); ok {
				ret = append(ret, &singleVariant{ie.ie})
			} else {
				return nil
			}
		}
		if len(ret) == 0 {
			return nil
		}
		if len(ret) == 1 {
			return ret[0]
		}
		return &variant{sources[which], ret}
	}

	var ret []maybeVariant
	for i := 0; i < max[which]; i++ {
		for _, source := range sourcesDedup[sources[which]] {
			chosen[source] = i
		}
		variant := v.simplifyRec(sources, max, sourcesDedup, chosen, which+1)
		if variant != nil {
			ret = append(ret, variant)
		}
	}
	if len(ret) == 0 {
		return nil
	}
	if len(ret) == 1 {
		return ret[0]
	}
	return &variant{sources[which], ret}
}

func (v *astVariant) simplify() maybeVariant {
	sourcesAST, maxAST := getVariantSources([]maybeASTVariant{v})
	sourcesDedup := make(map[int][]astFragment)
	maxDedup := make(map[int]int)
	for i := range sourcesAST {
		reg := sourcesAST[i].Register()
		sourcesDedup[reg] = append(sourcesDedup[reg], sourcesAST[i])
		maxDedup[reg] = maxAST[i]
	}
	var sources, max []int

	for k := range maxDedup {
		sources = append(sources, k)
	}
	sort.Ints(sources)
	for _, k := range sources {
		max = append(max, maxDedup[k])
	}
	return v.simplifyRec(sources, max, sourcesDedup, make(map[astFragment]int), 0)
}

func (v *astVariant) GetSpecific(chooser map[astFragment]int) maybeASTVariant {
	return v.ies[chooser[v.source]].GetSpecific(chooser)
}

func (v *astVariant) String() string {
	ies := make([]string, len(v.ies))
	for i := range ies {
		ies[i] = fmt.Sprint(v.ies[i])
	}
	return fmt.Sprintf("$%d=%s:{%s}", v.source.Register(), v.source.Name(), strings.Join(ies, ", "))
}

func (v *astVariant) Source() astFragment {
	return v.source
}

func (v *astVariant) HasVariants() bool {
	return true
}

func (v *astVariant) SubVariants() []maybeASTVariant {
	return v.ies
}

type singleVariant struct {
	ie ipfix.InformationElement
}

func (s *singleVariant) simplify() maybeVariant {
	return s
}

func (s *singleVariant) IE() ipfix.InformationElement {
	return s.ie
}

func (s *singleVariant) SpecificIE(map[int]int) ipfix.InformationElement {
	return s.ie
}

func (s *singleVariant) GetSpecific(map[astFragment]int) maybeASTVariant {
	return s
}

func (s *singleVariant) String() string {
	return fmt.Sprint(s.ie)
}

func (s *singleVariant) Source() astFragment {
	return nil
}

func (s *singleVariant) HasVariants() bool {
	return false
}

func (s *singleVariant) SubVariants() []maybeASTVariant {
	return nil
}

type noVariant struct{}

func (n *noVariant) simplify() maybeVariant {
	panic("simplify called on noVariant")
}

func (n *noVariant) String() string {
	return "<illegal>"
}

func (n *noVariant) Source() astFragment {
	return nil
}

func (n *noVariant) HasVariants() bool {
	return false
}

func (n *noVariant) SubVariants() []maybeASTVariant {
	return nil
}

func (n *noVariant) GetSpecific(map[astFragment]int) maybeASTVariant {
	return n
}
