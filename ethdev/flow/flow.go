package flow

/*
#include <stdint.h>
*/
import "C"

import (
	"unsafe"

	"github.com/yerden/go-dpdk/common"
)

type ItemSpec interface {
	ItemType() uint32
	DefaultMask() ItemSpec
	CStruct(common.Allocator) unsafe.Pointer
}

func boolUint32(b bool) C.uint32_t {
	if b {
		return 1
	}
	return 0
}
