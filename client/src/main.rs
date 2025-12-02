use std::io::{self, Read, Write};
use std::net::{TcpStream};
use std::time::Duration;

use pqcrypto::kem::mlkem512;
use pqcrypto::traits::kem::PublicKey as _;

// const BUF_SIZE: u32 = 1024 * 1024;

// Returns array of bytes read from socket, which will
// be the full packet
fn read_socket(socket: &mut TcpStream) -> io::Result<Vec<u8>>{
    let mut len_arr: [u8; 4] = [0; 4];
    socket.read_exact(&mut len_arr)?;
    let len: u32 = u32::from_be_bytes(len_arr);

    let mut data: Vec<u8> = vec![0; len as usize];
    socket.read_exact(&mut data)?;
    
    Ok(data)
}

// Writes an entire buffer of data to the socket
fn write_socket(socket: &mut TcpStream, buffer: &[u8]) -> io::Result<()>{
    let len: u32 = buffer.len() as u32;
    let len_arr: [u8; 4] = len.to_be_bytes();
    socket.write_all(&len_arr)?;
    socket.write_all(buffer)?;
    Ok(())
}

fn main() -> io::Result<()> {
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
    
    // socket.write_all()?;
    // socket.write_all(&client_pkey.as_bytes())?;

    // Make encryption and decryption functions
    

    // let (shared_key, ct) = mlkem512::encapsulate(&pkey);
    println!("Hello, world!");
    Ok(())
}
