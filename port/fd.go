package port

/*
#include <rte_config.h>
#include <rte_errno.h>

#include <rte_port_fd.h>
*/
import "C"

import (
	"unsafe"

	"github.com/yerden/go-dpdk/mempool"
)

// FdIn input port built on top of valid non-blocking file
// descriptor.
type FdIn struct {
	// Pre-initialized buffer pool.
	*mempool.Mempool

	// File descriptor.
	Fd uintptr

	// Maximum Transfer Unit (MTU)
	MTU uint32
}

// Create implements ConfigIn interface.
func (rd *FdIn) Create(socket int) (*InOps, *In) {
	ops := (*InOps)(&C.rte_port_fd_reader_ops)
	rc := &C.struct_rte_port_fd_reader_params{
		fd:      C.int(rd.Fd),
		mtu:     C.uint32_t(rd.MTU),
		mempool: (*C.struct_rte_mempool)(unsafe.Pointer(rd.Mempool)),
	}

	return createIn(ops, unsafe.Pointer(rc), socket)
}

// FdOut is an output port built on top of valid non-blocking file
// descriptor.
type FdOut struct {
	// File descriptor.
	Fd uintptr

	// If NoDrop set writer makes Retries attempts to write packets to
	// ring.
	NoDrop bool

	// If NoDrop set and Retries is 0, number of retries is unlimited.
	Retries uint32
}

// Create implements ConfigOut interface.
func (wr *FdOut) Create(socket int) (ops *OutOps, p *Out) {
	if !wr.NoDrop {
		ops = (*OutOps)(&C.rte_port_fd_writer_ops)
	} else {
		ops = (*OutOps)(&C.rte_port_fd_writer_nodrop_ops)
	}
	rc := &C.struct_rte_port_fd_writer_nodrop_params{
		fd:        C.int(wr.Fd),
		n_retries: C.uint32_t(wr.Retries),
	}
	return createOut(ops, unsafe.Pointer(rc), socket)
}
