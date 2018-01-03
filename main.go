package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ConfWatch/confwatchd/config"
	"github.com/ConfWatch/confwatchd/controllers"
	"github.com/ConfWatch/confwatchd/log"
	"github.com/ConfWatch/confwatchd/middlewares"
	"github.com/ConfWatch/confwatchd/models"

	"github.com/gin-gonic/autotls"
	"github.com/gin-gonic/gin"
	"github.com/michelloworld/ez-gin-template"
)

var (
	signals    = make(chan os.Signal, 1)
	confFile   = ""
	debug      = false
	logfile    = ""
	noColors   = false
	seedFolder = ""
	router     = (*gin.Engine)(nil)
)

func init() {
	flag.StringVar(&confFile, "config", "config.json", "JSON configuration file.")
	flag.StringVar(&seedFolder, "seed", seedFolder, "Seed the database with the data from this folder.")
	flag.BoolVar(&debug, "log-debug", debug, "Enable debug logs.")
	flag.StringVar(&logfile, "log-file", logfile, "Log messages to this file instead of standard error.")
	flag.BoolVar(&noColors, "log-colors-off", noColors, "Disable colored output.")
}

func setupSignals() {
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	s := <-signals
	log.Raw("\n")
	log.Importantf("RECEIVED SIGNAL: %s", s)
	os.Exit(1)
}

func setupLogging() {
	var err error

	log.WithColors = !noColors

	if logfile != "" {
		log.Output, err = os.Create(logfile)
		if err != nil {
			log.Fatal(err)
		}

		defer log.Output.Close()
	}

	if debug == true {
		log.MinLevel = log.DEBUG
	} else {
		log.MinLevel = log.INFO
	}
}

func main() {
	flag.Parse()

	go setupSignals()

	setupLogging()

	if confFile != "" {
		if err := config.Load(confFile); err != nil {
			log.Fatal(err)
		}
	}

	if err := models.Setup(config.Conf.Database); err != nil {
		log.Fatal(err)
	}

	if seedFolder != "" {
		log.Infof("Seeding database from %s ...", log.Bold(seedFolder))
		if err := models.Seed(seedFolder); err != nil {
			log.Fatal(err)
		}
		return
	}

	if config.Conf.Dev {
		log.Infof("Running in dev mode.")
	} else {
		log.Infof("Running in prod mode.")
	}
	gin.SetMode(gin.ReleaseMode)

	render := eztemplate.New()
	render.TemplatesDir = "views/"
	render.Layout = "layouts/base"
	render.Ext = ".html"
	render.Debug = config.Conf.Dev

	router = gin.New()

	router.HTMLRender = render.Init()
	router.Use(middlewares.Security())
	router.Use(middlewares.ServeStatic("/", "static", "index.html"))

	router.GET("/events", controllers.ListEvents)
	router.GET("/events/:event_name", controllers.ShowEvent)
	router.GET("/events/:event_name/editions/:edition_name", controllers.ShowEdition)

	address := fmt.Sprintf("%s:%d", config.Conf.Address, config.Conf.Port)
	if address[0] == ':' {
		address = "0.0.0.0" + address
	}

	log.Infof("%s v%s is running on %s ...", config.APP_NAME, config.APP_VERSION, log.Bold(config.Conf.Hosts[0]))
	if config.Conf.Dev {
		log.Fatal(router.Run(address))
	} else {
		log.Fatal(autotls.Run(router, config.Conf.Hosts...))
	}
}
