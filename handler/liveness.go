package handler

import (
	"fmt"
	"net/http"
)

// LivenessCheck will handle the process of requesting the liveness status
// from the server and returning the correct status depending on the response
// from lib in order to inform GTM of the state of the server
func LivenessCheck(w ResponseWriter, request *http.Request, ctx noAuthWebContext) (err error) {

	statusOK := true

	// TODO: Include liveness check function option which can be configured by the server
	livFunc := ctx.GetLivenessCheckFunc()
	if livFunc != nil {
		rttTime, err := (*livFunc)(ctx.GetDB())
		if err != nil {
			w.WriteHeader(http.StatusConflict)
			ctx.GetLogger().Error(err.Error())
			statusOK = false
		} else {
			if rttTime > ctx.conf.Database.MaxRTT {
				w.WriteHeader(http.StatusConflict)
				statusOK = false
			}
		}

		msg := fmt.Sprintf("Status OK? %t\nDB RTT: %f ms\n", statusOK, rttTime)

		_, writeErr := w.Write([]byte(msg))

		return writeErr
	}

	msg := "Status OK\n"

	_, writeErr := w.Write([]byte(msg))

	return writeErr

}
