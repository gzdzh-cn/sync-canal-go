package main

import (
	_ "sync-canal-go/internal/packed"

	"github.com/gogf/gf/v2/os/gctx"

	"sync-canal-go/internal/cmd"
)

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
