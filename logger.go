
package hoom

import (
	"os"

	"github.com/kmcsr/go-logger"
	logrusl "github.com/kmcsr/go-logger/logrus"
)

var loger logger.Logger = initLogger()

func initLogger()(logger.Logger){
	loger := logrusl.Logger
	loger.SetOutput(os.Stderr)
	return loger
}

func Logger()(logger.Logger){
	return loger
}

func SetLogger(l logger.Logger){
	loger = l
}
