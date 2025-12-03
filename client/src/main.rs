use std::result::Result;
use std::io::{self, Read, Write, Result as ResultIO};
use std::net::{TcpStream};
use std::time::Duration;

use pqcrypto::kem::mlkem512;
use pqcrypto::traits::kem::{Ciphertext, PublicKey, SharedSecret};

use aes_gcm::{AeadCore, Aes256Gcm, KeyInit, Nonce};
use aes_gcm::Error as ErrorAES;
use aes_gcm::aead::{Aead, OsRng};

const NONCE_SIZE: usize = 12;

// Returns array of bytes read from socket, which will
// be the full packet
fn read_socket(socket: &mut TcpStream) -> ResultIO<Vec<u8>>{
    let mut len_arr: [u8; 4] = [0; 4];
    socket.read_exact(&mut len_arr)?;
    let len: u32 = u32::from_be_bytes(len_arr);

    let mut data: Vec<u8> = vec![0; len as usize];
    socket.read_exact(&mut data)?;
    
    Ok(data)
}

// Writes an entire buffer of data to the socket
fn write_socket(socket: &mut TcpStream, buffer: &[u8]) -> ResultIO<()>{
    let len: u32 = buffer.len() as u32;
    let len_arr: [u8; 4] = len.to_be_bytes();
    socket.write_all(&len_arr)?;
    socket.write_all(buffer)?;
    Ok(())
}

// Takes a cipher and data and returns byte array of data encrypted
fn encrypt_fn(cipher: &Aes256Gcm, plaintext: &[u8]) -> Result<Vec<u8>, ErrorAES>{
    let nonce = Aes256Gcm::generate_nonce(&mut OsRng);
    cipher.encrypt(&nonce, plaintext)
}

fn decrypt_fn(cipher: &Aes256Gcm, nonce: &[u8; NONCE_SIZE] , ciphertext: &[u8]) -> (bool, Vec<u8>){
    let nonce = Nonce::from_slice(nonce);
    match cipher.decrypt(nonce, ciphertext) {
        Ok(pt) => (true, pt),
        Err(_) => (false, Vec::new())
    }
}

fn main() -> ResultIO<()> {
    // Connect to server and set socket operation timeouts
    let mut socket = TcpStream::connect("127.0.0.1:3000")?;
    socket.set_read_timeout(Some(Duration::from_secs(3)))?;
    socket.set_write_timeout(Some(Duration::from_secs(3)))?;

    // Generate client eph keys
    let (client_pkey, client_skey) = mlkem512::keypair();

    // Send client public key to server
    write_socket(&mut socket, &client_pkey.as_bytes())?;

    // Wait for server to respond
    // No cert:     pk2 + E(Sig(pk1 + pk2))
    // With cert:   pk2 + E(Sig(pk1 + pk2 + cert) + cert)
    let server_res = read_socket(&mut socket)?;

    // Get shared key and verify the signature
    let server_ct = mlkem512::Ciphertext::from_bytes(&server_res[..mlkem512::public_key_bytes()]).expect("Failed to convert server ct to Ciphertext");
    let server_hello_data = &server_res[mlkem512::public_key_bytes()..];

    let shared_key = mlkem512::decapsulate(&server_ct, &client_skey);
    let shared_key = shared_key.as_bytes();
    let cipher = Aes256Gcm::new_from_slice(shared_key).expect("Failed to create AesGcm cipher");
    
    let encrypt = |data: &[u8]| encrypt_fn(&cipher, data);

    // let (shared_key, ct) = mlkem512::encapsulate(&pkey);
    println!("Hello, world!");
    Ok(())
}
