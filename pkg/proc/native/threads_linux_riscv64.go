package native

import (
	"debug/elf"
	"fmt"
	"syscall"
	"unsafe"

	sys "golang.org/x/sys/unix"

	"github.com/go-delve/delve/pkg/proc"
	"github.com/go-delve/delve/pkg/proc/linutil"
)

func (thread *nativeThread) fpRegisters() ([]proc.Register, []byte, error) {
	var err error
	var riscv64_fpregs linutil.RISCV64PtraceFpRegs

	thread.dbp.execPtraceFunc(func() { err = ptraceGetFpRegset(thread.ID, &riscv64_fpregs) })
	fpregs := riscv64_fpregs.Decode()

	if err != nil {
		err = fmt.Errorf("could not get floating point registers: %v", err.Error())
	}

	return fpregs, riscv64_fpregs.Fregs, err
}

func (t *nativeThread) restoreRegisters(savedRegs proc.Registers) error {
	var restoreRegistersErr error

	sr := savedRegs.(*linutil.RISCV64Registers)
	t.dbp.execPtraceFunc(func() {
		restoreRegistersErr = ptraceSetGRegs(t.ID, sr.Regs)
		if restoreRegistersErr != syscall.Errno(0) {
			return
		}

		if sr.Fpregset != nil {
			iov := sys.Iovec{Base: &sr.Fpregset[0], Len: uint64(len(sr.Fpregset))}
			_, _, restoreRegistersErr = syscall.Syscall6(syscall.SYS_PTRACE, sys.PTRACE_SETREGSET, uintptr(t.ID), uintptr(elf.NT_FPREGSET), uintptr(unsafe.Pointer(&iov)), 0, 0)
		}
	})

	if restoreRegistersErr == syscall.Errno(0) {
		restoreRegistersErr = nil
	}

	return restoreRegistersErr
}
