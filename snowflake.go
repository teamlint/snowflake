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

const (
	DefaultStartTime int64 = 1288834974657 // 开始时间
	DefaultNodeBits  uint8 = 10            // 节点位数
	DefaultSeqBits   uint8 = 12            // 递增位数
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

// Snowflake
// +--------------------------------------------------------------------------+
// | 1 Bit Unused | 41 Bit Timestamp |  10 Bit NodeID  |   12 Bit Sequence ID |
// +--------------------------------------------------------------------------+
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

func New(opts ...Option) (*Snowflake, error) {
	// default
	options := Options{
		startTime: DefaultStartTime,
		nodeBits:  DefaultNodeBits,
		seqBits:   DefaultSeqBits,
	}
	// options
	for _, o := range opts {
		o(&options)
	}

	sf := Snowflake{opts: options}
	sf.node = sf.opts.node
	sf.nodeMax = -1 ^ (-1 << sf.opts.nodeBits)  // 1023
	sf.nodeMask = sf.nodeMax << sf.opts.seqBits // 4190208, 暂未使用
	sf.seqMask = -1 ^ (-1 << sf.opts.seqBits)   // 4095

	if sf.node < 0 || sf.node > sf.nodeMax {
		return nil, errors.New("Node number must be between 0 and " + strconv.FormatInt(sf.nodeMax, 10))
	}

	return &sf, nil
}

func MustNew(opts ...Option) *Snowflake {
	sf, err := New(opts...)
	if err != nil {
		panic(err)
	}
	return sf
}

//********************************************************************************
// Snowflake Options

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
// Snowflake

func (sf *Snowflake) ID() ID {
	sf.mu.Lock()

	elapsedTime := sf.elapsedTime()
	// TODO 需要判断消逝时间为负的情况

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

// MaxNode 返回可配置的最大节点值
func (sf *Snowflake) MaxNode() int64 {
	return sf.nodeMax
}

// elapsedTime 获取消逝时间
func (sf *Snowflake) elapsedTime() int64 {
	return time.Now().UnixNano()/1e6 - sf.opts.startTime
}

//********************************************************************************
// ID

// Time 获取 ID 表示的时间
func (f ID) Time(opts ...Option) int64 {
	options := Options{
		startTime: DefaultStartTime,
		nodeBits:  DefaultNodeBits,
		seqBits:   DefaultSeqBits,
	}
	for _, opt := range opts {
		opt(&options)
	}
	return int64(f)>>(options.nodeBits+options.seqBits) + options.startTime
}

// Node() 获取 ID 表示的节点值
func (f ID) Node(opts ...Option) int64 {
	options := Options{
		startTime: DefaultStartTime,
		nodeBits:  DefaultNodeBits,
		seqBits:   DefaultSeqBits,
	}
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
	options := Options{
		startTime: DefaultStartTime,
		nodeBits:  DefaultNodeBits,
		seqBits:   DefaultSeqBits,
	}
	for _, opt := range opts {
		opt(&options)
	}
	seqMask := -1 ^ (-1 << options.seqBits) // 4095
	return int64(f) & int64(seqMask)
}

//********************************************************************************
// Codec

// Base32
const encodeBase32Map = "ybndrfg8ejkmcpqxot1uwisza345h769"

var (
	decodeBase32Map [256]byte
	// ErrInvalidBase32 is returned by ParseBase32 when given an invalid []byte
	ErrInvalidBase32 = errors.New("invalid base32")
)

// Base58
const encodeBase58Map = "123456789abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ"

var (
	decodeBase58Map [256]byte
	// ErrInvalidBase58 is returned by ParseBase58 when given an invalid []byte
	ErrInvalidBase58 = errors.New("invalid base58")
)

// A JSONSyntaxError is returned from UnmarshalJSON if an invalid ID is provided.
type JSONSyntaxError struct{ original []byte }

func (j JSONSyntaxError) Error() string {
	return fmt.Sprintf("invalid snowflake ID %q", string(j.original))
}

func (f ID) Int64() int64 {
	return int64(f)
}

// ParseInt64 converts an int64 into a snowflake ID
func ParseInt64(id int64) ID {
	return ID(id)
}

// String returns a string of the snowflake ID
func (f ID) String() string {
	return strconv.FormatInt(int64(f), 10)
}

// ParseString converts a string into a snowflake ID
func ParseString(id string) (ID, error) {
	i, err := strconv.ParseInt(id, 10, 64)
	return ID(i), err

}

// Base2 returns a string base2 of the snowflake ID
func (f ID) Base2() string {
	return strconv.FormatInt(int64(f), 2)
}

// ParseBase2 converts a Base2 string into a snowflake ID
func ParseBase2(id string) (ID, error) {
	i, err := strconv.ParseInt(id, 2, 64)
	return ID(i), err
}

// Base32 uses the z-base-32 character set but encodes and decodes similar
// to base58, allowing it to create an even smaller result string.
// NOTE: There are many different base32 implementations so becareful when
// doing any interoperation.
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

// ParseBase32 parses a base32 []byte into a snowflake ID
// NOTE: There are many different base32 implementations so becareful when
// doing any interoperation.
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

// Base36 returns a base36 string of the snowflake ID
func (f ID) Base36() string {
	return strconv.FormatInt(int64(f), 36)
}

// ParseBase36 converts a Base36 string into a snowflake ID
func ParseBase36(id string) (ID, error) {
	i, err := strconv.ParseInt(id, 36, 64)
	return ID(i), err
}

// Base58 returns a base58 string of the snowflake ID
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

// ParseBase58 parses a base58 []byte into a snowflake ID
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

// Base64 returns a base64 string of the snowflake ID
func (f ID) Base64() string {
	return base64.StdEncoding.EncodeToString(f.Bytes())
}

// ParseBase64 converts a base64 string into a snowflake ID
func ParseBase64(id string) (ID, error) {
	b, err := base64.StdEncoding.DecodeString(id)
	if err != nil {
		return -1, err
	}
	return ParseBytes(b)

}

// Bytes returns a byte slice of the snowflake ID
func (f ID) Bytes() []byte {
	return []byte(f.String())
}

// ParseBytes converts a byte slice into a snowflake ID
func ParseBytes(id []byte) (ID, error) {
	i, err := strconv.ParseInt(string(id), 10, 64)
	return ID(i), err
}

// IntBytes returns an array of bytes of the snowflake ID, encoded as a
// big endian integer.
func (f ID) IntBytes() [8]byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(f))
	return b
}

// ParseIntBytes converts an array of bytes encoded as big endian integer as
// a snowflake ID
func ParseIntBytes(id [8]byte) ID {
	return ID(int64(binary.BigEndian.Uint64(id[:])))
}

// MarshalJSON returns a json byte array string of the snowflake ID.
func (f ID) MarshalJSON() ([]byte, error) {
	buff := make([]byte, 0, 22)
	buff = append(buff, '"')
	buff = strconv.AppendInt(buff, int64(f), 10)
	buff = append(buff, '"')
	return buff, nil
}

// UnmarshalJSON converts a json byte array of a snowflake ID into an ID type.
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

//********************************************************************************
// Package

func init() {

	for i := 0; i < len(encodeBase58Map); i++ {
		decodeBase58Map[i] = 0xFF
	}

	for i := 0; i < len(encodeBase58Map); i++ {
		decodeBase58Map[encodeBase58Map[i]] = byte(i)
	}

	for i := 0; i < len(encodeBase32Map); i++ {
		decodeBase32Map[i] = 0xFF
	}

	for i := 0; i < len(encodeBase32Map); i++ {
		decodeBase32Map[encodeBase32Map[i]] = byte(i)
	}
}
