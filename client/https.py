import json

class HttpsPayload():
    def __init__(self, starter: str, headers: dict, body: str):
        self.__starter = starter
        self.__headers = headers
        self.__body = body

    def print(self):
        print(self.__starter)
        for key, val in self.__headers.items():
            print(f"{key}: {val}")
        print()
        if 'Content-Type' in self.__headers and self.__headers['Content-Type'] == 'application/json':
            # TODO: print pretty json
            pass
        else:
            print(self.__body)
          
class HttpsRequest(HttpsPayload):
    def send(self) -> HttpsPayload:
        # TODO: send request and return response
        pass

