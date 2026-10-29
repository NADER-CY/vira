// Command testsummary reads `go test -json` output on stdin and prints pass/fail/skip counts and failures.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
)

type event struct {
	Action  string
	Package string
	Test    string
	Output  string
}

func main() {
	failOnFailure := flag.Bool("fail-on-failure", false, "exit 1 when any test or package failed")
	flag.Parse()
	counts := map[string]int{}
	output := map[string][]string{}
	var failed []string
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	for sc.Scan() {
		var e event
		if json.Unmarshal(sc.Bytes(), &e) != nil {
			continue
		}
		key := e.Package + "." + e.Test
		switch e.Action {
		case "output":
			output[key] = append(output[key], e.Output)
		case "pass", "fail", "skip":
			if e.Test == "" {
				if e.Action == "fail" && counts["pkgfail:"+e.Package] == 0 {
					counts["pkgfail:"+e.Package]++
					failed = append(failed, e.Package+" (package)")
				}
				continue
			}
			counts[e.Action]++
			if e.Action == "fail" {
				failed = append(failed, key)
			}
		}
	}
	sort.Strings(failed)
	for _, f := range failed {
		fmt.Println("FAIL", f)
		for _, l := range output[f] {
			fmt.Print("    ", l)
		}
	}
	fmt.Printf("tests: %d passed, %d failed, %d skipped\n", counts["pass"], counts["fail"], counts["skip"])
	if *failOnFailure && len(failed) > 0 {
		os.Exit(1)
	}
}
