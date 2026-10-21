// Command bench times two sl binaries against the same directories, with the
// runs interleaved, and prints medians.
//
//	go run ./bench -a old/sl -b release/bin/sl /usr/bin /etc
//
// Interleaving matters: a machine that is compiling in the background drifts by
// tens of percent between one finished sweep and the next, and a sequential
// "run A a hundred times, then B a hundred times" comparison invents changes
// that are not there. Alternating the two binaries keeps the drift common to
// both.
//
// Timings are wall clock from exec to exit, so process start (about 1.4 ms, of
// which ~1.0 ms is fork/exec) is inside the numbers. For hot-path measurements
// without that offset use the in-package benchmarks instead: make bench.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"time"
)

type result struct{ d time.Duration }

func median(xs []time.Duration) time.Duration {
	s := append([]time.Duration(nil), xs...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	if len(s) == 0 {
		return 0
	}
	if len(s)%2 == 1 {
		return s[len(s)/2]
	}
	return (s[len(s)/2-1] + s[len(s)/2]) / 2
}

func run(bin string, args []string) (time.Duration, error) {
	cmd := exec.Command(bin, args...)
	cmd.Stdout = nil // /dev/null: we measure the listing, not the terminal
	cmd.Stderr = os.Stderr
	t0 := time.Now()
	err := cmd.Run()
	return time.Since(t0), err
}

func main() {
	a := flag.String("a", "", "baseline binary")
	b := flag.String("b", "", "candidate binary")
	n := flag.Int("n", 30, "samples per binary per directory")
	flags := flag.String("flags", "-1", "flags to pass to both binaries")
	flag.Parse()
	dirs := flag.Args()
	if *a == "" || len(dirs) == 0 {
		fmt.Fprintln(os.Stderr, "usage: bench -a BASE [-b NEW] [-flags \"-1\"] DIRECTORY...")
		os.Exit(2)
	}
	args := []string{"--no-color"}
	args = append(args, splitFlags(*flags)...)

	fmt.Printf("%-20s %9s %9s %9s %6s\n", "directory", "base", "new", "delta", "runs")
	for _, d := range dirs {
		args := append(args, d)
		base := make([]time.Duration, 0, *n)
		var cand []time.Duration
		for i := 0; i < *n; i++ {
			t, err := run(*a, args)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s %s: %v\n", *a, d, err)
				os.Exit(1)
			}
			base = append(base, t)
			if *b != "" {
				t, err := run(*b, args)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s %s: %v\n", *b, d, err)
					os.Exit(1)
				}
				cand = append(cand, t)
			}
		}
		mb := median(base)
		if *b == "" {
			fmt.Printf("%-20s %9.2fms\n", d, float64(mb.Microseconds())/1000)
			continue
		}
		mn := median(cand)
		fmt.Printf("%-20s %8.2fms %8.2fms %+7.1f%% %8d\n",
			d, float64(mb.Microseconds())/1000, float64(mn.Microseconds())/1000,
			(float64(mn)/float64(mb)-1)*100, *n)
	}
}

// splitFlags cuts a flag string on spaces; quoting is not needed for the flags
// used here and pretending to implement a shell would be worse than this.
func splitFlags(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ' ' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
