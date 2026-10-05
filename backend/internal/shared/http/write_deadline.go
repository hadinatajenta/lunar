package http

import (
	"errors"
	"net/http"
	"time"
)

func ExtendWriteDeadline(w http.ResponseWriter, extension time.Duration) error {
	err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(extension))
	if errors.Is(err, http.ErrNotSupported) {
		return nil
	}
	return err
}
