package runtimekit

import (
	"fmt"
	"io"
	"net/http"
)

func (rt *runtime) httpHandler(handler value) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := &structValue{Type: "Req", Fields: map[string]any{"method": r.Method, "path": r.URL.Path, "body": readBody(r), "ctx": "request"}}
		var resp value
		var err error
		resp, err = rt.invokeCallable(handler, []value{req})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resp, err = terminalValue(resp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		sv, ok := resp.(*structValue)
		if !ok {
			http.Error(w, "handler did not return Resp", http.StatusInternalServerError)
			return
		}
		if headers, ok := sv.Fields["headers"].(map[string]any); ok {
			for k, v := range headers {
				w.Header().Set(k, fmt.Sprint(v))
			}
		}
		code, _ := asInt(sv.Fields["code"])
		if code == 0 {
			code = 200
		}
		w.WriteHeader(code)
		_, _ = io.WriteString(w, fmt.Sprint(sv.Fields["body"]))

	})
}
