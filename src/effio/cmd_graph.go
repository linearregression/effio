package effio

import (
	"encoding/hex"
	"hash/fnv"
	"log"
	"os"
	"path"
	"regexp"
	"sort"
)

// -excl has higher priority than -incl so you can -incl and then
// pare it down with -excl
func (cmd *Cmd) GraphSuite() {
	var idFlag, pathFlag, outFlag, outdir, inclFlag, exclFlag string
	var listFlag bool
	var err error
	var excl, incl *regexp.Regexp

	fs := cmd.FlagSet
	fs.StringVar(&pathFlag, "path", "suites/", "suite path, as generated by effio make")
	fs.StringVar(&idFlag, "id", "", "Id of the test suite")
	fs.StringVar(&outFlag, "out", "all", "name of the directory that will contain graphs: -path/-id/-out")
	fs.StringVar(&inclFlag, "incl", "", "regex matching tests to include in graph")
	fs.StringVar(&exclFlag, "excl", "", "regex matching tests to exclude from graph")
	fs.BoolVar(&listFlag, "list", false, "print a list of included tests and exit without processing")
	fs.Parse(cmd.Args)

	// whitelist
	if len(inclFlag) > 0 {
		incl, err = regexp.Compile(inclFlag)
		if err != nil {
			log.Fatalf("-incl '%s': regex could not be compiled: %s\n", inclFlag, err)
		}
	}

	// blacklist, applied after the whitelist
	if len(exclFlag) > 0 {
		excl, err = regexp.Compile(exclFlag)
		if err != nil {
			log.Fatalf("-excl '%s': regex could not be compiled: %s\n", exclFlag, err)
		}
	}

	// use full paths internally
	// TODO: this currently only supports relative paths and will break on rooted paths
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Could not get working directory: %s\n", err)
	}
	suite_dir := path.Join(wd, pathFlag)

	// load up the suite
	jsonPath := path.Join(wd, pathFlag, idFlag, "suite.json")
	_, err = os.Stat(jsonPath)
	if err != nil {
		log.Fatalf("Could not stat file '%s': %s\n", jsonPath, err)
	}
	s := LoadSuiteJson(jsonPath)

	// filter out unwanted tests from s.Tests
	// could be more clever here but KISS
	var tests []Test
	for _, test := range s.Tests {
		// when no -incl is specified, all tests are included by default
		keep := true
		if len(inclFlag) > 0 {
			// but when one is specified, -incl becomes a whitelisting RE
			keep = false
			if incl.MatchString(test.Name) {
				keep = true
			}
		}

		// blacklist RE always works the same and always comes after -incl
		if len(exclFlag) > 0 && excl.MatchString(test.Name) {
			keep = false
		}

		if keep {
			tests = append(tests, test)
		}
	}

	// swap out the test list
	s.Tests = tests

	// if incl/excl are used and an 'out' name isn't specified, make one
	// based on a hash of all names in the test so it's consistent and automatic
	if outFlag == "all" && (len(inclFlag) > 0 || len(exclFlag) > 0) {
		sort.Sort(s.Tests) // sort to ensure the hash is as consistent as possible
		hash := fnv.New64()
		for _, test := range s.Tests {
			hash.Write([]byte(test.Name)) // docs: never returns an error
		}
		name := hex.EncodeToString(hash.Sum(nil))
		outdir = path.Join(wd, pathFlag, idFlag, name)
		log.Printf("output will be written to '%s'\n", outdir)
	} else {
		outdir = path.Join(wd, pathFlag, idFlag, outFlag)
	}

	if listFlag {
		for _, test := range s.Tests {
			log.Println(test.Name)
		}
		return
	}

	s.GraphSizes(suite_dir, outdir)
	//s.Graph(suite_dir, outdir)
}
