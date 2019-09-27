package quickfilter_test

import (
	"fmt"
	"testing"

	"github.com/jussi-kalliokoski/quickfilter"
)

func Test(t *testing.T) {
	t.Run("Add and Iterate", func(t *testing.T) {
		data := generateData(20)
		qf := quickfilter.New(len(data))

		for i := range data {
			if data[i].index%2 == 0 {
				qf = qf.Add(i)
			}
		}
		newData := make([]mockData, 0, qf.Len())
		for it := qf.Iterate(); !it.Done(); it = it.Next() {
			newData = append(newData, data[it.Value()])
		}

		validate(t, len(data), newData)
	})

	t.Run("Fill and Iterate", func(t *testing.T) {
		data := generateData(20)
		qf := quickfilter.New(len(data))
		expectedLen := len(data)

		qf = qf.Fill()
		newData := make([]mockData, 0, qf.Len())
		for it := qf.Iterate(); !it.Done(); it = it.Next() {
			newData = append(newData, data[it.Value()])
		}
		receivedLen := len(newData)

		if expectedLen != receivedLen {
			t.Errorf("expected %d, got %d", expectedLen, receivedLen)
		}
	})

	t.Run("Clear and Iterate", func(t *testing.T) {
		data := generateData(20)
		qf := quickfilter.New(len(data))
		expectedLen := 0

		for i := range data {
			if data[i].index%2 == 0 {
				qf = qf.Add(i)
			}
		}
		qf = qf.Clear()
		newData := make([]mockData, 0, qf.Len())
		for it := qf.Iterate(); !it.Done(); it = it.Next() {
			newData = append(newData, data[it.Value()])
		}
		receivedLen := len(newData)

		if expectedLen != receivedLen {
			t.Errorf("expected %d, got %d", expectedLen, receivedLen)
		}
	})

	t.Run("Fill and Copy", func(t *testing.T) {
		data := generateData(20)
		qf := quickfilter.NewFilled(len(data))
		expectedLen := len(data)

		qf2 := qf.Copy()
		newData := make([]mockData, 0, qf.Len())
		for it := qf2.Iterate(); !it.Done(); it = it.Next() {
			newData = append(newData, data[it.Value()])
		}
		receivedLen := len(newData)

		if expectedLen != receivedLen {
			t.Errorf("expected %d, got %d", expectedLen, receivedLen)
		}
	})
}

func Example() {
	data := make([]int, 0, 8)
	for len(data) < cap(data) {
		data = append(data, len(data))
	}
	qf := quickfilter.New(len(data))
	for i := range data {
		if data[i]%2 == 0 {
			qf = qf.Add(i)
		}
	}
	newData := make([]int, 0, qf.Len())
	for it := qf.Iterate(); !it.Done(); it = it.Next() {
		newData = append(newData, data[it.Value()])
	}
	// Output: [0 2 4 6]
	fmt.Println(newData)
}

func Example_union() {
	data := make([]int, 0, 16)
	for len(data) < cap(data) {
		data = append(data, len(data))
	}
	qf1 := quickfilter.New(len(data))
	for i := range data {
		if data[i]%2 == 0 {
			qf1 = qf1.Add(i)
		}
	}
	qf2 := quickfilter.New(len(data))
	for i := range data {
		if data[i]%3 == 0 {
			qf2 = qf2.Add(i)
		}
	}
	qf := quickfilter.New(len(data))
	qf = qf.UnionOf(qf1, qf2)
	newData := make([]int, 0, qf.Len())
	for it := qf.Iterate(); !it.Done(); it = it.Next() {
		newData = append(newData, data[it.Value()])
	}
	// Output: [0 2 3 4 6 8 9 10 12 14 15]
	fmt.Println(newData)
}

func Benchmark(b *testing.B) {
	const size = 20000

	b.Run("QuickFilter", func(b *testing.B) {
		b.ReportAllocs()
		for n := 0; n < b.N; n++ {
			data := generateData(size)
			qf := quickfilter.New(len(data))
			for i := range data {
				if data[i].index%2 == 0 {
					qf = qf.Add(i)
				}
			}
			newData := make([]mockData, 0, qf.Len())
			for it := qf.Iterate(); !it.Done(); it = it.Next() {
				newData = append(newData, data[it.Value()])
			}
			validate(b, size, newData)
		}
	})

	b.Run("dynamic allocations", func(b *testing.B) {
		b.ReportAllocs()
		for n := 0; n < b.N; n++ {
			data := generateData(size)
			newData := make([]mockData, 0)
			for i := range data {
				if data[i].index%2 == 0 {
					newData = append(newData, data[i])
				}
			}
			validate(b, size, newData)
		}
	})

	b.Run("in place", func(b *testing.B) {
		b.ReportAllocs()
		for n := 0; n < b.N; n++ {
			data := generateData(size)
			offset := 0
			for i := range data {
				if data[i].index%2 == 0 {
					data[i-offset] = data[i]
				} else {
					offset++
				}
			}
			data = data[:len(data)-offset]
			validate(b, size, data)
		}
	})

	b.Run("in place copied", func(b *testing.B) {
		b.ReportAllocs()
		for n := 0; n < b.N; n++ {
			data := generateData(size)
			newData := make([]mockData, len(data))
			copy(newData, data)
			offset := 0
			for i := range newData {
				if newData[i].index%2 == 0 {
					newData[i-offset] = newData[i]
				} else {
					offset++
				}
			}
			newData = newData[:len(newData)-offset]
			validate(b, size, newData)
		}
	})
}

type mockData struct {
	index int
	trash [1000]int
}

func generateData(dataLen int) []mockData {
	data := make([]mockData, 0, dataLen)
	for i := 0; i < dataLen; i++ {
		data = append(data, mockData{index: i})
		_ = data[i].trash
	}
	return data
}

func validate(tb testing.TB, oldLen int, newData []mockData) {
	tb.Helper()
	expectedLen := oldLen / 2
	receivedLen := len(newData)
	if expectedLen != receivedLen {
		tb.Fatalf("expected length %d, received %d", expectedLen, receivedLen)
	}
	for i := range newData {
		if newData[i].index%2 != 0 {
			tb.Fatalf("unexpected index %d", newData[i].index)
		}
	}
}
