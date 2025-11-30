use std::{io::Write, net::{TcpStream}};
use pqcrypto::kem::mlkem512;
use pqcrypto::traits::kem::PublicKey as _;

use libaes::Cipher;

fn main() -> std::io::Result<()> {
    // Generate client eph keys
    let (client_pkey, _client_skey) = mlkem512::keypair();

    // Connect to server and send public key and its length
    let mut socket = TcpStream::connect("127.0.0.1:443")?;
    socket.write_all(&mlkem512::public_key_bytes().to_be_bytes())?;
    socket.write_all(&client_pkey.as_bytes())?;

    // Wait for server to respond
    // No cert:     pk2 + E(Sig(pk1 + pk2))
    // With cert:   pk2 + E(Sig(pk1 + pk2 + cert) + cert)

    // Make encryption and decryption functions
    let aes_cipher = Cipher::new_256(b"TODO: Key of 32 bytes here alrig");
    let encrypt = |iv: &[u8; 16], pt: &[u8]| aes_cipher.cbc_encrypt(iv, pt);
    let decrypt = |iv: &[u8; 16], ct: &[u8]| aes_cipher.cbc_decrypt(iv, ct);

    // let (shared_key, ct) = mlkem512::encapsulate(&pkey);
    println!("Hello, world!");
    Ok(())
}
