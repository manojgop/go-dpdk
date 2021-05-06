package ring_test

import (
	"sync"
	"syscall"
	"testing"
	"unsafe"
	//"fmt"

	"golang.org/x/sys/unix"

	"github.com/yerden/go-dpdk/common"
	"github.com/yerden/go-dpdk/eal"
	"github.com/yerden/go-dpdk/ring"
)

var initEAL = common.DoOnce(func() error {
	var set unix.CPUSet
	err := unix.SchedGetaffinity(0, &set)
	if err == nil {
		_, err = eal.Init([]string{"test",
			"-c", common.NewMap(&set).String(),
			"-m", "128",
			"--no-huge",
			"--no-pci",
			"--master-lcore", "0"})
	}
	return err
})

func TestRingCreate(t *testing.T) {
	assert := common.Assert(t, true)

	assert(initEAL() == nil)
	err := eal.ExecOnMaster(func(ctx *eal.LcoreCtx) {
		r, err := ring.Create("test_ring", 1024, ring.OptSC,
			ring.OptSP, ring.OptSocket(ctx.SocketID()))
		assert(r != nil && err == nil, err)
		defer r.Free()
		r1, err := ring.Lookup("test_ring")
		assert(r == r1 && err == nil)
		_, err = ring.Lookup("test_ring_nonexistent")
		assert(err != nil)
	})
	assert(err == nil, err)
}

func TestRingInit(t *testing.T) {
	assert := common.Assert(t, true)

	assert(initEAL() == nil)
	err := eal.ExecOnMaster(func(ctx *eal.LcoreCtx) {
		_, err := ring.GetMemSize(1023)
		assert(err == syscall.EINVAL) // invalid count
		n, err := ring.GetMemSize(1024)
		assert(n > 0 && err == nil, err)

		ringData := make([]byte, n)
		r := (*ring.Ring)(unsafe.Pointer(&ringData[0]))
		err = r.Init("test_ring", 1024, ring.OptSC,
			ring.OptSP, ring.OptSocket(ctx.SocketID()))
		assert(err == nil, err)
	})
	assert(err == nil, err)
}

func TestRingNew(t *testing.T) {
	assert := common.Assert(t, true)

	n := 64
	r, err := ring.New("test_ring", uint(n))
	assert(r != nil && err == nil, err)
	defer r.Free() // should have no effect

	assert(r.IsEmpty())
	assert(!r.IsFull())
	assert(r.Cap() == uint(n)-1, r.Cap())
	assert(r.FreeCount() == r.Cap(), r.FreeCount(), r.Cap())

	// test array
	array := make([]int, r.Cap())

	for i := 0; i < int(r.Cap()); i++ {
		objIn := unsafe.Pointer(&array[i])
		assert(r.Count() == uint(i), r.Count())
		assert(r.FreeCount() == r.Cap()-uint(i), r.Count())
		assert(r.FreeCount()+r.Count() == r.Cap())
		cons, free := r.SpEnqueueBulk([]unsafe.Pointer{objIn})
		assert(cons == 1, cons, i)
		assert(free == uint32(r.FreeCount()), free, r.FreeCount())
	}

	assert(!r.IsEmpty())
	assert(r.IsFull())
	for i := 0; i < int(r.Cap()); i++ {
		objOut, ok := r.ScDequeue()
		assert(objOut == unsafe.Pointer(&array[i]) && ok, i)
	}

	assert(r.IsEmpty())
	assert(!r.IsFull())
	_, ok := r.ScDequeue()
	assert(!ok)
}

func TestRingNewErr(t *testing.T) {
	assert := common.Assert(t, true)

	r, err := ring.New("test_ring", 1023)
	assert(r == nil && err == syscall.EINVAL)
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

var burstSizeMax uint = 512
var bufSend = make([]unsafe.Pointer, burstSizeMax)
var bufRecv = make([]unsafe.Pointer, burstSizeMax)
var ring_ptr, ring_err = ring.New("testRing", burstSizeMax*4)

func benchmarkRingBurstUintptr(b *testing.B, burst int, n int) {
	var wg sync.WaitGroup

	sender := func(n int) {
		defer wg.Done()
		i := 0
		for i < n {
			//n, free := ring_ptr.SpEnqueueBulk(bufSend[:burstSize])
			//fmt.Printf("num enqueued = %d, free ring entries = %d\n", n, free);
			ring_ptr.SpEnqueueBulk(bufSend[:burst])
			i++
		}
	}

	receiver := func(n int) {
		defer wg.Done()
		i := 0
		for i < n {
			//n, available := ring_ptr.ScDequeueBulk(bufRecv)
			//fmt.Printf("num dequeued = %d, remain ring entries = %d\n", n, available);
			ring_ptr.ScDequeueBulk(bufRecv)
			i++
		}
	}

	wg.Add(2)
	go sender(n)
	go receiver(n)
	wg.Wait()
}

func BenchmarkRingBurst1Uintptr(b *testing.B) {
	assert := common.Assert(b, true)
	assert(initEAL() == nil)
	assert(ring_ptr != nil && ring_err == nil, ring_err)

	for i := 0; i < b.N; i++ {
		benchmarkRingBurstUintptr(b, 1, 1)
	}
}

func BenchmarkRingBurst128Uintptr(b *testing.B) {
	assert := common.Assert(b, true)
	assert(initEAL() == nil)
	assert(ring_ptr != nil && ring_err == nil, ring_err)

	for i := 0; i < b.N; i++ {
		benchmarkRingBurstUintptr(b, 128, 1)
	}
}

func BenchmarkRingBurst512Uintptr(b *testing.B) {
	assert := common.Assert(b, true)
	assert(initEAL() == nil)
	assert(ring_ptr != nil && ring_err == nil, ring_err)

	for i := 0; i < b.N; i++ {
		benchmarkRingBurstUintptr(b, 512, 1)
	}
}

func BenchmarkChanBlockUintptr(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchmarkChanBlockUintptr(b, 1)
	}
}

var ch = make(chan uintptr, burstSizeMax*4)

func benchmarkChanBlockUintptr(b *testing.B, n int) {
	var wg sync.WaitGroup

	sender := func(n int) {
		defer wg.Done()
		i := 0
		for i < n {
			ch <- uintptr(0xAABBCCDD)
			i++
		}
	}

	receiver := func(n int) {
		defer wg.Done()
		i := 0
		for i < n {
			<-ch
			i++
		}
	}

	wg.Add(2)
	go sender(n)
	go receiver(n)
	wg.Wait()
}

// Keep below section commented.
/*
func benchmarkRingUintptr(b *testing.B, burst int) {
	var wg sync.WaitGroup
	assert := common.Assert(b, true)

	assert(initEAL() == nil)

	r, err := ring.New("hello", 1024)
	assert(r != nil && err == nil, err)

	sender := func(n int) {
		defer wg.Done()
		i := 0
		buf := make([]unsafe.Pointer, burst)
		for i < n {
			k, _ := r.SpEnqueueBulk(buf[:burst])
			i += int(k)
		}
	}

	receiver := func(n int) {
		defer wg.Done()
		buf := make([]unsafe.Pointer, burst)
		i := 0
		for i < n {
			k, _ := r.ScDequeueBulk(buf)
			i += int(k)
		}
	}

	wg.Add(2)
	go sender(b.N)
	go receiver(b.N)
	wg.Wait()
}

func BenchmarkRingUintptr1(b *testing.B) {
	benchmarkRingUintptr(b, 1)
}

func BenchmarkRingUintptr20(b *testing.B) {
	benchmarkRingUintptr(b, 20)
}

func BenchmarkRingUintptr128(b *testing.B) {
	benchmarkRingUintptr(b, 128)
}

func BenchmarkRingUintptr512(b *testing.B) {
	benchmarkRingUintptr(b, 512)
}


func BenchmarkChanNonblockUintptr(b *testing.B) {
	var wg sync.WaitGroup

	ch := make(chan uintptr, 1024)

	sender := func(n int) {
		defer wg.Done()
		i := 0
		for i < n {
			select {
			case ch <- uintptr(0xAABBCCDD):
				i++
			default:
			}
		}
	}

	receiver := func(n int) {
		defer wg.Done()
		i := 0
		for i < n {
			select {
			case <-ch:
				i++
			default:
			}
		}
	}

	wg.Add(2)
	go sender(b.N)
	go receiver(b.N)
	wg.Wait()
}

func BenchmarkChanBlockUintptr(b *testing.B) {
	var wg sync.WaitGroup

	ch := make(chan uintptr, 1024)

	sender := func(n int) {
		defer wg.Done()
		i := 0
		for i < n {
			ch <- uintptr(0xAABBCCDD)
			i++
		}
	}

	receiver := func(n int) {
		defer wg.Done()
		i := 0
		for i < n {
			<-ch
			i++
		}
	}

	wg.Add(2)
	go sender(b.N)
	go receiver(b.N)
	wg.Wait()
}
 */
 