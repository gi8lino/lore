// lore-plugin develops ordinary Go plugins for Lore's WASI runtime.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/gi8lino/lore/pluginsdk/develop"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := develop.Run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "lore-plugin:", err)
		os.Exit(1)
	}
}
