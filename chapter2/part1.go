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

// Part 1: Adapt the hello-buffer.py eBPF program to output different trace messages for odd and even process IDs.

const PART1_BPF_CODE = `
BPF_PERF_OUTPUT(output);

struct data_t {
	int pid;
	int uid;
	char command[16];
	char message[12];
};

int hello(void *ctx) { 
	struct data_t data = {};
	char message[12];
	strcpy(message, "Odd PID");
	
	data.pid = bpf_get_current_pid_tgid() >> 32;
	data.uid = bpf_get_current_uid_gid() & 0xFFFFFFFF;
	if (data.pid%2==0) {
		strcpy(message, "Even PID");
	}

	bpf_get_current_comm(&data.command, sizeof(data.command));
	bpf_probe_read_kernel(&data.message, sizeof(data.message), message);

	output.perf_submit(ctx, &data, sizeof(data));
	return 0;
}
`

func part1(ctx context.Context) error {
	mod, err := bcc.NewModule(PART1_BPF_CODE, nil)
	if err != nil {
		return fmt.Errorf("new BCC: %w", err)
	}
	defer mod.Close()
	helpers.PrintOut("Init BPF module successful!")

	fd, err := mod.LoadKprobe("hello")
	if err != nil {
		return fmt.Errorf("load kprobe: %w", err)
	}
	helpers.PrintOut("Loaded Kprobe hello (fd=%d)", fd)

	syscall := bcc.GetSyscallFnName("execve")
	if err = mod.AttachKprobe(syscall, fd, -1); err != nil {
		return fmt.Errorf("attack kprobe: %w", err)
	}
	helpers.PrintOut("Attached kprobe to syscall %s", syscall)

	type data struct {
		pid     int32
		uid     int32
		command [16]byte
		message [12]byte
	}
	cb := helpers.NewCallback(func(raw []byte, _ int32) {
		d := (*data)(unsafe.Pointer(&raw[0]))
		cmd := (*C.char)(unsafe.Pointer(&d.command[0]))
		msg := (*C.char)(unsafe.Pointer(&d.message[0]))
		helpers.PrintOut("%d %d %s %s", d.pid, d.uid, C.GoString(cmd), C.GoString(msg))
	}, nil)

	if err = mod.OpenPerfBuffer("output", cb, 0); err != nil {
		return fmt.Errorf("open perf buffer: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		time.Sleep(500 * time.Millisecond)
		mod.PollPerfBuffer("output", 0)
	}
}
