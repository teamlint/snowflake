package snowflake

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
)

//********************************************************************************
// Snowflake
// +--------------------------------------------------------------------------+
// | 1 Bit Unused | 41 Bit Timestamp |  10 Bit NodeID  |   12 Bit Sequence ID |
// +--------------------------------------------------------------------------+

const (
	DefaultStartTime int64 = 1288834974657 // 开始时间, UTC 时间 2010-11-04 01:42:54
	DefaultNodeBits  uint8 = 10            // 节点位数
	DefaultSeqBits   uint8 = 12            // 递增位数

	MaxBits uint8 = 64 // 最大位数
)

// Options 配置项
type Options struct {
	startTime int64 // 开始时间, 默认 1288834974657, 单位毫秒, UTC 时间 2010-11-04 01:42:54
	node      int64 // 节点 ID, 默认 0 - 1023

	timeBits uint8 // 时间位数, 默认 42 位, 使用 41 位, 首位保留未使用
	nodeBits uint8 // 节点位数, 默认 10 位
	seqBits  uint8 // 递增序列位数, 默认 12 位
}

type Option func(*Options)

type Snowflake struct {
	mu   sync.Mutex
	opts Options

	time int64 // 时间值
	node int64 // 节点值
	seq  int64 // 序列值

	nodeMax  int64
	nodeMask int64
	seqMask  int64
}

type ID int64

//********************************************************************************
// Codec

const (
	// Base32
	encodeBase32Map = "ybndrfg8ejkmcpqxot1uwisza345h769"
	// Base58
	encodeBase58Map = "123456789abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ"
)

var (
	// Base32
	decodeBase32Map [256]byte
	// ErrInvalidBase32 is returned by ParseBase32 when given an invalid []byte
	ErrInvalidBase32 = errors.New("invalid base32")
	// Base58
	decodeBase58Map [256]byte
	// ErrInvalidBase58 is returned by ParseBase58 when given an invalid []byte
	ErrInvalidBase58 = errors.New("invalid base58")
)

// A JSONSyntaxError is returned from UnmarshalJSON if an invalid ID is provided.
type JSONSyntaxError struct{ original []byte }

func (j JSONSyntaxError) Error() string {
	return fmt.Sprintf("invalid snowflake ID %q", string(j.original))
}

//********************************************************************************
// Package

func init() {
	// Base32
	for i := 0; i < len(encodeBase32Map); i++ {
		decodeBase32Map[i] = 0xFF
	}
	for i := 0; i < len(encodeBase32Map); i++ {
		decodeBase32Map[encodeBase32Map[i]] = byte(i)
	}
	// Base58
	for i := 0; i < len(encodeBase58Map); i++ {
		decodeBase58Map[i] = 0xFF
	}
	for i := 0; i < len(encodeBase58Map); i++ {
		decodeBase58Map[encodeBase58Map[i]] = byte(i)
	}
}

// New 创建 Snowflake 实例
func New(opts ...Option) (*Snowflake, error) {
	// default
	options := defaultOptions()
	// options
	for _, o := range opts {
		o(&options)
	}

	sf := Snowflake{opts: options}
	sf.node = sf.opts.node
	sf.nodeMax = -1 ^ (-1 << sf.opts.nodeBits)  // 1023
	sf.nodeMask = sf.nodeMax << sf.opts.seqBits // 4190208, 暂未使用
	sf.seqMask = -1 ^ (-1 << sf.opts.seqBits)   // 4095
	// startTime check
	now := epoch(time.Now())
	if (now - sf.opts.startTime) < 0 {
		return nil, fmt.Errorf("StartTime number(%d) must be before now's epoch(%d)", sf.opts.startTime, now)
	}

	// node check
	if sf.node < 0 || sf.node > sf.nodeMax {
		return nil, errors.New("Node number must be between 0 and " + strconv.FormatInt(sf.nodeMax, 10))
	}

	return &sf, nil
}

// MustNew 创建 Snowflake 实例, 如果出错引发 Panic
func MustNew(opts ...Option) *Snowflake {
	sf, err := New(opts...)
	if err != nil {
		panic(err)
	}
	return sf
}

//********************************************************************************
// Snowflake Options

func defaultOptions() Options {
	return Options{
		startTime: DefaultStartTime,
		nodeBits:  DefaultNodeBits,
		seqBits:   DefaultSeqBits,
	}
}

// Node设置节点
func Node(node int64) Option {
	return func(o *Options) {
		o.node = node
	}
}

// StartTime 设置节点 ID
func StartTime(startTime int64) Option {
	return func(o *Options) {
		o.startTime = startTime
	}
}

// NodeBits 设置节点位数
func NodeBits(nodeBits uint8) Option {
	return func(o *Options) {
		o.nodeBits = nodeBits
	}
}

// SeqBits 设置序列位数
func SeqBits(seqBits uint8) Option {
	return func(o *Options) {
		o.seqBits = seqBits
	}
}

//********************************************************************************
// Snowflake 类型转化

// ParseInt64 转化 64 位整型到 Snowflake ID 类型
func ParseInt64(id int64) ID {
	return ID(id)
}

// ParseString 转化字符串类型到 ID 类型
func ParseString(id string) (ID, error) {
	i, err := strconv.ParseInt(id, 10, 64)
	return ID(i), err

}

// ParseBase2 转化 Base2 编码字符串到 ID 类型
func ParseBase2(id string) (ID, error) {
	i, err := strconv.ParseInt(id, 2, 64)
	return ID(i), err
}

// ParseBase32 转化 Base32 编码字节数组到 ID 类型
func ParseBase32(b []byte) (ID, error) {
	var id int64

	for i := range b {
		if decodeBase32Map[b[i]] == 0xFF {
			return -1, ErrInvalidBase32
		}
		id = id*32 + int64(decodeBase32Map[b[i]])
	}

	return ID(id), nil
}

// ParseBase36 转化 Base36 编码字符串到 ID 类型
func ParseBase36(id string) (ID, error) {
	i, err := strconv.ParseInt(id, 36, 64)
	return ID(i), err
}

// ParseBase58 转化 Base58 编码字节数组到 ID 类型
func ParseBase58(b []byte) (ID, error) {
	var id int64
	for i := range b {
		if decodeBase58Map[b[i]] == 0xFF {
			return -1, ErrInvalidBase58
		}
		id = id*58 + int64(decodeBase58Map[b[i]])
	}

	return ID(id), nil
}

// ParseBase64 转化 Base64 编码字节数组到 ID 类型
func ParseBase64(id string) (ID, error) {
	b, err := base64.StdEncoding.DecodeString(id)
	if err != nil {
		return -1, err
	}
	return ParseBytes(b)

}

// ParseBytes 转化字节数组到 ID 类型
func ParseBytes(id []byte) (ID, error) {
	i, err := strconv.ParseInt(string(id), 10, 64)
	return ID(i), err
}

// ParseIntBytes 转化 Big Endian 编码字节数组到 ID 类型
func ParseIntBytes(id [8]byte) ID {
	return ID(int64(binary.BigEndian.Uint64(id[:])))
}

// Epoch 获取指定时间的 64位 整形毫秒时间
func Epoch(t time.Time) int64 {
	return epoch(t)
}

func epoch(t time.Time) int64 {
	if t.IsZero() {
		return DefaultStartTime
	}
	return t.UnixNano() / 1e6
}

func toTime(epoch int64) time.Time {
	if epoch <= 0 {
		return time.Unix(DefaultStartTime/1e3, DefaultStartTime%1e3*1e6)
	}
	return time.Unix(epoch/1e3, epoch%1e3*1e6)
}

//********************************************************************************
// Snowflake

// ID 产生 ID
func (sf *Snowflake) ID() ID {
	sf.mu.Lock()

	elapsedTime := sf.elapsedTime()

	if sf.time == elapsedTime {
		sf.seq = (sf.seq + 1) & sf.seqMask
		// 如果当前序列超出12bit长度,即大于4095，则需要等待下一毫秒
		// 下一毫秒将使用sequence:0
		if sf.seq == 0 {
			for sf.time > elapsedTime {
				elapsedTime = sf.elapsedTime()
			}
		}
	} else {
		sf.seq = 0
	}

	sf.time = elapsedTime

	id := sf.time<<(sf.opts.nodeBits+sf.opts.seqBits) |
		sf.node<<sf.opts.seqBits |
		sf.seq

	sf.mu.Unlock()
	return ID(id)
}

// MaxTime 返回可生成的最大时间
func (sf *Snowflake) MaxTime() int64 {
	return -1 ^ (-1 << (MaxBits - sf.opts.nodeBits - sf.opts.seqBits - 1)) // 多减1, 首位保留未使用
}

// MaxNode 返回可生成的最大节点值
func (sf *Snowflake) MaxNode() int64 {
	return sf.nodeMax
}

// MaxSeq 返回可生成的最大序列值
func (sf *Snowflake) MaxSeq() int64 {
	return sf.seqMask
}

// StartTime 获取配置起始时间
func (sf *Snowflake) StartTime() int64 {
	return sf.opts.startTime
}

// Lifetime 返回可生成的生命
func (sf *Snowflake) Lifetime() time.Time {
	return toTime(sf.MaxTime() + sf.opts.startTime)
}

// elapsedTime 获取消逝时间
func (sf *Snowflake) elapsedTime() int64 {
	return epoch(time.Now()) - sf.opts.startTime
}

//********************************************************************************
// ID

// Time 获取 ID 表示的时间整型值
func (f ID) Time(opts ...Option) int64 {
	options := defaultOptions()
	for _, opt := range opts {
		opt(&options)
	}
	return int64(f)>>(options.nodeBits+options.seqBits) + options.startTime
}

// Time 获取 ID 表示的标准时间类型值
func (f ID) StdTime(opts ...Option) time.Time {
	options := defaultOptions()
	for _, opt := range opts {
		opt(&options)
	}
	return toTime(f.Time(opts...))
}

// Node() 获取 ID 表示的节点值
func (f ID) Node(opts ...Option) int64 {
	options := defaultOptions()
	for _, opt := range opts {
		opt(&options)
	}
	nodeMax := -1 ^ (-1 << options.nodeBits) // 1023
	nodeMask := nodeMax << options.seqBits   // 4190208
	// return int64(f) & (nodeMask >> options.nodeBits)
	return int64(f) & int64(nodeMask) >> options.seqBits
}

// Seq() 获取 ID 表示的序列值
func (f ID) Seq(opts ...Option) int64 {
	options := defaultOptions()
	for _, opt := range opts {
		opt(&options)
	}
	seqMask := -1 ^ (-1 << options.seqBits) // 4095
	return int64(f) & int64(seqMask)
}

// Int64 返回 64 位整型 ID
func (f ID) Int64() int64 {
	return int64(f)
}

// String 返回字符串类型 ID
func (f ID) String() string {
	return strconv.FormatInt(int64(f), 10)
}

// Base2 返回 Base2 编码 ID
func (f ID) Base2() string {
	return strconv.FormatInt(int64(f), 2)
}

// Base64 返回 Base64 编码 ID
func (f ID) Base64() string {
	return base64.StdEncoding.EncodeToString(f.Bytes())
}

// Base32 返回Base32 编码 ID
func (f ID) Base32() string {
	if f < 32 {
		return string(encodeBase32Map[f])
	}

	b := make([]byte, 0, 12)
	for f >= 32 {
		b = append(b, encodeBase32Map[f%32])
		f /= 32
	}
	b = append(b, encodeBase32Map[f])

	for x, y := 0, len(b)-1; x < y; x, y = x+1, y-1 {
		b[x], b[y] = b[y], b[x]
	}

	return string(b)
}

// Base36 返回 Base36 编码 ID
func (f ID) Base36() string {
	return strconv.FormatInt(int64(f), 36)
}

// Base58 返回 Base58 编码 ID
func (f ID) Base58() string {
	if f < 58 {
		return string(encodeBase58Map[f])
	}

	b := make([]byte, 0, 11)
	for f >= 58 {
		b = append(b, encodeBase58Map[f%58])
		f /= 58
	}
	b = append(b, encodeBase58Map[f])

	for x, y := 0, len(b)-1; x < y; x, y = x+1, y-1 {
		b[x], b[y] = b[y], b[x]
	}

	return string(b)
}

// Bytes 返回字节数组类型 ID
func (f ID) Bytes() []byte {
	return []byte(f.String())
}

// IntBytes 返回使用 Big Endian 编码字节数组类型 ID
func (f ID) IntBytes() [8]byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(f))
	return b
}

// MarshalJSON ID 类型编码到 JSON 字节数组
func (f ID) MarshalJSON() ([]byte, error) {
	buff := make([]byte, 0, 22)
	buff = append(buff, '"')
	buff = strconv.AppendInt(buff, int64(f), 10)
	buff = append(buff, '"')
	return buff, nil
}

// UnmarshalJSON 转化 JSON 字节数组到 ID 类型
func (f *ID) UnmarshalJSON(b []byte) error {
	if len(b) < 3 || b[0] != '"' || b[len(b)-1] != '"' {
		return JSONSyntaxError{b}
	}

	i, err := strconv.ParseInt(string(b[1:len(b)-1]), 10, 64)
	if err != nil {
		return err
	}

	*f = ID(i)
	return nil
}
