/*
This package is a utility that strips diff content from a git log.

This small utility reads a git log from standard input, and writes it to
standard output preserving only four kinds of line:

"commit ..."  <- names the commit
"--- ..."     <- at the top of each file
"+++ ..."     <- at the top of each file
"@@ ..."      <- at the start of each hunk

It edits each "@@ -0,0 +1,3 @@" line so its second "@@" is followed by a
dash instead of a space ("@@-", which never happens in real diffs) so
our blame input routine knows to not expect "+" and "-" lines to follow.

This can be useful if you are doing experiments on the history of very
large repositories, and the raw git log output would be too large to
store and re-scan as you are developing.  The git log should be produced
with options similar to those used in gitops.go, like:

git log -U0 --format='commit %H' --no-prefix --no-renames --reverse --no-ext-diff --no-textconv --first-parent -m

*/
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	re, _ := regexp.Compile(`@@ -(\d+),?(\d*) \+(\d+),?(\d*) `)

	scanner := bufio.NewScanner(os.Stdin)

	const maxCapacity = 100*1024*1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "commit ") {
			fmt.Print(line + "\n")
		} else if strings.HasPrefix(line, "--- ") {
			fmt.Print(line + "\n")
		} else if strings.HasPrefix(line, "+++ ") {
			fmt.Print(line + "\n")
		} else if strings.HasPrefix(line, "@@ ") {
			rest := line[3:]
			i := strings.Index(rest, " @@")
			fmt.Printf("@@ %s @@-\n", rest[:i])

			result_slice := re.FindStringSubmatch(line)
			//old_start, _ := strconv.Atoi(result_slice[1])
			old_length := 1
			if len(result_slice[2]) > 0 {
				old_length, _ = strconv.Atoi(result_slice[2])
			}
			//new_start, _ := strconv.Atoi(result_slice[3])
			new_length := 1
			if len(result_slice[4]) > 0 {
				new_length, _ = strconv.Atoi(result_slice[4])
			}
			lines_to_skip := old_length + new_length
			for i := 0; i < lines_to_skip; i++ {
				scanner.Scan()
			}
		}
	}
	err := scanner.Err()
	if err != nil {
		log.Fatal(err)
	}
}
