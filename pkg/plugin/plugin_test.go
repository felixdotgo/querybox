package plugin

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

type authMock struct{}

func (a *authMock) Info() (InfoResponse, error) { return InfoResponse{}, nil }
func (a *authMock) Exec(ExecRequest) (ExecResponse, error) { return ExecResponse{}, nil }
func (a *authMock) AuthForms(AuthFormsRequest) (AuthFormsResponse, error) {
	f := AuthForm{Key: "basic", Name: "Basic", Fields: []*AuthField{{Type: AuthField_TEXT, Name: "host", Label: "Host"}}}
	return AuthFormsResponse{Forms: map[string]*AuthForm{"basic": &f}}, nil
}

func TestServeCLI_AuthForms(t *testing.T) {
	// capture stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	oldStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	// set args
	oldArgs := os.Args
	os.Args = []string{"plugin", "authforms"}
	defer func() { os.Args = oldArgs }()

	// run ServeCLI
	ServeCLI(&authMock{})

	_ = w.Close()
	out, _ := io.ReadAll(r)
	var resp AuthFormsResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatalf("invalid json: %v (raw=%s)", err, string(out))
	}
	f, ok := resp.Forms["basic"]
	if !ok {
		t.Fatalf("expected basic form")
	}
	if strings.ToLower(f.Name) != "basic" {
		t.Fatalf("unexpected form name: %s", f.Name)
	}
}
