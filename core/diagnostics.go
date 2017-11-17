package core

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/utils"
	"github.com/signalfx/neo-agent/utils/network"
	log "github.com/sirupsen/logrus"
)

// VersionLine should be populated by the startup logic to contain version
// information that can be reported in diagnostics.
var VersionLine string

// Serves the diagnostic status on the domain socket
func (a *Agent) serveDiagnosticInfo(socketPath string) error {
	if a.diagnosticSocketStop != nil {
		a.diagnosticSocketStop()
	}

	var err error
	a.diagnosticSocketStop, err = network.RunSimpleSocketServer(socketPath, func(_ net.Conn) string {
		return a.DiagnosticText()
	}, func(err error) {
		log.WithFields(log.Fields{
			"path":  socketPath,
			"error": err,
		}).Error("Problem with diagnostic socket")
	})
	return err
}

// DiagnosticText returns a simple textual output of the agent's status
func (a *Agent) DiagnosticText() string {
	return fmt.Sprintf(
		"NeoAgent Status"+
			"\n===============\n"+
			"\nVersion: %s"+
			"\nAgent Configuration:"+
			"\n%s\n\n"+
			"%s\n"+
			"%s\n"+
			"%s",
		VersionLine,
		utils.IndentLines(configAsDiagnosticText(a.lastConfig), 2),
		a.writer.DiagnosticText(),
		a.observers.DiagnosticText(),
		a.monitors.DiagnosticText())

}

// Converts the config structure to a human friendly output
func configAsDiagnosticText(conf *config.Config) string {
	s, _ := yaml.Marshal(conf)
	return string(s)
}

func (a *Agent) serveInternalMetrics(socketPath string) error {
	if a.internalMetricsSocketStop != nil {
		a.internalMetricsSocketStop()
	}

	var err error
	a.internalMetricsSocketStop, err = network.RunSimpleSocketServer(socketPath, func(_ net.Conn) string {
		jsonOut, err := json.MarshalIndent(a.InternalMetrics(), "", "  ")
		if err != nil {
			log.WithError(err).Error("Could not serialize internal metrics to JSON")
			return "[]"
		}
		return string(jsonOut)
	}, func(err error) {
		log.WithFields(log.Fields{
			"path":  socketPath,
			"error": err,
		}).Error("Problem with internal metrics socket")
	})
	return err
}

func (a *Agent) InternalMetrics() []*datapoint.Datapoint {
	out := make([]*datapoint.Datapoint, 0)
	out = append(out, a.writer.InternalMetrics()...)
	out = append(out, a.observers.InternalMetrics()...)
	out = append(out, a.monitors.InternalMetrics()...)

	for i := range out {
		if out[i].Dimensions == nil {
			out[i].Dimensions = make(map[string]string)
		}

		out[i].Dimensions["host"] = a.lastConfig.Hostname
		out[i].Timestamp = time.Now()
	}
	return out
}
