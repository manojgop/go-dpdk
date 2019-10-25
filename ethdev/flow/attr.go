package flow

/*
#include <stdint.h>
#include <rte_config.h>
#include <rte_flow.h>

void set_flow_attr_ingress(struct rte_flow_attr *attr) {
	attr->ingress = 1;
}

void set_flow_attr_egress(struct rte_flow_attr *attr) {
	attr->egress = 1;
}

void set_flow_attr_transfer(struct rte_flow_attr *attr) {
	attr->transfer = 1;
}
*/
import "C"

import (
	"unsafe"

	"github.com/yerden/go-dpdk/common"
)

type Attr struct {
	// Priority group.
	Group uint32

	// Rule priority level within group.
	Priority uint32

	// Rule applies to ingress traffic.
	Ingress bool

	// Rule applies to egress traffic.
	Egress bool

	// Instead of simply matching the properties of traffic as it
	// would appear on a given DPDK port ID, enabling this attribute
	// transfers a flow rule to the lowest possible level of any
	// device endpoints found in the pattern.
	//
	// When supported, this effectively enables an application to
	// re-route traffic not necessarily intended for it (e.g. coming
	// from or addressed to different physical ports, VFs or
	// applications) at the device level.
	//
	// It complements the behavior of some pattern items such as
	// RTE_FLOW_ITEM_TYPE_PHY_PORT and is meaningless without them.
	//
	// When transferring flow rules, ingress and egress attributes
	// keep their original meaning, as if processing traffic emitted
	// or received by the application.
	Transfer bool
}

func (a *Attr) CStruct(mem common.Allocator) unsafe.Pointer {
	var p *C.struct_rte_flow_attr
	common.MallocT(mem, &p)
	p.group = C.uint32_t(a.Group)
	p.priority = C.uint32_t(a.Priority)
	if a.Ingress {
		C.set_flow_attr_ingress(p)
	}
	if a.Egress {
		C.set_flow_attr_egress(p)
	}
	if a.Transfer {
		C.set_flow_attr_transfer(p)
	}
	return unsafe.Pointer(p)
}
