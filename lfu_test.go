package gcache

import (
	"fmt"
	"testing"
	"time"
)

func TestLFUGet(t *testing.T) {
	size := 1000
	numbers := 1000

	gc := buildTestLoadingCache(t, TypeLfu, size, loader)
	testSetCache(t, gc, numbers)
	testCacheGet(t, gc, numbers)
}

func TestLoadingLFUGet(t *testing.T) {
	size := 1000
	numbers := 1000

	gc := buildTestLoadingCache(t, TypeLfu, size, loader)
	testLoadingCacheGet(t, gc, numbers)
}

func TestLFULength(t *testing.T) {
	gc := buildTestLoadingCache(t, TypeLfu, 1000, loader)
	gc.Get(defaultCtx, "test1")
	gc.Get(defaultCtx, "test2")
	length := gc.Len(true)
	expectedLength := 2
	if length != expectedLength {
		t.Errorf("Expected length is %v, not %v", length, expectedLength)
	}
}

func TestLFUEvictItem(t *testing.T) {
	cacheSize := 10
	numbers := 11
	gc := buildTestLoadingCache(t, TypeLfu, cacheSize, loader)

	for i := 0; i < numbers; i++ {
		_, err := gc.Get(defaultCtx, fmt.Sprintf("Key-%d", i))
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

func TestLFUGetIFPresent(t *testing.T) {
	testGetIFPresent(t, TypeLfu)
}

func TestLFUHas(t *testing.T) {
	gc := buildTestLoadingCacheWithExpiration(t, TypeLfu, 2, 10*time.Millisecond)

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
