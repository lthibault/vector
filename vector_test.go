package vector_test

import (
	"testing"

	"github.com/lthibault/vector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVector(t *testing.T) {
	t.Parallel()
	t.Helper()

	const n = 4096
	var v vector.Vector[int]

	t.Run("ZeroValue", func(t *testing.T) {
		assert.Zero(t, v.Len(), "zero-value vector should have zero length")
		assert.Zero(t, v.Pop(), "popping empty vector should return zero-value vector")
	})

	t.Run("Append", func(t *testing.T) {
		for i := 0; i < n; i++ {
			v = v.Append(i)
		}

		require.NotNil(t, v, "vector should be non-nil")
		require.Equal(t, n, v.Len(), "should contain %d elements", n)
		require.Zero(t, v.At(0), "first element should be zero")
		require.Equal(t, n-1, v.At(n-1), "last element should be %d", n-1)

		v2 := v.Append()
		assert.Equal(t, v, v2, "append with no args should no-op")
	})

	t.Run("Pop", func(t *testing.T) {
		for i := n - 1; i >= 0; i-- {
			v = v.Pop()
			require.Equal(t, i, v.Len())
		}

		require.Zero(t, v, "should be zero-value vector")
	})
}

func TestBulkAppend(t *testing.T) {
	t.Parallel()

	var v vector.Vector[int]
	v = v.Append(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	assert.Equal(t, 10, v.Len(), "should bulk-insert 10 elements")
}

func TestGetSet(t *testing.T) {
	t.Parallel()
	t.Helper()

	const n = 4096

	is := make([]int, n)
	for i := range is {
		is[i] = i
	}

	v := vector.New(is...)

	t.Run("Overwrite", func(t *testing.T) {
		for i := 0; i < n; i++ {
			v = v.Set(i, -i)
		}

		for i := 0; i < n; i++ {
			assert.True(t, v.At(i) <= 0, "value should be overwritten")
		}
	})

	t.Run("Append", func(t *testing.T) {
		v2 := v.Set(n, -1)

		assert.NotEqual(t, v, v2, "should not mutate v")
		assert.Equal(t, n+1, v2.Len(), "should not append to vector")
		assert.Equal(t, -1, v2.At(n), "should return appended value")
	})

	t.Run("OutOfBounds", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() { v.At(9001) },
			"should panic when out of bounds")
		assert.Panics(t, func() { v.At(-1) },
			"should panic when out of bounds")
		assert.Panics(t, func() { v.Set(9001, 9001) },
			"should panic when out of bounds")
		assert.Panics(t, func() { v.Set(-1, 9001) },
			"should panic when out of bounds")
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	const n = 4096

	is := make([]int, n)
	for i := range is {
		is[i] = i
	}

	v := vector.New(is...)
	assert.Equal(t, n, v.Len(), "should have length of %d", n)

	for i := 0; i < n; i++ {
		assert.Equal(t, i, v.At(i))
	}
}

func TestBuilder(t *testing.T) {
	t.Parallel()
	t.Helper()

	const n = 4096
	b := vector.NewBuilder[int]()

	t.Run("ZeroValue", func(t *testing.T) {
		assert.Zero(t, b.Len(), "zero-value vector should have zero length")
	})

	t.Run("Append", func(t *testing.T) {
		for i := 0; i < n; i++ {
			b.Append(i)
		}

		require.NotNil(t, b, "vector should be non-nil")
		require.Equal(t, n, b.Len(), "should contain %d elements", n)
		require.Zero(t, b.Vector().At(0), "first element should be zero")
		require.Equal(t, n-1, b.Vector().At(n-1), "last element should be %d", n-1)
	})

	t.Run("Pop", func(t *testing.T) {
		v := b.Vector()

		for i := n - 1; i >= 0; i-- {
			v = v.Pop()
			require.Equal(t, i, v.Len())
		}

		require.Zero(t, v, "should be zero-value vector")
	})
}
