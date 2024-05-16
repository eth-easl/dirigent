package main

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"net/http"
	"os"
	"strconv"
	"time"
)

// static double SQRTSD (double x) {
//     double r;
//     __asm__ ("sqrtsd %1, %0" : "=x" (r) : "x" (x));
//     return r;
// }
import "C"

const ExecUnit int = 1e2

var machineName string

func takeSqrts() C.double {
	var tmp C.double // Circumvent compiler optimizations
	for i := 0; i < ExecUnit; i++ {
		tmp = C.SQRTSD(C.double(10))
	}
	return tmp
}

func busySpin(multiplier, runtimeMilli uint32) {
	totalIterations := int(multiplier * runtimeMilli)

	for i := 0; i < totalIterations; i++ {
		takeSqrts()
	}
}

func busyLoopFor(timeLeftMilliseconds uint32, start time.Time, mpl int) {
	timeConsumedMilliseconds := uint32(time.Since(start).Milliseconds())
	if timeConsumedMilliseconds < timeLeftMilliseconds {
		timeLeftMilliseconds -= timeConsumedMilliseconds
		if timeLeftMilliseconds > 0 {
			busySpin(uint32(mpl), timeLeftMilliseconds)
		}
	}
}

func rootHandler(w http.ResponseWriter, req *http.Request) {
	workload := req.Header.Get("workload")
	function := req.Header.Get("function")
	requestedCpu := req.Header.Get("requested_cpu")
	multiplier := req.Header.Get("multiplier")

	switch workload {
	case "empty":
		responseBytes, _ := json.Marshal(FunctionResponse{
			Status:        "OK - EMPTY",
			Function:      function,
			MachineName:   machineName,
			ExecutionTime: 0,
		})

		_, _ = w.Write(responseBytes)
		w.WriteHeader(http.StatusOK)
	case "trace":
		tlm, err := strconv.Atoi(requestedCpu)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		mpl, err := strconv.Atoi(multiplier)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		start := time.Now()
		busyLoopFor(uint32(tlm), start, mpl)

		responseBytes, _ := json.Marshal(FunctionResponse{
			Status:        "OK",
			Function:      function,
			MachineName:   machineName,
			ExecutionTime: time.Since(start).Microseconds(),
		})

		_, _ = w.Write(responseBytes)
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

type FunctionResponse struct {
	Status        string `json:"Status"`
	Function      string `json:"Function"`
	MachineName   string `json:"MachineName"`
	ExecutionTime int64  `json:"ExecutionTime"`
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

type Multiplexer struct {
	handlers map[string]func(w http.ResponseWriter, r *http.Request)
	Handler  http.HandlerFunc
}

func NewMultiplexer() *Multiplexer {
	multiplexer := &Multiplexer{
		handlers: make(map[string]func(w http.ResponseWriter, r *http.Request)),
	}
	multiplexer.init()
	return multiplexer
}

func (mux *Multiplexer) init() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// you can allow only http2 request
		//if r.ProtoMajor != 2 {
		//	log.Println("HTTP/2 request is rejected")
		//	w.WriteHeader(http.StatusInternalServerError)
		//	return
		//}

		f := mux.GetHandler(r.URL.Path)
		if f == nil {
			log.Println("unknown path", r.URL.Path)
			return
		}
		f(w, r)
	})
	mux.Handler = handler
}

func (mux *Multiplexer) GetHandler(path string) func(w http.ResponseWriter, r *http.Request) {
	f, ok := mux.handlers[path]
	if ok {
		return f
	}
	return nil
}

func (mux *Multiplexer) HandleFunc(path string, f func(w http.ResponseWriter, r *http.Request)) {
	mux.handlers[path] = f
}

func busyLoopOnStartup() {
	loopForMs := 0
	loopForMsString, ok := os.LookupEnv("COLD_START_BUSY_LOOP_MS")
	if ok {
		loopForMs, _ = strconv.Atoi(loopForMsString)
	}

	multiplier := 102
	multiplierString, ok := os.LookupEnv("ITER_MULTIPLIER")
	if ok {
		multiplier, _ = strconv.Atoi(multiplierString)
	}

	start := time.Now()
	busyLoopFor(uint32(loopForMs), start, multiplier)
	log.Debugf("Spent %d ms on startup", time.Since(start).Milliseconds())
}

func StartHTTPServer() {
	var err error
	machineName, err = os.Hostname()
	if err != nil {
		log.Fatal("Failed to get HOSTNAME environmental variable.")
	}

	busyLoopOnStartup()

	mux := NewMultiplexer()
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/health", healthHandler)

	server := &http.Server{
		Addr:    "0.0.0.0:80",
		Handler: h2c.NewHandler(mux.Handler, &http2.Server{}),
	}

	err = server.ListenAndServe()
	if err != nil {
		log.Fatal("Failed to set up an HTTP server - ", err)
	}
}
