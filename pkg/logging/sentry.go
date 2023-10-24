package logging

import (
	"github.com/evalphobia/logrus_sentry"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func ConfigureReporter(logger Logger, dsn, env string, tags map[string]string) {
	hook, err := logrus_sentry.NewSentryHook(dsn, []logrus.Level{
		logrus.WarnLevel,
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
	})
	if err != nil {
		panic(errors.Wrap(err, "ConfigureReporter"))
	}
	hook.SetTagsContext(tags)
	hook.StacktraceConfiguration.Enable = true
	hook.SetEnvironment(env)
	AddHookToLogger(logger, hook)
}
