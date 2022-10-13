package mock

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/golang/mock/gomock"
)

// DetachMode defines the mode for detaching mock calls.
type DetachMode int

const (
	// None mode to not detach mode.
	None DetachMode = 0
	// Head mode to detach head, i.e. do not order mock calls after predecessor
	// mock calls provided via context.
	Head DetachMode = 1
	// Tail mode to deteach tail, i.e. do not order mock calls before successor
	// mock calls provided via context.
	Tail DetachMode = 2
	// Both mode to detach tail and head, i.e. do neither order mock calls after
	// predecessor nor before successor provided via context.
	Both DetachMode = 3
)

// String return string representation of detach mode.
func (m DetachMode) String() string {
	switch m {
	case None:
		return "None"
	case Head:
		return "Head"
	case Tail:
		return "Tail"
	case Both:
		return "Both"
	default:
		return "Unknown"
	}
}

type (
	// Call alias for `gomock.Call`
	Call = gomock.Call
	// Controller alias for `gomock.Controller`
	Controller = gomock.Controller

	// chain is the type to signal that mock calls must and will be orders in a
	// chain of mock calls.
	chain any
	// parallel is the type to signal that mock calls must and will be orders
	// in a parallel set of mock calls.
	parallel any
	// detachHead is the type to signal that the leading mock call must and
	// will be detached from its predecessor.
	detachHead any
	// detachTail is the type to signal that the trailing mock call must and
	// will be detached from its successor.
	detachTail any
	// detachBoth is the type to signal that the mock call must and will be
	// deteched from its predecessor as well as from its successor.
	detachBoth any
)

// SetupFunc common mock setup function signature.
type SetupFunc func(*Mocks) any

// Mocks common mock handler.
type Mocks struct {
	ctrl  *Controller
	wg    *sync.WaitGroup
	mocks map[reflect.Type]any
}

// NewMock creates a new mock handler using given test reporter (`*testing.T`).
func NewMock(t gomock.TestReporter) *Mocks {
	return &Mocks{
		ctrl:  gomock.NewController(t),
		wg:    &sync.WaitGroup{},
		mocks: map[reflect.Type]any{},
	}
}

// Expect configures the mock handler to expect the given mock function calls.
func (mocks *Mocks) Expect(fncalls SetupFunc) *Mocks {
	if fncalls != nil {
		Setup(fncalls)(mocks)
	}
	return mocks
}

// WaitGroup returns the `WaitGroup` of the mock handler to wait at when the
// tests comprises mock calls in detached `go-routines`.
func (mocks *Mocks) WaitGroup() *sync.WaitGroup {
	return mocks.wg
}

// Get resolves the actual mock from the mock handler by providing the
// constructor function generated by `gomock` to create a new mock.
func Get[T any](mocks *Mocks, creator func(*Controller) *T) *T {
	ctype := reflect.TypeOf(creator)
	tmock, ok := mocks.mocks[ctype]
	if ok && tmock != nil {
		return tmock.(*T)
	}
	tmock = creator(mocks.ctrl)
	mocks.mocks[ctype] = tmock
	return tmock.(*T)
}

// Setup creates only a lazily ordered set of mock calls that is detached from
// the parent setup by returning no calls for chaining. The mock calls created
// by the setup are only validated in so far in relation to each other, that
// `gomock` delivers results for the same mock call receiver in the order
// provided during setup.
func Setup[T any](calls ...func(*T) any) func(*T) any {
	return func(mock *T) any {
		for _, call := range calls {
			inOrder([]*Call{}, []detachBoth{call(mock)})
		}
		return nil
	}
}

// Chain creates a single chain of mock calls that is validated by `gomock`.
// If the execution order deviates from the order defined in the chain, the
// test validation fails. The method returns the full mock calls tree to allow
// chaining with other ordered setup method.
func Chain[T any](fncalls ...func(*T) any) func(*T) any {
	return func(mock *T) any {
		calls := make([]chain, 0, len(fncalls))
		for _, fncall := range fncalls {
			calls = chainCalls(calls, fncall(mock))
		}
		return calls
	}
}

// Parallel creates a set of parallel set of mock calls that is validated by
// `gomock`. While the parallel setup provids some freedom, this still defines
// constrainst with repect to parent and child setup methods, e.g. when setting
// up parallel chains in a chain, each parallel chains needs to follow the last
// mock call and finish before the following mock call.
//
// If the execution order deviates from the order defined by the parallel
// context, the test validation fails. The method returns the full set of mock
// calls to allow combining them with other ordered setup methods.
func Parallel[T any](fncalls ...func(*T) any) func(*T) any {
	return func(mock *T) any {
		calls := make([]parallel, 0, len(fncalls))
		for _, fncall := range fncalls {
			calls = append(calls, fncall(mock).(parallel))
		}
		return calls
	}
}

// Detach detach given mock call setup using given detach mode. It is possible
// to detach the mock call from the preceeding mock calls (`Head`), from the
// succeeding mock calls (`Tail`), or from both as used in `Setup`.
func Detach[T any](mode DetachMode, fncall func(*T) any) func(*T) any {
	return func(mock *T) any {
		switch mode {
		case None:
			return fncall(mock)
		case Head:
			return []detachHead{fncall(mock)}
		case Tail:
			return []detachTail{fncall(mock)}
		case Both:
			return []detachBoth{fncall(mock)}
		default:
			panic(ErrDetachMode(mode))
		}
	}
}

// Sub returns the sub slice of mock calls starting at index `from` up to index
// `to` inclduing. A negative value is used to calculate an index from the end
// of the slice. If the index of `from` is higher as the index `to`, the
// indexes are automatically switched. The returned sub slice of mock calls
// keeps its original semantic.
func Sub[T any](from, to int, fncall func(*T) any) func(*T) any {
	return func(mock *T) any {
		calls := fncall(mock)
		switch calls := any(calls).(type) {
		case *Call:
			inOrder([]*Call{}, calls)
			return GetSubSlice(from, to, []any{calls})
		case []chain:
			inOrder([]*Call{}, calls)
			return GetSubSlice(from, to, calls)
		case []parallel:
			inOrder([]*Call{}, calls)
			return GetSubSlice(from, to, calls)
		case []detachBoth:
			panic(ErrDetachNotAllowed(Both))
		case []detachHead:
			panic(ErrDetachNotAllowed(Head))
		case []detachTail:
			panic(ErrDetachNotAllowed(Tail))
		case nil:
			return nil
		default:
			panic(ErrNoCall(calls))
		}
	}
}

// GetSubSlice returns the sub slice of mock calls starting at index `from`
// up to index `to` inclduing. A negative value is used to calculate an index
// from the end of the slice. If the index `from` is after the index `to`, the
// indexes are automatically switched.
func GetSubSlice[T any](from, to int, calls []T) any {
	from = getPos(from, calls)
	to = getPos(to, calls)
	if from > to {
		return calls[to : from+1]
	} else if from < to {
		return calls[from : to+1]
	}
	return calls[from]
}

// getPos returns the actual call position evaluating negative positions
// from the back of the mock call slice.
func getPos[T any](pos int, calls []T) int {
	len := len(calls)
	if pos < 0 {
		pos = len + pos
		if pos < 0 {
			return 0
		}
		return pos
	} else if pos < len {
		return pos
	}
	return len - 1
}

// chainCalls joins arbitray slices, single mock calls, and parallel mock calls
// into a single mock call slice and slice of mock slices. If the provided mock
// calls do not contain mock calls or slices of them, the join fails with a
// `panic`.
func chainCalls(calls []chain, more ...any) []chain {
	for _, call := range more {
		switch call := any(call).(type) {
		case *Call:
			calls = append(calls, call)
		case []chain:
			calls = append(calls, call...)
		case []parallel:
			calls = append(calls, call)
		case []detachBoth:
			calls = append(calls, call)
		case []detachHead:
			calls = append(calls, call)
		case []detachTail:
			calls = append(calls, call)
		case nil:
		default:
			panic(ErrNoCall(call))
		}
	}
	return calls
}

// inOrder creates an order of the given mock call using given anchors as
// predecessor and return the mock call as next anchor. The created order
// depends on the actual type of the mock call (slice).
func inOrder(anchors []*Call, call any) []*Call {
	switch call := any(call).(type) {
	case *Call:
		return inOrderCall(anchors, call)
	case []parallel:
		return inOrderParallel(anchors, call)
	case []chain:
		return inOrderChain(anchors, call)
	case []detachBoth:
		return inOrderDetachBoth(anchors, call)
	case []detachHead:
		return inOrderDetachHead(anchors, call)
	case []detachTail:
		return inOrderDetachTail(anchors, call)
	case nil:
		return anchors
	default:
		panic(ErrNoCall(call))
	}
}

// inOrderCall creates an order for the given mock call using the given achors
// as predecessor and resturn the call as next anchor.
func inOrderCall(anchors []*Call, call *Call) []*Call {
	if len(anchors) != 0 {
		for _, anchor := range anchors {
			if anchor != call {
				call.After(anchor)
			}
		}
	}
	return []*Call{call}
}

// inOrderChain creates a chain order of the given mock calls using given
// anchors as predecessor and return the last mocks call as next anchor.
func inOrderChain(anchors []*Call, calls []chain) []*Call {
	for _, call := range calls {
		anchors = inOrder(anchors, call)
	}
	return anchors
}

// inOrderParallel creates a parallel order the given mock calls using the
// anchors as predecessors and return list of all (last) mock calls as next
// anchors.
func inOrderParallel(anchors []*Call, calls []parallel) []*Call {
	nanchors := make([]*Call, 0, len(calls))
	for _, call := range calls {
		nanchors = append(nanchors, inOrder(anchors, call)...)
	}
	return nanchors
}

// inOrderDetachBoth creates a detached set of mock calls without using the
// anchors as predecessor nor returning the last mock calls as next anchor.
func inOrderDetachBoth(anchors []*Call, calls []detachBoth) []*Call {
	for _, call := range calls {
		inOrder(nil, call)
	}
	return anchors
}

// inOrderDetachHead creates a head detached set of mock calls without using
// the anchors as predecessor. The anchors are forwarded together with the new
// mock calls as next anchors.
func inOrderDetachHead(anchors []*Call, calls []detachHead) []*Call {
	for _, call := range calls {
		anchors = append(anchors, inOrder(nil, call)...)
	}
	return anchors
}

// inOrderDetachTail creates a tail detached set of mock calls using the
// anchors as predessors but without adding the mock calls as next anchors.
// The provided anchors are provided as next anchors.
func inOrderDetachTail(anchors []*Call, calls []detachTail) []*Call {
	for _, call := range calls {
		inOrder(anchors, call)
	}
	return anchors
}

// ErrNoCall creates an error with given call type to panic on inorrect call
// type.
func ErrNoCall(call any) error {
	return fmt.Errorf("type [%v] is not based on *gomock.Call",
		reflect.TypeOf(call))
}

// ErrDetachMode creates an error that the given detach mode is not supported.
func ErrDetachMode(mode DetachMode) error {
	return fmt.Errorf("detach mode [%v] is not supported", mode)
}

// ErrDetachNotAllowed creates an error that detach.
func ErrDetachNotAllowed(mode DetachMode) error {
	return fmt.Errorf("detach [%v] not supported in sub", mode)
}
