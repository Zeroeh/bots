package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"strconv"
)

const (
	//Public key used to encrypt email / password for hello packet
	RSAPublicKey = "-----BEGIN PUBLIC KEY-----\n" +
		"MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDCKFctVrhfF3m2Kes0FBL/JFeO\n" +
		"cmNg9eJz8k/hQy1kadD+XFUpluRqa//Uxp2s9W2qE0EoUCu59ugcf/p7lGuL99Uo\n" +
		"SGmQEynkBvZct+/M40L0E0rZ4BVgzLOJmIbXMp0J4PnPcb6VLZvxazGcmSfjauC7\n" +
		"F3yWYqUbZd/HCBtawwIDAQAB\n" +
		"-----END PUBLIC KEY-----"
	//RC4 hex keys for (de)ciphering the packet streams
	//RC4OutgoingKey = "\x6a\x39\x57\x0c\xc9\xde\x4e\xc7\x1d\x64\x82\x18\x94"
	//RC4IncomingKey = "\xc7\x93\x32\xb1\x97\xf9\x2b\xa8\x5e\xd2\x81\xa0\x23"
	RC4OutgoingKey = "\x5a\x4d\x20\x16\xbc\x16\xdc\x64\x88\x31\x94\xff\xd9"
	RC4IncomingKey = "\xc9\x1d\x9e\xec\x42\x01\x60\x73\x0d\x82\x56\x04\xe0"
)

//Encrypts the outgoing stream
/*func CipherData(data []byte, ciph *Cipher) {
	//buffer := make([]byte, len(*data))
	ciph.XorKeyStreamGeneric(data, data)
	//return buffer
}*/

func CipherReset(ciph *Cipher) {
	ciph.Reset()
}

func CreateKeyPair() (*Cipher, *Cipher) {
	send, _ := NewCipher([]byte(RC4OutgoingKey))
	recv, _ := NewCipher([]byte(RC4IncomingKey))
	return send, recv
}

func EncryptString(s string) string {
	byteString := []byte(s)
	block, _ := pem.Decode([]byte(RSAPublicKey))
	pubInterface, _ := x509.ParsePKIXPublicKey(block.Bytes)
	pub := pubInterface.(*rsa.PublicKey)
	encBytes, _ := rsa.EncryptPKCS1v15(rand.Reader, pub, byteString)
	return EncodeString(string(encBytes))
}

func EncodeString(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func (c *Ciphers) Kill() {
	c.SendCipher.Reset()
	c.RecvCipher.Reset()
}

type Ciphers struct {
	SendCipher *Cipher
	RecvCipher *Cipher
}

type Cipher struct {
	s    [256]uint32
	i, j uint8
}

type KeySizeError int

func (k KeySizeError) Error() string {
	return "crypto/rc4: invalid key size " + strconv.Itoa(int(k))
}

func NewCipher(key []byte) (*Cipher, error) {
	k := len(key)
	if k < 1 || k > 256 {
		return nil, KeySizeError(k)
	}
	var c Cipher
	for i := 0; i < 256; i++ {
		c.s[i] = uint32(i)
	}
	var j uint8
	for i := 0; i < 256; i++ {
		j += uint8(c.s[i]) + key[i%k]
		c.s[i], c.s[j] = c.s[j], c.s[i]
	}
	return &c, nil
}

func (c *Cipher) Reset() {
	for i := range c.s {
		c.s[i] = 0
	}
	c.i, c.j = 0, 0
}

func (c *Cipher) XorKeyStreamGeneric(dst, src []byte) {
	i, j := c.i, c.j
	for k, v := range src {
		i++
		j += uint8(c.s[i])
		c.s[i], c.s[j] = c.s[j], c.s[i]
		dst[k] = v ^ uint8(c.s[uint8(c.s[i]+c.s[j])])
	}
	c.i, c.j = i, j
}
