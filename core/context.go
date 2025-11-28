package core

import (
	"github.com/gocrud/app/config"
	"github.com/gocrud/app/logging"
)

// ConfigurationContext 提供配置期间所需的受限能力
// 仅暴露只读方法，避免在配置阶段进行副作用操作
type ConfigurationContext interface {
	GetConfiguration() config.Configuration
	GetEnvironment() Environment
	GetLogger() logging.Logger
}
