package helpers

import (
	"fmt"
	"os"
)

var (
	stdout = os.Stdout
	stderr = os.Stderr
)

func PrintOut(format string, arg ...any) { print(stdout, format, arg...) }
func PrintErr(format string, arg ...any) { print(stderr, format, arg...) }

func print(fd *os.File, format string, arg ...any) { fmt.Fprintf(fd, format+"\n", arg...) }
