// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copied from: https://github.com/golang/tools/blob/master/cmd/benchcmp/compare.go#L21:6 (cannot import cmd/)
package main

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"text/tabwriter"

	"golang.org/x/tools/benchmark/parse"
)

// BenchCmp is a pair of benchmarks.
type BenchCmp struct {
	Before *parse.Benchmark
	After  *parse.Benchmark
}

// Correlate correlates benchmarks from two BenchSets.
func Correlate(before, after parse.Set) (cmps []BenchCmp, warnings []string) {
	cmps = make([]BenchCmp, 0, len(after))
	for name, beforebb := range before {
		afterbb := after[name]
		if len(beforebb) != len(afterbb) {
			warnings = append(warnings, fmt.Sprintf("ignoring %s: before has %d instances, after has %d", name, len(beforebb), len(afterbb)))
			continue
		}
		for i, beforeb := range beforebb {
			afterb := afterbb[i]
			cmps = append(cmps, BenchCmp{Before: beforeb, After: afterb})
		}
	}
	return
}

func (c BenchCmp) Name() string           { return c.Before.Name }
func (c BenchCmp) String() string         { return fmt.Sprintf("<%s, %s>", c.Before, c.After) }
func (c BenchCmp) Measured(flag int) bool { return (c.Before.Measured & c.After.Measured & flag) != 0 }
func (c BenchCmp) DeltaNsPerOp() Delta    { return Delta{c.Before.NsPerOp, c.After.NsPerOp} }
func (c BenchCmp) DeltaMBPerS() Delta     { return Delta{c.Before.MBPerS, c.After.MBPerS} }
func (c BenchCmp) DeltaAllocedBytesPerOp() Delta {
	return Delta{float64(c.Before.AllocedBytesPerOp), float64(c.After.AllocedBytesPerOp)}
}
func (c BenchCmp) DeltaAllocsPerOp() Delta {
	return Delta{float64(c.Before.AllocsPerOp), float64(c.After.AllocsPerOp)}
}

// Delta is the before and after value for a benchmark measurement.
// Both must be non-negative.
type Delta struct {
	Before float64
	After  float64
}

// mag calculates the magnitude of a change, regardless of the direction of
// the change. mag is intended for sorting and has no independent meaning.
func (d Delta) mag() float64 {
	switch {
	case d.Before != 0 && d.After != 0 && d.Before >= d.After:
		return d.After / d.Before
	case d.Before != 0 && d.After != 0 && d.Before < d.After:
		return d.Before / d.After
	case d.Before == 0 && d.After == 0:
		return 1
	default:
		// 0 -> 1 or 1 -> 0
		// These are significant changes and worth surfacing.
		return math.Inf(1)
	}
}

// Changed reports whether the benchmark quantities are different.
func (d Delta) Changed() bool { return d.Before != d.After }

// Float64 returns After / Before. If Before is 0, Float64 returns
// 1 if After is also 0, and +Inf otherwise.
func (d Delta) Float64() float64 {
	switch {
	case d.Before != 0:
		return d.After / d.Before
	case d.After == 0:
		return 1
	default:
		return math.Inf(1)
	}
}

// Percent formats a Delta as a percent change, ranging from -100% up.
func (d Delta) Percent() string {
	return fmt.Sprintf("%+.2f%%", 100*d.Float64()-100)
}

// Multiple formats a Delta as a multiplier, ranging from 0.00x up.
func (d Delta) Multiple() string {
	return fmt.Sprintf("%.2fx", d.Float64())
}

func (d Delta) String() string {
	return fmt.Sprintf("Δ(%f, %f)", d.Before, d.After)
}

// ByParseOrder sorts BenchCmps to match the order in
// which the Before benchmarks were presented to Parse.
type ByParseOrder []BenchCmp

func (x ByParseOrder) Len() int           { return len(x) }
func (x ByParseOrder) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }
func (x ByParseOrder) Less(i, j int) bool { return x[i].Before.Ord < x[j].Before.Ord }

// lessByDelta provides lexicographic ordering:
//   * largest delta by magnitude
//   * alphabetic by name
func lessByDelta(i, j BenchCmp, calcDelta func(BenchCmp) Delta) bool {
	iDelta, jDelta := calcDelta(i).mag(), calcDelta(j).mag()
	if iDelta != jDelta {
		return iDelta < jDelta
	}
	return i.Name() < j.Name()
}

// ByDeltaNsPerOp sorts BenchCmps lexicographically by change
// in ns/op, descending, then by benchmark name.
type ByDeltaNsPerOp []BenchCmp

func (x ByDeltaNsPerOp) Len() int           { return len(x) }
func (x ByDeltaNsPerOp) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }
func (x ByDeltaNsPerOp) Less(i, j int) bool { return lessByDelta(x[i], x[j], BenchCmp.DeltaNsPerOp) }

// ByDeltaMBPerS sorts BenchCmps lexicographically by change
// in MB/s, descending, then by benchmark name.
type ByDeltaMBPerS []BenchCmp

func (x ByDeltaMBPerS) Len() int           { return len(x) }
func (x ByDeltaMBPerS) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }
func (x ByDeltaMBPerS) Less(i, j int) bool { return lessByDelta(x[i], x[j], BenchCmp.DeltaMBPerS) }

// ByDeltaAllocedBytesPerOp sorts BenchCmps lexicographically by change
// in B/op, descending, then by benchmark name.
type ByDeltaAllocedBytesPerOp []BenchCmp

func (x ByDeltaAllocedBytesPerOp) Len() int      { return len(x) }
func (x ByDeltaAllocedBytesPerOp) Swap(i, j int) { x[i], x[j] = x[j], x[i] }
func (x ByDeltaAllocedBytesPerOp) Less(i, j int) bool {
	return lessByDelta(x[i], x[j], BenchCmp.DeltaAllocedBytesPerOp)
}

// ByDeltaAllocsPerOp sorts BenchCmps lexicographically by change
// in allocs/op, descending, then by benchmark name.
type ByDeltaAllocsPerOp []BenchCmp

func (x ByDeltaAllocsPerOp) Len() int      { return len(x) }
func (x ByDeltaAllocsPerOp) Swap(i, j int) { x[i], x[j] = x[j], x[i] }
func (x ByDeltaAllocsPerOp) Less(i, j int) bool {
	return lessByDelta(x[i], x[j], BenchCmp.DeltaAllocsPerOp)
}

// formatNs formats ns measurements to expose a useful amount of
// precision. It mirrors the ns precision logic of testing.B.
func formatNs(ns float64) string {
	prec := 0
	switch {
	case ns < 10:
		prec = 2
	case ns < 100:
		prec = 1
	}
	return strconv.FormatFloat(ns, 'f', prec, 64)
}

// Copied from https://github.com/golang/tools/blob/master/cmd/benchcmp/benchcmp.go.
//
// flag.Bool("changed", false, "show only benchmarks that have changed")
// flag.Bool("mag", false, "sort benchmarks by magnitude of change")
func Render(wrt io.Writer, cmps []BenchCmp, magSort, changedOnly bool, oldHeader string) {
	w := new(tabwriter.Writer)
	w.Init(wrt, 0, 0, 5, ' ', 0)
	defer w.Flush()

	var header bool // Has the header has been displayed yet for a given block?

	if magSort {
		sort.Sort(ByDeltaNsPerOp(cmps))
	} else {
		sort.Sort(ByParseOrder(cmps))
	}
	for _, cmp := range cmps {
		if !cmp.Measured(parse.NsPerOp) {
			continue
		}
		if delta := cmp.DeltaNsPerOp(); !changedOnly || delta.Changed() {
			if !header {
				fmt.Fprintf(w, "benchmark\t%s ns/op\tnew ns/op\tdelta\n", oldHeader)
				header = true
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", cmp.Name(), formatNs(cmp.Before.NsPerOp), formatNs(cmp.After.NsPerOp), delta.Percent())
		}
	}

	header = false
	if magSort {
		sort.Sort(ByDeltaMBPerS(cmps))
	}
	for _, cmp := range cmps {
		if !cmp.Measured(parse.MBPerS) {
			continue
		}
		if delta := cmp.DeltaMBPerS(); !changedOnly || delta.Changed() {
			if !header {
				fmt.Fprintf(w, "\nbenchmark\t%s MB/s\tnew MB/s\tspeedup\n", oldHeader)
				header = true
			}
			fmt.Fprintf(w, "%s\t%.2f\t%.2f\t%s\n", cmp.Name(), cmp.Before.MBPerS, cmp.After.MBPerS, delta.Multiple())
		}
	}

	header = false
	if magSort {
		sort.Sort(ByDeltaAllocsPerOp(cmps))
	}
	for _, cmp := range cmps {
		if !cmp.Measured(parse.AllocsPerOp) {
			continue
		}
		if delta := cmp.DeltaAllocsPerOp(); !changedOnly || delta.Changed() {
			if !header {
				fmt.Fprintf(w, "\nbenchmark\t%s allocs\tnew allocs\tdelta\n", oldHeader)
				header = true
			}
			fmt.Fprintf(w, "%s\t%d\t%d\t%s\n", cmp.Name(), cmp.Before.AllocsPerOp, cmp.After.AllocsPerOp, delta.Percent())
		}
	}

	header = false
	if magSort {
		sort.Sort(ByDeltaAllocedBytesPerOp(cmps))
	}
	for _, cmp := range cmps {
		if !cmp.Measured(parse.AllocedBytesPerOp) {
			continue
		}
		if delta := cmp.DeltaAllocedBytesPerOp(); !changedOnly || delta.Changed() {
			if !header {
				fmt.Fprintf(w, "\nbenchmark\t%s bytes\tnew bytes\tdelta\n", oldHeader)
				header = true
			}
			fmt.Fprintf(w, "%s\t%d\t%d\t%s\n", cmp.Name(), cmp.Before.AllocedBytesPerOp, cmp.After.AllocedBytesPerOp, cmp.DeltaAllocedBytesPerOp().Percent())
		}
	}
}
