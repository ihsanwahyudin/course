// Command coursecheck validates every course under a folder the way the API
// does at startup (structure, lessons, material, translations, links between
// courses and the hand-written lesson steps), without a database.
//
//	go run ./cmd/coursecheck ../../packages/courses
package main

import (
	"fmt"
	"os"

	"fluentpath/api/internal/curriculum"
	"fluentpath/api/internal/tutor"
)

func main() {
	root := "../../packages/courses"
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	cat, err := curriculum.LoadCatalog(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	failed := false
	for _, c := range cat.Courses {
		if err := tutor.ValidateAuthored(c); err != nil {
			fmt.Fprintf(os.Stderr, "course %q lesson steps:\n%v\n", c.Slug, err)
			failed = true
			continue
		}
		ready := 0
		for _, n := range c.Nodes {
			if n.Ready() {
				ready++
			}
		}
		fmt.Printf("OK %s: %d nodes, %d ready\n", c.Slug, len(c.Nodes), ready)
	}
	if failed {
		os.Exit(1)
	}
}
