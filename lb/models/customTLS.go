// this file creates a custom server to mimic TLS handshake with PQC key exchange + AES-GCM encryption
package models

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"bytes"
	"bufio"

	"github.com/cloudflare/circl/kem/mlkem/mlkem512"
	"github.com/cloudflare/circl/sign/mldsa/mldsa44"
)


// ML-DSA-44 server signing keys
var (
	serverSigningKey *mldsa44.PrivateKey
	serverPublicKey  *mldsa44.PublicKey
)


func InitializeServerKeys() error {
    log.Println("Generating ephemeral server signing keys (ML-DSA-44)...")
    
    // Generate new key pair
    pub, priv, err := mldsa44.Scheme().GenerateKey()
    if err != nil {
        return fmt.Errorf("failed to generate keys: %v", err)
    }
    
    // Store in global variables
    serverSigningKey = priv.(*mldsa44.PrivateKey)
    serverPublicKey = pub.(*mldsa44.PublicKey)
    
    return nil
}

func readSocket(conn net.Conn) ([]byte, error) {
    var length uint32
    if err := binary.Read(conn, binary.BigEndian, &length); err != nil {
        return nil, err
    }
    
    data := make([]byte, length)
    _, err := io.ReadFull(conn, data)
    return data, err
}

func writeSocket(conn net.Conn, data []byte) error {
    if err := binary.Write(conn, binary.BigEndian, uint32(len(data))); err != nil {
        return err
    }
    _, err := conn.Write(data)
    return err
}

// encrypt encrypts plaintext using AES-256-GCM
func encrypt(key []byte, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}


	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

    // Prepend nonce to ciphertext
	ciphertext := aesgcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decrypt nonce + ciphertext using AES-256-GCM
func decrypt(key []byte, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := 12
	nonce := data[:nonceSize]
	ciphertext := data[nonceSize:]

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// HandleClient handles a single custom TLS connection
func HandleClient(conn net.Conn, lb *LoadBalancer) {

    // Read client public key
    clientPubKeyBytes, err := readSocket(conn)
    if err != nil {
        log.Printf("Error reading client key: %v", err)
		conn.Close()
        return
    }

    // Format client public key
    clientPubKey, err := mlkem512.Scheme().UnmarshalBinaryPublicKey(clientPubKeyBytes)
    if err != nil {
		conn.Close()
        return
    }

    // Generate shared secret, and encapsulate server key via client's public key
    ct, sharedSecret, err := mlkem512.Scheme().Encapsulate(clientPubKey)
    if err != nil {
		conn.Close()
        return
    }

    // signature of clientPubKey + server kem
    dataToSign := append(clientPubKeyBytes, ct...)
    signature := mldsa44.Scheme().Sign(serverSigningKey, dataToSign, nil)

    // send ciphertext + server public key + signature 
    serverPubKeyBytes, _ := serverPublicKey.MarshalBinary()
    response := append(ct, serverPubKeyBytes...)
    response = append(response, signature...)
    
    if err := writeSocket(conn, response); err != nil {
		conn.Close()
        return
    }

    // recieve encrypted data using shared secret
    encryptedRequest, err := readSocket(conn)
    if err != nil {
		conn.Close()
        return
    }

    // decrypt request using shared secret
    plainRequest, err := decrypt(sharedSecret, encryptedRequest)
    if err != nil {
		conn.Close()
        return
    }

    // convert to bufio format for http request
    reader := bufio.NewReader(bytes.NewReader(plainRequest))
    httpReq, err := http.ReadRequest(reader)
    if err != nil {
		conn.Close()
        return
    }

    // use custo ResponseWriter to capture response
    customWriter := &customResponseWriter{
        header: make(http.Header),
        status: http.StatusOK,
        body:   &bytes.Buffer{},
    }
    lb.handleRequest(customWriter, httpReq)

    // Step 10: Build HTTP response
    var httpResponse strings.Builder
    httpResponse.WriteString(fmt.Sprintf("HTTP/1.1 %d %s\r\n", customWriter.status, http.StatusText(customWriter.status)))
    for key, values := range customWriter.header {
        httpResponse.WriteString(fmt.Sprintf("%s: %s\r\n", key, values[0]))
    }
    httpResponse.WriteString("\r\n")
    httpResponse.Write(customWriter.body.Bytes())

    // Step 11: Encrypt and send response
    encryptedResponse, err := encrypt(sharedSecret, []byte(httpResponse.String()))
    if err != nil {
		conn.Close()
        return
    }

    writeSocket(conn, encryptedResponse)
	conn.Close()
}


func StartCustomTLSServer(port int, lb *LoadBalancer, stop <-chan struct{}) error {
    
		// Initialize server signing keys before starting
	if err := InitializeServerKeys(); err != nil {
		return fmt.Errorf("failed to initialize server keys: %v", err)
	}
	
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        return err
    }
    log.Println("Server listening...")

    go func() {
        <-stop
        log.Println("Shutting down...")
        listener.Close()
    }()

    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Println("Stopped accepting connections:", err)
            return nil
        }
        go HandleClient(conn, lb)
    }
}
