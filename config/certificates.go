package config

/*
Utilities to create a server with a self signed x509 certificate

*/

import (
	"fmt"
	"io/ioutil"
	"net"
	"os/exec"
	"strings"

	"github.com/simonlangowski/lightning1/crypto"
)

/*
func CreateServer(addr string, id int64) *Server {
	ip := addr
	if strings.Contains(ip, ":") {
		ip = strings.Split(ip, ":")[0]
	}

	name := pkix.Name{
		CommonName:   ip,
		Organization: []string{"MIT"},
		Country:      []string{"US"},
	}

	certDer, certPriv := SelfSignedCertificate(name)
	keyDer, err := x509.MarshalECPrivateKey(certPriv)
	if err != nil {
		panic(err)
	}

	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDer})
	keyPem := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDer})

	return CreateServerWithCertificate(addr, id, certPem, keyPem)
}
*/

// If the certificate exists in the map reuse it, otherwise create a new certificate
func CreateServerWithExisting(addr string, id int64, servers map[int64]*Server) *Server {
	cert, key := FindIdentity(addr, servers)
	if cert != nil {
		return CreateServerWithCertificate(addr, id, cert, key)
	} else {
		cert, key := CreateCertificate(addr)
		return CreateServerWithCertificate(addr, id, cert, key)
	}
}

func CreateCertificate(addr string) ([]byte, []byte) {
	ip := IP(addr)
	cmd := exec.Command("sh", "../certificates/certificate.sh", ip, addr)
	cmd.Dir = "../certificates"
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
	return GetCertificate(addr), GetCertificateKey(addr)
}

func CreateServerWithCertificate(addr string, id int64, cert, key []byte) *Server {
	priv, pub := crypto.NewDHKeyPair()
	ver, sign := crypto.NewSigningKeyPair()

	s := &Server{
		Address:         addr,
		Id:              id,
		Identity:        cert,
		PrivateIdentity: key,
		PublicKey:       pub.Bytes(),
		PrivateKey:      priv.Bytes(),
		VerificationKey: ver,
		SignatureKey:    sign,
	}
	return s
}

func GetCertificate(addr string) []byte {
	cert, err := ioutil.ReadFile(fmt.Sprintf("../certificates/cert%s.pem", addr))
	if err != nil {
		panic(err)
	}
	return cert
}

func GetCertificateKey(addr string) []byte {
	key, err := ioutil.ReadFile(fmt.Sprintf("../certificates/key%s.pem", addr))
	if err != nil {
		panic(err)
	}
	return key
}

/*

func CreateServer(addr string, id int64) *Server {
	ip := addr
	if strings.Contains(ip, ":") {
		ip = strings.Split(ip, ":")[0]
	}
	if ip == "localhost" {
		ip = "127.0.0.1" // net.LookupHost(ip)
	}
	cmd := exec.Command("sh", "./certificate.sh", ip)
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
	certPem, err := ioutil.ReadFile("cert.pem")
	if err != nil {
		panic(err)
	}
	keyPem, err := ioutil.ReadFile("key.pem")
	if err != nil {
		panic(err)
	}
	return CreateServerWithCertificate(addr, id, certPem, keyPem)
}
*/

/*
func SelfSignedCertificate(name pkix.Name) ([]byte, *ecdsa.PrivateKey) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}

	// if old certificate is not valid any more
	// or first time registering, generate a new cert
	serial, err := rand.Int(rand.Reader, (&big.Int{}).Exp(big.NewInt(2), big.NewInt(159), nil))
	if err != nil {
		panic(err)
	}

	now := time.Now()
	expiration := time.Now().Add(8760 * time.Hour) // year from now

	template := &x509.Certificate{
		SerialNumber: serial,
		Issuer:       name,
		Subject:      name,

		PublicKeyAlgorithm: x509.ECDSA,
		PublicKey:          priv.PublicKey,

		IsCA:     true,
		KeyUsage: x509.KeyUsageCertSign,

		NotBefore: now.Add(-10 * time.Minute).UTC(),
		NotAfter:  expiration.UTC(),

		// ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		// BasicConstraintsValid: true,
	}

	if ip := net.ParseIP(name.CommonName); ip != nil {
		template.IPAddresses = []net.IP{ip}
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		panic(err)
	}
	return der, priv
}
*/

func Port(addr string) string {
	return ":" + strings.Split(addr, ":")[1]
}

func Host(addr string) string {
	if strings.Contains(addr, ":") {
		return strings.Split(addr, ":")[0]
	}
	return addr
}

func IP(addr string) string {
	h := Host(addr)
	ip := net.ParseIP(h)
	if ip == nil {
		a, err := net.LookupHost(h)
		if err != nil {
			panic(err)
		}
		ip = net.ParseIP(a[0])
		if ip == nil {
			panic("Could not get ip address")
		}
	}
	return ip.String()
}

func FindIdentity(addr string, servers map[int64]*Server) ([]byte, []byte) {
	for _, server := range servers {
		if server.Address != addr {
			continue
		}
		return server.Identity, server.PrivateIdentity
	}
	return nil, nil
}
