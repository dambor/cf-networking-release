package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"

	"silk-controller/models"
	"silk-controller/store"

	"code.cloudfoundry.org/lager"
	"github.com/tedsuo/rata"
)

type Leases struct {
	Logger lager.Logger
	Store  store.Datastore
}

func (l *Leases) handleError(w http.ResponseWriter, action string, err error) {
	l.Logger.Error(action, err)

	statusCode := http.StatusBadRequest
	if err == store.NotFoundError {
		statusCode = http.StatusNotFound
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"action": action,
		"error":  err.Error(),
	})
}

func (l *Leases) BuildRouter() (http.Handler, error) {
	routes := rata.Routes{
		{Name: "list", Method: "GET", Path: "/leases"},
		{Name: "acquire", Method: "POST", Path: "/leases"},
		{Name: "renew", Method: "PUT", Path: "/leases/:leaseID"},
		{Name: "delete", Method: "DELETE", Path: "/leases/:leaseID"},
	}

	handlers := rata.Handlers{
		"list":    http.HandlerFunc(l.List),
		"acquire": http.HandlerFunc(l.Acquire),
		"renew":   http.HandlerFunc(l.Renew),
		"delete":  http.HandlerFunc(l.Delete),
	}
	return rata.NewRouter(routes, handlers)
}

func (l *Leases) List(w http.ResponseWriter, httpRequest *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	leases, err := l.Store.List()
	if err != nil {
		l.handleError(w, "list", err)
		return
	}

	json.NewEncoder(w).Encode(leases)
}

func (l *Leases) Acquire(w http.ResponseWriter, httpRequest *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var newLeaseRequest models.NewLeaseRequest
	err := json.NewDecoder(httpRequest.Body).Decode(&newLeaseRequest)
	if err != nil {
		l.handleError(w, "decode-request", err)
		return
	}

	response, err := l.Store.Acquire(newLeaseRequest)
	if err != nil {
		l.handleError(w, "acquire", err)
		return
	}

	json.NewEncoder(w).Encode(response)
}

var leaseIDRegexp = regexp.MustCompile(`/leases/(\w*)`)

func getLeaseID(req *http.Request) string {
	matches := leaseIDRegexp.FindStringSubmatch(req.URL.Path)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (l *Leases) Renew(w http.ResponseWriter, httpRequest *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	renewalRequest := models.Lease{ID: getLeaseID(httpRequest)}
	err := json.NewDecoder(httpRequest.Body).Decode(&renewalRequest)
	if err != nil {
		l.handleError(w, "decode-request", err)
		return
	}

	response, err := l.Store.Renew(renewalRequest)
	if err != nil {
		l.handleError(w, "renew", err)
		return
	}

	json.NewEncoder(w).Encode(response)
}

func (l *Leases) Delete(w http.ResponseWriter, httpRequest *http.Request) {
	err := l.Store.Delete(getLeaseID(httpRequest))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		l.handleError(w, "renew", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
