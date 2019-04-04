package quickfilter

import "testing"

func Test(t *testing.T) {
	t.Run("Add and Iterate", func(t *testing.T) {
		data := generateData(20)
		qf := New(len(data))

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
		qf := New(len(data))
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
		qf := New(len(data))
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
		qf := NewFilled(len(data))
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
	var data []int
	qf := New(len(data))
	for i := range data {
		if data[i]%2 == 0 {
			qf = qf.Add(i)
		}
	}
	newData := make([]int, 0, qf.Len())
	for it := qf.Iterate(); !it.Done(); it = it.Next() {
		newData = append(newData, data[it.Value()])
	}
}

func Benchmark(b *testing.B) {
	const size = 20000

	b.Run("QuickFilter", func(b *testing.B) {
		b.ReportAllocs()
		for n := 0; n < b.N; n++ {
			data := generateData(size)
			qf := New(len(data))
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
