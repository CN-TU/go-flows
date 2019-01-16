package flows

import (
	"errors"
	"fmt"
	"log"
	"strings"

	ipfix "github.com/CN-TU/go-ipfix"
)

// FeatureError gets returned from the feature parser and specifies the error message and includes information about feature number
type FeatureError struct {
	id   int
	name string
	err  error
}

// ID returns the feature id that caused the error
func (f FeatureError) ID() int {
	return f.id
}

func (f FeatureError) Error() string {
	return fmt.Sprintf("Feature #%d \"%s\": %s", f.id, f.name, f.err)
}

type astFragment interface {
	fmt.Stringer
	Name() string
	MakeExportName() string
	ExportName() string
	Composite() string
	SetComposite(string)
	SetExport(string)
	Export() bool
	IsRaw() bool
	ID() int
	Control() bool
	SetControl(bool)
	Variants() maybeASTVariant
	SetVariants(maybeASTVariant)
	Type() maybeVariant
	SetType(maybeVariant)
	Register() int
	SetRegister(int)
	Copy() astFragment
	FeatureMaker() featureMaker
	Returns() FeatureType
	Arguments() []astFragment
	Data() interface{}
	build(ret FeatureType) error
	assign(b astFragment)
	resolve() error
}

func makeASTFragment(feature interface{}, input FeatureType, id int) (astFragment, error) {
	switch f := feature.(type) {
	case astFragment:
		return f, nil
	case string:
		return makeASTFeature(f, input, id)
	case []interface{}:
		name, ok := f[0].(string)
		if !ok {
			return nil, fmt.Errorf("calls must be of type string, but '%s' is '%T'", f[0], f[0])
		}
		return makeASTCall(name, f[1:], input, id)
	default:
		return makeASTConstant(f, id)
	}
}

type astBase struct {
	id         int
	name       string
	exportName string
	composite  string
	export     bool
	feature    featureMaker
	ret        FeatureType
	variants   maybeASTVariant
	t          maybeVariant
	control    bool
	resolved   bool
	register   int
}

func (a *astBase) SetComposite(c string) {
	a.composite = c
}

func (a *astBase) Composite() string {
	return a.composite
}

func (a *astBase) Control() bool {
	return a.control
}

func (a *astBase) SetControl(v bool) {
	a.control = v
}

func (a *astBase) Arguments() []astFragment {
	return nil
}

func (a *astBase) Returns() FeatureType {
	return a.ret
}

func (a *astBase) Data() interface{} {
	return nil
}

func (a *astBase) FeatureMaker() featureMaker {
	return a.feature
}

func (a *astBase) Register() int {
	return a.register
}

func (a *astBase) SetRegister(r int) {
	a.register = r
}

func (a *astBase) SetVariants(v maybeASTVariant) {
	a.resolved = true
	a.variants = v
}

func (a *astBase) Variants() maybeASTVariant {
	return a.variants
}

func (a *astBase) SetType(v maybeVariant) {
	a.t = v
}

func (a *astBase) Type() maybeVariant {
	return a.t
}

func (a *astBase) ID() int {
	return a.id
}

func (a *astBase) assign(b astFragment) {
	if b.Export() {
		a.SetExport(b.ExportName())
	}
}

func (a *astBase) ExportName() string {
	return a.exportName
}

func (a *astBase) IsRaw() bool {
	return false
}

func (a *astBase) SetExport(name string) {
	a.export = true
	a.exportName = name
}

func (a *astBase) Export() bool {
	return a.export
}

func (a *astBase) String() string {
	return a.name
}

func (a *astBase) Name() string {
	return a.name
}

type astEmpty struct{}

func (a *astEmpty) SetComposite(string) {
	panic("SetComposite called on astEmpty")
}

func (a *astEmpty) Composite() string {
	panic("Composite called on astEmpty")
}

func (a *astEmpty) Control() bool {
	panic("Control called on astEmpty")
}

func (a *astEmpty) SetControl(v bool) {
	panic("SetControl called on astEmpty")
}

func (a *astEmpty) Arguments() []astFragment {
	panic("Arguments called on astEmpty")
}

func (a *astEmpty) Data() interface{} {
	return nil
}

func (a *astEmpty) FeatureMaker() featureMaker {
	panic("FeatureMaker called on astEmpty")
}

func (a *astEmpty) Register() int {
	panic("Register called on astEmpty")
}

func (a *astEmpty) SetRegister(int) {
	panic("SetRegister called on astEmpty")
}

func (a *astEmpty) assign(astFragment) {
	panic("assign called on astEmpty")
}

func (a *astEmpty) SetVariants(v maybeASTVariant) {
	panic("SetVariants called on astEmpty")
}

func (a *astEmpty) Variants() maybeASTVariant {
	panic("Variants called on astEmpty")
}

func (a *astEmpty) SetType(v maybeVariant) {
	panic("SetType called on astEmpty")
}

func (a *astEmpty) Type() maybeVariant {
	panic("Type called on astEmpty")
}

func (a *astEmpty) ID() int {
	return 0
}

func (a *astEmpty) SetExport(string) {
	panic("SetExport called on astEmpty")
}

func (a *astEmpty) Export() bool {
	return false
}

func (a *astEmpty) resolve() error {
	return nil
}

type astRaw struct {
	astEmpty
}

func (a *astRaw) IsRaw() bool {
	return true
}

type astRawPacket struct {
	astRaw
}

func (a *astRawPacket) Returns() FeatureType {
	return RawPacket
}

func (a *astRawPacket) Name() string {
	return "RawPacket"
}

func (a *astRawPacket) ExportName() string {
	return "RawPacket"
}

func (a *astRawPacket) MakeExportName() string {
	return "RawPacket"
}

func (a *astRawPacket) String() string {
	return "RawPacket"
}

func (a *astRawPacket) Copy() astFragment {
	return &astRawPacket{}
}

var errInput = errors.New("wrong input type")

func (a *astRawPacket) build(ret FeatureType) error {
	if ret != RawPacket {
		return errInput
	}
	return nil
}

func makeASTRaw(input FeatureType) (astFragment, error) {
	switch input {
	case RawPacket:
		return &astRawPacket{}, nil
	default:
		return nil, fmt.Errorf("%s input not implemented", input)
	}
}

type astRegister struct {
	astEmpty
	register int
	ret      FeatureType
}

func (a *astRegister) Register() int {
	return a.register
}

func (a *astRegister) IsRaw() bool {
	return false
}

func (a *astRegister) Returns() FeatureType {
	return a.ret
}

func (a *astRegister) Name() string {
	return "astRegister"
}

func (a *astRegister) ExportName() string {
	return "astRegister"
}

func (a *astRegister) MakeExportName() string {
	return "astRegister"
}

func (a *astRegister) String() string {
	return fmt.Sprintf("$%d", a.register)
}

func (a *astRegister) build(ret FeatureType) error {
	panic("build called on astRegister")
}

func (a *astRegister) Copy() astFragment {
	return &astRegister{register: a.register}
}

func makeASTRegister(register int, ret FeatureType) astFragment {
	return &astRegister{register: register, ret: ret}
}

type astConstant struct {
	astBase
	value interface{}
}

func (a *astConstant) Data() interface{} {
	return a.value
}

func (a *astConstant) String() string {
	return fmt.Sprintf("<Const>%v", a.value)
}

func (a *astConstant) build(ret FeatureType) (err error) {
	a.feature, err = newConstantMetaFeature(a.value)
	return
}

func (a *astConstant) MakeExportName() string {
	return a.name
}

func (a *astConstant) resolve() error {
	a.variants = toVariant(a.feature.ie, a)
	a.resolved = true
	return nil
}

func (a *astConstant) Copy() astFragment {
	return &astConstant{
		astBase: a.astBase,
		value:   a.value,
	}
}

func makeASTConstant(value interface{}, id int) (astFragment, error) {
	switch value.(type) {
	case bool, float64, int64, uint64, int:
	default:
		return nil, fmt.Errorf("can't use type %T of value '%s' as feature", value, value)
	}
	return &astConstant{
		astBase: astBase{
			id:   id,
			name: fmt.Sprint(value),
		},
		value: value,
	}, nil
}

type astCall struct {
	astBase
	args []astFragment
}

func (a *astCall) Arguments() []astFragment {
	return a.args
}

func (a *astCall) Copy() astFragment {
	r := &astCall{
		astBase: a.astBase,
		args:    make([]astFragment, len(a.args)),
	}
	for i := range r.args {
		r.args[i] = a.args[i].Copy()
	}
	return r
}

func makeASTCall(name string, args []interface{}, input FeatureType, id int) (astFragment, error) {
	r := &astCall{
		astBase: astBase{
			id:   id,
			name: name,
		},
	}

	r.args = make([]astFragment, len(args))
	var err error
	for i := range args {
		r.args[i], err = makeASTFragment(args[i], input, id)
		if err != nil {
			return nil, err
		}
	}
	return r, nil
}

type multiError []error

func (me multiError) Error() string {
	var ret []string
	for _, e := range me {
		ret = append(ret, e.Error())
	}
	return strings.Join(ret, " or ")
}

func (a *astCall) build(ret FeatureType) error {
	if a.control {
		candidates := getFeatures(a.name, ControlFeature, 1)
		if len(candidates) == 0 {
			return fmt.Errorf("couldn't find control feature '%s'", a.name)
		}
		a.feature = candidates[0]
		a.ret = ControlFeature
		return nil
	}
	candidates := getFeatures(a.name, ret, len(a.args))
	if len(candidates) == 0 {
		return fmt.Errorf("couldn't find feature '%s' returning %s with %d argument(s)", a.name, ret, len(a.args))
	}
	var errs multiError
	var err error
CANDIDATES:
	for _, candidate := range candidates {
		argtypes := make([]FeatureType, len(a.args))
		for i := range candidate.arguments {
			argtypes[i] = candidate.arguments[i]
			if argtypes[i] == MatchType {
				argtypes[i] = ret
			}
		}
		if argtypes[len(candidate.arguments)-1] == Ellipsis {
			a := argtypes[len(candidate.arguments)-2]
			for i := len(candidate.arguments) - 1; i < len(argtypes); i++ {
				argtypes[i] = a
			}
		}
		for i := range a.args {
			err = a.args[i].build(argtypes[i])
			if err != nil {
				errs = append(errs, err)
				continue CANDIDATES
			}
		}
		a.feature = candidate
		a.ret = ret
		for i := range a.args {
			err := a.args[i].build(argtypes[i])
			if err != nil {
				return err
			}
		}
		return nil
	}
	return errs
}

func (a *astCall) MakeExportName() string {
	var args []string
	for _, arg := range a.args {
		if !arg.IsRaw() {
			args = append(args, arg.MakeExportName())
		}
	}
	if len(args) == 0 {
		return a.name
	}
	return fmt.Sprintf("%s(%s)", a.name, strings.Join(args, ","))
}

func (a *astCall) resolve() error {
	// already resolved (e.g. was a composite feature)
	if a.resolved {
		return nil
	}

	// resolve arguments
	for _, arg := range a.args {
		err := arg.resolve()
		if err != nil {
			return err
		}
	}

	// selections don't have types (=return Raw*)
	if a.ret == Selection {
		return nil
	}

	if a.feature.function {
		// resolved function
		if a.feature.resolver != nil {
			variants := make([]maybeASTVariant, len(a.args))
			for i := range variants {
				variants[i] = a.args[i].Variants()
			}
			variant, err := resolveASTVariants(a.feature.resolver, variants...)
			if err != nil {
				return err
			}
			a.SetVariants(variant)
			return nil
		}
		// typed function
		if a.feature.ie.Type != ipfix.IllegalType {
			// no variant typed function
			a.SetVariants(toVariant(a.feature.ie, a))
			return nil
		}
		// unresolved function
		if len(a.args) == 1 {
			// 1 argument: default is to just use the type of the argument
			a.SetVariants(a.args[0].Variants())
			return nil
		}
		if len(a.args) == 2 {
			// 2 argument: default is to just upconvert
			variants, err := resolveASTVariants(UpConvertInformationElements, a.args[0].Variants(), a.args[1].Variants())
			if err != nil {
				return err
			}
			a.SetVariants(variants)
			return nil
		}
		return fmt.Errorf("can't resolve type of %s with %d arguments", a.name, len(a.args))
	}

	// variant feature
	if len(a.feature.variants) != 0 {
		a.SetVariants(toVariant(a.feature.variants, a))
		return nil
	}

	// feature
	a.SetVariants(toVariant(a.feature.ie, a))
	return nil
}

func makeASTFeature(name string, input FeatureType, id int) (astFragment, error) {
	source, err := makeASTRaw(input)
	if err != nil {
		return nil, err
	}
	return makeASTCall(name, []interface{}{source}, input, id)
}

func (a *astCall) String() string {
	args := make([]string, len(a.args))
	for i := range a.args {
		args[i] = a.args[i].String()
	}
	t := fmt.Sprintf("<%s>", a.ret)
	if a.ret == 0 {
		t = ""
	}
	return fmt.Sprintf("%s%s(%s)", t, a.name, strings.Join(args, ", "))
}

type ast struct {
	ret            FeatureType
	input          FeatureType
	filter         []string
	filterFeatures []MakeFeature
	fragments      []astFragment
	exporter       []Exporter
}

// makeAST builds a basic ast for the given feature specification (no verification done yet)
func makeAST(features []interface{}, control, filter []string, exporter []Exporter, input, ret FeatureType) (*ast, error) {
	r := &ast{
		ret:      ret,
		input:    input,
		filter:   filter,
		exporter: exporter,
	}

	r.fragments = make([]astFragment, len(features))
	var err error
	for i := range features {
		r.fragments[i], err = makeASTFragment(features[i], input, i+1)
		if err != nil {
			return r, FeatureError{r.fragments[i].ID(), fmt.Sprint(r.fragments[i]), err}
		}
		r.fragments[i].SetExport(r.fragments[i].MakeExportName())
	}
	for i, feature := range control {
		frag, err := makeASTFragment(feature, input, i+1)
		frag.SetControl(true)
		if err != nil {
			return r, FeatureError{frag.ID(), fmt.Sprint(frag), err}
		}
		r.fragments = append(r.fragments, frag)
	}

	return r, nil
}

func (a ast) String() string {
	ret := &strings.Builder{}
	fmt.Fprintf(ret, "%s -> %s =>", a.input, a.ret)
	for _, exporter := range a.exporter {
		fmt.Fprintf(ret, " %s", exporter.ID())
	}
	fmt.Fprintln(ret)
	for _, fragment := range a.fragments {
		exp := ""
		if fragment.Export() {
			var t interface{}
			t = fragment.Type()
			if t == nil {
				t = fragment.Variants()
			}
			ts := fmt.Sprint(t)
			if t == nil {
				ts = "no type"
			}
			exp = fmt.Sprintf(" -> %s {%s}", fragment.ExportName(), ts)
		}
		fmt.Fprintf(ret, "\t%02d: $%d = %s%s\n", fragment.ID(), fragment.Register(), fragment, exp)
	}
	if len(a.filter) > 0 {
		fmt.Fprintln(ret, "Filters: ", strings.Join(a.filter, ", "))
	}
	return ret.String()
}

func makeExpandedError(fragment astFragment, err error) error {
	short := fragment.ExportName()
	expanded := fragment.MakeExportName()
	if short != expanded {
		short = fmt.Sprintf("%s => %s", short, expanded)
	}
	return FeatureError{fragment.ID(), short, err}
}

func makeFeatureError(fragment astFragment, err error) error {
	return FeatureError{fragment.ID(), fragment.ExportName(), err}
}

// build inserts actual feature definitions in the call chain
func (a *ast) build() error {
	for _, fragment := range a.fragments {
		if err := fragment.build(a.ret); err != nil {
			return makeExpandedError(fragment, err)
		}
	}
	if len(a.filter) > 0 {
		a.filterFeatures = make([]MakeFeature, len(a.filter))
		for i, filter := range a.filter {
			candidates := getFeatures(filter, RawPacket, 1)
			if len(candidates) == 0 {
				return fmt.Errorf("couldn't find filter feature '%s'", filter)
			}
			a.filterFeatures[i] = candidates[0].make
		}
	}
	return nil
}

func expandASTCall(c *astCall, input FeatureType) error {
	var err error
	for i := range c.args {
		c.args[i], err = expandASTFragment(c.args[i], input)
		if err != nil {
			return err
		}
	}
	return nil
}

func expandASTFragment(f astFragment, input FeatureType) (astFragment, error) {
	c, ok := f.(*astCall)
	if !ok {
		return f, nil
	}
	spec, ok := compositeFeatures[f.Name()]
	if !ok {
		if err := expandASTCall(c, input); err != nil {
			return nil, err
		}
		return f, nil
	}
	n, err := makeASTFragment(spec.definition, input, c.ID())
	if err != nil {
		return nil, err
	}
	// fix export
	n.assign(c)
	// directly set to resolved and type - we loose information of original fragment here
	n.SetVariants(toVariant(spec.ie, n)) //SMELL: composite has no variants
	n.SetComposite(f.Name())
	// recurse down for compositeFeatures that consist of compositeFeatures
	c, ok = n.(*astCall)
	if ok {
		if err := expandASTCall(c, input); err != nil {
			return nil, err
		}
	}
	return n, nil
}

// expand expands macros
func (a *ast) expand() error {
	var err error
	for i := range a.fragments {
		a.fragments[i], err = expandASTFragment(a.fragments[i], a.input)
		if err != nil {
			return makeFeatureError(a.fragments[i], err)
		}
	}
	return nil
}

func expandSelectASTFragment(f astFragment, input FeatureType) (astFragment, error) {
	c, ok := f.(*astCall)
	if !ok {
		return f, nil
	}
	var err error
	for i := range c.args {
		c.args[i], err = expandSelectASTFragment(c.args[i], input)
		if err != nil {
			return f, err
		}
	}
	if c.name == "select" || c.name == "select_slice" {
		source, err := makeASTRaw(input)
		if err != nil {
			return f, err
		}
		c.args = append(c.args, source)
	}
	return f, nil
}

// expandSelect expands select* functions (= add input as last argument)
func (a *ast) expandSelect() error {
	var err error
	for i := range a.fragments {
		a.fragments[i], err = expandSelectASTFragment(a.fragments[i], a.input)
		if err != nil {
			return makeFeatureError(a.fragments[i], err)
		}
	}
	return nil
}

// lowerASTFragments modifies the ast to emulate map and apply
func lowerASTFragment(f astFragment, replacement astFragment) (astFragment, error) {
	//check if sentinel and replace with replacement if non nil
	if f.IsRaw() && replacement != nil {
		return replacement, nil
	}
	c, ok := f.(*astCall)
	if !ok {
		return f, nil
	}
	var err error
	if c.name == "map" || c.name == "apply" {
		to := c.args[0]
		selection := c.args[1]

		// first handle eventual apply/map inside selection
		selection, err = lowerASTFragment(selection, replacement)
		if err != nil {
			return nil, err
		}

		// now lower the selection into to
		to, err = lowerASTFragment(to, selection)
		if err != nil {
			return nil, err
		}
		to.assign(f)
		return to, nil
	}
	for i := range c.args {
		c.args[i], err = lowerASTFragment(c.args[i], replacement)
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

// lowerASTFragments modifies the ast to emulate map and apply
func (a *ast) lower() error {
	var err error
	for i := range a.fragments {
		a.fragments[i], err = lowerASTFragment(a.fragments[i], nil)
		if err != nil {
			return makeExpandedError(a.fragments[i], err)
		}
	}
	return nil
}

// resolve does type resolution
func (a *ast) resolve() error {
	for _, fragment := range a.fragments {
		if fragment.Control() {
			continue
		}
		err := fragment.resolve()
		if err != nil {
			return makeExpandedError(fragment, err)
		}
	}
	return nil
}

func simplifyFragments(fragment astFragment, subtrees map[string]astFragment, out *[]astFragment, register *int) error {
	if fragment.IsRaw() {
		return nil
	}

	if c, ok := fragment.(*astCall); ok {
		for _, arg := range c.args {
			err := simplifyFragments(arg, subtrees, out, register)
			if err != nil {
				return err
			}
		}
	}

	name := fragment.String()
	if f, ok := subtrees[name]; ok {
		if fragment.Export() {
			return fmt.Errorf("exporting feature twice not allowed (first occurance in #%d)", f.ID())
		}
		fragment.SetRegister(f.Register())
		return nil
	}

	cpy := fragment.Copy()
	if c, ok := cpy.(*astCall); ok {
		for i := range c.args {
			name := c.args[i].String()
			if found, ok := subtrees[name]; ok {
				c.args[i] = makeASTRegister(found.Register(), found.Returns())
			}
		}
	}
	subtrees[name] = cpy
	cpy.SetRegister(*register)
	fragment.SetRegister(*register)
	*register++
	if cpy.Export() {
		cpy.SetType(fragment.Variants().simplify())
	}
	*out = append(*out, cpy)
	return nil
}

// merge merges common subtrees
func (a *ast) simplify() error {
	subtrees := make(map[string]astFragment)
	var out []astFragment
	register := 0
	for _, fragment := range a.fragments {
		err := simplifyFragments(fragment, subtrees, &out, &register)
		if err != nil {
			return makeExpandedError(fragment, err)
		}
	}
	a.fragments = out
	return nil
}

// compile builds and optimizes a final call chain
func (a *ast) compile(verbose bool) error {
	if verbose {
		log.Println("Phase #1 [parsing]:")
		log.Println(a)
	}
	if verbose {
		log.Println("Phase #2 [composite expansion]:")
	}
	if err := a.expand(); err != nil {
		return err
	}
	if verbose {
		log.Println(a)
		log.Println("Phase #3 [building]:")
	}
	if err := a.build(); err != nil {
		return err
	}
	if verbose {
		log.Println(a)
		log.Println("Phase #4 [expand select]:")
	}
	if err := a.expandSelect(); err != nil {
		return err
	}
	if verbose {
		log.Println(a)
		log.Println("Phase #5 [lower map/apply]:")
	}
	if err := a.lower(); err != nil {
		return err
	}
	if verbose {
		log.Println(a)
		log.Println("Phase #6 [type resolution]:")
	}
	if err := a.resolve(); err != nil {
		return err
	}
	if verbose {
		log.Println(a)
		log.Println("Phase #7 [simplify]")
	}
	if err := a.simplify(); err != nil {
		return err
	}
	if verbose {
		log.Println(a)
	}

	return nil
}

func (a *ast) convert() (features []MakeFeature, filters []MakeFeature, args [][]int, tocall [][]int, ctrl *control) {
	ctrl = &control{}
	args = make([][]int, len(a.fragments))
	tocall = make([][]int, len(a.fragments))
	for _, fragment := range a.fragments {
		fm := fragment.FeatureMaker()
		features = append(features, fm.make)
		if fragment.Control() {
			ctrl.control = append(ctrl.control, fragment.Register())
			continue
		}

		for _, arg := range fragment.Arguments() {
			if arg.IsRaw() {
				ctrl.event = append(ctrl.event, fragment.Register())
			} else {
				args[fragment.Register()] = append(args[fragment.Register()], arg.Register())
				tocall[arg.Register()] = append(tocall[arg.Register()], fragment.Register())
			}
		}

		if fragment.Export() {
			ctrl.export = append(ctrl.export, fragment.Register())
		}

		if len(fm.variants) > 0 {
			ctrl.variant = append(ctrl.variant, fragment.Register())
		}
	}
	filters = a.filterFeatures
	return
}
