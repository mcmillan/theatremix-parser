// Command theatremix-parser converts a TheatreMix .tmix show file into JSON.
//
//	theatremix-parser [flags] [FILE]
//
// FILE is a .tmix show file; when omitted or "-", the file is read from
// stdin. The JSON document is written to stdout. See docs/ for the format
// specification and the shape of the output.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/mcmillan/theatremix-parser/tmix"
)

// version is overridden at build time: -ldflags "-X main.version=1.2.3".
var version = "dev"

const (
	exitOK       = 0
	exitError    = 1
	exitViolated = 2
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("theatremix-parser", flag.ContinueOnError)
	fs.SetOutput(stderr)
	compact := fs.Bool("compact", false, "emit single-line JSON instead of indented")
	validate := fs.Bool("validate", false, "check the show against the format invariants; report violations on stderr and exit 2 if any")
	showVersion := fs.Bool("version", false, "print the version and exit")
	fs.Usage = func() {
		fmt.Fprintf(stderr, "usage: theatremix-parser [flags] [FILE]\n\n"+
			"Reads a TheatreMix .tmix show file (FILE, or stdin when omitted or \"-\")\n"+
			"and writes a JSON representation of it to stdout.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitError
	}
	if *showVersion {
		fmt.Fprintln(stdout, "theatremix-parser", version)
		return exitOK
	}
	if fs.NArg() > 1 {
		fs.Usage()
		return exitError
	}

	var (
		show *tmix.Show
		err  error
	)
	if path := fs.Arg(0); path == "" || path == "-" {
		var data []byte
		if data, err = io.ReadAll(stdin); err == nil {
			show, err = tmix.OpenBytes(data)
		}
	} else {
		show, err = tmix.OpenFile(path)
	}
	if err != nil {
		fmt.Fprintf(stderr, "theatremix-parser: %v\n", err)
		return exitError
	}

	code := exitOK
	if *validate {
		for _, v := range tmix.Validate(show) {
			fmt.Fprintln(stderr, "violation:", v)
			code = exitViolated
		}
	}

	enc := json.NewEncoder(stdout)
	enc.SetEscapeHTML(false)
	if !*compact {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(show); err != nil {
		fmt.Fprintf(stderr, "theatremix-parser: %v\n", err)
		return exitError
	}
	return code
}
