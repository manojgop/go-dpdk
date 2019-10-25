package flow

/*
#include <stdint.h>
#include <rte_config.h>
#include <rte_flow.h>

void item_raw_set_relative(void *p, uint32_t d) {
	struct rte_flow_item_raw *it = p;
	it->relative = (d != 0);
}
void item_raw_set_search(void *p, uint32_t d) {
	struct rte_flow_item_raw *it = p;
	it->search = (d != 0);
}
const void *item_any_default_mask() {
	return &rte_flow_item_any_mask;
}
const void *item_raw_default_mask() {
	return &rte_flow_item_raw_mask;
}
const void *item_vf_default_mask() {
	return &rte_flow_item_vf_mask;
}
const void *item_port_id_default_mask() {
	return &rte_flow_item_port_id_mask;
}
const void *item_phy_port_default_mask() {
	return &rte_flow_item_phy_port_mask;
}
*/
import "C"

import (
	"encoding/binary"
	"net"
	"unsafe"

	"github.com/yerden/go-dpdk/common"
)

type Item C.struct_rte_flow_item

type ItemAny struct {
	Num uint32
}

func (s *ItemAny) ItemType() uint32      { return ItemTypeAny }
func (s *ItemAny) DefaultMask() ItemSpec { return &ItemAny{Num: 0x00000000} }
func (s *ItemAny) CStruct(mem common.Allocator) unsafe.Pointer {
	var p *C.struct_rte_flow_item_any
	common.MallocT(mem, &p)
	p.num = C.uint32_t(s.Num)
	return unsafe.Pointer(p)
}

type ItemRaw struct {
	Relative bool
	Search   bool
	Reserved uint32
	Offset   int32
	Limit    uint16
	Length   uint16 // XXX set in mask only; auto in spec/last
	Pattern  []byte
}

func (s *ItemRaw) ItemType() uint32 { return ItemTypeRaw }
func (s *ItemRaw) DefaultMask() ItemSpec {
	return &ItemRaw{
		Relative: true,
		Search:   true,
		Reserved: 0x3fffffff,
		Offset:   -1, //0xffffffff,
		Limit:    0xffff,
		Length:   0xffff,
		Pattern:  nil,
	}
}

func (s *ItemRaw) CStruct(mem common.Allocator) unsafe.Pointer {
	var p *C.struct_rte_flow_item_raw
	common.MallocT(mem, &p)
	C.item_raw_set_relative(unsafe.Pointer(p), boolUint32(s.Relative))
	C.item_raw_set_search(unsafe.Pointer(p), boolUint32(s.Search))
	p.offset = C.int32_t(s.Offset)
	p.limit = C.uint16_t(s.Limit)
	p.length = C.uint16_t(len(s.Pattern))
	p.pattern = (*C.uchar)(common.CBytes(mem, s.Pattern))
	return unsafe.Pointer(p)
}

type ItemEth struct {
	Dst, Src net.HardwareAddr
	Type     uint16
}

func (s *ItemEth) ItemType() uint32 { return ItemTypeEth }

func (s *ItemEth) DefaultMask() ItemSpec {
	return &ItemEth{
		Dst:  net.HardwareAddr([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
		Src:  net.HardwareAddr([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
		Type: 0}
}

func (s *ItemEth) CStruct(mem common.Allocator) unsafe.Pointer {
	var p *C.struct_rte_flow_item_eth
	common.MallocT(mem, &p)
	// copy dst and src MACs
	common.CopyFromBytes(unsafe.Pointer(&p.dst.addr_bytes[0]), s.Dst, 6)
	common.CopyFromBytes(unsafe.Pointer(&p.src.addr_bytes[0]), s.Src, 6)
	common.PutUint16(binary.BigEndian, unsafe.Pointer(&p._type), s.Type)
	return unsafe.Pointer(p)
}

type ItemPhyPort struct {
	Index uint32
}

func (s *ItemPhyPort) ItemType() uint32      { return ItemTypePhyPort }
func (s *ItemPhyPort) DefaultMask() ItemSpec { return &ItemPhyPort{Index: 0x00000000} }
func (s *ItemPhyPort) CStruct(mem common.Allocator) unsafe.Pointer {
	var p *C.struct_rte_flow_item_phy_port
	common.MallocT(mem, &p)
	p.index = C.uint32_t(s.Index)
	return unsafe.Pointer(p)
}

type ItemPortId struct {
	ID uint32
}

func (s *ItemPortId) ItemType() uint32      { return ItemTypePortId }
func (s *ItemPortId) DefaultMask() ItemSpec { return &ItemPortId{ID: 0xffffffff} }
func (s *ItemPortId) CStruct(mem common.Allocator) unsafe.Pointer {
	var p *C.struct_rte_flow_item_port_id
	common.MallocT(mem, &p)
	p.id = C.uint32_t(s.ID)
	return unsafe.Pointer(p)
}

type ItemVf struct {
	ID uint32
}

func (s *ItemVf) ItemType() uint32      { return ItemTypeVf }
func (s *ItemVf) DefaultMask() ItemSpec { return &ItemVf{ID: 0x00000000} }
func (s *ItemVf) CStruct(mem common.Allocator) unsafe.Pointer {
	var p *C.struct_rte_flow_item_vf
	common.MallocT(mem, &p)
	p.id = C.uint32_t(s.ID)
	return unsafe.Pointer(p)
}
