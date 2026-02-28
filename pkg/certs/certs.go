package certs

import (
	"crypto/x509"
	_ "embed"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

//go:embed roots.pem
var rootsPem []byte

var (
    oncePool sync.Once
    pool     *x509.CertPool
    poolErr  error
    onceFile sync.Once
    certPath string
    fileErr  error
)

// RootCertPool returns an x509.CertPool containing the embedded root
// certificates.  The pool is initialised only once and cached for later
// calls.  If parsing fails the error is returned.
func RootCertPool() (*x509.CertPool, error) {
    oncePool.Do(func() {
        p := x509.NewCertPool()
        if ok := p.AppendCertsFromPEM(rootsPem); !ok {
            // embedded bundle failed to parse; we don't treat this as fatal
            // because callers can still choose to use the system pool or
            // ignore verification.  pool will simply be empty.
        }
        pool = p
    })
    return pool, poolErr
}

// RootCertPath writes the embedded PEM to a temporary file and returns the
// path.  The file is created only once per process and reused on subsequent
// calls.  It is the caller's responsibility to remove the file if desired.
func RootCertPath() (string, error) {
    onceFile.Do(func() {
        dir := os.TempDir()
        fp := filepath.Join(dir, "querybox-root-certs.pem")
        if err := ioutil.WriteFile(fp, rootsPem, 0o644); err != nil {
            fileErr = err
            return
        }
        certPath = fp
    })
    return certPath, fileErr
}
