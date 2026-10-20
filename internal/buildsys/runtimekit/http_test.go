package runtimekit

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPResultBoundary(t *testing.T) {
	spec := resultSpec("")
	spec.Modules[0].Funcs = append(spec.Modules[0].Funcs,
		FuncSpec{Module: "main", Name: "http_fail", ReturnType: "R[http.Resp,nil]", BlockBody: true, Params: []ParamSpec{{Name: "req", Type: "http.Req"}}, Body: "return Err(error:nil)"},
		FuncSpec{Module: "main", Name: "http_success", ReturnType: "R[http.Resp,str]", BlockBody: true, Params: []ParamSpec{{Name: "req", Type: "http.Req"}, {Name: "prefix", Type: "str"}}, Body: `return Ok(value:http.text(code:201,body:prefix + req.path))`},
	)
	rt, err := newRuntime(spec, nil, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	success := &partialValue{Target: functionValue{Module: "main", Name: "http_success"}, Bound: map[string]value{"prefix": "received:"}, Signature: FunctionType{Module: "main", Params: []ParamSpec{{Name: "req", Type: "http.Req"}}, ReturnType: "R[http.Resp,str]"}}
	for _, tc := range []struct {
		handler value
		status  int
		body    string
	}{
		{functionValue{Module: "main", Name: "http_fail"}, 500, "unhandled Err: <nil>"},
		{success, 201, "received:/path"},
		{ascribeValue(success, "Fn[(req:http.Req)->R[http.Resp,str]]", "main"), 201, "received:/path"},
	} {
		recorder := httptest.NewRecorder()
		rt.httpHandler(tc.handler).ServeHTTP(recorder, httptest.NewRequest("GET", "/path", nil))
		if recorder.Code != tc.status || !strings.Contains(recorder.Body.String(), tc.body) {
			t.Fatalf("got %d: %s", recorder.Code, recorder.Body.String())
		}
	}
}
