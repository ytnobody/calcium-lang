package vm

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ytnobody/calcium-lang/pkg/compiler"
	"github.com/ytnobody/calcium-lang/pkg/value"
)

func TestSecretHttpPostSinkEndToEnd(t *testing.T) {
	var received string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received = string(body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	input := `use core.http!;
url = "` + server.URL + `";
cap = Secret.capability(http.post_secret, {url: url});
s = Secret.wrap("SECRET-BODY");
Secret.with(cap) {
    s !!!> http.post_secret(url, cap, "text/plain", {})
};`

	program := parse(input)
	comp := compiler.New()
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compile error: %s", err)
	}

	machine := New(comp.Constants())
	machine.SetSecretWritePolicy([]string{"entry"})
	if err := machine.Run(comp.Bytecode().Instructions); err != nil {
		t.Fatalf("runtime error: %s", err)
	}

	if received != "SECRET-BODY" {
		t.Fatalf("expected trusted sink to send Secret body, got %q", received)
	}
	result := machine.LastPoppedStackElem()
	if result.Type != value.TYPE_SUCCESS {
		t.Fatalf("expected success result, got %s: %s", result.Type, result.String())
	}
}
