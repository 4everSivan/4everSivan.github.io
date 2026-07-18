---
title: "Goroutine"
lastmod: "2024-08-16T09:58:22+08:00"
---

[TOC]

## 1. 简介

首先，让我们以一个简单的方式来解释什么是 Goroutine。Goroutine 是 Go 语言的一个特别功能，它就像是小型的工作任务，可以让我们同时处理很多事情，而不需要浪费太多计算机资源。可以把它想象成比传统方式更聪明的方式来处理多项工作，而不会让计算机变得超级忙碌。这种功能让 Go 语言在处理大量同时执行的工作时变得非常强大。  

## 2. 快速入门

使用 Goroutine 只需要创建一个函数，然后在要使用 Goroutine 的函数前面使用 `go` 关键字即可完成。可以参考以下用例：

```go
package main

import (
	"fmt"
)

func main() {
	go sayHello()
}

func sayHello() {
	fmt.Println("Hello, Goroutine!")
}
```

## 3. 同步等待

通常情况下，我们希望主程序能够等待所有的 Goroutine 完成，以确保结果的完整性。这就是 `WaitGroup` 的作用：

### 3.1 `sync.WaitGroup`

这个功能主要用于让主程序等待所有的 Goroutine 完成，然后再继续执行接下来的程序。主要有以下几种方法：

- `Add(delta int)`：用于增加计数器的值，表示有多少个Goroutine需要等待。
- `Done()`：用于减少计数器的值，表示一个 Goroutine 已经完成。通常在 Goroutine 执行完后使用 `Done`。
- `Wait()`：用于等待计数器归零。当计数器的值为零时，`Wait` 函数会返回，并允许主程序继续执行。

可以参考以下用例：

```go
package main

import (
	"fmt"
	"sync"
	"time"
)

func printSomethings(thing int, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Printf("%d --------- start.\n", thing)
	time.Sleep(1 * time.Second)
	fmt.Printf("%d --------- end.\n", thing)
}

func main() {

	var wg sync.WaitGroup
	wg.Add(5)

	for i := 0; i < 5; i++ {
		go printSomethings(i, &wg)
	}
	wg.Wait()
}
```

## 4. 特性和限制

### 4.1 特性

* 资源消耗极低：Goroutine 的创建相对轻量，主要消耗少量栈空间。这意味着你可以创建大量的 Goroutine，而不必担心资源耗尽的问题。

* 有效的线程管理：当一个 Goroutine 被阻塞时，相应的管理线程将被搁置，但运行时会将其他 Goroutine 分配给这个线程，使其继续执行其他工作。这种机制确保了线程的高效使用，避免了资源浪费。

* 最大线程数限制：你可以通过**设置 `$GOMAXPROCS`** 来限制系统中的线程数量，确保它们不会无节制地增加。这有助于避免系统资源的过度消耗。

### 4.2 限制

* Goroutine 数量限制：理论上，Go 语言可以创建极多的 Goroutine，但实际上，系统的可用资源（内存和CPU）是有限的。因此，你需要谨慎控制 Goroutine 的数量，以避免过多的并发造成资源耗尽或性能下降。

* 竞争条件和死锁：Goroutine 的并发操作需要谨慎处理共享资源，否则可能出现竞争条件（race condition）和死锁（deadlock）。这虽然不是直接的限制，但在 Goroutine 的设计和使用中需要特别注意，以确保程序的正确性。

## 5. 特殊情况

对下面这个用例：

```go
package main

import (
    "fmt"
    "time"
)

func main() {
    for i := 0; i < 3; i++ {
        go func() {
            fmt.Println(i)
        }()
    }

    time.Sleep(1 * time.Second)
}
```

输出结果：

```
3
3
3
```

预期结果：

```
// 随机生成 0-2
0
1
2
```

问题原因：

1. 所有 Goroutine 代码片段中的 `i` 是同一个变量，待循环结束的时候，它的值为 `3`。
2. `main()` 循环结束后才开始并发执行的新生成的 Goroutine。

修复：

```go
package main

import (
    "fmt"
    "time"
)

func main() {
    for i := 0; i < 3; i++ {
        go func(v int) {
            fmt.Println(v)
        }(i)
    }

    time.Sleep(1 * time.Second)
}
```

通过方法传参的方式，将 `i` 的值拷贝到新的变量 `v` 中，而在每一个 goroutine 都对应了一个属于自己作用域的 `v` 变量， 所以最终打印结果为随机的 `0,1,2`。
