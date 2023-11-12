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
Part 3: The hello-tail.py eBPF program is an example of a program that attaches to the sys_enter
raw tracepoint that is hit whenever any syscall is called. Change hello-map.py to show the
total number of syscalls made by each user ID, by attaching it to that same sys_enter raw tracepoint.
*/

const PART3_BPF_CODE = `
struct data_t {
	u64 uid;
	int syscall;
};

BPF_HASH(output, struct data_t, u64);

int hello(struct bpf_raw_tracepoint_args *ctx) {
	struct data_t data = {};
	data.syscall = ctx->args[1];
	data.uid = bpf_get_current_uid_gid() & 0xFFFFFFFF;
	output.increment(data);
	return 0;
}
`

func part3(ctx context.Context) error {
	mod, err := bcc.NewModule(PART3_BPF_CODE, nil)
	if err != nil {
		return fmt.Errorf("new BCC module: %w", err)
	}
	defer mod.Close()

	fd, err := mod.LoadRawTracepoint("hello")
	if err != nil {
		return fmt.Errorf("load raw tracepoint: %w", err)
	}

	if err = mod.AttachRawTracepoint("sys_enter", fd); err != nil {
		return fmt.Errorf("attack raw tracepoint: %w", err)
	}

	output := bcc.NewTable(mod.TableId("output"), mod)

	type data struct {
		uid     uint64
		syscall int32
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		time.Sleep(500 * time.Millisecond)
		it := output.Iter()
		for it.Next() {
			data := (*data)(unsafe.Pointer(&it.Key()[0]))
			cnt := (*uint64)(unsafe.Pointer(&it.Leaf()[0]))
			helpers.PrintOut("syscall=%d uid=%d cnt=%d", data.syscall, data.uid, *cnt)
		}
		helpers.PrintOut("-------------------")
	}
}
