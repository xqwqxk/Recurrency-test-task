package transporthttp

import (
	"net/http"

	"github.com/gorilla/mux"

	swaggerdocs "example.com/taskservice/internal/transport/http/docs"
	httphandlers "example.com/taskservice/internal/transport/http/handlers"
)

func NewRouter(
	taskHandler *httphandlers.TaskHandler,
	recurrenceHandler *httphandlers.RecurrenceHandler,
	docsHandler *swaggerdocs.Handler,
) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/swagger/openapi.json", docsHandler.ServeSpec).Methods(http.MethodGet)
	router.HandleFunc("/swagger/", docsHandler.ServeUI).Methods(http.MethodGet)
	router.HandleFunc("/swagger", docsHandler.RedirectToUI).Methods(http.MethodGet)

	api := router.PathPrefix("/api/v1").Subrouter()

	// Task CRUD
	api.HandleFunc("/tasks", taskHandler.Create).Methods(http.MethodPost)
	api.HandleFunc("/tasks", taskHandler.List).Methods(http.MethodGet)
	api.HandleFunc("/tasks/{id:[0-9]+}", taskHandler.GetByID).Methods(http.MethodGet)
	api.HandleFunc("/tasks/{id:[0-9]+}", taskHandler.Update).Methods(http.MethodPut)
	api.HandleFunc("/tasks/{id:[0-9]+}", taskHandler.Delete).Methods(http.MethodDelete)

	// Recurrence rules — task-scoped endpoints
	api.HandleFunc("/tasks/{id:[0-9]+}/recurrence", recurrenceHandler.CreateRule).Methods(http.MethodPost)
	api.HandleFunc("/tasks/{id:[0-9]+}/recurrence", recurrenceHandler.GetRuleByTask).Methods(http.MethodGet)
	api.HandleFunc("/tasks/{id:[0-9]+}/recurrence/next", recurrenceHandler.NextOccurrences).Methods(http.MethodPost)

	// Recurrence rules — direct access by rule ID
	api.HandleFunc("/recurrence/{recurrence_id:[0-9]+}", recurrenceHandler.GetRule).Methods(http.MethodGet)
	api.HandleFunc("/recurrence/{recurrence_id:[0-9]+}", recurrenceHandler.UpdateRule).Methods(http.MethodPut)
	api.HandleFunc("/recurrence/{recurrence_id:[0-9]+}", recurrenceHandler.DeleteRule).Methods(http.MethodDelete)

	return router
}
