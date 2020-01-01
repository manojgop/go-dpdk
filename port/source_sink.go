package port

/*
#include <rte_config.h>
#include <rte_port_source_sink.h>
*/
import "C"

import (
	"unsafe"

	"github.com/yerden/go-dpdk/mempool"
)

// compile time checks
var _ = []ConfigIn{
	&Source{},
}

var _ = []ConfigOut{
	&Sink{},
}

// Source is an input port that can be used to generate packets.
type Source struct {
	// Pre-initialized buffer pool.
	*mempool.Mempool

	// The full path of the pcap file to read packets from.
	Filename string

	// The number of bytes to be read from each packet in the pcap file. If
	// this value is 0, the whole packet is read; if it is bigger than packet
	// size, the generated packets will contain the whole packet.
	BytesPerPacket uint32
}

// Create implements ConfigIn interface.
func (rd *Source) Create(socket int) (*InOps, *In) {
	ops := (*InOps)(&C.rte_port_source_ops)
	rc := &C.struct_rte_port_source_params{}
	rc.mempool = (*C.struct_rte_mempool)(unsafe.Pointer(rd.Mempool))
	rc.n_bytes_per_pkt = C.uint32_t(rd.BytesPerPacket)
	if rd.Filename != "" {
		rc.file_name = C.CString(rd.Filename)
		defer C.free(unsafe.Pointer(rc.file_name))
	}
	return createIn(ops, unsafe.Pointer(rc), socket)
}

// Sink is an output port that drops all packets written to it.
type Sink struct {
	// The full path of the pcap file to write the packets to.
	Filename string

	// The maximum number of packets write to the pcap file. If this value is
	// 0, the "infinite" write will be carried out.
	MaxPackets uint32
}

// Create implements ConfigOut interface.
func (wr *Sink) Create(socket int) (*OutOps, *Out) {
	ops := (*OutOps)(&C.rte_port_sink_ops)
	rc := &C.struct_rte_port_sink_params{}
	rc.max_n_pkts = C.uint32_t(wr.MaxPackets)
	if wr.Filename != "" {
		rc.file_name = C.CString(wr.Filename)
		defer C.free(unsafe.Pointer(rc.file_name))
	}
	return createOut(ops, unsafe.Pointer(rc), socket)
}
