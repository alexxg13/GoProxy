package app

import (
	"GoProxy/config"
	"context"
	"fmt"

	"github.com/alexxg13/go-utils/logger"
	"github.com/google/uuid"
)

func Run(cfg *config.Config) {
	ctx := context.Background()

	ctx = context.WithValue(ctx, logger.LoggerEnvKey, cfg.Log.Level)
	ctx = logger.WithRequestID(ctx, uuid.NewString())
	ctx = logger.WithTraceID(ctx, uuid.NewString())

	ctx, err := logger.New(ctx)
	if err != nil {
		fmt.Printf("failed to create logger: %v\n", err)
		return
	}
	defer logger.GetLogger(ctx).Sync()

}
