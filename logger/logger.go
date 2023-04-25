package logger

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	formatConsole = "console"
	formatJSON    = "json"

	levelDebug = "DEBUG"
	levelInfo  = "INFO"
	levelWarn  = "WARN"
	levelError = "ERROR"
)

var defaultLogger *zap.Logger

func getLogger() *zap.Logger {
	checkNil()
	return defaultLogger.WithOptions(zap.AddCallerSkip(1))
}

// Open initial log settings
// print the debug level log in the terminal, example: Init()
// print the info level log in the terminal, example: Init(WithLevel("info"))
// print the json format, debug level log in the terminal, example: Init(WithFormat("json"))
// output the log to the file out.log, using the default cut log-related parameters, debug-level log, example: Init(WithSave())
// output the log to the specified file, custom set the log file cut log parameters, json format, debug level log, example:
// Open(
//
//	  WithFormat("json"),
//	  WithSave(true,
//
//			WithFileName("my.log"),
//			WithFileMaxSize(5),
//			WithFileMaxBackups(5),
//			WithFileMaxAge(10),
//			WithFileIsCompression(true),
//		))

func Open(opts ...Option) (*zap.Logger, error) {
	o := defaultOptions()
	o.apply(opts...)
	isSave := o.isSave
	levelName := o.level
	encoding := o.encoding

	var err error
	var zapLog *zap.Logger
	//var str string
	if !isSave {
		zapLog, err = log2Terminal(levelName, encoding)
		if err != nil {
			panic(err)
		}
		//	str = fmt.Sprintf("initialize logger finish, config is output to 'terminal', format=%s, level=%s", encoding, levelName)
	} else {
		zapLog = log2File(encoding, levelName, o.fileConfig)
		//	str = fmt.Sprintf("initialize logger finish, config is output to 'file', format=%s, level=%s, file=%s", encoding, levelName, o.fileConfig.filename)
	}
	if o.stacktrace != "" {
		zapLog = zapLog.WithOptions(zap.AddStacktrace(getLevelSize(o.stacktrace)))
	}
	return zapLog, err
}

func Init(opts ...Option) (*zap.Logger, error) {

	var err error

	defaultLogger, err = Open(opts...)

	return defaultLogger, err
}

func log2Terminal(levelName string, encoding string) (*zap.Logger, error) {
	js := fmt.Sprintf(`{
      		"level": "%s",
            "encoding": "%s",
      		"outputPaths": ["stdout"],
            "errorOutputPaths": ["stdout"]
		}`, levelName, encoding)

	var config zap.Config
	err := json.Unmarshal([]byte(js), &config)
	if err != nil {
		return nil, err
	}

	config.EncoderConfig = zap.NewProductionEncoderConfig()
	config.EncoderConfig.EncodeTime = timeFormatter                // default time format
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder // logging levels in the log file using upper case letters
	config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder // default caller format
	config.DisableStacktrace = true

	return config.Build()
}

func log2File(encoding string, levelName string, fo *fileOptions) *zap.Logger {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder   // modify Time Encoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder // logging levels in the log file using upper case letters
	var encoder zapcore.Encoder
	if encoding == formatConsole { // console format
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else { // json format
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	ws := zapcore.AddSync(&lumberjack.Logger{
		Filename:   fo.filename,      // file name
		MaxSize:    fo.maxSize,       // maximum file size (MB)
		MaxBackups: fo.maxBackups,    // maximum number of old files
		MaxAge:     fo.maxAge,        // maximum number of days for old documents
		Compress:   fo.isCompression, // whether to compress and archive old files
	})
	core := zapcore.NewCore(encoder, ws, getLevelSize(levelName))

	// add the function call information log to the log.
	return zap.New(core, zap.AddCaller())
}

// DEBUG(default), INFO, WARN, ERROR
func getLevelSize(levelName string) zapcore.Level {
	levelName = strings.ToUpper(levelName)
	switch levelName {
	case levelDebug:
		return zapcore.DebugLevel
	case levelInfo:
		return zapcore.InfoLevel
	case levelWarn:
		return zapcore.WarnLevel
	case levelError:
		return zapcore.ErrorLevel
	}
	return zapcore.DebugLevel
}

func timeFormatter(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

// GetWithSkip get defaultLogger, set the skipped caller value, customize the number of lines of code displayed
func GetWithSkip(skip int) *zap.Logger {
	checkNil()
	return defaultLogger.WithOptions(zap.AddCallerSkip(skip))
}

// Get logger
func Get() *zap.Logger {
	checkNil()
	return defaultLogger
}

func checkNil() {
	if defaultLogger == nil {
		_, err := Init() // default output to console
		if err != nil {
			panic(err)
		}
	}
}
