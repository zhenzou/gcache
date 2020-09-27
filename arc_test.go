package gcache

import (
	"fmt"
	"testing"
	"time"
)

func TestARCGet(t *testing.T) {
	size := 1000
	gc := buildTestCache(t, TypeArc, size)
	testSetCache(t, gc, size)
	testCacheGet(t, gc, size)
}

func TestLoadingARCGet(t *testing.T) {
	size := 1000
	numbers := 1000
	testLoadingCacheGet(t, buildTestLoadingCache(t, TypeArc, size, loader), numbers)
}

func TestARCLength(t *testing.T) {
	gc := buildTestLoadingCacheWithExpiration(t, TypeArc, 2, time.Millisecond)
	gc.Get(defaultCtx, "test1")
	gc.Get(defaultCtx, "test2")
	gc.Get(defaultCtx, "test3")
	length := gc.Len(true)
	expectedLength := 2
	if length != expectedLength {
		t.Errorf("Expected length is %v, not %v", expectedLength, length)
	}
	time.Sleep(time.Millisecond)
	gc.Get(defaultCtx, "test4")
	length = gc.Len(true)
	expectedLength = 1
	if length != expectedLength {
		t.Errorf("Expected length is %v, not %v", expectedLength, length)
	}
}

func TestARCEvictItem(t *testing.T) {
	cacheSize := 10
	numbers := cacheSize + 1
	gc := buildTestLoadingCache(t, TypeArc, cacheSize, loader)

	for i := 0; i < numbers; i++ {
		_, err := gc.Get(defaultCtx, fmt.Sprintf("Key-%d", i))
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

func TestARCPurgeCache(t *testing.T) {
	cacheSize := 10
	purgeCount := 0
	gc := New(cacheSize).
		ARC().
		LoaderFunc(loader).
		PurgeVisitorFunc(func(k, v interface{}) {
			purgeCount++
		}).
		Build()

	for i := 0; i < cacheSize; i++ {
		_, err := gc.Get(defaultCtx, fmt.Sprintf("Key-%d", i))
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	}

	gc.Purge()

	if purgeCount != cacheSize {
		t.Errorf("failed to purge everything")
	}
}

func TestARCGetIFPresent(t *testing.T) {
	testGetIFPresent(t, TypeArc)
}

func TestARCHas(t *testing.T) {
	gc := buildTestLoadingCacheWithExpiration(t, TypeArc, 2, 10*time.Millisecond)

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
