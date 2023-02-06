package util

import (
	"log/syslog"
)

var LogWriter *syslog.Writer

func init() {
	LogWriter, _ = syslog.New(syslog.LOG_NOTICE, "cgar_collect")
}
