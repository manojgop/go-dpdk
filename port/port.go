/*
Package port wraps RTE port library.

Please refer to DPDK Programmer's Guide for reference and caveats.
*/
package port

/*
#include <rte_config.h>
#include <rte_port.h>

void *go_rd_create(void *ops_table, void *params, int socket_id)
{
	struct rte_port_in_ops *ops = ops_table;
	return ops->f_create(params, socket_id);
}

int go_rd_free(void *ops_table, void *port)
{
	struct rte_port_in_ops *ops = ops_table;
	return ops->f_free(port);
}

void *go_wr_create(void *ops_table, void *params, int socket_id)
{
	struct rte_port_out_ops *ops = ops_table;
	return ops->f_create(params, socket_id);
}

int go_wr_free(void *ops_table, void *port)
{
	struct rte_port_out_ops *ops = ops_table;
	return ops->f_free(port);
}
*/
import "C"

import (
	"unsafe"

	"github.com/yerden/go-dpdk/common"
)

type opaqueData [0]byte

// InOps describes input port interface defining the input port
// operation.
type InOps C.struct_rte_port_in_ops

// OutOps describes output port interface defining the output port
// operation.
type OutOps C.struct_rte_port_out_ops

// In is an input port instance.
type In opaqueData

// Out is an output port instance.
type Out opaqueData

// ConfigIn implements reader port capability which allows to read
// packets from it.
type ConfigIn interface {
	// Create returns pointer to statically allocated call table
	// and a pointer to the opaque port struct.
	Create(socket int) (*InOps, *In)
}

// ConfigOut implements writer port capability which allows to
// write packets to it.
type ConfigOut interface {
	// Create returns pointer to statically allocated call table
	// and a pointer to the opaque port struct.
	Create(socket int) (*OutOps, *Out)
}

// XXX: we need to wrap calls which are not performance bottlenecks.

func err(n ...interface{}) error {
	if len(n) == 0 {
		return common.RteErrno()
	}

	return common.IntToErr(n[0])
}

func createIn(ops *InOps, arg unsafe.Pointer, socket int) (*InOps, *In) {
	return ops, (*In)(C.go_rd_create(unsafe.Pointer(ops), arg, C.int(socket)))
}

func createOut(ops *OutOps, arg unsafe.Pointer, socket int) (*OutOps, *Out) {
	return ops, (*Out)(C.go_wr_create(unsafe.Pointer(ops), arg, C.int(socket)))
}

// Free releases all memory allocated when creating port instance.
func (ops *InOps) Free(p *In) error {
	return err(C.go_rd_free(unsafe.Pointer(ops), unsafe.Pointer(p)))
}

// Free releases all memory allocated when creating port instance.
func (ops *OutOps) Free(p *Out) error {
	return err(C.go_wr_free(unsafe.Pointer(ops), unsafe.Pointer(p)))
}
