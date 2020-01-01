package port

/*
#include <rte_config.h>
#include <rte_port.h>
#include <rte_port_ethdev.h>
*/
import "C"

import (
	"unsafe"
)

// compile time checks
var _ = []ConfigIn{
	&EthdevIn{},
}

var _ = []ConfigOut{
	&EthdevOut{},
}

// EthdevIn is an input port built on top of pre-initialized NIC
// RX queue.
type EthdevIn struct {
	// Configured Ethernet port and RX queue ID.
	PortID, QueueID uint16
}

// Create implements ConfigIn interface.
func (rd *EthdevIn) Create(socket int) (*InOps, *In) {
	ops := (*InOps)(&C.rte_port_ethdev_reader_ops)
	rc := &C.struct_rte_port_ethdev_reader_params{
		port_id:  C.uint16_t(rd.PortID),
		queue_id: C.uint16_t(rd.QueueID),
	}
	return createIn(ops, unsafe.Pointer(rc), socket)
}

// EthdevOut is an output port built on top of pre-initialized NIC
// TX queue.
type EthdevOut struct {
	// Configured Ethernet port and TX queue ID.
	PortID, QueueID uint16

	// Recommended burst size for NIC TX queue.
	TxBurstSize uint32

	// If NoDrop set writer makes Retries attempts to write packets to
	// NIC TX queue.
	NoDrop bool

	// If NoDrop set and Retries is 0, number of retries is unlimited.
	Retries uint32
}

// Create implements ConfigOut interface.
func (wr *EthdevOut) Create(socket int) (ops *OutOps, p *Out) {
	if !wr.NoDrop {
		ops = (*OutOps)(&C.rte_port_ethdev_writer_ops)
	} else {
		ops = (*OutOps)(&C.rte_port_ethdev_writer_nodrop_ops)
	}
	// NOTE: struct rte_port_ethdev_writer_params is a subset of struct
	// rte_port_ethdev_writer_nodrop_params, so we may simply use the latter
	// for it would fit regardless of NoDrop flag.
	rc := &C.struct_rte_port_ethdev_writer_nodrop_params{
		port_id:     C.uint16_t(wr.PortID),
		queue_id:    C.uint16_t(wr.QueueID),
		tx_burst_sz: C.uint32_t(wr.TxBurstSize),
		n_retries:   C.uint32_t(wr.Retries),
	}
	return createOut(ops, unsafe.Pointer(rc), socket)
}
