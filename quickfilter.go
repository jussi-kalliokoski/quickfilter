// Package quickfilter provides a QuickFilter module for performing filtering
// of slices with minimal allocations. It works by allocating a bit array that
// stores the indices of the elements to be added to the resulting slice. This
// means that the total number of allocations for the filtering operation will
// be a static two (2), `len(sourceSlice)/8` bytes for the bit array and
// `len(resultSlice)*size_of_element` for the results array. This helps with
// providing predictable performance for your filter operations.
//
// QuickFilter should not be used blindly due to the readability overhead, and
// it may not even be the fastest option available. Usually it's more efficient
// to do the operation in-place if you can mutate the original slice. You can
// see an example of this in the benchmarks. You might also consider using
// resource pooling instead. As always, optimize when needed and benchmark to
// find the best solution.
//
// For reference, here are the benchmark results on a MacBook Pro (early 2015,
// 3.1GHz i7, 16GB memory) of filtering a large arbitrary payload:
//
//   goos: darwin
//   goarch: amd64
//   pkg: github.com/jussi-kalliokoski/quickfilter
//   Benchmark/QuickFilter-4                       30          45195222 ns/op        240249702 B/op         5 allocs/op
//   Benchmark/dynamic_allocations-4               10         104337103 ns/op        612835667 B/op        25 allocs/op
//   Benchmark/in_place-4                          30          39128758 ns/op        160161952 B/op         3 allocs/op
//
// As we can see, QuickFilter is more than twice as fast as using dynamic
// allocations, allocates almost a third of the memory, and is almost as fast
// as doing the operation in-place.
package quickfilter

import (
	"math/bits"
)

// QuickFilter is a utility module that stores offsets and allows you to
// iterate over them.
type QuickFilter struct {
	len       int
	sourceLen int
	bits      []uint
}

// New returns a new QuickFilter with enough space reserved to store sourceLen
// offsets.
//
// In a filtering operation, sourceLen should be the len() of the original
// slice.
func New(sourceLen int) QuickFilter {
	lastIndex, _ := offsets(sourceLen - 1)
	return QuickFilter{
		sourceLen: sourceLen,
		bits:      make([]uint, lastIndex+1),
	}
}

// NewFilled returns a new QuickFilter, like New, but instead of all items
// being left out by default, all are included in the filter result. This
// allows for doing multi-level filtering where the results from one filter
// function are passed to the next as a QuickFilter.
func NewFilled(sourceLen int) QuickFilter {
	return New(sourceLen).Fill()
}

// Add an index to the offset list.
//
// The original QuickFilter is no longer usable and must be replaced with the
// returned one. This approach prevents the QuickFilter from escaping to the
// heap.
func (qf QuickFilter) Add(index int) QuickFilter {
	index, mask := offsets(index)
	qf.bits[index] |= mask
	qf.len++
	return qf
}

// Delete an index from the offset list.
//
// The original QuickFilter is no longer usable and must be replaced with the
// returned one. This approach prevents the QuickFilter from escaping to the
// heap.
func (qf QuickFilter) Delete(index int) QuickFilter {
	index, mask := offsets(index)
	oldValue := qf.bits[index]
	if oldValue&mask == 0 {
		return qf
	}
	qf.bits[index] = oldValue ^ mask
	qf.len--
	return qf
}

// Len returns the number of offsets stored.
func (qf QuickFilter) Len() int {
	return qf.len
}

// Cap returns the maximum number of values that can be stored.
func (qf QuickFilter) Cap() int {
	return qf.sourceLen
}

// Clear the entries in the QuickFilter.
//
// The original QuickFilter is no longer usable and must be replaced with the
// returned one. This approach prevents the QuickFilter from escaping to the
// heap.
func (qf QuickFilter) Clear() QuickFilter {
	for i := range qf.bits {
		qf.bits[i] = 0
	}
	qf.len = 0
	return qf
}

// Fill the QuickFilter to contain all the entries in the source slice.
//
// The original QuickFilter is no longer usable and must be replaced with the
// returned one. This approach prevents the QuickFilter from escaping to the
// heap.
func (qf QuickFilter) Fill() QuickFilter {
	for i := 0; i < len(qf.bits); i++ {
		qf.bits[i] = ^uint(0)
	}
	qf.len = qf.sourceLen
	return qf
}

// Copy a QuickFilter with its results to a new QuickFilter.
func (qf QuickFilter) Copy() QuickFilter {
	return New(qf.Cap()).CopyFrom(qf)
}

// CopyFrom copies the set values from an existing QuickFilter.
//
// If the two QuickFilters are not of the same Cap(), the receiver will be
// resized.
//
// The original QuickFilter is no longer usable and must be replaced with the
// returned one. This approach prevents the QuickFilter from escaping to the
// heap.
func (qf QuickFilter) CopyFrom(qf2 QuickFilter) QuickFilter {
	qf = qf.Resize(qf2.Cap())
	qf.len = qf2.len
	copy(qf.bits, qf2.bits)
	return qf
}

// Resize a QuickFilter to a new source length. Will allocate a new backing
// buffer if the source length won't fit in the old one.
//
// The original QuickFilter is no longer usable and must be replaced with the
// returned one. This approach prevents the QuickFilter from escaping to the
// heap.
func (qf QuickFilter) Resize(sourceLen int) QuickFilter {
	lastIndex, _ := offsets(sourceLen - 1)
	bitsLen := lastIndex + 1
	qf.sourceLen = sourceLen
	if cap(qf.bits) < bitsLen {
		qf.bits = make([]uint, bitsLen)
	} else {
		qf.bits = qf.bits[:bitsLen]
	}
	return qf
}

// UnionOf fills the QuickFilter with the set values in one or both of the
// provided QuickFilters.
//
// The receiver and passed QuickFilters must all be the same size or this will
// panic.
//
// The original QuickFilter is no longer usable and must be replaced with the
// returned one. This approach prevents the QuickFilter from escaping to the
// heap.
func (qf QuickFilter) UnionOf(qf1, qf2 QuickFilter) QuickFilter {
	if len(qf.bits) != len(qf1.bits) || len(qf.bits) != len(qf2.bits) {
		panic("receiver and passed QuickFilters must be the same size")
	}
	qf.len = 0
	for i := range qf.bits {
		qf.bits[i] = qf1.bits[i] | qf2.bits[i]

		if i == len(qf.bits) - 1 {
			qf.len += onesCountLastWord(qf.bits[i], qf.sourceLen % bits.UintSize)
		} else {
			qf.len += bits.OnesCount(qf.bits[i])
		}
	}
	return qf
}

// IntersectionOf fills the QuickFilter with the set values in one or both of
// the provided QuickFilters.
//
// The receiver and passed QuickFilters must all be the same size or this will
// panic.
//
// The original QuickFilter is no longer usable and must be replaced with the
// returned one. This approach prevents the QuickFilter from escaping to the
// heap.
func (qf QuickFilter) IntersectionOf(qf1, qf2 QuickFilter) QuickFilter {
	if len(qf.bits) != len(qf1.bits) || len(qf.bits) != len(qf2.bits) {
		panic("receiver and passed QuickFilters must be the same size")
	}
	qf.len = 0
	for i := range qf.bits {
		qf.bits[i] = qf1.bits[i] & qf2.bits[i]

		if i == len(qf.bits) - 1 {
			qf.len += onesCountLastWord(qf.bits[i], qf.sourceLen % bits.UintSize)
		} else {
			qf.len += bits.OnesCount(qf.bits[i])
		}
	}
	return qf
}

// Has returns a boolean indicating whether the QuickFilter has the bit at
// given index set.
func (qf QuickFilter) Has(index int) bool {
	wordIndex, mask := offsets(index)
	return qf.bits[wordIndex]&mask > 0
}

// Iterate over the stored offsets.
func (qf QuickFilter) Iterate() Iterator {
	return Iterator{
		index:     -1,
		sourceLen: qf.sourceLen,
		bits:      qf.bits,
	}.Next()
}

// Iterator over the offsets of a QuickFilter.
type Iterator struct {
	index     int
	sourceLen int
	bits      []uint
}

// Done returns a boolean indicating whether the Iterator has been exhausted.
func (it Iterator) Done() bool {
	return it.index >= it.sourceLen
}

// Next returns the Iterator at the next offset.
func (it Iterator) Next() Iterator {
	it.index++
	for it.index < it.sourceLen {
		index, mask := offsets(it.index)
		if it.bits[index]&mask > 0 {
			return it
		}
		if it.bits[index] == 0 {
			// fast path for empty words
			index++
			it.index = index * bits.UintSize
			continue
		}
		it.index++
	}
	return it
}

// Value returns the currently found offset.
func (it Iterator) Value() int {
	return it.index
}

func offsets(pos int) (index int, mask uint) {
	return pos / bits.UintSize, 1 << (uint(pos) % bits.UintSize)
}

// onesCountLastWord returns the number of onces bits in the word
// taking into account the number of bits used.
//
// First we reverse the word and shift by the number of unused bits,
// then we reverse back to have the word with unused bits set to zeros.
func onesCountLastWord(word uint, usedBits int) int {
	return bits.OnesCount(
		bits.Reverse(
			bits.Reverse(word) << uint(bits.UintSize - usedBits),
		),
	)
}
