package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/afero"
	yhubcli "github.com/willpinha/yhub/internal/cli"
	gitpkg "github.com/willpinha/yhub/internal/git"
)

func main() {
	app := yhubcli.NewApp(afero.NewOsFs(), gitpkg.New())
	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
