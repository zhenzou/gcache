package gcache

import (
	"fmt"
	"testing"
	"time"
)

func TestSimpleGet(t *testing.T) {
	size := 1000
	gc := buildTestCache(t, TypeSimple, size)
	testSetCache(t, gc, size)
	testCacheGet(t, gc, size)
}

func TestLoadingSimpleGet(t *testing.T) {
	size := 1000
	numbers := 1000
	testLoadingCacheGet(t, buildTestLoadingCache(t, TypeSimple, size, loader), numbers)
}

func TestSimpleLength(t *testing.T) {
	gc := buildTestLoadingCache(t, TypeSimple, 1000, loader)
	gc.Get(defaultCtx, "test1")
	gc.Get(defaultCtx, "test2")
	length := gc.Len(true)
	expectedLength := 2
	if length != expectedLength {
		t.Errorf("Expected length is %v, not %v", length, expectedLength)
	}
}

func TestSimpleEvictItem(t *testing.T) {
	cacheSize := 10
	numbers := 11
	gc := buildTestLoadingCache(t, TypeSimple, cacheSize, loader)

	for i := 0; i < numbers; i++ {
		_, err := gc.Get(defaultCtx, fmt.Sprintf("Key-%d", i))
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

func TestSimpleUnboundedNoEviction(t *testing.T) {
	numbers := 1000
	sizeTracker := 0
	gcu := buildTestLoadingCache(t, TypeSimple, 0, loader)

	for i := 0; i < numbers; i++ {
		currentSize := gcu.Len(true)
		if currentSize != sizeTracker {
			t.Errorf("Excepted cache size is %v not %v", currentSize, sizeTracker)
		}

		_, err := gcu.Get(defaultCtx, fmt.Sprintf("Key-%d", i))
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		sizeTracker++
	}
}

func TestSimpleGetIFPresent(t *testing.T) {
	testGetIFPresent(t, TypeSimple)
}

func TestSimpleHas(t *testing.T) {
	gc := buildTestLoadingCacheWithExpiration(t, TypeSimple, 2, 10*time.Millisecond)

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
