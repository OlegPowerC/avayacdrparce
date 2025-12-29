#!/usr/bin/env python3
import keyring
import argparse

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description="Set passord in Keyring")
    parser.add_argument('-u', type=str, help="User name", required=True)
    parser.add_argument('-k', type=str, help="KeyChainName", required=True)
    args = parser.parse_args()
    if len(args.u) < 4:
        print("Too short username")
        exit(1)
    if len(args.k) < 4:
        print("Too short keychainname")
        exit(1)
    keyring.delete_password(args.k,args.u)
