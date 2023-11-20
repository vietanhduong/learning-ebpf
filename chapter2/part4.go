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
Part 4:
The RAW_TRACEPOINT_PROBE macro provided by BCC simplifies attaching to raw tracepoints, telling the
user space BCC code to automatically attach it to a specified tracepoint. Try it in hello-tail.py, like this:
• Replace the definition of the hello() function with RAW_TRACEPOINT_PROBE(sys_enter).
• Remove the explicit attachment call b.attach_raw_tracepoint() from the Python code.
You should see that BCC automatically creates the attachment and the program works exactly the same.
This is an example of the many convenient macros that BCC provides.
*/

const PART4_BPF_CODE = `
struct data_t {
	u64 uid;
	int syscall;
};

BPF_HASH(output, struct data_t, u64);

RAW_TRACEPOINT_PROBE(sched_switch) 
{
	struct data_t data = {};
	data.syscall = ctx->args[1];
	data.uid = bpf_get_current_uid_gid() & 0xFFFFFFFF;
	output.increment(data);
	return 0;
}
`

func part4(ctx context.Context) error {
	mod, err := bcc.NewModule(PART4_BPF_CODE)
	if err != nil {
		return fmt.Errorf("new BCC module: %w", err)
	}
	defer mod.Close()

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
		time.Sleep(time.Second)
		it := output.Iter()
		for it.Next() {
			data := (*data)(unsafe.Pointer(&it.Key()[0]))
			cnt := (*uint64)(unsafe.Pointer(&it.Leaf()[0]))
			helpers.PrintOut("syscall=%d uid=%d cnt=%d", data.syscall, data.uid, *cnt)
		}
		helpers.PrintOut("-------------------")
	}
}
