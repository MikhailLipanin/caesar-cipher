// verify that all the imports have our preferred alias(es).
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/term"
)

var (
	importAliases = flag.String("import-aliases", "cmd/preferredimports/.import-aliases", "json file with import aliases")
	fix           = flag.Bool("fix", false, "update file with the preferred aliases for imports")
	skipDirs      = flag.String("skip-dirs", "a^", "regex for directories to skip")
	// nolint:lll
	skipFiles  = flag.String("skip-files", "(.*.gen.go$)|(.*.pb.go$)|(^federation.go$)|(^generated.go$)|(^mock_.*)", "regex for files to skip")
	isTerminal = term.IsTerminal(int(os.Stdout.Fd()))
	logPrefix  = ""
	aliases    = map[*regexp.Regexp]string{}
)

type analyzer struct {
	fset      *token.FileSet // positions are relative to fset
	ctx       build.Context
	failed    bool
	skipFiles *regexp.Regexp
	donePaths map[string]interface{}
}

func newAnalyzer() *analyzer {
	ctx := build.Default
	ctx.CgoEnabled = true

	a := &analyzer{
		fset:      token.NewFileSet(),
		ctx:       ctx,
		donePaths: make(map[string]interface{}),
	}

	return a
}

// collect extracts test metadata from a file.
func (a *analyzer) collect(dir string) {
	if _, ok := a.donePaths[dir]; ok {
		return
	}
	a.donePaths[dir] = nil

	// Create the AST by parsing src.
	fs, err := parser.ParseDir(a.fset, dir, nil,
		parser.ParseComments|parser.SkipObjectResolution)

	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR(syntax)", logPrefix, err)
		a.failed = true
		return
	}

	for _, p := range fs {
		// returns first error, but a.handleError deals with it
		files := a.filterFiles(p.Files)
		for _, file := range files {
			replacements := make(map[string]string)
			pathToFile := a.fset.File(file.Pos()).Name()
			for _, imp := range file.Imports {
				importPath := strings.ReplaceAll(imp.Path.Value, "\"", "")
				pathSegments := strings.Split(importPath, "/")
				suffixAlias := pathSegments[len(pathSegments)-1]
				importName := suffixAlias
				if imp.Name != nil {
					importName = imp.Name.Name
				}
				for re, template := range aliases {
					match := re.FindStringSubmatchIndex(importPath)
					if match == nil {
						// No match.
						continue
					}
					if match[0] > 0 || match[1] < len(importPath) {
						// Not a full match.
						continue
					}
					alias := string(re.ExpandString(nil, template, importPath, match))
					if alias != importName {
						if !*fix {
							fmt.Fprintf(os.Stderr, "%sERROR wrong alias for import %q should be %q in file %s\n",
								logPrefix, importPath, alias, pathToFile)
							a.failed = true
						}
						replacements[importName] = alias
						// This condition need if we want to import a package without named alias.
						if alias == suffixAlias {
							imp.Name = nil
						} else {
							if imp.Name != nil {
								imp.Name.Name = alias
							} else {
								imp.Name = ast.NewIdent(alias)
							}
						}
					}
					break
				}
			}

			if len(replacements) > 0 {
				if *fix {
					fmt.Printf("%sReplacing imports with aliases in file %s\n", logPrefix, pathToFile)
					for key, value := range replacements {
						renameImportUsages(file, key, value)
					}
					// ast.SortImports(a.fset, file)
					var buffer bytes.Buffer
					if err = format.Node(&buffer, a.fset, file); err != nil {
						panic(fmt.Sprintf("Error formatting ast node after rewriting import.\n%s\n", err.Error()))
					}

					fileInfo, err := os.Stat(pathToFile)
					if err != nil {
						panic(fmt.Sprintf("Error stat'ing file: %s\n%s\n", pathToFile, err.Error()))
					}

					err = os.WriteFile(pathToFile, buffer.Bytes(), fileInfo.Mode())
					if err != nil {
						panic(fmt.Sprintf("Error writing file: %s\n%s\n", pathToFile, err.Error()))
					}
				}
			}
		}
	}
}

func renameImportUsages(f *ast.File, old, new string) {
	// use this to avoid renaming the package declaration, eg:
	//   given: package foo; import foo "bar"; foo.Baz, rename foo->qux
	//   yield: package foo; import qux "bar"; qux.Baz
	var pkg *ast.Ident

	ancestors := make([]ast.Node, 0)

	// Rename top-level old to new, both unresolved names
	// (probably defined in another file) and names that resolve
	// to a declaration we renamed.
	ast.Inspect(f, func(node ast.Node) bool {
		switch id := node.(type) {
		case *ast.File:
			pkg = id.Name
		case *ast.Ident:
			if pkg != nil && id == pkg {
				return false
			}
			if len(ancestors) > 0 {
				// Here, we are checking, that current ast.Ident is an export from package.
				// We can say that, if the current ast.Ident is an expression X inside ast.SelectorExpr
				// model.
				if parentNode, ok := ancestors[len(ancestors)-1].(*ast.SelectorExpr); ok {
					if parentNode.X == id && id.Name == old {
						id.Name = new
					}
				}
			}
		}
		if node == nil {
			// Pop, since we're done with this node and its children.
			ancestors = ancestors[:len(ancestors)-1]
		} else {
			// Push this node on the stack, since its children will be visited next.
			ancestors = append(ancestors, node)
		}
		return true
	})
}

func (a *analyzer) filterFiles(fs map[string]*ast.File) []*ast.File {
	files := make([]*ast.File, 0, len(fs))
	for name, file := range fs {
		baseName := path.Base(name)
		if a.skipFiles.MatchString(baseName) {
			continue
		}
		files = append(files, file)
	}
	return files
}

type collector struct {
	dirs     []string
	skipDirs *regexp.Regexp
}

// handlePath walks the filesystem recursively, collecting directories,
// ignoring some unneeded directories (hidden/vendored) that are handled
// specially later.
func (c *collector) handlePath(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if info.IsDir() {
		// Ignore hidden directories (.git, .cache, etc)
		if len(path) > 1 && path[0] == '.' ||
			c.skipDirs.MatchString(path) {
			return filepath.SkipDir
		}
		c.dirs = append(c.dirs, path)
	}
	return nil
}

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		args = append(args, ".")
	}

	skipDirs, err := regexp.Compile(*skipDirs)
	if err != nil {
		log.Fatalf("Error compiling regex: %v", err)
	}
	skipFiles, err := regexp.Compile(*skipFiles)
	if err != nil {
		log.Fatalf("Error compiling regex: %v", err)
	}
	c := collector{skipDirs: skipDirs}
	for _, arg := range args {
		err := filepath.Walk(arg, c.handlePath)
		if err != nil {
			log.Fatalf("Error walking: %v", err)
		}
	}
	sort.Strings(c.dirs)

	if len(*importAliases) > 0 {
		bytes, err := os.ReadFile(*importAliases)
		if err != nil {
			log.Fatalf("Error reading import aliases: %v", err)
		}
		var stringAliases map[string]string
		err = json.Unmarshal(bytes, &stringAliases)
		if err != nil {
			log.Fatalf("Error loading aliases: %v", err)
		}
		for pattern, name := range stringAliases {
			re, err := regexp.Compile(pattern)
			if err != nil {
				log.Fatalf("Error parsing import path pattern %q as regular expression: %v", pattern, err)
			}
			aliases[re] = name
		}
	}

	if isTerminal {
		logPrefix = "\r" // clear status bar when printing
	}
	log.Println("checking-imports...")

	a := newAnalyzer()
	a.skipFiles = skipFiles
	for _, dir := range c.dirs {
		a.collect(dir)
	}
	if a.failed {
		log.Fatalf("!!! Please see \".import-aliases\" for the preferred aliases for imports.")
	}
	log.Println("Done!")
}
