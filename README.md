# Snowflake

### ID Format

Snowflake is a distributed unique ID generator inspired by [Twitter's Snowflake](https://blog.twitter.com/2010/announcing-snowflake).

By default, a Snowflake ID is composed of
* The ID as a whole is a 63 bit integer stored in an int64
* 43 bits are used to store a timestamp with millisecond precision, using a custom epoch.default  is 1288834974657(UTC 2010-11-04 01:42:54). The lifetime (**278** years) is longer than that of [Twitter's Snowflake](https://blog.twitter.com/2010/announcing-snowflake) (69 years)
* 10 bits are used to store a node id - a range from 0 through 1023.
* 10 bits are used to store a sequence number - a range from 0 through 1023.

### Custom Format

You can alter the number of bits used for the node id and step number (sequence) by `NodeBits(nodeBits uint8)` and `SeqBits(seqBits uint8)` option function values. Remember that There is a maximum of 22 bits available that can be shared between these two values. You do not have to use all 22 bits.

```go
New(NodeBits(16),SeqBits(6))
```

### Custom Start Time

By default this package uses the Twitter Epoch of 1288834974657(UTC Nov 04 2010 01:42:54). You can set your own epoch value by `StartTime(startTime int64)` option function.

### How it Works
Each time you generate an ID, it works, like this.
* A timestamp with millisecond precision is stored using 43 bits of the ID.
* Then the NodeID is added in subsequent bits.
* Then the Sequence Number is added, starting at 0 and incrementing for each ID generated in the same millisecond. If you generate enough IDs in the same millisecond that the sequence would roll over or overfill then the generate function will pause until the next millisecond.

The default format shown below.
```
+--------------------------------------------------------------------------+
| 1 Bit Unused | 43 Bit Timestamp |  10 Bit NodeID  |   10 Bit Sequence ID |
+--------------------------------------------------------------------------+
```

Using the default settings, this allows for 1024 unique IDs to be generated every millisecond, per Node ID.

## Getting Started

### Installing

This assumes you already have a working Go environment, if not please see
[this page](https://golang.org/doc/install) first.

```sh
go get github.com/teamlint/snowflake
```

### Usage

Import the package into your project then construct a new snowflake Node using a
unique node number. The default settings permit a node number range from 0 to 1023.
If you have set a custom NodeBits value, you will need to calculate what your 
node number range will be. With the node object call the ID() method to 
generate and return a unique snowflake ID. 

Keep in mind that each node you create must have a unique node number, even 
across multiple servers.  If you do not keep node numbers unique the generator 
cannot guarantee unique IDs across all nodes.

The function New creates a new Snowflake instance.

```go
func New(opts ...Option) (*Snowflake, error)
```

You can configure Snowflake by the Option function:

The Option function:

```go
type Option func(*Options)

type Options struct {
	startTime int64 // 开始时间, 默认 1288834974657, 单位毫秒, UTC 时间 2010-11-04 01:42:54
	node      int64 // 节点 ID, 0 - 1023, 默认 0, 优先使用环境变量 SNOWFLAKE_NODE, 其次使用私有 IP 地址进行节点掩码计算

	timeBits uint8 // 时间位数, 默认 43 位
	nodeBits uint8 // 节点位数, 默认 10 位
	seqBits  uint8 // 递增序列位数, 默认 10 位
}
```

The option function provided are as follows:

- Node option `func Node(node int64) Option`
- StartTime option ` func StartTime(startTime int64) Option`
- Node bits option ` func NodeBits(nodeBits uint8) Option`
- Sequence bits option `func SeqBits(seqBits uint8) Option`
- Verbose option `func Verbose() Option`

In order to get a new unique ID, you just have to call the method ID.

```go
func (sf *Snowflake) ID() ID
```

**Example Program:**

```go
package main

import (
	"fmt"

	"github.com/teamlint/snowflake"
)

func main() {
  opts := []Option{Verbose(), NodeBits(8), StartTime(1314220021721)}
	sf, err := snowflake.New(opts...)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Generate a snowflake ID.
	id := sf.ID()

	// Print out the ID in a few different ways.
	fmt.Printf("Int64  ID: %d\n", id)
	fmt.Printf("String ID: %s\n", id)
	fmt.Printf("Base2  ID: %s\n", id.Base2())
	fmt.Printf("Base64 ID: %s\n", id.Base64())

	// Print out the ID's timestamp
	fmt.Printf("ID Time  : %d\n", id.Time(opts...))

	// Print out the ID's node number
	fmt.Printf("ID Node  : %d\n", id.Node(opts...))

	// Print out the ID's sequence number
	fmt.Printf("ID Step  : %d\n", id.Seq(opts...))

  // Generate and print, all in one.
  fmt.Printf("ID       : %d\n", sf.ID().Int64())
}
```

### Performance

With default settings, this snowflake generator should be sufficiently fast 
enough on most systems to generate 1024 unique ID's per millisecond. This is 
the maximum that the snowflake ID format supports. That is, around 110-112 
nanoseconds per operation. 

Since the snowflake generator is single threaded the primary limitation will be
the maximum speed of a single processor on your system.

To benchmark the generator on your system run the following command inside the
snowflake package directory.

```sh
$ go test -run=none -bench=.
```
