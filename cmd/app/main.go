package main

import (
	"context"
	"fmt"
	"os"

	"diplomaMeetHelper/internal/app"
)

func main() {
	if err := app.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
