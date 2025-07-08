package deps

import (
	"github.com/and161185/loyalty/internal/auth"
	"go.uber.org/zap"
)

type Deps struct {
	Logger       *zap.SugaredLogger
	TokenManager *auth.TokenManager
}

func NewDependencies(secretKey string) *Deps {
	logCfg := zap.NewProductionConfig()
	logCfg.OutputPaths = []string{"stdout", "server.log"}

	logger := zap.Must(logCfg.Build())

	deps := Deps{Logger: logger.Sugar(), TokenManager: auth.NewTokenManager(secretKey)}

	return &deps
}
