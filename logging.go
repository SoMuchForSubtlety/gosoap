package gosoap

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
)

type CommunicationLogger interface {
	LogRequest(operation string, header http.Header, body []byte)
	LogResponse(operation string, header http.Header, body []byte)
}

func NewSlogAdapter(logger *slog.Logger, level slog.Level) CommunicationLogger {
	if logger == nil {
		logger = slog.Default()
	}
	return &slogAdapter{
		logger: logger,
		level:  level,
	}
}

type slogAdapter struct {
	logger *slog.Logger
	level  slog.Level
}

func (s *slogAdapter) LogRequest(operation string, header http.Header, body []byte) {
	s.logger.Log(context.Background(), s.level, string(body),
		slog.String("operation", operation),
		slog.Any("header", concatheader(header)),
	)
}

func (s *slogAdapter) LogResponse(operation string, header http.Header, body []byte) {
	s.logger.Log(context.Background(), s.level, string(body),
		slog.String("operation", operation),
		slog.Any("header", concatheader(header)),
	)
}

func concatheader(h http.Header) map[string]string {
	res := make(map[string]string, len(h))
	for k, v := range h {
		// https://www.rfc-editor.org/rfc/rfc9110.html#name-field-lines-and-combined-fi
		res[k] = strings.Join(v, ", ")
	}
	return res
}
