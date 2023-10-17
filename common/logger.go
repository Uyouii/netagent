package common

import (
	"context"
	"log"
)

type Logf func(format string, fields ...interface{})

type GetLogfFunc func(ctx context.Context) Logf

func GetErrorf(ctx context.Context) Logf {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds | log.LstdFlags)
	return log.Printf
}

func GetInfof(ctx context.Context) Logf {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds | log.LstdFlags)
	return log.Printf
}

func GetLogFuns(ctx context.Context) (Logf, Logf) {
	return GetInfof(ctx), GetErrorf(ctx)
}
