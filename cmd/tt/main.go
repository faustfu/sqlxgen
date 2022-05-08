package main

import (
	"flag"
	"fmt"
	"golang.org/x/tools/go/packages"
	"log"
	"os"
	"strings"
	"time"
)

var (
	packageNamesStr = flag.String("packageNames", "", "comma-separated list of package names; must be set")
)

func main() {
	flag.Parse()
	if len(*packageNamesStr) == 0 {
		flag.Usage()
		os.Exit(2)
	}
	log.Printf("args: %q\n", os.Args)

	packageNames := strings.Split(*packageNamesStr, ",")

	ps, err := loadPackages(packageNames, []string{})
	if err != nil {
		log.Fatal(err)
	}

	if len(ps) != 1 {
		log.Fatalf("error: %d packages found", len(ps))
	}

	p := *ps[0]
	log.Println(p)
}

func loadPackages(patterns []string, tags []string) ([]*packages.Package, error) {
	p := newProfile(fmt.Sprintf("loadPackages: %v", patterns))
	defer p.stop()

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles |
			packages.NeedImports |
			packages.NeedTypes |
			packages.NeedSyntax | packages.NeedTypesInfo |
			packages.NeedDeps,
		Tests:      false,
		BuildFlags: []string{fmt.Sprintf("-tags=%s", strings.Join(tags, " "))},
	}

	return packages.Load(cfg, patterns...)
}

type Profile struct {
	task      string
	startTime time.Time
}

func (p *Profile) stop() {
	du := time.Now().Sub(p.startTime)
	log.Printf("profile: %s: %s\n", p.task, du)
}

func newProfile(task string) *Profile {
	return &Profile{task: task, startTime: time.Now()}
}
