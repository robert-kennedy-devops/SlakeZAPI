package whatsapp

import (
	"fmt"

	"github.com/whatsapp-saas/api/pkg/logger"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type whatsmeowLogger struct {
	base   *logger.Logger
	module string
}

func newWhatsmeowLogger(base *logger.Logger, module string) waLog.Logger {
	return &whatsmeowLogger{base: base, module: module}
}

func (l *whatsmeowLogger) Warnf(msg string, args ...interface{}) {
	l.base.Warn(fmt.Sprintf(msg, args...), map[string]interface{}{"component": l.module})
}

func (l *whatsmeowLogger) Errorf(msg string, args ...interface{}) {
	l.base.Error(fmt.Sprintf(msg, args...), map[string]interface{}{"component": l.module})
}

func (l *whatsmeowLogger) Infof(msg string, args ...interface{}) {
	l.base.Info(fmt.Sprintf(msg, args...), map[string]interface{}{"component": l.module})
}

func (l *whatsmeowLogger) Debugf(msg string, args ...interface{}) {
	l.base.Debug(fmt.Sprintf(msg, args...), map[string]interface{}{"component": l.module})
}

func (l *whatsmeowLogger) Sub(module string) waLog.Logger {
	return &whatsmeowLogger{
		base:   l.base,
		module: fmt.Sprintf("%s/%s", l.module, module),
	}
}
