package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/afero"
	yhubcli "github.com/willpinha/yhub/internal/cli"
)

func main() {
	app := yhubcli.NewApp(afero.NewOsFs())
	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
