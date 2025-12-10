use std::env::args;
use std::io::{Read, Result as ResultIO, Write};
use std::net::{TcpStream, ToSocketAddrs};
use std::sync::Arc;
use std::time::Duration;

use pqcrypto::kem::mlkem512::{ciphertext_bytes, decapsulate, keypair};
use pqcrypto::sign::mldsa44;
use pqcrypto::traits::kem::{Ciphertext, PublicKey as KemPKey, SharedSecret};
use pqcrypto::traits::sign::{PublicKey as SignPKey, SignedMessage};

use aes_gcm::aead::{Aead, OsRng};
use aes_gcm::{AeadCore, Aes256Gcm, Error as ErrorAES, KeyInit, Nonce};

use rustls::pki_types::ServerName;

/// Returns array of bytes read from socket, which will
/// be the full packet
fn read_socket(socket: &mut TcpStream) -> ResultIO<Vec<u8>> {
    // First, read first 4 bytes for length of data to get (N bytes)
    let mut len_arr: [u8; 4] = [0; 4];
    socket.read_exact(&mut len_arr)?;

    // Then, read N bytes from socket
    let len: u32 = u32::from_be_bytes(len_arr);
    let mut data: Vec<u8> = vec![0; len as usize];
    socket.read_exact(&mut data)?;

    Ok(data)
}

/// Writes an entire buffer of data to the socket
fn write_socket(socket: &mut TcpStream, buffer: &[u8]) -> ResultIO<()> {
    let len: u32 = buffer.len() as u32;
    let len_arr: [u8; 4] = len.to_be_bytes();

    // Send 4 bytes for amount of data to send, then send all the data
    socket.write_all(&len_arr)?;
    socket.write_all(buffer)?;
    Ok(())
}

/// Encrypts the plaintext with the given cipher.
/// Returns nonce + ciphertext if OK, error if fail to encrypt
fn encrypt_fn(cipher: &Aes256Gcm, plaintext: &[u8]) -> Result<Vec<u8>, ErrorAES> {
    let nonce = Aes256Gcm::generate_nonce(&mut OsRng);

    // Try to encrypt and concat nonce + ciphertext, or throw error
    match cipher.encrypt(&nonce, plaintext) {
        Ok(mut ct) => {
            let mut full_vec = nonce.to_vec();
            full_vec.append(&mut ct);
            Ok(full_vec)
        }
        Err(e) => Err(e),
    }
}

/// Decrypts the nonce + ciphertext with the given cipher.
/// Returns plaintext if OK, error if fail to decrypt
fn decrypt_fn(cipher: &Aes256Gcm, ciphertext: &[u8]) -> Result<Vec<u8>, ErrorAES> {
    // The first 12 bytes is nonce
    let nonce = Nonce::from_slice(&ciphertext[..12]);

    // Try to decrypt or throw error
    cipher.decrypt(nonce, &ciphertext[12..])
}

/// Makes an HTTPS connection to domain
fn do_https(domain: String, endpoint: String) {
    // The next 3 variables were borrowed from rustls documentation under "Getting started"
    // https://docs.rs/rustls/latest/rustls/
    let root_store =
        rustls::RootCertStore::from_iter(webpki_roots::TLS_SERVER_ROOTS.iter().cloned());

    let client_config = rustls::ClientConfig::builder()
        .with_root_certificates(root_store)
        .with_no_client_auth();
    let config = Arc::new(client_config);

    let name: ServerName<'_> = domain
        .clone()
        .try_into()
        .expect("Failed to convert into ServerName");

    let mut conn =
        rustls::ClientConnection::new(config, name).expect("Failed to create client connection");

    let mut addrs = (domain.clone(), 443 as u16)
        .to_socket_addrs()
        .expect("Failed to convert domain to IP");
    let sock_addr = addrs.next().unwrap();
    let mut sock = TcpStream::connect_timeout(&sock_addr, Duration::from_secs(3))
        .expect("Failed to make socket");
    sock.set_read_timeout(Some(Duration::from_secs(10)))
        .unwrap();
    sock.set_write_timeout(Some(Duration::from_secs(10)))
        .unwrap();

    let mut stream = rustls::Stream::new(&mut conn, &mut sock);

    // Setup HTTP headers for GET
    let headers = [
        ("Accept", "text/*; application/*; multipart/*".to_string()),
        ("Accept-Encoding", "identity".to_string()),
        ("Accept-Language", "en-US".to_string()),
        ("Connection", "close".to_string()),
        ("Host", format!("{domain}:443")),
        ("User-Agent", "rust-socket".to_string()),
    ]
    .iter()
    .fold(String::new(), |acc, tup| {
        format!("{}{}: {}\r\n", acc, tup.0, tup.1)
    });

    // Format the HTTP GET request
    let mut req = format!("GET {endpoint} HTTP/1.1\r\n");
    req.push_str(&headers);
    req.push_str("\r\n");

    // Prints the HTTP request
    print!("\n{}", req.clone());

    // Send the request
    stream
        .write_all(req.as_bytes())
        .expect("Failed to write to socket");

    println!("Sent request, now waiting response...\n");

    // Read response
    let mut res = String::new();
    match stream.read_to_string(&mut res) {
        Ok(_) => print!("{}", res),
        Err(e) => {
            println!("{}", res);
            println!("ERR while reading response: {}", e.to_string())
        }
    }
}

fn main() -> Result<(), String> {
    let cli_args: Vec<String> = args().collect();

    if cli_args.len() == 1 {
        println!("Usage: client <host> [endpoint]");
        println!("Args:");
        println!("  host        Domain name (ex: httpbin.io) or local IP address and port (ex: 127.0.0.1:3000)");
        println!("  endpoint    If given external domain name, endpoint to GET (default is /)");
        println!("Examples:");
        println!("  client httpbin.io");
        println!("  client www.randomnumberapi.com /api/v1.0/random?count=7");
        println!("  client 127.0.0.1:7443 /");
        return Ok(());
    }

    if !cli_args[1].contains(":") {
        let endpoint = {
            if cli_args.len() == 3 {
                cli_args[2].clone()
            } else {
                "/".to_string()
            }
        };
        do_https(cli_args[1].clone(), endpoint);
        return Ok(());
    }

    // Connect to server and set socket operation timeouts
    let mut socket = TcpStream::connect(&cli_args[1]).expect("Failed to open socket");
    socket
        .set_read_timeout(Some(Duration::from_secs(3)))
        .expect("Failed to set read timeout");
    socket
        .set_write_timeout(Some(Duration::from_secs(3)))
        .expect("Failed to set write timeout");

    // Generate client public and secret keys with ML-KEM-512
    let (client_pkey, client_skey) = keypair();

    // Send client public key
    write_socket(&mut socket, &client_pkey.as_bytes())
        .expect("Failed to send our public key over socket");

    // Recieve: ciphertext (KEM) + server_public_key + signature (plaintext)
    let server_res = read_socket(&mut socket)
        .expect("Failed to read server response");

    // Parse the response components
    let kem_ct_size = ciphertext_bytes();
    let server_pubkey_size = 1312; // ML-DSA-44 public key size

    // Extract components: server_pubkey + kem_ct + signature
    let server_pubkey_bytes = &server_res[..server_pubkey_size];
    let server_kem_ct = &server_res[server_pubkey_size..server_pubkey_size + kem_ct_size];
    let signature = &server_res[server_pubkey_size + kem_ct_size..]; 


    // Convert KEM ciphertext
    let server_ct = Ciphertext::from_bytes(server_kem_ct)
        .expect("Failed to convert server ct to Ciphertext");

    // Derive shared key with server encapsulated secret + client secret key
    let shared_secret = decapsulate(&server_ct, &client_skey);
    let shared_secret = shared_secret.as_bytes();
    
    // Parse server's public key
    let server_id_pk = mldsa44::PublicKey::from_bytes(server_pubkey_bytes)
        .expect("Failed to parse server public key");

    // signed data format: client_pubkey || server_pub_key || server_encapsulated_secret
    let data_to_sign = [client_pkey.as_bytes(), server_pubkey_bytes, server_kem_ct].concat();
    
    // ML-DSA SignedMessage format: signature || message
    let mut signed_message_bytes = signature.to_vec();
    signed_message_bytes.extend_from_slice(&data_to_sign);
    
    let signed_message = mldsa44::SignedMessage::from_bytes(&signed_message_bytes)
        .expect("Failed to create signed message");

    let verified_data = mldsa44::open(&signed_message, &server_id_pk)
        .expect("Invalid server signature");

    // Verify that the message extracted from the signature matches what we expect
    if verified_data != data_to_sign {
        panic!("Signature verification mismatch");
    }

    // Create cipher for application data encryption
    let cipher = Aes256Gcm::new_from_slice(shared_secret)
        .expect("Failed to create AesGcm cipher");
    
    // Create encryption/decryption functions
    let encrypt = |plaintext: &[u8]| encrypt_fn(&cipher, plaintext);
    let decrypt = |ciphertext: &[u8]| decrypt_fn(&cipher, ciphertext);

    // Send HTTP request over encrypted channel
    let endpoint = &cli_args[2];
    let host = cli_args[1].split(':').next().unwrap_or("localhost");
    
    let http_request = format!(
        "GET {} HTTP/1.1\r\nHost: {}\r\nConnection: close\r\nUser-Agent: rust-pq-client\r\n\r\n",
        endpoint, host
    );
    
    println!("Sending HTTP request:\n{}", http_request);
    
    // Encrypt and send HTTP request via AES-GCM
    let encrypted_request = encrypt(http_request.as_bytes())
        .expect("Failed to encrypt request");
    write_socket(&mut socket, &encrypted_request)
        .expect("Failed to send encrypted request");
    
    // Receive and decrypt HTTP response via AES-GCM
    let encrypted_response = read_socket(&mut socket)
        .expect("Failed to read encrypted response");
    let decrypted_response = decrypt(&encrypted_response)
        .expect("Failed to decrypt response");
    
    let response_str = String::from_utf8_lossy(&decrypted_response);
    println!("\nHTTP Response:\n{}", response_str);
    
    Ok(())
}
