package snowflake

import (
	"bytes"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

func TestMain(t *testing.T) {
	// Epoch = 1314220021721
	// 2002-02-26 13:14:52.099+08
	// Epoch = 1014729292099
}

//******************************************************************************
// General Test funcs

func TestNew(t *testing.T) {
	// Default
	_, err := New(NodeBits(8))
	if err != nil {
		t.Fatalf("error snowflake.New %s", err)
	}
	MustNew()
	// Overflow
	_, err = New(Node(2000))
	if err == nil {
		t.Fatal("no error snowflake.New with Node option")
	}
	// set node
	sf, err := New(Node(135))
	if err != nil {
		t.Fatalf("error snowflake.New %s", err)
	}
	if sf.Node() != 135 {
		t.Fatalf("error snowflake.New with Node option 135, but it is %d", sf.Node())
	}
}
func TestBits(t *testing.T) {
	// startTime, _ := time.Parse(time.RFC3339, "2020-01-02T15:04:05.678Z")
	// t.Log(startTime)
	// opts := []Option{Verbose(), NodeBits(8), StartTime(Epoch(startTime))}
	opts := []Option{Verbose(), SeqBits(10)}
	sf := MustNew(opts...)
	id := sf.ID()
	t.Logf("ID[%v] = %v, timestamp = %d, time = %v, elapsedTime = %v \n\n ", id.Node(opts...), id.Int64(), id.Time(opts...), id.StdTime(opts...), id.Time(opts...)-sf.StartTime())

	for i := 0; i < 5; i++ {
		rand.Seed(time.Now().UnixNano())
		maxNodeBits := uint8(rand.Intn(int(MaxNotTimeBits - 1)))
		maxSeqBits := MaxNotTimeBits - maxNodeBits
		t.Logf("maxNodeBits = %d, maxSeqBits = %d\n", maxNodeBits, maxSeqBits)
		nodeBits := uint8(rand.Intn(int(maxNodeBits) + 1))
		seqBits := uint8(rand.Intn(int(maxSeqBits)) + 1)
		opts := []Option{NodeBits(nodeBits), SeqBits(seqBits)}
		sf = MustNew(opts...)
		id = sf.ID()
		t.Logf("ID[%v] = %v, timestamp = %d, time = %v, elapsedTime = %v \n\n ", id.Node(opts...), id.Int64(), id.Time(opts...), id.StdTime(opts...), id.Time(opts...)-sf.StartTime())
	}
}
func TestEnv(t *testing.T) {
	MustNew(Env())
}
func TestOption(t *testing.T) {
	opts := []Option{Node(12), StartTime(1014729292099), NodeBits(5), SeqBits(12)}
	sf := MustNew(opts...)
	id := sf.ID()
	t.Logf("[TestOption] ID=%v, [%13d|%04d|%04d], Time=%v, elapsedTime=%d, MaxTime=%d, MaxNode=%d, MaxSeq=%d, lifetime=%v\n",
		id, id.Time(opts...), id.Node(opts...), id.Seq(opts...),
		id.StdTime(opts...),
		id.Time(opts...)-sf.StartTime(), sf.MaxTime(), sf.MaxNode(), sf.MaxSeq(), sf.Lifetime(),
	)

	opts = []Option{Node(13)}
	sf = MustNew(opts...)
	id = sf.ID()
	t.Logf("[TestOption] ID=%v, [%13d|%04d|%04d], Time=%v, elapsedTime=%d, MaxTime=%d, MaxNode=%d, MaxSeq=%d, lifetime=%v\n",
		id, id.Time(opts...), id.Node(opts...), id.Seq(opts...),
		id.StdTime(opts...),
		id.Time(opts...)-sf.StartTime(), sf.MaxTime(), sf.MaxNode(), sf.MaxSeq(), sf.Lifetime(),
	)

	opts = []Option{Node(14), NodeBits(9)}
	sf = MustNew(opts...)
	id = sf.ID()
	t.Logf("[TestOption] ID=%v, [%13d|%04d|%04d], Time=%v, elapsedTime=%d, MaxTime=%d, MaxNode=%d, MaxSeq=%d, lifetime=%v\n",
		id, id.Time(opts...), id.Node(opts...), id.Seq(opts...),
		id.StdTime(opts...),
		id.Time(opts...)-sf.StartTime(), sf.MaxTime(), sf.MaxNode(), sf.MaxSeq(), sf.Lifetime(),
	)

	opts = []Option{Node(15), StartTime(Epoch(time.Now()))}
	sf = MustNew(opts...)
	id = sf.ID()
	t.Logf("[TestOption] ID=%v, [%13d|%04d|%04d], Time=%v, elapsedTime=%d, MaxTime=%d, MaxNode=%d, MaxSeq=%d, lifetime=%v\n",
		id, id.Time(opts...), id.Node(opts...), id.Seq(opts...),
		id.StdTime(opts...),
		id.Time(opts...)-sf.StartTime(), sf.MaxTime(), sf.MaxNode(), sf.MaxSeq(), sf.Lifetime(),
	)
}

// lazy check if Generate will create duplicate IDs
// would be good to later enhance this with more smarts
func TestDuplicateID(t *testing.T) {
	sf, _ := New(Node(1))

	var x, y ID
	for i := 0; i < 1000000; i++ {
		y = sf.ID()
		if x == y {
			t.Errorf("x(%d) & y(%d) are the same", x, y)
		}
		x = y
	}
}

func TestRace(t *testing.T) {
	// opts := []Option{Node(1), SeqBits(8), Verbose()}
	opts := []Option{Node(1), SeqBits(10)}
	sf := MustNew(opts...)

	for j := 0; j < 1000; j++ {
		go func(t *testing.T, j int, sf *Snowflake) {
			// 不同协程使用不同节点 ID,避免同一时间产生相同 ID
			// sf2 := MustNew(Node(int64(j)))
			for i := 0; i < 100; i++ {
				// id := sf2.ID()
				// 如果使用多个routine,传入指定实例
				// sf.ID()
				id := sf.ID()
				// t.Logf("[Race.Rou][%v.%v] ID=%v, [%13d|%04d|%04d]\n", j, i, id, id.Time(), id.Node(), id.Seq())
				t.Logf("[Race.Rou][%v.%v] ID=%v, [%13d|%04d|%04d]\n", j, i, id, id.Time(opts...), id.Node(opts...), id.Seq(opts...))
			}
		}(t, j, sf)
		for i := 0; i < 1000; i++ {
			// sf.ID()
			id := sf.ID()
			t.Logf("[Race.For][%v.%v] ID=%v, [%13d|%04d|%04d]\n", j, i, id, id.Time(opts...), id.Node(opts...), id.Seq(opts...))
		}
	}

}

//******************************************************************************
// Converters/Parsers Test funcs
// We should have funcs here to test conversion both ways for everything

func TestPrintAll(t *testing.T) {
	sf, err := New()
	if err != nil {
		t.Fatalf("error creating Snowflake, %s", err)
	}

	id := sf.ID()

	t.Logf("Int64    : %#v", id.Int64())
	t.Logf("String   : %#v", id.String())
	t.Logf("Base2    : %#v", id.Base2())
	t.Logf("Base32   : %#v", id.Base32())
	t.Logf("Base36   : %#v", id.Base36())
	t.Logf("Base58   : %#v", id.Base58())
	t.Logf("Base64   : %#v", id.Base64())
	t.Logf("Bytes    : %#v", id.Bytes())
	t.Logf("IntBytes : %#v", id.IntBytes())

}

func TestTime(t *testing.T) {
	sf, err := New()
	if err != nil {
		t.Fatalf("error creating Snowflake, %s", err)
	}

	id := sf.ID()
	t.Logf("Snowflake now id[%d] timestamp = %v, std time = %v\n", id, id.Time(), id.StdTime())
	var i int64 = 1
	var max int64 = ^(-1 << 63)
	// max = 1e19
	t.Logf("max=%d\n", max)
	for i < max {
		oid := ParseInt64(i)
		// bits := i / 10
		if i < 0 {
			break
		}
		if i >= 1e18 {
			i = i + 1e18
		} else {
			i = i * 10
		}
		t.Logf("Snowflake node[%3d] id[%19d] timestamp = %13d, std time = %v\n", oid.Node(), oid, oid.Time(), oid.StdTime())
	}
}

func TestInt64(t *testing.T) {
	sf, err := New()
	if err != nil {
		t.Fatalf("error creating Snowflake, %s", err)
	}

	oID := sf.ID()
	i := oID.Int64()

	pID := ParseInt64(i)
	if pID != oID {
		t.Fatalf("pID %v != oID %v", pID, oID)
	}

	mi := int64(332680650168468485) // db generated
	pID = ParseInt64(mi)
	if pID.Int64() != mi {
		t.Fatalf("pID %v != mi %v", pID.Int64(), mi)
	}
	t.Logf("[TestInt64] id=%d, stdtime=%v, time=%d, node=%d, seq=%d\n", pID, pID.StdTime(), pID.Time(), pID.Node(), pID.Seq())

}

func TestString(t *testing.T) {
	node, err := New()
	if err != nil {
		t.Fatalf("error creating Snowflake, %s", err)
	}

	oID := node.ID()
	si := oID.String()

	pID, err := ParseString(si)
	if err != nil {
		t.Fatalf("error parsing, %s", err)
	}

	if pID != oID {
		t.Fatalf("pID %v != oID %v", pID, oID)
	}

	ms := `1116766490855473152`
	_, err = ParseString(ms)
	if err != nil {
		t.Fatalf("error parsing, %s", err)
	}

	ms = `1112316766490855473152`
	_, err = ParseString(ms)
	if err == nil {
		t.Fatalf("no error parsing %s", ms)
	}
}

func TestBase2(t *testing.T) {
	sf, err := New()
	if err != nil {
		t.Fatalf("error creating Snowflake, %s", err)
	}

	oID := sf.ID()
	i := oID.Base2()

	pID, err := ParseBase2(i)
	if err != nil {
		t.Fatalf("error parsing, %s", err)
	}
	if pID != oID {
		t.Fatalf("pID %v != oID %v", pID, oID)
	}

	ms := `111101111111101110110101100101001000000000000000000000000000`
	_, err = ParseBase2(ms)
	if err != nil {
		t.Fatalf("error parsing, %s", err)
	}

	ms = `1112316766490855473152`
	_, err = ParseBase2(ms)
	if err == nil {
		t.Fatalf("no error parsing %s", ms)
	}
}

func TestBase32(t *testing.T) {
	sf, err := New()
	if err != nil {
		t.Fatalf("error creating Snowflake, %s", err)
	}

	for i := 0; i < 100; i++ {
		id := sf.ID()
		b32i := id.Base32()
		psf, err := ParseBase32([]byte(b32i))
		if err != nil {
			t.Fatal(err)
		}
		if id != psf {
			t.Fatal("Parsed does not match String.")
		}
	}
}

func TestBase36(t *testing.T) {
	sf, err := New()
	if err != nil {
		t.Fatalf("error creating Snowflake, %s", err)
	}

	oID := sf.ID()
	i := oID.Base36()

	pID, err := ParseBase36(i)
	if err != nil {
		t.Fatalf("error parsing, %s", err)
	}
	if pID != oID {
		t.Fatalf("pID %v != oID %v", pID, oID)
	}

	ms := `8hgmw4blvlkw`
	_, err = ParseBase36(ms)
	if err != nil {
		t.Fatalf("error parsing, %s", err)
	}

	ms = `68h5gmw443blv2lk1w`
	_, err = ParseBase36(ms)
	if err == nil {
		t.Fatalf("no error parsing, %s", err)
	}
}

func TestBase58(t *testing.T) {
	sf, err := New()
	if err != nil {
		t.Fatalf("error creating Snowflake, %s", err)
	}

	for i := 0; i < 10; i++ {
		id := sf.ID()
		b58 := id.Base58()
		psf, err := ParseBase58([]byte(b58))
		if err != nil {
			t.Fatal(err)
		}
		if id != psf {
			t.Fatal("Parsed does not match String.")
		}
	}
}

func TestBase64(t *testing.T) {
	sf, err := New()
	if err != nil {
		t.Fatalf("error creating Snowflake, %s", err)
	}

	oID := sf.ID()
	i := oID.Base64()

	pID, err := ParseBase64(i)
	if err != nil {
		t.Fatalf("error parsing, %s", err)
	}
	if pID != oID {
		t.Fatalf("pID %v != oID %v", pID, oID)
	}

	ms := `MTExNjgxOTQ5NDY2MDk5NzEyMA==`
	_, err = ParseBase64(ms)
	if err != nil {
		t.Fatalf("error parsing, %s", err)
	}

	ms = `MTExNjgxOTQ5NDY2MDk5NzEyMA`
	_, err = ParseBase64(ms)
	if err == nil {
		t.Fatalf("no error parsing, %s", err)
	}
}

func TestBytes(t *testing.T) {
	sf, err := New()
	if err != nil {
		t.Fatalf("error creating Snowflake, %s", err)
	}

	oID := sf.ID()
	i := oID.Bytes()

	pID, err := ParseBytes(i)
	if err != nil {
		t.Fatalf("error parsing, %s", err)
	}
	if pID != oID {
		t.Fatalf("pID %v != oID %v", pID, oID)
	}

	ms := []byte{0x31, 0x31, 0x31, 0x36, 0x38, 0x32, 0x31, 0x36, 0x37, 0x39, 0x35, 0x37, 0x30, 0x34, 0x31, 0x39, 0x37, 0x31, 0x32}
	_, err = ParseBytes(ms)
	if err != nil {
		t.Fatalf("error parsing, %#v", err)
	}

	ms = []byte{0xFF, 0xFF, 0xFF, 0x31, 0x31, 0x31, 0x36, 0x38, 0x32, 0x31, 0x36, 0x37, 0x39, 0x35, 0x37, 0x30, 0x34, 0x31, 0x39, 0x37, 0x31, 0x32}
	_, err = ParseBytes(ms)
	if err == nil {
		t.Fatalf("no error parsing, %#v", err)
	}
}

func TestIntBytes(t *testing.T) {
	sf, err := New()
	if err != nil {
		t.Fatalf("error creating Snowflake, %s", err)
	}

	oID := sf.ID()
	i := oID.IntBytes()

	pID := ParseIntBytes(i)
	if pID != oID {
		t.Fatalf("pID %v != oID %v", pID, oID)
	}

	ms := [8]uint8{0xf, 0x7f, 0xc0, 0xfc, 0x2f, 0x80, 0x0, 0x0}
	mi := int64(1116823421972381696)
	pID = ParseIntBytes(ms)
	if pID.Int64() != mi {
		t.Fatalf("pID %v != mi %v", pID.Int64(), mi)
	}

}

//******************************************************************************
// Marshall Test Methods

func TestMarshalJSON(t *testing.T) {
	id := ID(13587)
	expected := "\"13587\""

	bytes, err := id.MarshalJSON()
	if err != nil {
		t.Fatalf("Unexpected error during MarshalJSON")
	}

	if string(bytes) != expected {
		t.Fatalf("Got %s, expected %s", string(bytes), expected)
	}
}

func TestMarshalsIntBytes(t *testing.T) {
	id := ID(13587).IntBytes()
	expected := []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x35, 0x13}
	if !bytes.Equal(id[:], expected) {
		t.Fatalf("Expected ID to be encoded as %v, got %v", expected, id)
	}
}

func TestUnmarshalJSON(t *testing.T) {
	tt := []struct {
		json        string
		expectedID  ID
		expectedErr error
	}{
		{`"13587"`, 13587, nil},
		{`1`, 0, JSONSyntaxError{[]byte(`1`)}},
		{`"invalid`, 0, JSONSyntaxError{[]byte(`"invalid`)}},
	}

	for _, tc := range tt {
		var id ID
		err := id.UnmarshalJSON([]byte(tc.json))
		if !reflect.DeepEqual(err, tc.expectedErr) {
			t.Fatalf("Expected to get error '%s' decoding JSON, but got '%s'", tc.expectedErr, err)
		}

		if id != tc.expectedID {
			t.Fatalf("Expected to get ID '%s' decoding JSON, but got '%s'", tc.expectedID, id)
		}
	}
}

// ****************************************************************************
// Benchmark Methods

func BenchmarkParseBase32(b *testing.B) {
	sf, _ := New(Node(1))
	id := sf.ID()
	b32i := id.Base32()

	b.ReportAllocs()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		ParseBase32([]byte(b32i))
	}
}
func BenchmarkBase32(b *testing.B) {
	sf, _ := New(Node(1))
	id := sf.ID()

	b.ReportAllocs()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		id.Base32()
	}
}
func BenchmarkParseBase58(b *testing.B) {
	sf, _ := New(Node(1))
	id := sf.ID()
	b58 := id.Base58()

	b.ReportAllocs()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		ParseBase58([]byte(b58))
	}
}
func BenchmarkBase58(b *testing.B) {
	sf, _ := New(Node(1))
	id := sf.ID()

	b.ReportAllocs()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		id.Base58()
	}
}
func BenchmarkGenerate(b *testing.B) {
	sf, _ := New(Node(1))

	b.ReportAllocs()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = sf.ID()
	}
}

func BenchmarkGenerateMaxSequence(b *testing.B) {
	sf, _ := New(NodeBits(1), SeqBits(21))

	b.ReportAllocs()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = sf.ID()
	}
}

func BenchmarkUnmarshal(b *testing.B) {
	sf, _ := New(Node(1))
	id := sf.ID()
	bytes, _ := id.MarshalJSON()

	var id2 ID

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = id2.UnmarshalJSON(bytes)
	}
}

func BenchmarkMarshal(b *testing.B) {
	sf, _ := New(Node(1))
	id := sf.ID()

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, _ = id.MarshalJSON()
	}
}
