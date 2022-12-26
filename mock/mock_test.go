package mock_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tkrop/go-testing/mock"
	"github.com/tkrop/go-testing/perm"
	"github.com/tkrop/go-testing/test"
)

//go:generate mockgen -package=mock_test -destination=mock_iface_test.go -source=mock_test.go  IFace

type IFace interface {
	CallA(string)
	CallB(string) string
}

func CallA(input string) mock.SetupFunc {
	return func(mocks *mock.Mocks) any {
		return mock.Get(mocks, NewMockIFace).EXPECT().
			CallA(input).Times(mocks.Times(1)).
			Do(mocks.GetDone(1))
	}
}

func CallB(input string, output string) mock.SetupFunc {
	return func(mocks *mock.Mocks) any {
		return mock.Get(mocks, NewMockIFace).EXPECT().
			CallB(input).Return(output).
			Times(mocks.Times(1)).Do(mocks.GetDone(1))
	}
}

func NoCall() mock.SetupFunc {
	return func(mocks *mock.Mocks) any {
		return mock.Get(mocks, NewMockIFace).EXPECT()
	}
}

func MockSetup(t gomock.TestReporter, mockSetup mock.SetupFunc) *mock.Mocks {
	return mock.NewMock(t).Expect(mockSetup)
}

func MockValidate(
	t test.Test, mocks *mock.Mocks,
	validate func(test.Test, *mock.Mocks),
	failing bool,
) {
	if failing {
		// we need to execute failing test synchronous, since we setup full
		// permutations instead of stopping setup on first failing mock calls.
		validate(t, mocks)
	} else {
		// Test proper usage of `WaitGroup` on non-failing validation.
		validate(t, mocks)
		mocks.Wait()
	}
}

func SetupPermTestABC(mocks *mock.Mocks) *perm.Test {
	iface := mock.Get(mocks, NewMockIFace)
	return perm.NewTest(mocks,
		perm.TestMap{
			"a": func(test.Test) { iface.CallA("a") },
			"b1": func(t test.Test) {
				assert.Equal(t, "c", iface.CallB("b"))
			},
			"b2": func(t test.Test) {
				assert.Equal(t, "d", iface.CallB("b"))
			},
			"c": func(test.Test) { iface.CallA("c") },
		})
}

func SetupPermTestABCD(mocks *mock.Mocks) *perm.Test {
	iface := mock.Get(mocks, NewMockIFace)
	return perm.NewTest(mocks,
		perm.TestMap{
			"a": func(test.Test) { iface.CallA("a") },
			"b": func(test.Test) { iface.CallA("b") },
			"c": func(t test.Test) {
				assert.Equal(t, "d", iface.CallB("c"))
			},
			"d": func(t test.Test) {
				assert.Equal(t, "e", iface.CallB("d"))
			},
		})
}

func SetupPermTestABCDEF(mocks *mock.Mocks) *perm.Test {
	iface := mock.Get(mocks, NewMockIFace)
	return perm.NewTest(mocks,
		perm.TestMap{
			"a": func(test.Test) { iface.CallA("a") },
			"b": func(test.Test) { iface.CallA("b") },
			"c": func(t test.Test) {
				assert.Equal(t, "d", iface.CallB("c"))
			},
			"d": func(t test.Test) {
				assert.Equal(t, "e", iface.CallB("d"))
			},
			"e": func(test.Test) { iface.CallA("e") },
			"f": func(test.Test) { iface.CallA("f") },
		})
}

var testSetupParams = perm.ExpectMap{
	"b2-b1-a-c": test.Failure,
	"b2-b1-c-a": test.Failure,
	"b2-c-b1-a": test.Failure,
	"b2-a-b1-c": test.Failure,
	"b2-c-a-b1": test.Failure,
	"b2-a-c-b1": test.Failure,
	"a-b2-b1-c": test.Failure,
	"c-b2-b1-a": test.Failure,
	"a-b2-c-b1": test.Failure,
	"c-b2-a-b1": test.Failure,
	"c-a-b2-b1": test.Failure,
	"a-c-b2-b1": test.Failure,
}

func TestSetup(t *testing.T) {
	perms := testSetupParams.Remain(test.Success)
	test.Map(t, perms).Run(func(t test.Test, expect test.Expect) {
		// Given
		name := strings.Split(t.Name(), "/")[1]
		perm := strings.Split(name, "-")
		mockSetup := mock.Setup(
			CallA("a"),
			mock.Setup(
				CallB("b", "c"),
				CallB("b", "d"),
			),
			CallA("c"),
		)
		mock := MockSetup(t, mockSetup)

		// When
		test := SetupPermTestABC(mock)

		// Then
		test.Test(t, perm, expect)
	})
}

var testChainParams = perm.ExpectMap{
	"a-b1-b2-c": test.Success,
}

func TestChain(t *testing.T) {
	perms := testChainParams.Remain(test.Failure)
	test.Map(t, perms).Run(func(t test.Test, expect test.Expect) {
		// Given
		name := strings.Split(t.Name(), "/")[1]
		perm := strings.Split(name, "-")
		mockSetup := mock.Chain(
			CallA("a"),
			mock.Chain(
				CallB("b", "c"),
				CallB("b", "d"),
			),
			CallA("c"),
		)
		mock := MockSetup(t, mockSetup)

		// When
		test := SetupPermTestABC(mock)

		// Then
		test.Test(t, perm, expect)
	})
}

var testSetupChainParams = perm.ExpectMap{
	"a-b-c-d": test.Success,
	"a-c-b-d": test.Success,
	"a-c-d-b": test.Success,
	"c-a-b-d": test.Success,
	"c-a-d-b": test.Success,
	"c-d-a-b": test.Success,
}

func TestSetupChain(t *testing.T) {
	perms := testSetupChainParams.Remain(test.Failure)
	test.Map(t, perms).Run(func(t test.Test, expect test.Expect) {
		// Given
		name := strings.Split(t.Name(), "/")[1]
		perm := strings.Split(name, "-")

		// Basic setup of two independent chains.
		mockSetup := mock.Setup(
			mock.Chain(
				CallA("a"),
				CallA("b"),
			),
			mock.Chain(
				CallB("c", "d"),
				CallB("d", "e"),
			),
		)
		mock := MockSetup(t, mockSetup)

		// When
		test := SetupPermTestABCD(mock)

		// Then
		test.Test(t, perm, expect)
	})
}

func TestChainSetup(t *testing.T) {
	perms := testSetupChainParams.Remain(test.Failure)
	test.Map(t, perms).Run(func(t test.Test, expect test.Expect) {
		// Given
		name := strings.Split(t.Name(), "/")[1]
		perm := strings.Split(name, "-")

		// Frail setup to detach a (sub-)chain.
		mockSetup := mock.Chain(
			CallA("a"),
			CallA("b"),
			mock.Setup( // detaching (sub-)chain.
				mock.Chain(
					CallB("c", "d"),
					CallB("d", "e"),
				),
			),
		)
		mock := MockSetup(t, mockSetup)

		// When
		test := SetupPermTestABCD(mock)

		// Then
		test.Test(t, perm, expect)
	})
}

var testParallelChainParams = perm.ExpectMap{
	"a-b-c-d-e-f": test.Success,
	"a-b-c-e-d-f": test.Success,
	"a-b-e-c-d-f": test.Success,
	"a-c-b-d-e-f": test.Success,
	"a-c-d-b-e-f": test.Success,
	"a-c-d-e-b-f": test.Success,
	"a-c-b-e-d-f": test.Success,
	"a-c-e-d-b-f": test.Success,
	"a-c-e-b-d-f": test.Success,
	"a-e-b-c-d-f": test.Success,
	"a-e-c-b-d-f": test.Success,
	"a-e-c-d-b-f": test.Success,
}

func TestParallelChain(t *testing.T) {
	perms := testParallelChainParams.Remain(test.Failure)
	test.Map(t, perms).Run(func(t test.Test, expect test.Expect) {
		// Given
		name := strings.Split(t.Name(), "/")[1]
		perm := strings.Split(name, "-")
		mockSetup := mock.Chain(
			CallA("a"),
			mock.Parallel(
				CallA("b"),
				mock.Chain(
					CallB("c", "d"),
					CallB("d", "e"),
				),
				mock.Parallel(
					CallA("e"),
				),
			),
			CallA("f"),
		)
		mock := MockSetup(t, mockSetup)

		// When
		test := SetupPermTestABCDEF(mock)

		// Then
		test.Test(t, perm, expect)
	})
}

var testChainSubParams = perm.ExpectMap{
	"a-b-c-d-e-f": test.Success,
	"a-c-b-d-e-f": test.Success,
	"a-c-d-b-e-f": test.Success,
	"a-c-d-e-b-f": test.Success,
	"f-a-b-c-d-e": test.Success,
	"a-f-b-c-d-e": test.Success,
	"a-b-f-c-d-e": test.Success,
	"a-b-c-f-d-e": test.Success,
	"a-b-c-d-f-e": test.Success,
	"a-c-d-e-f-b": test.Success,

	"b-a-c-d-e-f": test.Failure,
	"c-a-b-d-e-f": test.Failure,
	"d-a-b-c-e-f": test.Failure,
	"a-b-c-e-d-f": test.Failure,
	"a-b-d-e-c-f": test.Failure,
}

func TestChainSub(t *testing.T) {
	perms := testChainSubParams
	// perms := testChainSubParams.Remain(test.ExpectSuccess)
	test.Map(t, perms).Run(func(t test.Test, expect test.Expect) {
		// Given
		name := strings.Split(t.Name(), "/")[1]
		perm := strings.Split(name, "-")
		mockSetup := mock.Chain(
			mock.Sub(0, 0, mock.Chain(
				CallA("a"),
				CallA("b"),
			)),
			mock.Sub(0, -1, mock.Parallel(
				CallB("c", "d"),
				CallB("d", "e"),
			)),
			mock.Sub(0, 0, CallA("e")),
			mock.Sub(2, 2, mock.Setup(CallA("f"))),
		)
		mock := MockSetup(t, mockSetup)

		// When
		test := SetupPermTestABCDEF(mock)

		// Then
		test.Test(t, perm, expect)
	})
}

var testDetachParams = perm.ExpectMap{
	"a-b-c-d": test.Success,
	"a-b-d-c": test.Success,
	"a-d-b-c": test.Success,
	"b-a-c-d": test.Success,
	"b-a-d-c": test.Success,
	"b-d-a-c": test.Success,
	"d-a-b-c": test.Success,
	"d-b-a-c": test.Success,
}

func TestDetach(t *testing.T) {
	perms := testDetachParams.Remain(test.Failure)
	test.Map(t, perms).Run(func(t test.Test, expect test.Expect) {
		// Given
		name := strings.Split(t.Name(), "/")[1]
		perm := strings.Split(name, "-")
		mockSetup := mock.Chain(
			mock.Detach(mock.None, CallA("a")),
			mock.Detach(mock.Head, CallA("b")),
			mock.Detach(mock.Tail, CallB("c", "d")),
			mock.Detach(mock.Both, CallB("d", "e")),
		)
		mock := MockSetup(t, mockSetup)

		// When
		test := SetupPermTestABCD(mock)

		// Then
		test.Test(t, perm, expect)
	})
}

type PanicParams struct {
	setup       mock.SetupFunc
	expectError error
}

var testPanicParams = map[string]PanicParams{
	"setup": {
		setup:       mock.Setup(NoCall()),
		expectError: mock.ErrNoCall(NewMockIFace(nil).EXPECT()),
	},
	"chain": {
		setup:       mock.Chain(NoCall()),
		expectError: mock.ErrNoCall(NewMockIFace(nil).EXPECT()),
	},
	"parallel": {
		setup:       mock.Parallel(NoCall()),
		expectError: mock.ErrNoCall(NewMockIFace(nil).EXPECT()),
	},
	"detach": {
		setup:       mock.Detach(4, NoCall()),
		expectError: mock.ErrDetachMode(4),
	},
	"sub": {
		setup:       mock.Sub(0, 0, NoCall()),
		expectError: mock.ErrNoCall(NewMockIFace(nil).EXPECT()),
	},
	"sub-head": {
		setup:       mock.Sub(0, 0, mock.Detach(mock.Head, NoCall())),
		expectError: mock.ErrDetachNotAllowed(mock.Head),
	},
	"sub-tail": {
		setup:       mock.Sub(0, 0, mock.Detach(mock.Tail, NoCall())),
		expectError: mock.ErrDetachNotAllowed(mock.Tail),
	},
	"sub-both": {
		setup:       mock.Sub(0, 0, mock.Detach(mock.Both, NoCall())),
		expectError: mock.ErrDetachNotAllowed(mock.Both),
	},
}

func TestPanic(t *testing.T) {
	test.Map(t, testPanicParams).Run(func(t test.Test, param PanicParams) {
		// Given
		defer func() {
			err := recover()
			assert.Equal(t, param.expectError, err)
		}()

		// When
		MockSetup(t, param.setup)

		// Then
		require.Fail(t, "not paniced")
	})
}

type GetSubSliceParams struct {
	slice       []any
	from, to    int
	expectSlice any
}

var testGetSubSliceParams = map[string]GetSubSliceParams{
	"first": {
		slice: []any{"a", "b", "c", "d", "e"},
		from:  0, to: 0,
		expectSlice: "a",
	},
	"last": {
		slice: []any{"a", "b", "c", "d", "e"},
		from:  -1, to: -1,
		expectSlice: "e",
	},
	"middle": {
		slice: []any{"a", "b", "c", "d", "e"},
		from:  2, to: 2,
		expectSlice: "c",
	},
	"begin": {
		slice: []any{"a", "b", "c", "d", "e"},
		from:  0, to: 2,
		expectSlice: []any{"a", "b", "c"},
	},
	"end": {
		slice: []any{"a", "b", "c", "d", "e"},
		from:  -3, to: -1,
		expectSlice: []any{"c", "d", "e"},
	},
	"all": {
		slice: []any{"a", "b", "c", "d", "e"},
		from:  0, to: -1,
		expectSlice: []any{"a", "b", "c", "d", "e"},
	},
	"sub": {
		slice: []any{"a", "b", "c", "d", "e"},
		from:  -2, to: 1,
		expectSlice: []any{"b", "c", "d"},
	},
	"out-of-bound": {
		slice: []any{"a", "b", "c", "d", "e"},
		from:  -7, to: 7,
		expectSlice: []any{"a", "b", "c", "d", "e"},
	},
}

func TestGetSubSlice(t *testing.T) {
	test.Map(t, testGetSubSliceParams).
		Run(func(t test.Test, param GetSubSliceParams) {
			// When
			slice := mock.GetSubSlice(param.from, param.to, param.slice)

			// Then
			assert.Equal(t, param.expectSlice, slice)
		})
}

type GetFuncParams struct {
	numargs int
	exist   bool
}

func call(fncall any, args int) {
	switch args {
	case 0:
		fncall.(func())()
	case 1:
		fncall.(func(any))(nil)
	case 2:
		fncall.(func(any, any))(nil, nil)
	case 3:
		fncall.(func(any, any, any))(nil, nil, nil)
	case 4:
		fncall.(func(any, any, any, any))(nil, nil, nil, nil)
	case 5:
		fncall.(func(
			any, any, any, any, any,
		))(nil, nil, nil, nil, nil)
	case 6:
		fncall.(func(
			any, any, any, any, any, any,
		))(nil, nil, nil, nil, nil, nil)
	case 7:
		fncall.(func(
			any, any, any, any, any, any, any,
		))(nil, nil, nil, nil, nil, nil, nil)
	case 8:
		fncall.(func(
			any, any, any, any, any, any, any, any,
		))(nil, nil, nil, nil, nil, nil, nil, nil)
	case 9:
		fncall.(func(
			any, any, any, any, any, any, any, any, any,
		))(nil, nil, nil, nil, nil, nil, nil, nil, nil)
	default:
		panic("not supported")
	}
}

var testGetFuncParams = map[string]GetFuncParams{
	"test 0 args":  {numargs: 0, exist: true},
	"test 1 args":  {numargs: 1, exist: true},
	"test 2 args":  {numargs: 2, exist: true},
	"test 3 args":  {numargs: 3, exist: true},
	"test 4 args":  {numargs: 4, exist: true},
	"test 5 args":  {numargs: 5, exist: true},
	"test 6 args":  {numargs: 6, exist: true},
	"test 7 args":  {numargs: 7, exist: true},
	"test 8 args":  {numargs: 8, exist: true},
	"test 9 args":  {numargs: 9, exist: true},
	"test 10 args": {numargs: 10},
	"test 11 args": {numargs: 11},
}

func TestGetDone(t *testing.T) {
	test.Map(t, testGetFuncParams).
		Run(func(t test.Test, param GetFuncParams) {
			// Given
			mocks := MockSetup(t, nil)
			mocks.Times(1)
			if !param.exist {
				defer func() { recover() }()
			}

			// When
			call(mocks.GetDone(param.numargs), param.numargs)

			// Then
			mocks.Wait()
			if !param.exist {
				assert.Fail(t, "not paniced on not supported argument number")
			}
		})
}

func TestGetPanic(t *testing.T) {
	test.Map(t, testGetFuncParams).
		Run(func(t test.Test, param GetFuncParams) {
			// Given
			mocks := MockSetup(t, nil)
			mocks.Times(1)
			defer func() {
				reason := recover()
				// Then
				if param.exist {
					require.Equal(t, "panic-test", reason)
					mocks.Wait()
				} else {
					assert.Equal(t, fmt.Sprintf(
						"argument number not supported: %d",
						param.numargs), reason)
				}
			}()

			// When
			call(mocks.GetPanic(param.numargs, "panic-test"), param.numargs)

			// Then
			assert.Fail(t, "not paniced on not supported argument number")
		})
}

func callVar(fncall any, args int) {
	switch args {
	case 1:
		fncall.(func(...any))(nil)
	case 2:
		fncall.(func(any, ...any))(nil, nil)
	case 3:
		fncall.(func(any, any, ...any))(nil, nil, nil)
	case 4:
		fncall.(func(any, any, any, ...any))(nil, nil, nil, nil)
	case 5:
		fncall.(func(
			any, any, any, any, ...any,
		))(nil, nil, nil, nil, nil)
	case 6:
		fncall.(func(
			any, any, any, any, any, ...any,
		))(nil, nil, nil, nil, nil, nil)
	case 7:
		fncall.(func(
			any, any, any, any, any, any, ...any,
		))(nil, nil, nil, nil, nil, nil, nil)
	case 8:
		fncall.(func(
			any, any, any, any, any, any, any, ...any,
		))(nil, nil, nil, nil, nil, nil, nil, nil)
	case 9:
		fncall.(func(
			any, any, any, any, any, any, any, any, ...any,
		))(nil, nil, nil, nil, nil, nil, nil, nil, nil)
	default:
		panic("not supported")
	}
}

var testGetVarFuncParams = map[string]GetFuncParams{
	"test 0 args":  {numargs: 0},
	"test 1 args":  {numargs: 1, exist: true},
	"test 2 args":  {numargs: 2, exist: true},
	"test 3 args":  {numargs: 3, exist: true},
	"test 4 args":  {numargs: 4, exist: true},
	"test 5 args":  {numargs: 5, exist: true},
	"test 6 args":  {numargs: 6, exist: true},
	"test 7 args":  {numargs: 7, exist: true},
	"test 8 args":  {numargs: 8, exist: true},
	"test 9 args":  {numargs: 9, exist: true},
	"test 10 args": {numargs: 10},
	"test 11 args": {numargs: 11},
}

func TestGetVarDone(t *testing.T) {
	test.Map(t, testGetVarFuncParams).
		Run(func(t test.Test, param GetFuncParams) {
			// Given
			mocks := MockSetup(t, nil)
			mocks.Times(1)
			if !param.exist {
				defer func() { recover() }()
			}

			// When
			callVar(mocks.GetVarDone(param.numargs), param.numargs)

			// Then
			mocks.Wait()
			if !param.exist {
				assert.Fail(t, "not paniced on not supported argument number")
			}
		})
}

func TestGetVarPanic(t *testing.T) {
	test.Map(t, testGetVarFuncParams).
		Run(func(t test.Test, param GetFuncParams) {
			// Given
			mocks := MockSetup(t, nil)
			mocks.Times(1)
			defer func() {
				reason := recover()
				// Then
				if param.exist {
					require.Equal(t, "panic-test", reason)
					mocks.Wait()
				} else {
					assert.Equal(t, fmt.Sprintf(
						"argument number not supported: %d",
						param.numargs), reason)
				}
			}()

			// When
			callVar(mocks.GetVarPanic(param.numargs, "panic-test"), param.numargs)

			// Then
			assert.Fail(t, "not paniced on not supported argument number")
		})
}

type FailureParam struct {
	expect test.Expect
	test   func(test.Test)
}

var testFailureParams = map[string]FailureParam{
	"success": {
		test:   func(test.Test) {},
		expect: test.Success,
	},

	"errorf": {
		test:   func(t test.Test) { t.Errorf("fail") },
		expect: test.Failure,
	},

	"fatalf": {
		test:   func(t test.Test) { t.Fatalf("fail") },
		expect: test.Failure,
	},

	"failnow": {
		test:   func(t test.Test) { t.FailNow() },
		expect: test.Failure,
	},

	"panic": {
		test:   func(t test.Test) { panic("panic") },
		expect: test.Failure,
	},
}

func TestFailures(t *testing.T) {
	test.Map(t, testFailureParams).
		Run(func(t test.Test, param FailureParam) {
			// Given
			mocks := mock.NewMock(t).Expect(CallA("a"))
			defer func() {
				if err := recover(); err != nil && err != "panic" {
					// Test thread will not wait on failures.
					mocks.Wait()
				}
			}()

			// When
			param.test(t)

			// Then
			mock.Get(mocks, NewMockIFace).CallA("a")
			mocks.Wait()
		})
}
