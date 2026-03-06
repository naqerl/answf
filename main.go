package main

import (
        "errors"
        "fmt"
        "os"

        "github.com/naqerl/answf/internal/app"
        "github.com/naqerl/answf/internal/cli"
)

func main() {
        cfg, err := cli.Parse(os.Args[1:], os.Getenv)
        if err != nil {
                if errors.Is(err, cli.ErrUsage) {
                        cli.PrintUsage(os.Stderr)
                        os.Exit(0)
                }
                fmt.Fprintf(os.Stderr, "error: %v\n", err)
                os.Exit(2)
        }

        out, err := app.Run(cfg)
        if err != nil {
                fmt.Fprintf(os.Stderr, "error: %v\n", err)
                os.Exit(1)
        }

        fmt.Print(out)
}
