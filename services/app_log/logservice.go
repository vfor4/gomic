package applog

import (
	"io"
	stdlog "log"
	"net/http"
	"os"
	"time"
)

var applog *stdlog.Logger

type logPath string

func (lp logPath) Write(p []byte) (n int, err error) {
	f, err := os.OpenFile(time.Now().Format("2006.01.02"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return 0, nil
	}
	defer f.Close()
	return f.Write(p)
}

func Run(path string) {
	applog = stdlog.New(logPath(path), "", stdlog.LstdFlags)
}

func RegisterHandler() {
	http.HandleFunc("/log", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		msg, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		applog.Printf("%v\n", string(msg))
	})
}
