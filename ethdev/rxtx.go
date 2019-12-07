package ethdev

/*
#include <stdlib.h>

#include <rte_config.h>
#include <rte_ethdev.h>
*/
import "C"

import (
	"reflect"
	"unsafe"

	"github.com/yerden/go-dpdk/mbuf"
)

// TxBuffer is used to relieve pressure on TX queue by increasing
// latency.
type TxBuffer C.struct_rte_eth_dev_tx_buffer

// return x/y rounded to upper bound.
func fracCeilRound(x int, y int) int {
	n := x / y
	if y*n != x {
		n++
	}
	return n
}

// NewTxBuffer returns new buffer with cnt as the size.
func NewTxBuffer(cnt int) *TxBuffer {
	amount := TxBufferSize(cnt)
	ptrSize := unsafe.Sizeof(&mbuf.Mbuf{})
	data := make([]*mbuf.Mbuf, fracCeilRound(int(amount), int(ptrSize)))
	buf := (*TxBuffer)(unsafe.Pointer(&data[0]))
	buf.Init(cnt)
	return buf
}

// TxBufferSize returns size in bytes needed to allocate TxBuffer with
// mbufCnt buffers.
func TxBufferSize(mbufCnt int) uintptr {
	return unsafe.Sizeof(TxBuffer{}) + uintptr(mbufCnt)*unsafe.Sizeof(&mbuf.Mbuf{})
}

// RxBurst reads incoming packets into pkts with specified pid/qid.
// Returns number of packets read from RX queue.
func (pid Port) RxBurst(qid uint16, pkts []*mbuf.Mbuf) uint16 {
	return uint16(C.rte_eth_rx_burst(C.uint16_t(pid), C.uint16_t(qid),
		(**C.struct_rte_mbuf)(unsafe.Pointer(&pkts[0])), C.uint16_t(len(pkts))))
}

// TxBurst sends outgoing packets from pkts into specified pid/qid.
// Returns number of packets sent to TX queue.
func (pid Port) TxBurst(qid uint16, pkts []*mbuf.Mbuf) uint16 {
	return uint16(C.rte_eth_tx_burst(C.uint16_t(pid), C.uint16_t(qid),
		(**C.struct_rte_mbuf)(unsafe.Pointer(&pkts[0])), C.uint16_t(len(pkts))))
}

// TxBufferFlush sends packets from TxBuffer into TX queue.
// Returns number of packets sent to TX queue.
func (pid Port) TxBufferFlush(qid uint16, buf *TxBuffer) uint16 {
	return uint16(C.rte_eth_tx_buffer_flush(C.uint16_t(pid), C.uint16_t(qid),
		(*C.struct_rte_eth_dev_tx_buffer)(unsafe.Pointer(buf))))
}

// TxBuffer enqueues packet m for sending to TX queue using buf as a
// TX buffer.
// Returns: 0 = packet has been buffered for later transmission N > 0
// = packet has been buffered, and the buffer was subsequently
// flushed, causing N packets to be sent, and the error callback to be
// called for the rest.
func (pid Port) TxBuffer(qid uint16, buf *TxBuffer, m *mbuf.Mbuf) uint16 {
	return uint16(C.rte_eth_tx_buffer(C.uint16_t(pid), C.uint16_t(qid),
		(*C.struct_rte_eth_dev_tx_buffer)(unsafe.Pointer(buf)),
		(*C.struct_rte_mbuf)(unsafe.Pointer(m))))
}

// Mbufs returns slice containing mbufs of TxBuffer.
func (buf *TxBuffer) Mbufs() []*mbuf.Mbuf {
	var d []*mbuf.Mbuf
	b := (*C.struct_rte_eth_dev_tx_buffer)(unsafe.Pointer(buf))
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&d))
	sh.Data = uintptr(unsafe.Pointer(b)) + unsafe.Sizeof(*b)
	sh.Len = int(b.length)
	sh.Cap = int(b.size)
	return d
}

// Init initializes default values for buffered transmitting. cnt is
// the buffer size.
func (buf *TxBuffer) Init(cnt int) {
	b := (*C.struct_rte_eth_dev_tx_buffer)(unsafe.Pointer(buf))
	C.rte_eth_tx_buffer_init(b, C.uint16_t(cnt))
}
