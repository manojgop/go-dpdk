package flow

/*
#include <rte_config.h>
#include <rte_flow.h>
*/
import "C"

import (
	"unsafe"
)

type Error C.struct_rte_flow_error

func (e *Error) Error() string {
	return C.GoString(e.message)
}

func (e *Error) Unwrap() error {
	return ErrorType(e._type)
}

func (e *Error) Cause() unsafe.Pointer {
	return e.cause
}

type ErrorType uint

func (e ErrorType) Error() string {
	if s, ok := errStr[e]; ok {
		return s
	}
	return ""
}

var (
	errStr = make(map[ErrorType]string)
)

func registerErr(c uint, str string) ErrorType {
	et := ErrorType(c)
	errStr[et] = str
	return et
}

var (
	ErrTypeNone         = registerErr(C.RTE_FLOW_ERROR_TYPE_NONE, "No error.")
	ErrTypeUnspecified  = registerErr(C.RTE_FLOW_ERROR_TYPE_UNSPECIFIED, "Cause unspecified.")
	ErrTypeHandle       = registerErr(C.RTE_FLOW_ERROR_TYPE_HANDLE, "Flow rule (handle).")
	ErrTypeAttrGroup    = registerErr(C.RTE_FLOW_ERROR_TYPE_ATTR_GROUP, "Group field.")
	ErrTypeAttrPriority = registerErr(C.RTE_FLOW_ERROR_TYPE_ATTR_PRIORITY, "Priority field.")
	ErrTypeAttrIngress  = registerErr(C.RTE_FLOW_ERROR_TYPE_ATTR_INGRESS, "Ingress field.")
	ErrTypeAttrEgress   = registerErr(C.RTE_FLOW_ERROR_TYPE_ATTR_EGRESS, "Egress field.")
	ErrTypeAttrTransfer = registerErr(C.RTE_FLOW_ERROR_TYPE_ATTR_TRANSFER, "Transfer field.")
	ErrTypeAttr         = registerErr(C.RTE_FLOW_ERROR_TYPE_ATTR, "Attributes structure.")
	ErrTypeItemNum      = registerErr(C.RTE_FLOW_ERROR_TYPE_ITEM_NUM, "Pattern length.")
	ErrTypeItemSpec     = registerErr(C.RTE_FLOW_ERROR_TYPE_ITEM_SPEC, "Item specification.")
	ErrTypeItemLast     = registerErr(C.RTE_FLOW_ERROR_TYPE_ITEM_LAST, "Item specification range.")
	ErrTypeItemMask     = registerErr(C.RTE_FLOW_ERROR_TYPE_ITEM_MASK, "Item specification mask.")
	ErrTypeItem         = registerErr(C.RTE_FLOW_ERROR_TYPE_ITEM, "Specific pattern item.")
	ErrTypeActionNum    = registerErr(C.RTE_FLOW_ERROR_TYPE_ACTION_NUM, "Number of actions.")
	ErrTypeActionConf   = registerErr(C.RTE_FLOW_ERROR_TYPE_ACTION_CONF, "Action configuration.")
	ErrTypeAction       = registerErr(C.RTE_FLOW_ERROR_TYPE_ACTION, "Specific action.")
)
