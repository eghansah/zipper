package main

import (
	"fmt"
	"net/http"
	"os"
	"sundry_reports/utils"

	"github.com/go-chi/chi/v5"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type config struct {
	Host       string `mapstructure:"HOST"`
	Port       int    `mapstructure:"PORT"`
	URLPrefix  string `mapstructure:"URL_PREFIX"`
	LogPath    string `mapstructure:"LOG_PATH"`
	ZipCommand string `mapstructure:"ZIP_COMMAND"`
}

type service struct {
	config *config
	logger *zap.SugaredLogger
	router *chi.Mux
	svr    *http.Server
}

func getLogEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	// The format time can be customized
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	// encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewJSONEncoder(encoderConfig)
}

func main() {
	var (
		cfg config
		err error
	)

	MANDATORY_ENV_VARS := []string{}

	viper.AutomaticEnv()
	for _, k := range MANDATORY_ENV_VARS {
		if !viper.IsSet(k) {
			panic(fmt.Sprintf("'%s' environment variable needs to be set", k))
		}
	}

	//Bind env vars
	for _, k := range utils.GetMapstructureTags(config{}) {
		viper.BindEnv(k, k)
		// fmt.Println(k)
	}

	hooks := mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	)

	//Set default config values
	cfg.Host = "127.0.0.1"
	cfg.Port = 5000

	//Read config from env variables
	err = viper.Unmarshal(&cfg, viper.DecodeHook(hooks))
	if err != nil {
		panic(err)
	}

	// log.Printf("Config ===> %+v\n", cfg)

	logfile := utils.LogFile{
		LogFileName: "service",
		LogPath:     cfg.LogPath,
	}

	var syncer zapcore.WriteSyncer
	if cfg.LogPath == "" {
		syncer = zap.CombineWriteSyncers(os.Stdout)
	} else {
		syncer = zap.CombineWriteSyncers(os.Stdout, &logfile)
	}
	encoder := getLogEncoder()
	core := zapcore.NewCore(encoder, syncer, zapcore.DebugLevel)
	// Print function lines
	logger := zap.New(core, zap.AddCaller()).Sugar()

	svc := service{config: &cfg, logger: logger}
	svc.Init(cfg)
	svc.Run()

}
