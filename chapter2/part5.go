package main

import (
	"context"
	"fmt"
	"time"
	"unsafe"

	"github.com/vietanhduong/go-bpf/bcc"
	"github.com/vietanhduong/learning-ebpf/helpers"
)

/*
Part 5:
You could further adapt hello_map.py so that the key in the hash table identifies a
particular syscall (rather than a particular user). The output will show how many
times that syscall has been called across the whole system.
*/

const PART5_BPF_CODE = `
BPF_HASH(counter, int, u64);

int hello(struct bpf_raw_tracepoint_args *ctx) {
	counter.increment(ctx->args[1]);
  return 0;
}
`

func part5(ctx context.Context) error {
	mod, err := bcc.NewModule(PART5_BPF_CODE)
	if err != nil {
		return fmt.Errorf("new BCC module: %w", err)
	}
	defer mod.Close()

	fd, err := mod.LoadRawTracepoint("hello")
	if err != nil {
		return fmt.Errorf("load raw tracepoint: %w", err)
	}

	if err = mod.AttachRawTracepoint("sys_enter", fd); err != nil {
		return fmt.Errorf("attach raw tracepoint: %w", err)
	}

	counter := bcc.NewTable(mod.TableId("counter"), mod)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		time.Sleep(time.Second)
		it := counter.Iter()
		for it.Next() {
			syscall := (*int32)(unsafe.Pointer(&it.Key()[0]))
			cnt := (*uint64)(unsafe.Pointer(&it.Leaf()[0]))
			helpers.PrintOut("syscall=%d cnt=%d", *syscall, *cnt)
		}
		helpers.PrintOut("-----------")
	}
}
