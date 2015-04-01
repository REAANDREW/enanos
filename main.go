package main

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"gopkg.in/alecthomas/kingpin.v1"
	"os"
	"time"
)

const (
	ENV_ENANOS_PORT         string = "ENANOS_PORT"
	ENV_ENANOS_VERBOSE      string = "ENANOS_VERBOSE"
	ENV_ENANOS_HOST         string = "ENANOS_HOST"
	ENV_ENANOS_MIN_SLEEP    string = "ENANOS_MIN_SLEEP"
	ENV_ENANOS_MAX_SLEEP    string = "ENANOS_MAX_SLEEP"
	ENV_ENANOS_RANDOM_SLEEP string = "ENANOS_RANDOM_SLEEP"
	ENV_ENANOS_MIN_SIZE     string = "ENANOS_MIN_SIZE"
	ENV_ENANOS_MAX_SIZE     string = "ENANOS_MAX_SIZE"
	ENV_ENANOS_RANDOM_SIZE  string = "ENANOS_RANDOM_SIZE"
	ENV_ENANOS_DEAD_TIME    string = "ENANOS_DEAD_TIME"
)

var (
	verbose     = kingpin.Flag("verbose", "Enable verbose mode.").Short('v').OverrideDefaultFromEnvar(ENV_ENANOS_VERBOSE).Bool()
	port        = kingpin.Flag("port", "the port to host the server on").Default("8000").Short('p').OverrideDefaultFromEnvar(ENV_ENANOS_PORT).Int()
	host        = kingpin.Flag("host", "this host for enanos to bind to").Default("0.0.0.0").OverrideDefaultFromEnvar(ENV_ENANOS_HOST).String()
	minSleep    = kingpin.Flag("min-sleep", "the minimum sleep time for the wait endpoint e.g. 5ms, 5s, 5m etc...").Default("1s").OverrideDefaultFromEnvar(ENV_ENANOS_MIN_SLEEP).String()
	maxSleep    = kingpin.Flag("max-sleep", "the maximum sleep time for the wait endpoint e.g. 5ms, 5s, 5m etc...").Default("60s").OverrideDefaultFromEnvar(ENV_ENANOS_MAX_SLEEP).String()
	randomSleep = kingpin.Flag("random-sleep", "whether to sleep a random time between min and max or just the max").Default("false").OverrideDefaultFromEnvar(ENV_ENANOS_RANDOM_SLEEP).Bool()
	minSize     = kingpin.Flag("min-size", "the minimum size of response body for the content_size endpoint e.g. 5B, 5KB, 5MB etc...").Default("10KB").OverrideDefaultFromEnvar(ENV_ENANOS_MIN_SIZE).String()
	maxSize     = kingpin.Flag("max-size", "the maximum size of response body for the content_size endpoint e.g. 5B, 5KB, 5MB etc...").Default("100KB").OverrideDefaultFromEnvar(ENV_ENANOS_MAX_SIZE).String()
	randomSize  = kingpin.Flag("random-size", "whether to return a random sized payload between min and max or just max").Default("false").OverrideDefaultFromEnvar(ENV_ENANOS_RANDOM_SIZE).Bool()
	deadTime    = kingpin.Flag("dead-time", "the time which the server should remain dead before coming back online").Default("5s").OverrideDefaultFromEnvar(ENV_ENANOS_DEAD_TIME).String()
	content     = kingpin.Flag("content", "the content to return for OK responses").Default("hello world").String()
	headers     = kingpin.Flag("header", "response headers to be returned. Key:Value").Short('H').Strings()
	config      = kingpin.Flag("config", "config file used to configure enanos.  Supported providers include file.").Default("empty").Short('c').String()
)

func main() {
	kingpin.Version("1.1.0")
	kingpin.CommandLine.Help = `Enanos is an investigation tool in the form of a HTTP server with several endpoints that can be used to substitute the actual http service dependencies of a system.  This tool allows you to see how a system will perform against varying un-stable http services, each which exhibit different effects.
	
	/success		- will return a 200 response code
	/server_error		- will return a random 5XX response code 
	/content_size		- will return a 200 response code but a response body with a size between <minSize> and <maxSize>.  The content returned will be random or a mangled version of the content which has been configured to return i.e. it cannot guarantee to meet any content-types configured in that it will be malformed.
	/wait			- will return a 200 response code but only after a random sleep between <minSleep> and <maxSleep>
	/redirect		- will return a random 3XX response code.  If the response code is one which redirects then Bashful will return its own location to invite an infinite redirect loop
	/client_error		- will return a random 4XX response code
	/dead_or_alive	- will kill the server and only bring it back online after configured amount of time (ms) has passed

	/defined?code=<code>	- will return the specified http status code
	`
	kingpin.Parse()

	var commandLineArgs = CommandLineArgs{}
	commandLineArgs.content = *content
	commandLineArgs.deadTime = *deadTime
	commandLineArgs.headers = *headers
	commandLineArgs.host = *host
	commandLineArgs.maxSize = *maxSize
	commandLineArgs.maxWait = *maxSleep
	commandLineArgs.minSize = *minSize
	commandLineArgs.minWait = *minSleep
	commandLineArgs.port = *port
	commandLineArgs.randomSize = *randomSize
	commandLineArgs.randomWait = *randomSleep
	commandLineArgs.verbose = *verbose

	var snoozer Snoozer = createSnoozer()
	var responseBodyGenerator ResponseBodyGenerator = createResponseBodyGenerator()
	var responseCodeGenerator ResponseCodeGenerator = NewRandomResponseCodeGenerator(responseCodes_300, responseCodes_400, responseCodes_500)
	var argsReader = NewArgsConfigurationReader(commandLineArgs)
	var config = argsReader.Read()

	fmt.Println(fmt.Sprintf("Enanos Server listening on port %d", *port))
	StartEnanos(config, responseBodyGenerator, responseCodeGenerator, snoozer)
}

func createSnoozer() Snoozer {
	minSleepValue, minSleepErr := time.ParseDuration(*minSleep)
	maxSleepValue, maxSleepErr := time.ParseDuration(*maxSleep)

	if minSleepErr != nil || maxSleepErr != nil {
		fmt.Errorf("Invalid duration specified for min or max sleep")
		os.Exit(1)
	}

	if *randomSleep {
		return NewRandomSnoozer(minSleepValue, maxSleepValue)
	} else {
		return NewMaxSnoozer(maxSleepValue)
	}
}

func createResponseBodyGenerator() ResponseBodyGenerator {
	minSizeValue, minSizeErr := humanize.ParseBytes(*minSize)
	maxSizeValue, maxSizeErr := humanize.ParseBytes(*maxSize)

	if minSizeErr != nil || maxSizeErr != nil {
		fmt.Errorf("Invalid size specified for min or max size")
		os.Exit(1)
	}

	if *randomSize {
		return NewRandomResponseBodyGenerator(int(minSizeValue), int(maxSizeValue))
	} else {
		return NewMaxResponseBodyGenerator(int(maxSizeValue))
	}
}
