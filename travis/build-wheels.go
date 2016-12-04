package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

func main() {
	pips, _ := filepath.Glob("/opt/python/*/bin/pip")

	os.Setenv("PATH", os.Getenv("PATH")+":/usr/local/go/bin")

	// Compile wheels in parallel
	var wg sync.WaitGroup
	for _, pip := range pips {
		if strings.Contains(pip, "p26") {
			// We don't support Python 2.6
			continue
		}
		pip := pip // Local copy for closure
		wg.Add(1)
		go func() {
			defer wg.Done()
			run(pip, "install", "numpy")
			//run(pip, "wheel", "--only-binary", ":all", "/io/", "-w", "wheelhouse/")
			run(pip, "wheel", "--no-deps", "/io/", "-w", "wheelhouse/")
		}()
		break
	}
	wg.Wait()

	// Bundle external shared libraries into the wheels
	whls, _ := filepath.Glob("wheelhouse/*.whl")
	for _, whl := range whls {
		run("auditwheel", "repair", whl, "-w", "/io/wheelhouse/")
	}
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func run(program string, arg ...string) {
	cmd := exec.Command(program, arg...)
	fmt.Println(strings.Join(cmd.Args, " "))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	check(cmd.Run())
}
