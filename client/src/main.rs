use std::env::args;
use std::fs::File;
use std::io::{Read, Result as ResultIO, Write, stdin, stdout};
use std::net::{TcpStream, ToSocketAddrs};
use std::sync::Arc;
use std::time::Duration;

use pqcrypto::kem::mlkem512::{decapsulate, keypair, public_key_bytes};
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
        println!("  client 127.0.0.1:3000");
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

    // Generate client eph keys
    let (client_pkey, client_skey) = keypair();

    // Send client public key to server
    write_socket(&mut socket, &client_pkey.as_bytes())
        .expect("Failed to send our public key over socket");

    // Wait for server to respond
    // No cert:     pk2 + E(Sig(pk1 + pk2))
    let server_res = read_socket(&mut socket).expect("Failed to get server hello from socket");

    // Get shared key
    let server_pkey = &server_res[..public_key_bytes()];
    let server_ct =
        Ciphertext::from_bytes(server_pkey).expect("Failed to convert server ct to Ciphertext");
    let server_hello_data = &server_res[public_key_bytes()..];

    let shared_key = decapsulate(&server_ct, &client_skey);
    let shared_key = shared_key.as_bytes();
    let cipher = Aes256Gcm::new_from_slice(shared_key).expect("Failed to create AesGcm cipher");

    // Create the encryption and decryption functions partially
    let encrypt = |plaintext: &[u8]| encrypt_fn(&cipher, plaintext);
    let decrypt = |ciphertext: &[u8]| decrypt_fn(&cipher, ciphertext);

    // Decrypt the data and verify signature
    let server_sig = decrypt(server_hello_data).expect("Failed to decrypt hello");

    // TODO: get server's permanent public key
    let mut server_file =
        File::open("TODO_server_key.idk").expect("Failed to open server id public key file");
    let mut server_id_pk_bytes = Vec::new();
    server_file
        .read_to_end(&mut server_id_pk_bytes)
        .expect("Failed to read hardcoded server key file");
    let server_id_pk = mldsa44::PublicKey::from_bytes(server_id_pk_bytes.as_slice())
        .expect("Failed to convert file to public key");

    let sm = mldsa44::SignedMessage::from_bytes(server_sig.as_slice())
        .expect("Failed to convert bytes to SignedMessage");

    // Verify signature
    let cpk_spk = mldsa44::open(&sm, &server_id_pk).expect("Could not open signature!");
    let correct_cpk_spk = [client_pkey.as_bytes(), server_pkey].concat();
    assert!(cpk_spk == correct_cpk_spk, "Could not verify signature!");

    // Main message loop
    let mut pt_msg = String::new();
    let mut err_msg = String::new();
    loop {
        // Read msg from stdin
        pt_msg.clear();
        stdout().write(b"Type a message: ").unwrap();
        match stdin().read_line(&mut pt_msg) {
            Ok(_) => (),
            Err(e) => {
                err_msg = format!("Error reading stdin: {e}");
                break;
            }
        };

        // Client exit if typed "exit"
        if pt_msg == "exit" {
            break;
        }

        // Encrypt msg
        let ct_msg_vec = match encrypt(pt_msg.trim_end().as_bytes()) {
            Ok(ct) => ct,
            Err(e) => {
                err_msg = format!("Error reading stdin: {e}");
                break;
            }
        };

        // Send the message
        match write_socket(&mut socket, ct_msg_vec.as_slice()) {
            Ok(_) => (),
            Err(e) => {
                err_msg = format!("Error reading stdin: {e}");
                break;
            }
        }

        // Await response
        match read_socket(&mut socket) {
            Ok(res) => println!("{}", String::from_utf8(res).unwrap()),
            Err(e) => {
                err_msg = format!("Error reading stdin: {e}");
                break;
            }
        }
    }

    socket.shutdown(std::net::Shutdown::Both).unwrap();

    if !err_msg.is_empty() {
        Err(err_msg)
    } else {
        Ok(())
    }
}
