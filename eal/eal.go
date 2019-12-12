/*
Package eal wraps EAL initialization and provides some additional functionality
on top of that. Every CPU's logical core which is setup by EAL runs its own
function which essentially receives functions to execute via Go channel. So you
may run arbitrary Go code in the context of EAL thread.

EAL may be initialized via command line string, parsed command line string or a
set of Options.

Please note that some functions may be called only in EAL thread because of TLS
(Thread Local Storage) dependency.

API is a subject to change. Be aware.
*/
package eal

/*
#include <stdlib.h>

#include <rte_config.h>
#include <rte_eal.h>
#include <rte_errno.h>
#include <rte_lcore.h>

extern int lcoreFuncListener(void *arg);
*/
import "C"

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"runtime"
	"strings"
	//"sync"
	"unsafe"

	"github.com/yerden/go-dpdk/common"
)

// Maximum number of lcores configured during DPDK compile-time.
const (
	MaxLcore = C.RTE_MAX_LCORE
)

// The type of process in a linux, multi-process setup.
const (
	ProcAuto      = C.RTE_PROC_AUTO
	ProcPrimary   = C.RTE_PROC_PRIMARY
	ProcSecondary = C.RTE_PROC_SECONDARY
)

// Lcore is a per-lcore context and is supplied to function running to
// particular lcore.
type Lcore struct {
	// Value is a user-specified context. You may change it as you
	// will and it will persist across function invocations on
	// particular lcore.
	Value interface{}

	// channel to receive functions to execute.
	ch chan *lcoreJob

	// signal to kill current thread
	done bool
}

func err(n ...interface{}) error {
	if len(n) == 0 {
		return common.RteErrno()
	}

	return common.IntToErr(n[0])
}

// ID returns CPU logical core id. This function must be called only
// in EAL thread.
func (lc *Lcore) ID() uint {
	return uint(C.rte_lcore_id())
}

// SocketID returns NUMA socket where the current thread resides. This
// function must be called only in EAL thread.
func (lc *Lcore) SocketID() uint {
	return uint(C.rte_socket_id())
}

// String implements fmt.Stringer.
func (lc *Lcore) String() string {
	return fmt.Sprintf("lcore=%d socket=%d", lc.ID(), lc.SocketID())
}

// LcoreToSocket return socket id for given lcore ID.
func LcoreToSocket(id uint) uint {
	return uint(C.rte_lcore_to_socket_id(C.uint(id)))
}

type ealConfig struct {
	lcores map[uint]*Lcore
}

var (
	// goEAL is the storage for all EAL lcore threads configuration.
	goEAL = &ealConfig{make(map[uint]*Lcore)}
)

// panicCatcher launches lcore function and returns possible panic as
// an error.
func panicCatcher(fn func(*Lcore), lc *Lcore) (err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		log.Printf("panic on lcore %d: %v", lc.ID(), r)
		pc := make([]uintptr, 64)
		// this function is called from runtime package, so to
		// unwind the stack we may skip (1) runtime.Callers
		// function, (2) this caller function
		n := runtime.Callers(2, pc)
		err = &ErrLcorePanic{pc[:n], lc.ID(), fmt.Sprintf("%v", r)}
	}()
	fn(lc)
	return err
}

// to run as lcore_function_t
//export lcoreFuncListener
func lcoreFuncListener(unsafe.Pointer) C.int {
	id := uint(C.rte_lcore_id())
	lc := goEAL.lcores[id]
	log.Printf("lcore %d started", id)
	defer log.Printf("lcore %d exited", id)

	for job := range lc.ch {
		err := panicCatcher(job.fn, lc)
		if job.ret != nil {
			job.ret <- err
		}
		if lc.done {
			break
		}
	}
	return 0
}

// stop all lcores and call rte_eal_cleanup on master.
// warning: it will block infinitely if lcore functions are being
// executed on some lcores.
func ealDeInit() error {
	// Shutdown slaves
	for _, id := range Lcores(true) {
		ExecOnLcore(id, func(lc *Lcore) {
			lc.done = true
		})
	}

	// Shutdown master
	var e error
	err := ExecOnMaster(func(lc *Lcore) {
		lc.done = true
		e = err(C.rte_eal_cleanup())
	})
	if err != nil {
		return err
	}
	return e
}

// ErrLcorePanic is an error returned by ExecOnLcore in case lcore
// function panics.
type ErrLcorePanic struct {
	Pc      []uintptr
	LcoreID uint
	errStr  string
}

// Error implements error interface.
func (e *ErrLcorePanic) Error() string {
	return e.errStr
}

// PrintStack prints calling stack of the error into specified writer.
func (e *ErrLcorePanic) PrintStack(w io.Writer) {
	frames := runtime.CallersFrames(e.Pc)
	for {
		frame, more := frames.Next()
		if !more {
			break
		}
		// skipping everything from runtime package.
		if strings.HasPrefix(frame.Function, "runtime.") {
			continue
		}
		fmt.Fprintf(w, "... at %s:%d, %s\n", frame.File, frame.Line,
			frame.Function)
	}
}

// ErrLcoreInvalid is returned by ExecOnLcore in case the desired
// lcore ID is invalid.
var ErrLcoreInvalid = fmt.Errorf("Invalid logical core")

type lcoreJob struct {
	fn  func(*Lcore)
	ret chan<- error
}

// ExecOnLcoreAsync sends fn to execute on CPU logical core lcoreID,
// i.e. in EAL-owned thread on that lcore.
//
// Possible panic in lcore function will be intercepted and returned
// as an error of type ErrLcorePanic through ret channel specified by
// caller. If lcoreID is invalid, ErrLcoreInvalid error will be
// returned the same way.
//
// ret may be nil, in which case no error will be reported.
func ExecOnLcoreAsync(lcoreID uint, ret chan<- error, fn func(*Lcore)) {
	if lc, ok := goEAL.lcores[lcoreID]; ok {
		lc.ch <- &lcoreJob{fn, ret}
	} else if ret != nil {
		ret <- ErrLcoreInvalid
	}
}

// ExecOnLcore sends fn to execute on CPU logical core lcoreID, i.e.
// in EAL-owned thread on that lcore. Then it waits for the execution
// to finish and returns the execution result.
//
// Possible panic in lcore function will be intercepted and returned
// as an error of type ErrLcorePanic. If lcoreID is invalid,
// ErrLcoreInvalid error will be returned.
func ExecOnLcore(lcoreID uint, fn func(*Lcore)) error {
	ch := make(chan error, 1)
	ExecOnLcoreAsync(lcoreID, ch, fn)
	return <-ch
}

// ExecOnMasterAsync is a shortcut for ExecOnLcoreAsync with master
// lcore as a destination.
func ExecOnMasterAsync(ret chan<- error, fn func(*Lcore)) {
	ExecOnLcoreAsync(GetMasterLcore(), ret, fn)
}

// ExecOnMaster is a shortcut for ExecOnLcore with master lcore as a
// destination.
func ExecOnMaster(fn func(*Lcore)) error {
	return ExecOnLcore(GetMasterLcore(), fn)
}

type lcoresIter struct {
	i  C.uint
	sm C.int
}

func (iter *lcoresIter) next() bool {
	iter.i = C.rte_get_next_lcore(iter.i, iter.sm, 0)
	return iter.i < C.RTE_MAX_LCORE
}

// Lcores returns all lcores registered in EAL. If skipMaster is true,
// master lcore will not be included in the result.
func Lcores(skipMaster bool) (out []uint) {
	c := &lcoresIter{i: ^C.uint(0), sm: C.int(0)}

	if skipMaster {
		c.sm = 1
	}

	for c.next() {
		out = append(out, uint(c.i))
	}
	return out
}

// call rte_eal_init and launch lcoreFuncListener on all slave lcores
// should be run in master lcore thread only
func ealInitAndLaunch(args []string) error {
	mem := common.NewAllocatorSession(&common.StdAlloc{})
	defer mem.Flush()

	argc := C.int(len(args))
	argv := make([]*C.char, argc+1)
	for i := range args {
		argv[i] = (*C.char)(common.CString(mem, args[i]))
	}

	// initialize EAL
	if C.rte_eal_init(argc, &argv[0]) < 0 {
		return err()
	}

	// init per-lcore contexts
	for _, id := range Lcores(false) {
		goEAL.lcores[id] = &Lcore{ch: make(chan *lcoreJob)}
	}

	// lcore function
	fn := (*C.lcore_function_t)(C.lcoreFuncListener)

	// launch every EAL thread lcore function
	// it should be success since we've just called rte_eal_init()
	return err(C.rte_eal_mp_remote_launch(fn, nil, C.SKIP_MASTER))
}

// InitWithArgs initializes EAL as in rte_eal_init. Options are
// specified in a parsed command line string.
//
// This function initialized EAL and waits for executable functions on
// each of EAL-owned threads.
func InitWithArgs(args []string) error {
	ch := make(chan error, 1)
	log.Println("EAL parameters:", args)
	go func() {
		// we should initialize EAL and run EAL threads in a separate
		// goroutine because its thread is going to be acquired by EAL
		// and become master lcore thread
		runtime.LockOSThread()

		// initialize EAL and launch lcoreFuncListener on all slave
		// lcores, then report
		err := ealInitAndLaunch(args)
		if ch <- err; err == nil {
			// run on master lcore
			lcoreFuncListener(nil)
		}
	}()

	return <-ch
}

// Cleanup releases EAL-allocated resources, ensuring that no hugepage
// memory is leaked. It is expected that all DPDK applications call
// rte_eal_cleanup() before exiting. Not calling this function could
// result in leaking hugepages, leading to failure during
// initialization of secondary processes.
//
// All lcores are signalled to stop. Please make sure that lcore
// functions returned otherwise this function will block until that
// happens.
//
// This function should be called from outside of EAL threads.
func Cleanup() error {
	return ealDeInit()
}

func parseCmd(input string) ([]string, error) {
	s := bufio.NewScanner(strings.NewReader(input))
	s.Split(common.SplitFunc(common.DefaultSplitter))

	var argv []string
	for s.Scan() {
		argv = append(argv, s.Text())
	}
	return argv, s.Err()
}

// Init initializes EAL as in rte_eal_init. Options are
// specified in a unparsed command line string. This string is parsed
// and InitWithArgs is then called upon.
func Init(input string) error {
	argv, err := parseCmd(input)
	if err != nil {
		return err
	}
	return InitWithArgs(argv)
}

// InitWithParams initializes EAL as in rte_eal_init. Options are
// specified with arrays of parameters which are then joined
// and InitWithArgs is then called upon.
//
// program may be arbitrary name, you may want to set it to os.Args[0].
func InitWithParams(program string, p ...Parameter) error {
	return InitWithArgs(append([]string{program}, Join(p)...))
}

// HasHugePages tells if huge pages are activated.
func HasHugePages() bool {
	return int(C.rte_eal_has_hugepages()) != 0
}

// ProcessType returns the current process type.
func ProcessType() int {
	return int(C.rte_eal_process_type())
}

// LcoreCount returns number of CPU logical cores configured by EAL.
func LcoreCount() uint {
	return uint(C.rte_lcore_count())
}

// GetMasterLcore returns CPU logical core id where the master thread
// is executed.
func GetMasterLcore() uint {
	return uint(C.rte_get_master_lcore())
}
