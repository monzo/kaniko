package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/executor"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
)

func respond(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func executeCommand(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	if debug := req.URL.Query().Get("debug"); debug != "" {
		_ = util.ConfigureLogging("debug")
		logrus.Debugf("Serving request: %+v", req)
		return
	}
	var opts *config.KanikoOptions
	err := decoder.Decode(&opts)
	if err != nil {
		respond(w, http.StatusBadRequest, fmt.Sprintf("Could not parse JSON body: %v", err))
		return
	}
	logrus.Infof("Received opts: %+v", opts)
	if opts.NoPush {
		_, err := executor.DoBuild(opts)
		if err != nil {
			respond(w, http.StatusInternalServerError, fmt.Sprintf("Error building image: %v", err))
		}
		respond(w, http.StatusOK, "Success")
		return
	}
	if err := executor.CheckPushPermissions(opts); err != nil {
		respond(w, http.StatusBadRequest, fmt.Sprintf("Error checking push permissions: %v", err))
		return
	}
	if err := os.Chdir("/"); err != nil {
		respond(w, http.StatusInternalServerError, fmt.Sprintf("Error changing to root dir: %v", err))
		return
	}
	image, err := executor.DoBuild(opts)
	if err != nil {
		respond(w, http.StatusInternalServerError, fmt.Sprintf("Error building image: %v", err))
		return
	}
	if err := executor.DoPush(image, opts); err != nil {
		respond(w, http.StatusInternalServerError, fmt.Sprintf("Error pushing image: %v", err))
	}
}



func main() {
	if err := util.ConfigureLogging("info"); err != nil {
		panic(err)
	}
	logrus.Info("Starting server")
	http.HandleFunc("/", executeCommand)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
