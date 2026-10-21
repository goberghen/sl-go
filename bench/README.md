# Measuring sl

Two harnesses, because "faster" is two different questions.

## 1. In-process: `make bench`

`src/bench_test.go` runs the pipeline with `io.Discard` as the output, so no
process start and no terminal are involved. This is where a change to a matcher
or to `scan.go` shows up without noise.

The two families:

* `BenchmarkList*` — the whole listing of a real directory:
  `/usr/share/doc` (many small trees), `/usr/include` (deep, many headers),
  `/etc`, `/usr/share/man` (every name carries a wrapper suffix), `/usr/bin`
  (thousands of extensionless names), `/usr/lib`.
* `BenchmarkMatcher*` versus `BenchmarkMatcher*Regexp` — every hand-written
  matcher next to the regular expression it replaced, over a fixed 1200-name
  corpus built from the names that actually occur in package directories.
  Keeping the old pattern in the tree is deliberate: a fast path that is no
  longer fast should fail its own comparison.

Run with `-count 5` and look at the median; single runs on a shared machine
move by 10% either way.

## 2. Wall clock: `make bench-ab A=old/sl B=new/sl DIRS="/usr/bin /etc"`

`bench/bench.go` execs two real binaries, alternating them run by run, and
prints medians. Alternating is the whole point: a sequential "A a hundred
times, then B a hundred times" comparison on a machine that is doing something
else in the background invents changes that are not there. Process start
(about 1.4 ms, ~1.0 ms of it fork/exec) is inside these numbers, which is why
they are larger than the in-process ones; for a utility that is started by a
shell each time, that offset is part of the cost the user feels.

Use identical flags for both binaries (`-flags`, default `-1`).

## What the numbers are not

* They are for one machine (2 vCPU container, warm page cache). Directory
  contents differ wildly between systems; the shape of the result is what
  travels, not the microseconds.
* `BenchmarkList*` deliberately does not include `--annotate` (an external
  process per entry) and does not touch a terminal.
* The corpus is fixed, so a matcher benchmark cannot notice that a new matcher
  became quadratic in name length. The pipeline benchmarks can.

## Recorded results

`bench/results/` holds the raw output of the runs quoted in `VERDIKT.md`:

| file | what |
| --- | --- |
| `pipeline.txt` | `BenchmarkList*`, 1.1.4 after the round-eight changes |
| `matchers.txt` | matcher versus regexp, per 1200 names |
| `ab-one-stat.txt` | wall clock, before/after the one-`stat` fix |

Median of the matcher runs, ns per name (1200-name corpus, the mix found in package directories):

| matcher | strings | regexp | ratio |
| --- | ---: | ---: | ---: |
| `notableName` | 49 | 1090 | 22× |
| `ignorableName` | 14 | 444 | 31× |
| `devName` | 23 | 446 | 19× |
| `devDirName` | 34 | 922 | 27× |
| `manPageName` | 20 | 229 | 12× |
| `wrapperName` | 12 | 124 | 10× |
| `autoBkupBase` | 8 | 710 | 89× |
| `autosearchSpan` | 21 | 308 | 15× |
| `normalizeKey` | 141 | 2069 | 15× |

The end-to-end effect is smaller than these ratios suggest, because most names
never reach the expensive matchers (the suffix table answers first) and the
time is dominated by `readdir` and `stat`.
