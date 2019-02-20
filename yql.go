package main

import (
	"flag"
	"fmt"
	"github.com/go-yaml/yaml"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/scanner"
)

type keyElement struct {
	key   string
	index int
	isMap bool // true if key is used, else index is used for array
	valid bool // true when scanner is happy
}
type argStruct struct {
	quiet    bool
	filepath string
	useStdin bool
	cmd      string
	key      string
	keypath  []keyElement
	val      string
	progname string
}

type datamap map[interface{}]interface{}

var args argStruct

func main() {
	var err error
	args, err = parseCmd()
	if err != nil {
		usage(err)
		return
	}
	// Attempt to read the data file
	b, err := ioutil.ReadFile(args.filepath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read data file %s : %v\n", args.filepath, err)
		return
	}
	var data datamap = make(datamap)
	err = yaml.Unmarshal(b, &data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to parse data file %s : %v\n", args.filepath, err)
		return
	}
	if args.cmd == "get" {
		get(data, args.keypath)
	} else if args.cmd == "set" {
		data, err = set(data, args.keypath, args.val)
		if err == nil {
			// write the datamap back out to the file TODO
			fmt.Printf("%v\n", data)
		} else {
			errprint("%v\n", err)
		}
	}
	// fmt.Printf("%#v\n", args)
}

// Get the value at the keypath in the data
func get(data datamap, path []keyElement) {
	var val interface{} = data
	pathname := "root"
	for _, elt := range path {
		if elt.isMap {
			m, ok := val.(datamap)
			if !ok {
				errprint("%s is not a map.\n", pathname)
				return
			}
			val, ok = m[elt.key]
			if !ok {
				errprint("%s.%s not found\n", pathname, elt.key)
			}
			pathname = elt.key
		} else {
			m, ok := val.([]interface{})
			if !ok {
				errprint("%s is not an array.\n", pathname)
				return
			}
			if elt.index >= len(m) {
				errprint("%s[%d] index out of bounds\n", pathname, elt.index)
				return
			}
			val = m[elt.index]
		}
	}
	fmt.Printf("%v\n", val)
}

// Set the given value v at the keypath in the data
func set(data datamap, path []keyElement, v string) (err error) {
	var val interface{} = data
	pathname := "root"
	for idx, elt := range path {
		if elt.isMap {
			m, ok := val.(datamap)
			if !ok {
				return fmt.Errorf("%s is not a map.\n", pathname)
			}
			val, ok = m[elt.key]
			if !ok {
				// Need to create a new thing at this key. It can be only a new map
				// or a place for a value
				if idx >= len(path)-1 {
					// last item
					m[elt.key] = new(interface{})
					val = m[elt.key]
				} else {
					nextElt := path[idx+1]
					if nextElt.isMap {
						m[elt.key] = new(datamap)
						val = m[elt.key]
					} else {
						return fmt.Errorf("Cannot create an array at %s\n", elt.key)
					}
				}
			}
			pathname = elt.key
		} else {
			m, ok := val.([]interface{})
			if !ok {
				return fmt.Errorf("%s is not an array.\n", pathname)
			}
			if elt.index >= len(m) {
				return fmt.Errorf("Index (%d) out of range. Cannot grow arrays", elt.index)
			}
			val = m[elt.index]
		}
	}
	// Afer crawling the path, set the val
	fmt.Printf("%s is currently %v\n", pathname, val)
	i, e := strconv.Atoi(v)
	if e == nil {
		val = i
	} else {
		val = v
	}
	fmt.Printf("%s is now %v\n", pathname, val)
	rdata = data
	return rdata, nil
}

func errprint(f string, aargs ...interface{}) {
	if !args.quiet {
		fmt.Fprintf(os.Stderr, f, aargs...)
	}
}
func parseCmd() (args argStruct, err error) {
	var quiet = flag.Bool("q", false, "quiet warnings")
	var filePath = flag.String("f", "", "set db filepath (or use YQL_FILE env variable)")
	var useStdin = flag.Bool("stdin", false, "read val from stdin for Set command")
	args.progname = filepath.Base(os.Args[0])
	flag.Parse()
	args.quiet = *quiet
	args.useStdin = *useStdin
	args.filepath = os.Getenv("YQL_FILE")
	if *filePath != "" {
		args.filepath = *filePath
	}

	pathIndex := 0
	switch args.progname {
	case "yql":
		pathIndex = 1          // if an action is required first, path is later in the args
		args.cmd = flag.Arg(0) // will be "" if no arg
	case "yset":
		args.cmd = "set"
	case "yget":
		args.cmd = "get"
	}
	args.key = flag.Arg(pathIndex)
	args.val = flag.Arg(pathIndex + 1)
	if args.cmd != "set" && args.cmd != "get" {
		err = fmt.Errorf("Requires 'get' or 'set' command")
		return
	}
	if args.cmd == "set" && args.val == "" && !args.useStdin {
		err = fmt.Errorf("Requires value after argument or -stdin flag")
		return
	}
	args.keypath, err = parseKeypath(args.key)
	return
}

// Given a keypath, parse it for its map and index components
func parseKeypath(keypath string) (result []keyElement, err error) {
	if keypath == "" {
		err = fmt.Errorf("Requires keypath argument")
		return
	}
	if keypath[0] == '.' || keypath[0] == '[' {
		err = fmt.Errorf("First element of keypath must be a map index")
		return
	}
	var s scanner.Scanner
	s.Init(strings.NewReader(keypath))
	s.Filename = "cmdline"

	for elt := nextElement(&s); elt.valid; elt = nextElement(&s) {
		result = append(result, elt)
	}
	return
}

// Scan the given reader for a keyElement. This may be
// <symbol>
// '.'<symbol>
// <symbol><EOF>
// [int]
func nextElement(s *scanner.Scanner) (elt keyElement) {
	s.Mode = scanner.ScanIdents | scanner.ScanInts | scanner.GoTokens
	indexOk := true

	tok := s.Scan()
	if tok == '.' {
		indexOk = false
		tok = s.Scan()
	}
	if tok == scanner.Ident {
		elt.isMap = true
		elt.key = s.TokenText()
		elt.valid = true
		return
	}
	if !indexOk {
		return
	}
	if tok == '[' {
		// Beginning of an index
		tok = s.Scan()
		if tok != scanner.Int {
			errprint("expected int after [ in keypath\n")
			return // not valid
		} else {
			elt.index, _ = strconv.Atoi(s.TokenText())
			// Finally, require a closing bracket
			tok = s.Scan()
			if tok != ']' {
				errprint("expected ] after index in keypath\n")
				return
			} else {
				elt.valid = true
				return
			}
		}
	}
	return
}

func usage(err error) {
	switch args.progname {
	case "yql":
		fmt.Fprintf(os.Stderr, "%s\nUsage: yql set|get [-q] [-f fpath] [-stdin] keypath [val]\n-q\t quiets warnings\n-f fpath to data file\n-stdin\tread value from stdin not cmdline\n", err.Error())
	case "yset":
		fmt.Fprintf(os.Stderr, "%s\nUsage: yset [-q] [-f fpath] [-stdin] keypath [val]\n-q\t quiets warnings\n-f fpath to data file\n-stdin\tread value from stdin not cmdline\n", err.Error())
	case "yget":
		fmt.Fprintf(os.Stderr, "%s\nUsage: yget [-q] [-f fpath] keypath \n-q\t quiets warnings\n-f fpath to data file\n", err.Error())
	default:
		fmt.Fprintf(os.Stderr, "%s is an unknown name for this program\n", os.Args[0])
	}
}
