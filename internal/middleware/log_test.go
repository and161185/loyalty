package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestLogMiddleware(t *testing.T) {
	var buf bytes.Buffer

	// кастомный логгер, пишет в буфер
	encoderCfg := zap.NewDevelopmentEncoderConfig()
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderCfg),
		zapcore.AddSync(&buf),
		zapcore.DebugLevel,
	)
	logger := zap.New(core).Sugar()

	body := `{"hello":"world"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	handler := LogMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("response"))
	}))

	handler.ServeHTTP(rr, req)

	time.Sleep(10 * time.Millisecond)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "method=POST") {
		t.Error("лог не содержит метод")
	}
	if !strings.Contains(logOutput, "status=201") {
		t.Error("лог не содержит статус")
	}
	if !strings.Contains(logOutput, `body={"hello":"world"}`) {
		t.Error("лог не содержит тело запроса")
	}
	if !strings.Contains(logOutput, "outputheaders=") {
		t.Error("лог не содержит заголовки ответа")
	}
}
