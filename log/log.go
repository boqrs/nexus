package log

import (
	"fmt"
	"io"

	"github.com/boqrs/zeus/log"
	rotate2 "github.com/boqrs/zeus/log/writer/rotate"
)

type LogConfig struct {
	Level string `json:"level" yaml:"level" mapstructure:"level"`
	Dir   string `json:"dir" yaml:"dir" mapstructure:"dir"`
}

func InitLogger(cfg LogConfig) (log.Logger, error) {
	var level log.Level
	level, has := levelString[cfg.Level]
	if !has {
		return nil, fmt.Errorf("log level is error")
	}

	if len(cfg.Dir) == 0 {
		fmt.Println("log dir is empty, be careful~~~~~~~~~~~~~~~~")
	}

	var wr io.Writer
	var err error
	if len(cfg.Dir) > 0 {
		//writer option
		wopts := []rotate2.Option{
			rotate2.WithLogDir(cfg.Dir),
			rotate2.WithLogSubDir("info"),
			rotate2.WithLogSubDir("error"),
			rotate2.WithLogSubDir("debug"),
		}

		wr, err = rotate2.NewWriter(wopts...)
		if err != nil {
			panic(err)
			return nil, err
		}
	}

	opts := []log.Option{
		log.WithLevelEnabler(level),
		log.WithEncoderCfg(log.NewEncoderConfig()),
		log.AddCallerSkip(1),
		log.AddCaller(),
		log.SetProjectName("communal"),
	}

	if len(cfg.Dir) > 0 {
		opts = append(opts, log.WithWriter(wr))
	}

	return log.New(log.ZapLogger, opts...)
}

var levelString = map[string]log.Level{
	"debug": log.DebugLevel,
	"info":  log.InfoLevel,
	"error": log.ErrorLevel,
	"fatal": log.FatalLevel,
}
