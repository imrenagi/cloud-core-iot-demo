package web

import (
	"encoding/json"
	"errors"
	"net/http"

	alphaErrors "github.com/imrenagi/iot-demo-server/pkg/errors"
)

func writeSuccessResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	jsonData, _ := json.Marshal(data)
	w.Write(jsonData)
}

func writeFailResponseFromError(w http.ResponseWriter, err error) {
	var statusCode int
	if errors.Is(err, alphaErrors.ErrInvalidArguments) {
		statusCode = http.StatusBadRequest
	} else {
		statusCode = http.StatusInternalServerError
	}

	type Error struct {
		StatusCode int    `json:"error_code"`
		Message    string `json:"error_message"`
	}

	errorMsg := Error{
		Message:    err.Error(),
		StatusCode: statusCode,
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(errorMsg.StatusCode)
	jsonData, _ := json.Marshal(errorMsg)
	w.Write(jsonData)
}
