import json
import socket

class HttpsPayload():
    def __init__(self, starter: str, headers: dict, body: str):
        self.__starter = starter
        self.__headers = headers
        self.__body = body

    # Print the payload (and make it pretty if JSON body)
    def print(self):
        print(self.__starter)
        for key, val in self.__headers.items():
            print(f"{key}: {val}")
        print()
        if 'Content-Type' in self.__headers and self.__headers['Content-Type'] == 'application/json':
            json.dumps(self.__body, indent=4)
        else:
            print(self.__body)
          
class TlsSession():
    def __init__(self, domainOrIP: str):
        self.__host = domainOrIP

    # Use "with" statement to connect to the host
    def __enter__(self):
        self.__socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.__socket.connect((self.__host, 443))
        # TODO: TLS here
        return self

    # Close the socket when exiting the with statement
    def __exit__(self, e_type, e_val, e_traceback):
        self.__socket.shutdown(socket.SHUT_RDWR)
        self.__socket.close()