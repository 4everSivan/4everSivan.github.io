---
title: "Go"
lastmod: "2024-08-16T11:15:05+08:00"
---

[TOC]

## 1. 命名规范

命名规则涉及变量、常量、全局函数、结构、接⼝、⽅法等的命名。 Go语⾔从语法层⾯进⾏了以下限定：任何需要对外暴露的名字必须以⼤写字母开头，不需要对外暴露的则应该以⼩写字母开头。

### 1.1 大小写区分

1. 当命名（包括常量、变量、类型、函数名、结构字段等等）以⼀个⼤写字母开头，如：Analysize，那么使⽤这种形式的标识符的对象

就可以被外部包的代码所使⽤（客户端程序需要先导⼊这个包），这被称为**导出**（像⾯向对象语⾔中的 public）；

2. 命名如果以⼩写字母开头，则对包外是不可见的，但是他们在整个包的内部是可见并且可⽤的，这被称为**未导出**（像⾯向对象语⾔中的 private ）

### 1.2 包名称

保持package的名字和⽬录保持⼀致，尽量采取有意义的包名，简短，有意义，尽量和标准库不要冲突。包名应该为⼩写单词，不要使⽤下划线或者混合⼤⼩写。

```go
package domain
package main
```

### 1.3 ⽂件命名

尽量采取有意义的⽂件名，简短，有意义，应该为⼩写单词，使⽤下划线分隔各个单词。

```go
approve_service.go
```

### 1.4 结构体命名

采⽤驼峰命名法，⾸字母根据访问控制⼤写或者⼩写

struct 申明和初始化格式采⽤多⾏，例如下⾯：

```go
type MainConfig struct {
  Port string `json:"port"`
  Address string `json:"address"`
}
config := MainConfig{"1234", "123.221.134"}
```

### 1.5 接⼝命名

命名规则基本和上⾯的结构体类型

单个函数的结构名以 “er” 作为后缀，例如 Reader , Writer 。

```go
type Reader interface {
Read(p []byte) (n int, err error)
}
```

### 1.6 变量命名

和结构体类似，变量名称⼀般遵循驼峰法，⾸字母根据访问控制原则⼤写或者⼩写，但遇到特有名词时，需要遵循以下规则：

如果变量为私有，且特有名词为⾸个单词，则使⽤⼩写，如 appService

若变量类型为 bool 类型，则名称应以 Has, Is, Can 或 Allow 开头

```go
var isExist bool
var hasConflict bool
var canManage bool
var allowGitHook bool
```

### 1.7 常量命名

常量均需使⽤全部⼤写字母组成，并使⽤下划线分词

```go
const APP_URL = "https://www.baidu.com"
```

如果是枚举类型的常量，需要先创建相应类型：

```go
type Scheme string const (
  HTTP  Scheme = "http"
  HTTPS Scheme = "https"
)
```



## 2. 数据类型

数据类型的出现是为了把数据分成所需内存大小不同的数据，编程的时候需要用大数据的时候才需要申请大内存，就可以充分利用内存。

Go 语言按类别有以下几种数据类型：

| 序号 | 类型和描述                                                   |
| :--- | :----------------------------------------------------------- |
| 1    | **布尔型:** 布尔型的值只可以是常量 true 或者 false。一个简单的例子：var b bool = true。 |
| 2    | **数字类型:** 整型 int 和浮点型 float32、float64，Go 语言支持整型和浮点型数字，并且支持复数，其中位的运算采用补码。 |
| 3    | **字符串类型:** 字符串就是一串固定长度的字符连接起来的字符序列。Go 的字符串是由单个字节连接起来的。Go 语言的字符串的字节使用 UTF-8 编码标识 Unicode 文本。 |
| 4    | **派生类型:** 包括：(a) 指针类型（Pointer）(b) 数组类型(c) 结构化类型(struct)(d) Channel 类型(e) 函数类型(f) 切片类型(g) 接口类型（interface）(h) Map 类型 |

### 2.1 [数字类型](Go数据类型-数字类型)

Go 也有基于架构的类型，例如：int、uint 和 uintptr。

| 序号 | 类型和描述                                                   |
| :--- | :----------------------------------------------------------- |
| 1    | **uint8** 无符号 8 位整型 (0 到 255)                         |
| 2    | **uint16** 无符号 16 位整型 (0 到 65535)                     |
| 3    | **uint32** 无符号 32 位整型 (0 到 4294967295)                |
| 4    | **uint64** 无符号 64 位整型 (0 到 18446744073709551615)      |
| 5    | **int8** 有符号 8 位整型 (-128 到 127)                       |
| 6    | **int16** 有符号 16 位整型 (-32768 到 32767)                 |
| 7    | **int32** 有符号 32 位整型 (-2147483648 到 2147483647)       |
| 8    | **int64** 有符号 64 位整型 (-9223372036854775808 到 9223372036854775807) |

### 2.2 浮点型

| 序号 | 类型和描述                        |
| :--- | :-------------------------------- |
| 1    | **float32** IEEE-754 32位浮点型数 |
| 2    | **float64** IEEE-754 64位浮点型数 |
| 3    | **complex64** 32 位实数和虚数     |
| 4    | **complex128** 64 位实数和虚数    |

### 2.3 其他数字类型

以下列出了其他更多的数字类型：

| 序号 | 类型和描述                               |
| :--- | :--------------------------------------- |
| 1    | **byte** 类似 uint8                      |
| 2    | **rune** 类似 int32                      |
| 3    | **uint** 32 或 64 位                     |
| 4    | **int** 与 uint 一样大小                 |
| 5    | **uintptr** 无符号整型，用于存放一个指针 |

### 2.4 [派生类型](Go数据类型-派生类型)

| 序号 | 类型和描述            |
| ---- | --------------------- |
| 1    | 指针类型（Pointer）   |
| 2    | 数组类型              |
| 3    | 结构化类型(struct)    |
| 4    | Channel 类型          |
| 5    | 函数类型              |
| 6    | 切片类型              |
| 7    | 接口类型（interface） |
| 8    | Map 类型              |

## 3. 变量

Go 语言变量名由字母、数字、下划线组成，其中首个字符不能为数字。

声明变量的一般形式是使用 var 关键字：

```go
var identifier type
```

可以一次声明多个变量：

```go
var identifier1, identifier2 type
```

第三种方式：

```go
v_name := value
```

> ⚠️注意：**这种方式如果变量已经使用 var 声明过了，再使用 \**:=\** 声明变量，就产生编译错误**

使用：

```go
package main
import "fmt"
func main() {
    var a string = "Runoob"
    fmt.Println(a)

    var b, c int = 1, 2
    fmt.Println(b, c)
}
```

### 3.1 值类型和引用类型

所有像 int、float、bool 和 string 这些基本类型都属于值类型，使用这些类型的变量直接指向存在内存中的值：

![4.4.2_fig4.1](https://www.runoob.com/wp-content/uploads/2015/06/4.4.2_fig4.1.jpgrawtrue)

当使用等号 `=` 将一个变量的值赋值给另一个变量时，如：`j = i`，实际上是在内存中将 i 的值进行了拷贝：

![4.4.2_fig4.2](https://www.runoob.com/wp-content/uploads/2015/06/4.4.2_fig4.2.jpgrawtrue)

你可以通过 `&i` 来获取变量 `i` 的内存地址，例如：`0xf840000040`（每次的地址都可能不一样）。

值类型变量的值存储在堆中。

内存地址会根据机器的不同而有所不同，甚至相同的程序在不同的机器上执行后也会有不同的内存地址。因为每台机器可能有不同的存储器布局，并且位置分配也可能不同。

更复杂的数据通常会需要使用多个字，这些数据一般使用引用类型保存。

一个引用类型的变量 r1 存储的是 r1 的值所在的内存地址（数字），或内存地址中第一个字所在的位置。

![4.4.2_fig4.3](https://www.runoob.com/wp-content/uploads/2015/06/4.4.2_fig4.3.jpgrawtrue)

这个内存地址称之为指针，这个指针实际上也被存在另外的某一个值中。

同一个引用类型的指针指向的多个字可以是在连续的内存地址中（内存布局是连续的），这也是计算效率最高的一种存储形式；也可以将这些字分散存放在内存中，每个字都指示了下一个字所在的内存地址。

当使用赋值语句 r2 = r1 时，只有引用（地址）被复制。

如果 r1 的值被改变了，那么这个值的所有引用都会指向被修改后的内容，在这个例子中，r2 也会受到影响。

### 3.2 空白标识符

`_` 称之为空白标识符，空白标识符在函数返回值时的使用：

```go
package main

import "fmt"

func main() {
  _,numb,strs := numbers() //只获取函数返回值的后两个
  fmt.Println(numb,strs)
}

//一个可以返回多个值的函数
func numbers()(int,int,string){
  a , b , c := 1 , 2 , "str"
  return a,b,c
}
```

输出结果：

```go
2 str
```



## 4. 常量

常量是一个简单值的标识符，在程序运行时，不会被修改的量。

常量中的数据类型只可以是布尔型、数字型（整数型、浮点型和复数）和字符串型。

常量的定义格式：

```go
const identifier [type] = value
```

你可以省略类型说明符 [type]，因为编译器可以根据变量的值来推断其类型。

- 显式类型定义： `const b string = "abc"`
- 隐式类型定义： `const b = "abc"`

多个相同类型的声明可以简写为：

```go
const c_name1, c_name2 = value1, value2
```

### 4.1 iota

`iota` 是 `go` 语言的常量计数器，只能在常量的表达式中使用。`iota` 在 `const` 关键字出现时将被重置为 `0`。`const` 中每新增一行常量声明将使 `iota` 计数一次(`iota` 可理解为 `const` 语句块中的行索引)。 使用 `iota` 能简化定义，在定义枚举时很有用。

举个例子：

```go
const (
  n1 = iota //0
  n2        //1
  n3        //2
  n4        //3
)
```

几个常见的iota示例:

1. 使用 `_` 跳过某些值

   ```go
   const (
     n1 = iota //0
     n2        //1
     _
     n4        //3
   )
   ```

2. `iota`声明中间插队

   ```go
   const (
     n1 = iota //0
     n2 = 100  //100
     n3 = iota //2
     n4        //3
   )
   const n5 = iota //0
   ```

3. 定义数量级 （这里的`<<`表示左移操作，`1<<10`表示将`1`的二进制表示向左移`10`位，也就是由`1`变成了`10000000000`，也就是十进制的`1024`。同理`2<<2`表示将`2`的二进制表示向左移`2`位，也就是由`10`变成了`1000`，也就是十进制的`8`。）

   ```go
   const (
     _  = iota
     KB = 1 << (10 * iota)
     MB = 1 << (10 * iota)
     GB = 1 << (10 * iota)
     TB = 1 << (10 * iota)
     PB = 1 << (10 * iota)
   )
   ```

4. 多个`iota`定义在一行

   ```go
   const (
     a, b = iota + 1, iota + 2 //1,2
     c, d                      //2,3
     e, f                      //3,4
   )
   ```

## 5. 并发

### 5.1 [Goroutine](Go Goroutine)

Goroutine 是 Go 语言的一个特别功能，它就像是小型的工作任务，可以让我们同时处理很多事情，而不需要浪费太多计算机资源。可以把它想象成比传统方式更聪明的方式来处理多项工作，而不会让计算机变得超级忙碌。这种功能让 Go 语言在处理大量同时执行的工作时变得非常强大。  

与线程的区别：

* **资源消耗**：Goroutine比系统级线程更轻量，占用更少的内存，并且创建和销毁的成本更低。
* **调度管理**：Goroutine由Go的运行时管理，而不是由操作系统管理。Go运行时可以在一个操作系统线程上调度多个Goroutine。
* **并发模型**：Go语言的并发模型基于CSP（Communicating Sequential Processes），通过channel（通道）来实现Goroutine之间的通信，而不是使用共享内存和锁的方式。

Goroutine 非常适合用来处理大量需要并发执行的任务，比如：

* 网络请求的并发处理
* 数据的并行处理
* 定时任务的异步执行

### 5.2 [Context](Go Context)

go context是Go语言中的一个标准库，主要用于在处理并发编程时管理跨Goroutine的请求生命周期。它提供了一种控制信号传递、超时、取消和元数据传递的机制，非常适合在多个Goroutine之间传递数据、信号和执行控制。

典型使用场景：

* **网络请求处理**：在处理HTTP请求时，可以使用context来控制请求的生命周期，确保在超时或取消时及时终止相关操作。
* **数据库操作**：在数据库查询或写入过程中，使用context可以确保在超时或取消时终止长时间未完成的数据库操作。
* **微服务通信**：在服务间通信中，context可以用于传递追踪信息、取消信号等，确保服务之间的操作一致性。

## 8. 关键字

### 8.1 Package

go语言的包(package)是多个Go源码的集合，go语言有很多内置包，比如fmt，os，io等。我们也可以自定义包。在一个go语言程序中使用其它包的对象或者函数时，首先要通过 import 引入它。

该文件夹下面的所有go文件都要在代码的第一行添加如下代码，声明该文件归属的包。

```go
package 包名
```

注意事项：

* 一个文件夹下面直接包含的文件只能归属一个package，同样一个package的文件不能在多个文件夹下。
* 包名可以不和文件夹的名字一样，包名不能包含 - 符号。
* 包名为main的包为应用程序的入口包，这种包编译后会得到一个可执行文件，而编译不包含main包的源代码则不会得到可执行文件。

### 8.2 Import

#### 8.2.1 引入包的路径
第一种方式相对路径：

```go
import "./module" // 引入的包在当前文件同一目录的 module 目录
```

第二种方式绝对路径：

```go
import "LearnGo/init" // 引入的包在 gopath/src/LearnGo/init 目录。
```

#### 8.2.2 引入包的特殊方式

下面展示一些特殊的 import 方式。

* 点操作

  这个点操作的含义就是这个包导入之后，在调用这个包的函数时，可以省略前缀的包名。

  ```go
  import . "fmt"
  // 例如：fmt.Println("hello world") 可以省略的写成 Println("hello world")。
  ```

* 别名操作

  别名操作就是可以把包命名成另一个容易记忆的名字。

  ```go
  import f "fmt"
  // 别名操作的话调用包函数时前缀变成了我们的前缀，即 f.Println("hello world")。
  ```

* `_` 操作

  `_` 操作是一个让很多人费解的操作符，例如：

  ```go
  import _ "github.com/go-sql-driver/mysql"
  // _操作其实是引入该包，而不直接使用包里面的函数，而是调用了该包里面的 init 函数。
  ```


## 9. 单元测试

Go语言拥有一套单元测试和性能测试系统，仅需要添加很少的代码就可以快速测试一段需求代码。

go test 命令，会自动读取源码目录下面名为 `*_test.go` 的文件，生成并运行测试用的可执行文件。输出的信息类似下面所示的样子：

```sh
ok archive/tar 0.011s
FAIL archive/zip 0.022s
ok compress/gzip 0.033s
...
```

性能测试系统可以给出代码的性能数据，帮助测试者分析性能问题。

> ⚠️ 注意：单元测试（unit testing），是指对软件中的最小可测试单元进行检查和验证。对于单元测试中单元的含义，一般要根据实际情况去判定其具体含义，如C语言中单元指一个函数，Java 里单元指一个类，图形化的软件中可以指一个窗口或一个菜单等。总的来说，单元就是人为规定的最小的被测功能模块。

单元测试是在软件开发过程中要进行的最低级别的测试活动，软件的独立单元将在与程序的其他部分相隔离的情况下进行测试。

### 9.1 创建

要开始一个单元测试，需要准备一个 go 源码文件，在命名文件时需要让文件必须以 `_test` 结尾。默认的情况下，`go test` 命令不需要任何的参数，它会自动把你源码包下面所有 test 文件测试完毕，当然你也可以带上参数。

这里介绍几个常用的参数：

- -bench regexp 执行相应的 benchmarks，例如 -bench=.；
- -cover 开启测试覆盖率；
- -run regexp 只运行 regexp 匹配的函数，例如 -run=Array 那么就执行包含有 Array 开头的函数；
- -v 显示测试的详细命令。

单元测试源码文件可以由多个测试用例组成，每个测试用例函数需要以`Test`为前缀，例如：

```go
func TestXXX( t *testing.T ) {
  ... ...
}
```

- 测试用例文件不会参与正常源码编译，不会被包含到可执行文件中。
- 测试用例文件使用`go test`指令来执行，没有也不需要 main() 作为函数入口。所有在以 `_test` 结尾的源码内以 `Test` 开头的函数会自动被执行。
- 测试用例可以不传入 `*testing.T` 参数。

### 9.2 命令

#### 9.2.1 单元测试命令行

单元测试使用 go test 命令启动，例如：

```go
$ go test helloworld_test.go
ok          command-line-arguments        0.003s
$ go test -v helloworld_test.go
=== RUN   TestHelloWorld
--- PASS: TestHelloWorld (0.00s)
        helloworld_test.go:8: hello world
PASS
ok          command-line-arguments        0.004s
```

代码说明如下：

- 第 1 行，在 go test 后跟 helloworld_test.go 文件，表示测试这个文件里的所有测试用例。
- 第 2 行，显示测试结果，ok 表示测试通过，command-line-arguments 是测试用例需要用到的一个包名，0.003s 表示测试花费的时间。
- 第 3 行，显示在附加参数中添加了`-v`，可以让测试时显示详细的流程。
- 第 4 行，表示开始运行名叫 TestHelloWorld 的测试用例。
- 第 5 行，表示已经运行完 TestHelloWorld 的测试用例，PASS 表示测试成功。
- 第 6 行打印字符串 hello world。

#### 9.2.2 运行指定单元测试用例

`go test`指定文件时默认执行文件内的所有测试用例。可以使用`-run`参数选择需要的测试用例单独执行，参考下面的代码。

```go
package code11_3
import "testing"
func TestA(t *testing.T) {
    t.Log("A")
}
func TestAK(t *testing.T) {
    t.Log("AK")
}
func TestB(t *testing.T) {
    t.Log("B")
}
func TestC(t *testing.T) {
    t.Log("C")
}
```

这里指定 TestA 进行测试：

```go
$ go test -v -run TestA select_test.go
=== RUN   TestA
--- PASS: TestA (0.00s)
        select_test.go:6: A
=== RUN   TestAK
--- PASS: TestAK (0.00s)
        select_test.go:10: AK
PASS
ok          command-line-arguments        0.003s
```

TestA 和 TestAK 的测试用例都被执行，原因是`-run`跟随的测试用例的名称支持正则表达式，使用 `-run TestA$` 即可只执行 TestA 测试用例。

#### 9.2.3 标记单元测试结果

当需要终止当前测试用例时，可以使用 FailNow，参考下面的代码。

```go
func TestFailNow(t *testing.T) {
  t.FailNow()
}
```

还有一种只标记错误不终止测试的方法，代码如下：

```go
func TestFail(t *testing.T) {
  fmt.Println("before fail")
  t.Fail()    
  fmt.Println("after fail")
}
```

测试结果如下：

```sh
=== RUN   TestFail
before fail
after fail
--- FAIL: TestFail (0.00s)
FAIL
exit status 1
FAIL        command-line-arguments        0.002s
```

从日志中看出，第 5 行调用 Fail() 后测试结果标记为失败，但是第 7 行依然被程序执行了。

#### 9.2.4 单元测试日志

每个测试用例可能并发执行，使用 testing.T 提供的日志输出可以保证日志跟随这个测试上下文一起打印输出。testing.T 提供了几种日志输出方法，详见下表所示。

| 方  法 | 备  注                           |
| ------ | -------------------------------- |
| Log    | 打印日志，同时结束测试           |
| Logf   | 格式化打印日志，同时结束测试     |
| Error  | 打印错误日志，同时结束测试       |
| Errorf | 格式化打印错误日志，同时结束测试 |
| Fatal  | 打印致命日志，同时结束测试       |
| Fatalf | 格式化打印致命日志，同时结束测试 |

### 9.3 benchmark

基准测试可以测试一段程序的运行性能及耗费 CPU 的程度。Go语言中提供了基准测试框架，使用方法类似于单元测试，使用者无须准备高精度的计时器和各种分析工具，基准测试本身即可以打印出非常标准的测试报告。

#### 9.3.1 使用

```go
package code11_3
import "testing"
func Benchmark_Add(b *testing.B) {
    var n int
    for i := 0; i < b.N; i++ {
        n++
    }
}
```

这段代码使用基准测试框架测试加法性能。第 7 行中的 b.N 由基准测试框架提供。测试代码需要保证函数可重入性及无状态，也就是说，测试代码不使用全局变量等带有记忆性质的数据结构。避免多次运行同一段代码时的环境不一致，不能假设 N 值范围。

开启基准测试：

```sh
$ go test -v -bench=. benchmark_test.go
goos: linux
goarch: amd64
Benchmark_Add-4           20000000         0.33 ns/op
PASS
ok          command-line-arguments        0.700s
```

代码说明如下：

- 第 1 行的`-bench=.`表示运行 benchmark_test.go 文件里的所有基准测试，和单元测试中的`-run`类似。
- 第 4 行中显示基准测试名称，2000000000 表示测试的次数，也就是 testing.B 结构中提供给程序使用的 N。“0.33 ns/op”表示每一个操作耗费多少时间（纳秒）。

> ⚠️ **注意：**Windows 下使用 go test 命令行时，`-bench=.`应写为`-bench="."`。

#### 9.3.2 基准测试原理

基准测试框架对一个测试用例的默认测试时间是 1 秒。开始测试时，当以 Benchmark 开头的基准测试用例函数返回时还不到 1 秒，那么 testing.B 中的 N 值将按 1、2、5、10、20、50……递增，同时以递增后的值重新调用基准测试用例函数。

#### 9.3.3 自定义测试时间

通过`-benchtime`参数可以自定义测试时间，例如：

```sh
$ go test -v -bench=. -benchtime=5s benchmark_test.go
goos: linux
goarch: amd64
Benchmark_Add-4           10000000000                 0.33 ns/op
PASS
ok          command-line-arguments        3.380s
```

#### 9.3.4 测试内存

基准测试可以对一段代码可能存在的内存分配进行统计，下面是一段使用字符串格式化的函数，内部会进行一些分配操作。

```go
func Benchmark_Alloc(b *testing.B) {
	for i := 0; i < b.N; i++ {
  	fmt.Sprintf("%d", i)    
  }
}
```

在命令行中添加 `-benchmem` 参数以显示内存分配情况，参见下面的指令：

```sh
$ go test -v -bench=Alloc -benchmem benchmark_test.go
goos: linux
goarch: amd64
Benchmark_Alloc-4 20000000 109 ns/op 16 B/op 2 allocs/op
PASS
ok          command-line-arguments        2.311s
```

代码说明如下：

- 第 1 行的代码中 `-bench` 后添加了 Alloc，指定只测试 `Benchmark_Alloc()` 函数。
- 第 4 行代码的 `16 B/op` 表示每一次调用需要分配 16 个字节，`2 allocs/op` 表示每一次调用有两次分配。

#### 9.3.5 控制计时器

有些测试需要一定的启动和初始化时间，如果从 Benchmark() 函数开始计时会很大程度上影响测试结果的精准性。testing.B 提供了一系列的方法可以方便地控制计时器，从而让计时器只在需要的区间进行测试。我们通过下面的代码来了解计时器的控制。

```go
func Benchmark_Add_TimerControl(b *testing.B) {
    // 重置计时器
    b.ResetTimer()
    // 停止计时器
    b.StopTimer()
    // 开始计时器
    b.StartTimer()
    var n int
    for i := 0; i < b.N; i++ {
        n++
    }
}
```

从 `Benchmark()` 函数开始，Timer 就开始计数。`StopTimer()` 可以停止这个计数过程，做一些耗时的操作，通过 `StartTimer()` 重新开始计时。ResetTimer() 可以重置计数器的数据。
