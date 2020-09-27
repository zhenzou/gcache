package gcache

import (
	"fmt"
	"testing"
	"time"
)

func TestLRUGet(t *testing.T) {
	size := 1000
	gc := buildTestCache(t, TypeLru, size)
	testSetCache(t, gc, size)
	testCacheGet(t, gc, size)
}

func TestLoadingLRUGet(t *testing.T) {
	size := 1000
	gc := buildTestLoadingCache(t, TypeLru, size, loader)
	testLoadingCacheGet(t, gc, size)
}

func TestLRULength(t *testing.T) {
	gc := buildTestLoadingCache(t, TypeLru, 1000, loader)
	gc.Get(defaultCtx, "test1")
	gc.Get(defaultCtx, "test2")
	length := gc.Len(true)
	expectedLength := 2
	if length != expectedLength {
		t.Errorf("Expected length is %v, not %v", length, expectedLength)
	}
}

func TestLRUEvictItem(t *testing.T) {
	cacheSize := 10
	numbers := 11
	gc := buildTestLoadingCache(t, TypeLru, cacheSize, loader)

	for i := 0; i < numbers; i++ {
		_, err := gc.Get(defaultCtx, fmt.Sprintf("Key-%d", i))
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

func TestLRUGetIFPresent(t *testing.T) {
	testGetIFPresent(t, TypeLru)
}

func TestLRUHas(t *testing.T) {
	gc := buildTestLoadingCacheWithExpiration(t, TypeLru, 2, 10*time.Millisecond)

	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			gc.Get(defaultCtx, "test1")
			gc.Get(defaultCtx, "test2")

			if gc.Existed("test0") {
				t.Fatal("should not have test0")
			}
			if !gc.Existed("test1") {
				t.Fatal("should have test1")
			}
			if !gc.Existed("test2") {
				t.Fatal("should have test2")
			}

			time.Sleep(20 * time.Millisecond)

			if gc.Existed("test0") {
				t.Fatal("should not have test0")
			}
			if gc.Existed("test1") {
				t.Fatal("should not have test1")
			}
			if gc.Existed("test2") {
				t.Fatal("should not have test2")
			}
		})
	}
}
