# GCache

Cache library for golang. It supports expirable Cache, LFU, LRU and ARC.


[![go report card](https://goreportcard.com/badge/github.com/zhenzou/gcache "go report card")](https://goreportcard.com/report/github.com/zhenzou/gcache)
[![test status](https://github.com/zhenzou/gcache/workflows/tests/badge.svg?branch=master "test status")](https://github.com/go-gorm/gorm/actions)
[![MIT license](https://img.shields.io/badge/license-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)
[![Go.Dev reference](https://img.shields.io/badge/go.dev-reference-blue?logo=go&logoColor=white)](https://pkg.go.dev/bluele/gcache?tab=doc)

## Features

* Supports expirable Cache, LFU, LRU and ARC.

* Goroutine safe.

* Supports event handlers which evict, purge, and add entry. (Optional)

* Automatically load cache if it doesn't exists. (Optional)

## Install

```
$ go get github.com/bluele/gcache
```

## Example

### Manually set a key-value pair.

```go
package main

import (
  "context"
"fmt"

  "github.com/bluele/gcache"
)

func main() {
  gc := gcache.New(20).
    LRU().
    Build()
  _ = gc.Set("key", "ok")
  value, err := gc.Get(context.Background(),"key")
  if err != nil {
    panic(err)
  }
  fmt.Println("Get:", value)
}
```

```
Get: ok
```

### Manually set a key-value pair, with an expiration time.

```go
package main

import (
  "context"
  "fmt"
  "time"

  "github.com/bluele/gcache"
)

func main() {
  gc := gcache.New(20).
    LRU().
    Build()
  gc.SetWithExpire("key", "ok", time.Second*10)
  value, _ := gc.Get(context.Background(),"key")
  fmt.Println("Get:", value)

  // Wait for value to expire
  time.Sleep(time.Second*10)

  value, err = gc.Get(context.Background(),"key")
  if err != nil {
    panic(err)
  }
  fmt.Println("Get:", value)
}
```

```
Get: ok
// 10 seconds later, new attempt:
panic: ErrKeyNotFound
```


### Automatically load value

```go
package main

import (
  "context"
  "fmt"

  "github.com/bluele/gcache"
)

func main() {
  gc := gcache.New(20).
    LRU().
    LoaderFunc(func(ctx context.Context,key interface{}) (interface{}, error) {
      return "ok", nil
    }).
    Build()
  value, err := gc.Get(context.Background(),"key")
  if err != nil {
    panic(err)
  }
  fmt.Println("Get:", value)
}
```

```
Get: ok
```

### Automatically load value with expiration

```go
package main

import (
  "context"
  "fmt"
  "time"

  "github.com/bluele/gcache"
)

func main() {
  var evictCounter, loaderCounter, purgeCounter int
  gc := gcache.New(20).
    LRU().
    LoaderExpireFunc(func(ctx context.Context,key interface{}) (interface{}, *time.Duration, error) {
      loaderCounter++
      expire := 1 * time.Second
      return "ok", &expire, nil
    }).
    EvictedFunc(func(key, value interface{}) {
      evictCounter++
      fmt.Println("evicted key:", key)
    }).
    PurgeVisitorFunc(func(key, value interface{}) {
      purgeCounter++
      fmt.Println("purged key:", key)
    }).
    Build()
  value, err := gc.Get(context.Background(),"key")
  if err != nil {
    panic(err)
  }
  fmt.Println("Get:", value)
  time.Sleep(1 * time.Second)
  value, err = gc.Get(context.Background(),"key")
  if err != nil {
    panic(err)
  }
  fmt.Println("Get:", value)
  gc.Purge()
  if loaderCounter != evictCounter+purgeCounter {
    panic("bad")
  }
}
```

```
Get: ok
evicted key: key
Get: ok
purged key: key
```


## Cache Algorithm

  * Least-Frequently Used (LFU)

  Discards the least frequently used items first.

  ```go
  func main() {
    // size: 10
    gc := gcache.New(10).
      LFU().
      Build()
    gc.Set("key", "value")
  }
  ```

  * Least Recently Used (LRU)

  Discards the least recently used items first.

  ```go
  func main() {
    // size: 10
    gc := gcache.New(10).
      LRU().
      Build()
    gc.Set("key", "value")
  }
  ```

  * Adaptive Replacement Cache (ARC)

  Constantly balances between LRU and LFU, to improve the combined result.

  detail: http://en.wikipedia.org/wiki/Adaptive_replacement_cache

  ```go
  func main() {
    // size: 10
    gc := gcache.New(10).
      ARC().
      Build()
    gc.Set("key", "value")
  }
  ```

  * SimpleCache (Default)

  SimpleCache has no clear priority for evict cache. It depends on key-value map order.

```go
package main
import (
	"context"
)

func main() {
  // size: 10
  gc := gcache.New(10).Build()
  gc.Set("key", "value")
  _, err := gc.Get(context.Background(),"key")
  if err != nil {
    panic(err)
  }
}
```

## Loading Cache

If specified `LoaderFunc`, values are automatically loaded by the cache, and are stored in the cache until either evicted or manually invalidated.

```go
package main
import (
  "context"
)

func main() {
  gc := gcache.New(10).
    LRU().
    LoaderFunc(func(key interface{}) (interface{}, error) {
      return "value", nil
    }).
    Build()
  v, _ := gc.Get(context.Background(),"key")
  // output: "value"
  fmt.Println(v)
}
```

GCache coordinates cache fills such that only one load in one process of an entire replicated set of processes populates the cache, then multiplexes the loaded value to all callers.

## Expirable cache

```go
package main

func main() {
  // LRU cache, size: 10, expiration: after a hour
  gc := gcache.New(10).
    LRU().
    Expiration(time.Hour).
    Build()
}
```

## Event handlers

### Evicted handler

Event handler for evict the entry.

```go
package main

func main() {
  gc := gcache.New(2).
    EvictedFunc(func(key, value interface{}) {
      fmt.Println("evicted key:", key)
    }).
    Build()
  for i := 0; i < 3; i++ {
    gc.Set(i, i*i)
  }
}
```

```
evicted key: 0
```

### Added handler

Event handler for add the entry.

```go
package main

func main() {
  gc := gcache.New(2).
    AddedFunc(func(key, value interface{}) {
      fmt.Println("added key:", key)
    }).
    Build()
  for i := 0; i < 3; i++ {
    gc.Set(i, i*i)
  }
}
```

```
added key: 0
added key: 1
added key: 2
```

# Author

**Jun Kimura**

* <http://github.com/bluele>
* <junkxdev@gmail.com>
