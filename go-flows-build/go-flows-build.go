package main

import (
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"html/template"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

var thisPath string
var modulePath string
var moduleImport = "github.com/CN-TU/go-flows/modules"

func init() {
	_, f, _, _ := runtime.Caller(0)
	thisPath = filepath.Dir(filepath.Dir(f))
	modulePath = filepath.Join(thisPath, "modules")
}

type module struct {
	name    string
	path    string
	imp     string
	builtin bool
}

func (m module) String() string {
	if m.builtin {
		return "+ " + m.name
	}
	return "  " + m.name
}

type modules []module

func (m modules) String() string {
	var ret = make([]string, len(m))
	for i, m := range m {
		ret[i] = fmt.Sprint(m)
	}
	return strings.Join(ret, "\n")
}

func listModules() (ret modules) {
	builtin := make(map[string]struct{})

	f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(thisPath, "builtin.go"), nil, parser.ImportsOnly)
	if err != nil {
		panic(fmt.Sprint("Couldn't read builtin.go: ", err))
	}
	for _, s := range f.Imports {
		full := strings.Split(strings.Trim(s.Path.Value, "\""), "/")
		builtin[strings.Join(full[len(full)-2:], ".")] = struct{}{}
	}

	var mtypes []os.FileInfo
	mtypes, err = ioutil.ReadDir(modulePath)
	if err != nil {
		panic(fmt.Sprint("Couldn't read module path: ", err))
	}
	for _, mt := range mtypes {
		if !mt.IsDir() {
			continue
		}
		var mods []os.FileInfo
		mods, err = ioutil.ReadDir(filepath.Join(modulePath, mt.Name()))
		if err != nil {
			panic(fmt.Sprint("Couldn't read '", mt.Name(), "' in module path: ", err))
		}
		for _, m := range mods {
			if !m.IsDir() {
				continue
			}
			name := fmt.Sprint(mt.Name(), ".", m.Name())
			_, b := builtin[name]
			ret = append(ret, module{
				name:    name,
				path:    filepath.Join(modulePath, mt.Name(), m.Name()),
				imp:     strings.Join([]string{moduleImport, mt.Name(), m.Name()}, "/"),
				builtin: b,
			})
		}
	}
	return
}

func getFiles(flags []string, path string) ([]string, string) {
	type List struct {
		GoFiles []string
		Target  string
		Name    string
	}
	args := append([]string{"list", "-json"}, flags...)
	cmd := exec.Command("go", args...)
	cmd.Dir = path
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	var l List
	err = json.Unmarshal(out, &l)
	if err != nil {
		panic(err)
	}
	ret := make([]string, len(l.GoFiles))
	for i, f := range l.GoFiles {
		ret[i] = filepath.Join(path, f)
	}
	name := l.Name
	if name == "" || name == "main" {
		name = filepath.Base(l.Target)
	}
	if name == "." {
		name = ""
	}
	return ret, name
}

func usage() {
	fmt.Printf(`Usage: %s [-flags] action [path/files]

Build plugin:
%s [-goflags] plugin path/files

Builds a plugin from a package (source directory) or files and writes
output to "go-flows.<name>.defs.so". All -goflags are passed to the
compiler (except for -o and --buildmode).

Build go-flows:
%s [-module] [+module] [-goflags] build [modules]

Builds the binary and includes or excludes specific modules. If modules is
given, then all excluded modules are built as plugins. The special word all
can be used to affect all modules.

The following modules are available (+ modules are default):
%s
`, os.Args[0], os.Args[0], os.Args[0], listModules())
}

func checkFlags(flags []string) (string, []string) {
	var name string
	var pos int
	for i, f := range flags {
		if strings.HasPrefix(f, "-buildmode") || strings.HasPrefix(f, "--buildmode") {
			fmt.Println("Flag '", f, "' not allowed")
			os.Exit(-1)
		}
		if f == "-o" {
			name = flags[i+1]
			pos = i
		}
	}
	if name != "" {
		flags = append(flags[:pos], flags[pos+2:]...)
	}
	return name, flags
}

var pluginTemplate = template.Must(template.New("plugin").Parse(`package main

import (
	_ "{{.}}"
)

func main() {}
`))

func plugin(flags []string, files []string, builtin modules, tempdir string) {
	if len(files) == 0 {
		usage()
		return
	}
	var name string
	n, flags := checkFlags(flags)
	if len(files) == 1 {
		for _, m := range builtin {
			if files[0] == m.name {
				name = m.name

				f := filepath.Join(tempdir, m.name+".go")

				pluginfile, err := os.Create(f)
				defer os.Remove(f)
				if err != nil {
					panic(fmt.Sprint("Error creating temporary file for plugin: ", err))
				}

				pluginTemplate.Execute(pluginfile, m.imp)
				pluginfile.Close()

				files[0] = f
				goto FOUND
			}
		}

		fi, err := os.Stat(files[0])
		if err != nil {
			panic(err)
		}
		if fi.IsDir() {
			files, name = getFiles(flags, files[0])
			if len(files) == 0 {
				panic(fmt.Sprint("No go files in directory ", files[0]))
			}
		}
	}

	if name == "" {
		name = filepath.Base(files[0])
		ext := filepath.Ext(name)
		name = name[:len(name)-len(ext)]
	}

FOUND:

	name = fmt.Sprintf("go-flows.%s.defs.so", name)
	if n != "" {
		name = n
	}
	args := append(append([]string{"build", "-o", name, "--buildmode=plugin"}, flags...), files...)
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	ret := cmd.Run()
	if ret != nil {
		panic(ret)
	}
}

var builtinTemplate = template.Must(template.New("builtin").Parse(`package main

import (
{{range $i, $m := .}}	_ "{{$m}}"
{{end}})
`))

func build(flags []string, args []string, builtin modules, tempdir string) {
	buildModules := len(args) > 0 && args[0] == "modules"

	var goflags []string
	modules := make(map[string]bool)
	imports := make(map[string]string)
	for _, mod := range builtin {
		modules[mod.name] = mod.builtin
		imports[mod.name] = mod.imp
	}
	for _, f := range flags {
		plus := f[0] == '+'
		minus := f[0] == '-'
		if plus || minus {
			if f[1:] == "all" {
				for i := range modules {
					modules[i] = plus
				}
			} else if _, ok := modules[f[1:]]; ok {
				modules[f[1:]] = plus
			} else {
				goflags = append(goflags, f)
			}
		} else {
			goflags = append(goflags, f)
		}
	}
	nonstd := false
	for _, mod := range builtin {
		if modules[mod.name] != mod.builtin {
			nonstd = true
			break
		}
	}
	if !nonstd {
		args := append([]string{"build"}, goflags...)
		cmd := exec.Command("go", args...)
		cmd.Dir = thisPath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		ret := cmd.Run()
		if ret != nil {
			panic(ret)
		}
		return
	}

	flist, _ := getFiles(goflags, thisPath)
	for i, f := range flist {
		if strings.HasSuffix(f, "builtin.go") {
			flist = append(flist[:i], flist[i+1:]...)
			break
		}
	}

	var importlist []string
	var buildlist []string
	for name, use := range modules {
		if use {
			importlist = append(importlist, imports[name])
		} else {
			buildlist = append(buildlist, name)
		}
	}

	f := filepath.Join(thisPath, "builderBuiltin.go")

	importfile, err := os.Create(f)
	defer os.Remove(f)
	if err != nil {
		panic(fmt.Sprint("Error creating temporary file for integrating modules: ", err))
	}

	builtinTemplate.Execute(importfile, importlist)
	importfile.Close()

	flist = append(flist, f)

	args = append(append([]string{"build", "-o", "go-flows"}, goflags...), flist...)
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	ret := cmd.Run()
	if ret != nil {
		panic(ret)
	}

	if buildModules {
		for _, m := range buildlist {
			fmt.Printf("Building %s...\n", m)
			plugin(goflags, []string{m}, builtin, tempdir)
		}
	}
}

func main() {
	if len(os.Args) == 1 {
		usage()
		return
	}

	var flags []string
	var action string
	args := os.Args[1:]

	for i, s := range args {
		if s[0] == '-' || s[0] == '+' {
			continue
		}
		if s == "plugin" || s == "build" {
			flags = args[:i]
			action = args[i]
			args = args[i+1:]
			break
		}
	}

	tempdir, err := ioutil.TempDir("", "")
	if err != nil {
		fmt.Println("Couldn't create temporary director: ", err)
		os.Exit(-1)
	}
	defer os.RemoveAll(tempdir)

	defer func() {
		if err := recover(); err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
					os.Exit(status.ExitStatus())
				}
				fmt.Println(err)
			} else {
				fmt.Println(err)
			}
			os.Exit(-1)
		}
	}()

	modules := listModules()

	switch action {
	case "plugin":
		plugin(flags, args, modules, tempdir)
	case "build":
		build(flags, args, modules, tempdir)
	default:
		usage()
	}
}
