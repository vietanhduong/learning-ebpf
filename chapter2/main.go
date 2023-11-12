package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/vietanhduong/learning-ebpf/helpers"
)

func main() {
	var part uint
	flag.UintVar(&part, "part", 0, "Part must be 1 to 5 (following by the chapter 2).")
	flag.Parse()

	if part < 1 || part > 5 {
		helpers.PrintErr("-part must be greater than 0 and less than 6")
		flag.Usage()
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM)
	defer stop()

	parts := [5]func(context.Context) error{
		part1,
		part2,
		part3,
		part4,
	}

	if fn := parts[part-1]; fn != nil {
		if err := fn(ctx); err != nil {
			helpers.PrintErr("Error: %v", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	helpers.PrintErr("Error: unimplement")
	os.Exit(1)
}
