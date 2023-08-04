package profiler

import (
	"cluster_manager/pkg/config"
	"log"
	"net/http"
	"runtime"
)

func SetupProfilerServer(config config.ProfilerConfig) {
	if config.Mutex {
		runtime.SetMutexProfileFraction(1)
	}

	if config.Enable {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}
}
