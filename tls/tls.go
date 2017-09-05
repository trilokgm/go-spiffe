package tls

import(
	"crypto/tls"
	"crypto/x509"

	"github.com/spiffe/go-spiffe/spiffe"
)

type TLSPeer struct {
	// Slice of permitted SPIFFE IDs
	SpiffeIDs []string

	TrustRoots *x509.CertPool
}

// NewTLSConfig creates a SPIFFE-compatible TLS configuration.
// We are opinionated towards mutual TLS. If you don't want
// mutual TLS, you'll need to update the returned config
func (t *TLSPeer) NewTLSConfig(certs []tls.Certificate) *tls.Config {
	config := &tls.Config{
		// Disable validation/verification because we perform
		// this step with custom logic in `verifyPeerCertificate`
		ClientAuth: tls.RequireAnyClientCert,
		InsecureSkipVerify: true,

		VerifyPeerCertificate: t.verifyPeerCertificate,
	}

	return config
}

// verifyPeerCertificate serves callbacks from TLS listeners/dialers. It performs
// SPIFFE-specific validation steps on behalf of the golang TLS library
func (t *TLSPeer) verifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) (err error) {
	// First, parse all received certs
	var certs []*x509.Certificate
	for _, rawCert := range rawCerts {
		cert, err := x509.ParseCertificate(rawCert)
		if err != nil {
			return err
		}

		certs = append(certs, cert)
	}

	// Perform path validation
	// Assume leaf is the first off the wire
	intermediates := x509.NewCertPool()
	for _, intermediate := range certs[1:] {
		intermediates.AddCert(intermediate)
	}
	err = spiffe.VerifyCertificate(certs[0], intermediates, t.TrustRoots)
	if err != nil {
		return err
	}

	// Look for a known SPIFFE ID in the leaf
	err = spiffe.MatchID(t.SpiffeIDs, certs[0])
	if err != nil {
		return err
	}

	// If we are here, then all is well
	return nil
}