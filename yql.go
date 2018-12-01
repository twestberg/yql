package main

import (
	"flag"
	"fmt"
	//"github.com/go-yaml/yaml"
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

var args argStruct

func main() {
	var err error
	args, err = parseCmd()
	if err != nil {
		usage(err)
		return
	}
	fmt.Printf("%#v\n", args)
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
			fmt.Fprintf(os.Stderr, "expected int after [ in keypath\n")
			return // not valid
		} else {
			elt.index, _ = strconv.Atoi(s.TokenText())
			// Finally, require a closing bracket
			tok = s.Scan()
			if tok != ']' {
				fmt.Fprintf(os.Stderr, "expected ] after index in keypath\n")
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
