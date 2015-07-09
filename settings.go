package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

const (
	VERSION = "0.9.4"
)

// Allows to specify multiple flags with same name and collects all values to array
type MultiOption []string

func (h *MultiOption) String() string {
	return fmt.Sprint(*h)
}

func (h *MultiOption) Set(value string) error {
	*h = append(*h, value)
	return nil
}

type AppSettings struct {
	verbose bool
	stats   bool

	splitOutput bool

	inputDummy  MultiOption
	outputDummy MultiOption

	inputTCP       MultiOption
	outputTCP      MultiOption
	outputTCPStats bool

	inputFile  MultiOption
	outputFile MultiOption

	inputRAW MultiOption

	inputHTTP  MultiOption
	outputHTTP MultiOption

	outputHTTPConfig HTTPOutputConfig
	modifierConfig   HTTPModifierConfig
}

var Settings AppSettings = AppSettings{}

func usage() {
	fmt.Printf("Gor is a simple http traffic replication tool written in Go. Its main goal is to replay traffic from production servers to staging and dev environments.\nProject page: https://github.com/buger/gor\nAuthor: <Leonid Bugaev> leonsbox@gmail.com\nCurrent Version: %s\n\n", VERSION)
	flag.PrintDefaults()
	os.Exit(2)
}

func init() {
	flag.Usage = usage

	flag.BoolVar(&Settings.verbose, "verbose", false, "Turn on verbose/debug output")
	flag.BoolVar(&Settings.stats, "stats", false, "Turn on queue stats output")

	flag.BoolVar(&Settings.splitOutput, "split-output", false, "By default each output gets same traffic. If set to `true` it splits traffic equally among all outputs.")

	flag.Var(&Settings.inputDummy, "input-dummy", "Used for testing outputs. Emits 'Get /' request every 1s")
	flag.Var(&Settings.outputDummy, "output-dummy", "Used for testing inputs. Just prints data coming from inputs.")

	flag.Var(&Settings.inputTCP, "input-tcp", "Used for internal communication between Gor instances. Example: \n\t# Receive requests from other Gor instances on 28020 port, and redirect output to staging\n\tgor --input-tcp :28020 --output-http staging.com")
	flag.Var(&Settings.outputTCP, "output-tcp", "Used for internal communication between Gor instances. Example: \n\t# Listen for requests on 80 port and forward them to other Gor instance on 28020 port\n\tgor --input-raw :80 --output-tcp replay.local:28020")
	flag.BoolVar(&Settings.outputTCPStats, "output-tcp-stats", false, "Report TCP output queue stats to console every 5 seconds.")

	flag.Var(&Settings.inputFile, "input-file", "Read requests from file: \n\tgor --input-file ./requests.gor --output-http staging.com")
	flag.Var(&Settings.outputFile, "output-file", "Write incoming requests to file: \n\tgor --input-raw :80 --output-file ./requests.gor")

	flag.Var(&Settings.inputRAW, "input-raw", "Capture traffic from given port (use RAW sockets and require *sudo* access):\n\t# Capture traffic from 8080 port\n\tgor --input-raw :8080 --output-http staging.com")

	flag.Var(&Settings.inputHTTP, "input-http", "Read requests from HTTP, should be explicitly sent from your application:\n\t# Listen for http on 9000\n\tgor --input-http :9000 --output-http staging.com")

	flag.Var(&Settings.outputHTTP, "output-http", "Forwards incoming requests to given http address.\n\t# Redirect all incoming requests to staging.com address \n\tgor --input-raw :80 --output-http http://staging.com")
	flag.IntVar(&Settings.outputHTTPConfig.workers, "output-http-workers", 0, "Gor uses dynamic worker scaling by default.  Enter a number to run a set number of workers.")
	flag.IntVar(&Settings.outputHTTPConfig.redirectLimit, "output-http-redirects", 0, "Enable how often redirects should be followed.")

	flag.BoolVar(&Settings.outputHTTPConfig.stats, "output-http-stats", false, "Report http output queue stats to console every 5 seconds.")

	flag.StringVar(&Settings.outputHTTPConfig.elasticSearch, "output-http-elasticsearch", "", "Send request and response stats to ElasticSearch:\n\tgor --input-raw :8080 --output-http staging.com --output-http-elasticsearch 'es_host:api_port/index_name'")

	flag.Var(&Settings.modifierConfig.headers, "http-set-header", "Inject additional headers to http reqest:\n\tgor --input-raw :8080 --output-http staging.com --http-set-header 'User-Agent: Gor'")
	flag.Var(&Settings.modifierConfig.headers, "output-http-header", "WARNING: `--output-http-header` DEPRECATED, use `--http-set-header` instead")

	flag.Var(&Settings.modifierConfig.params, "http-set-param", "Set request url param, if param already exists it will be overwritten:\n\tgor --input-raw :8080 --output-http staging.com --http-set-param api_key=1")

	flag.Var(&Settings.modifierConfig.methods, "http-allow-method", "Whitelist of HTTP methods to replay. Anything else will be dropped:\n\tgor --input-raw :8080 --output-http staging.com --http-allow-method GET --http-allow-method OPTIONS")
	flag.Var(&Settings.modifierConfig.methods, "output-http-method", "WARNING: `--output-http-method` DEPRECATED, use `--http-allow-method` instead")

	flag.Var(&Settings.modifierConfig.urlRegexp, "http-allow-url", "A regexp to match requests against. Filter get matched agains full url with domain. Anything else will be dropped:\n\t gor --input-raw :8080 --output-http staging.com --http-allow-url ^www.")
	flag.Var(&Settings.modifierConfig.urlRegexp, "output-http-url-regexp", "WARNING: `--output-http-url-regexp` DEPRECATED, use `--http-allow-url` instead")

	flag.Var(&Settings.modifierConfig.urlNegativeRegexp, "http-diallow-url", "A regexp to match requests against. Filter get matched agains full url with domain. Anything else will be dropped:\n\t gor --input-raw :8080 --output-http staging.com --http-disallow-url ^www.")

	flag.Var(&Settings.modifierConfig.urlRewrite, "http-rewrite-url", "Rewrite the request url based on a mapping:\n\tgor --input-raw :8080 --output-http staging.com --http-rewrite-url /v1/user/([^\\/]+)/ping:/v2/user/$1/ping")
	flag.Var(&Settings.modifierConfig.urlRewrite, "output-http-rewrite-url", "WARNING: `--output-http-rewrite-url` DEPRECATED, use `--http-rewrite-url` instead")

	flag.Var(&Settings.modifierConfig.headerFilters, "http-allow-header", "A regexp to match a specific header against. Requests with non-matching headers will be dropped:\n\t gor --input-raw :8080 --output-http staging.com --http-allow-header api-version:^v1")
	flag.Var(&Settings.modifierConfig.headerFilters, "output-http-header-filter", "WARNING: `--output-http-header-filter` DEPRECATED, use `--http-allow-header` instead")

	flag.Var(&Settings.modifierConfig.headerHashFilters, "http-allow-header-hash", "Takes a fraction of requests, consistently taking or rejecting a request based on the FNV32-1A hash of a specific header:\n\t gor --input-raw :8080 --output-http staging.com --http-allow-header-hash user-id:25%")
	flag.Var(&Settings.modifierConfig.headerHashFilters, "output-http-header-hash-filter", "WARNING: `output-http-header-hash-filter` DEPRECATED, use `--http-allow-header-hash` instead")

	flag.Var(&Settings.modifierConfig.paramHashFilters, "http-allow-param-hash", "Takes a fraction of requests, consistently taking or rejecting a request based on the FNV32-1A hash of a specific GET param:\n\t gor --input-raw :8080 --output-http staging.com --http-allow-param-hash user_id:25%")
}

func Debug(args ...interface{}) {
	if Settings.verbose {
		log.Print("[DEBUG] ")
		log.Println(args...)
	}
}
