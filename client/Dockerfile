
FROM rust:1.91.1

WORKDIR /usr/src/app
COPY . ./

RUN cargo build --release
WORKDIR target/release

ENTRYPOINT ["./client"]