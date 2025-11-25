from argparse import ArgumentParser
import pqcrypto 

parser = ArgumentParser(description='Make an HTTPS request to some server')
parser.add_argument('domainOrIP', help='Domain name or IP address of server (ex: google.com, 127.0.0.1)')
parser.add_argument('-X', '--method', default='GET', help='HTTP method to use')
parser.add_argument('-H', '--header', help='Header of request (as JSON string)')
parser.add_argument('-D', '--data', help='Body of request')

args = parser.parse_args()

print(args.method)
print(args.data)