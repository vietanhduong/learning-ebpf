package main

import (
	"context"
	"fmt"
	"time"
	"unsafe"

	"github.com/vietanhduong/go-bpf/bcc"
	"github.com/vietanhduong/learning-ebpf/helpers"
)

import "C"

/*
Part 2: Modify hello-map.py so that the eBPF code gets triggered by more than one syscall.
For example, openat() is commonly called to open files, and write() is called to write data
to a file. You can start by attaching the hello eBPF program to multiple syscall kprobes.
Then try having modified versions of the hello eBPF program for different syscalls,
demonstrating that you can access the same map from multiple different programs.
*/

const PART2_BPF_CODE = `
struct data_t {
	u64 uid;
	char syscall[256];
};

BPF_HASH(counter_table, struct data_t, u64);

static __always_inline int count_syscall(char* syscall, void* ctx) {
	struct data_t data = {};
	strcpy(data.syscall, syscall);
  data.uid = bpf_get_current_uid_gid() & 0xFFFFFFFF;
  counter_table.increment(data);
  return 0;
}


int CFG_OPEN_AT_SYSCALL_FN (void* ctx) {
	return count_syscall(CFG_OPEN_AT_SYSCALL, ctx);
}

int CFG_WRITE_SYSCALL_FN (void* ctx) {
	return count_syscall(CFG_WRITE_SYSCALL, ctx);
}
`

func part2(ctx context.Context) error {
	openat := bcc.GetSyscallFnName("openat")
	openatFn := fmt.Sprintf("hello_%s", openat)
	write := bcc.GetSyscallFnName("write")
	writeFn := fmt.Sprintf("hello_%s", write)
	mod, err := bcc.NewModule(PART2_BPF_CODE, []string{
		fmt.Sprintf("-DCFG_OPEN_AT_SYSCALL_FN=%s", openatFn),
		fmt.Sprintf(`-DCFG_OPEN_AT_SYSCALL="%s"`, openat),
		fmt.Sprintf("-DCFG_WRITE_SYSCALL_FN=%s", writeFn),
		fmt.Sprintf(`-DCFG_WRITE_SYSCALL="%s"`, write),
	})
	if err != nil {
		return fmt.Errorf("new bcc module: %w", err)
	}
	defer mod.Close()

	fd, err := mod.LoadKprobe(openatFn)
	if err != nil {
		return fmt.Errorf("load kprobe %s: %w", openatFn, err)
	}

	if err = mod.AttachKprobe(openat, fd, MAX_ACTIVE); err != nil {
		return fmt.Errorf("attach kprobe %s: %w", openat, err)
	}

	if fd, err = mod.LoadKprobe(writeFn); err != nil {
		return fmt.Errorf("load kprobe %s: %w", writeFn, err)
	}

	if err = mod.AttachKprobe(write, fd, MAX_ACTIVE); err != nil {
		return fmt.Errorf("attach kprobe %s: %w", write, err)
	}

	cnttbl := bcc.NewTable(mod.TableId("counter_table"), mod)

	type data struct {
		uid     uint64
		syscall [256]byte
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		time.Sleep(500 * time.Millisecond)
		it := cnttbl.Iter()
		for it.Next() {
			d := (*data)(unsafe.Pointer(&it.Key()[0]))
			syscall := (*C.char)(unsafe.Pointer(&d.syscall[0]))
			cnt := (*uint64)(unsafe.Pointer(&it.Leaf()[0]))
			helpers.PrintOut("syscall=%s uid=%d cnt=%d", C.GoString(syscall), d.uid, *cnt)
		}
		helpers.PrintOut("-------------------")
	}
}
