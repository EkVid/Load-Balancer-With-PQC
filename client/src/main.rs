use std::env::args;
use std::io::{Read, Result as ResultIO, Write};
use std::net::{TcpStream, ToSocketAddrs};
use std::sync::Arc;
use std::time::Duration;

use pqcrypto::kem::mlkem512::{decapsulate, keypair, public_key_bytes};
use pqcrypto::traits::kem::{Ciphertext, PublicKey, SharedSecret};

use aes_gcm::aead::{Aead, OsRng};
use aes_gcm::{AeadCore, Aes256Gcm, Error as ErrorAES, KeyInit, Nonce};

use rustls::pki_types::pem::PemObject;
use rustls::pki_types::{CertificateDer, ServerName};

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
    let mut root_store =
        rustls::RootCertStore::from_iter(
            webpki_roots::TLS_SERVER_ROOTS.iter().cloned()
        );

    // Open self-signed cert to allow
    let cert = CertificateDer::from_pem_file("self_cert.pem").unwrap();
    root_store.add(cert).unwrap();
    
    let client_config = rustls::ClientConfig::builder()
        .with_root_certificates(root_store)
        .with_no_client_auth();
    let config = Arc::new(client_config);

    let name: ServerName<'_> = domain
        .clone()
        .try_into()
        .expect("Failed to convert into ServerName");

    let mut conn = rustls::ClientConnection::new(config, name)
        .expect("Failed to create client connection");

    let mut addrs = (domain.clone(), 443 as u16)
        .to_socket_addrs()
        .expect("Failed to convert domain to IP");
    let sock_addr = addrs.next().unwrap();
    let mut sock = TcpStream::connect_timeout(
        &sock_addr, 
        Duration::from_secs(3))
        .expect("Failed to make socket");
    let _ = sock.set_read_timeout(Some(Duration::from_secs(10)));
    let _ = sock.set_write_timeout(Some(Duration::from_secs(10)));

    let mut stream = rustls::Stream::new(&mut conn, &mut sock);

    // Write GET message to server
    let req = format!("GET {} HTTP/1.0\r\nHost: {}\r\n\r\n", endpoint, domain);
    print!("{}", req.clone());
    let mut req = req.as_bytes();
    stream
        .write(&mut req)
        .unwrap();

    println!("Sent request, now waiting response...");

    // Read response
    let mut res  = String::new();
    stream.read_to_string(&mut res).unwrap();
    print!("{}", res);
}

fn main() -> ResultIO<()> {
    // Run with "cargo run <domain or IP+port> <endpoint>"
    let cli_args: Vec<String> = args().collect();

    if !cli_args[1].contains(":") {
        do_https(cli_args[1].clone(), cli_args[2].clone());
        return Ok(());
    }

    // Connect to server and set socket operation timeouts
    let mut socket = TcpStream::connect(&cli_args[1])?;
    socket.set_read_timeout(Some(Duration::from_secs(3)))?;
    socket.set_write_timeout(Some(Duration::from_secs(3)))?;

    // Generate client eph keys
    let (client_pkey, client_skey) = keypair();

    // Send client public key to server
    write_socket(&mut socket, &client_pkey.as_bytes())?;

    // Wait for server to respond
    // No cert:     pk2 + E(Sig(pk1 + pk2))
    let server_res = read_socket(&mut socket)?;

    // Get shared key
    let server_ct = Ciphertext::from_bytes(&server_res[..public_key_bytes()])
        .expect("Failed to convert server ct to Ciphertext");
    let server_hello_data = &server_res[public_key_bytes()..];

    let shared_key = decapsulate(&server_ct, &client_skey);
    let shared_key = shared_key.as_bytes();
    let cipher = Aes256Gcm::new_from_slice(shared_key).expect("Failed to create AesGcm cipher");

    // Create the encryption and decryption functions partially
    let encrypt = |plaintext: &[u8]| encrypt_fn(&cipher, plaintext);
    let decrypt = |ciphertext: &[u8]| decrypt_fn(&cipher, ciphertext);

    // Decrypt the data and verify signature
    let server_sig = decrypt(server_hello_data).expect("Failed to decrypt hello");

    // println!("Hello, world!");
    Ok(())
}
