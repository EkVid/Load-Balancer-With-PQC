# D58-Project

If you would like to run the client in a Docker container, run the follow commands
while in the client directory:

# Build the docker image named "client", which can take a while
/client$ docker build -t client .

# Run the container, passing host and endpoint like usual
/client$ docker run --rm client __CMD_ARGS__