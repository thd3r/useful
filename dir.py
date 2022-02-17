#!/usr/bin/env python3

import requests
import sys
import time
import argparse
import threading
from requests.packages.urllib3.exceptions import InsecureRequestWarning

requests.packages.urllib3.disable_warnings(InsecureRequestWarning)
threadLock = threading.Lock()
threads = []

class MyThread(threading.Thread):
    def __init__(self, domain, wordlist):
        threading.Thread.__init__(self)
        url = domain
        if not url.endswith('/'):
            url = url + '/'
        else:
            url = url
        self.url = url
        self.wordlist = wordlist

    def run(self):
        dirsearch(self.url, self.wordlist)

def get_headers():
    headers = {
        'User-Agent': 'Mozilla/5.0 (X11; Linux x86_64; rv:91.0) Gecko/20100101 Firefox/91.0'
    }
    return headers

def get_time():
    return time.strftime("%H:%M:%S")

def get_arguments():
    parser = argparse.ArgumentParser(description='\033[96mDirsearch\033[0m',
                                     usage='python3 main.py -u <target> -w <wordlist>')
    parser.add_argument("-u", "--url", type=str, help="Target url")
    parser.add_argument("-w", "--wordlist", type=str, help="Wordlist to brute force dir")

    args = parser.parse_args()
    if not args.url:
        parser.error("Please use --help for more information")
    elif not args.wordlist:
        parser.error("Please use --help for more information")
    return args

def dirsearch(url, wordlist):
    headers = get_headers()
    t = get_time()
    f = open(wordlist, 'r')
    for file in f:
        try:
            target = url + file.strip()
            resp = requests.get(target, headers=headers, verify=True)
            if resp.status_code == 200:
                threadLock.acquire()
                print("\033[92m[{}] {} -> {} {}\033[0m".format(t, resp.status_code, resp.status_code, resp.url))
                threadLock.release()

            elif resp.history == 301 or resp.history == 302:
                threadLock.acquire()
                print("\033[96m[{}] {} -> {} {}\033[0m".format(t, resp.history, resp.status_code, resp.url))
                threadLock.release()

            elif resp.status_code == 403 or resp.status_code == 409:
                threadLock.acquire()
                print("\033[95m[{}] {} -> {} {}\033[0m".format(t, resp.history, resp.status_code, file.strip()))
                threadLock.release()

            elif resp.status_code == 400 or resp.status_code == 404:
                pass

            else:
                threadLock.acquire()
                print("\033[91m[{}] {} -> {} {}\033[0m".format(t, resp.history, resp.status_code, resp.url))
                threadLock.release()

        except Exception as err:
            threadLock.acquire()
            print("Error: {}".format(err))
            threadLock.release()
            sys.exit()


def main():
    args = get_arguments()
    date_time = get_time()
    url = args.url
    wordlist = args.wordlist
    try:
        print("\n\033[93m[{}] \033[96mStarting:\033[0m\n".format(date_time))
        t = MyThread(url, wordlist)
        threads.append(t)
        t.start()

        for i in threads:
            i.join()
        print("\nDone!")
    except KeyboardInterrupt:
        print("")
        sys.exit()

if __name__ == '__main__':
    main()


