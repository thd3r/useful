#!/usr/bin/env python3

import requests
from requests.packages.urllib3 import disable_warnings

# Disable InsecureRequestWarning from urllib3
disable_warnings()

class bcolors:
    PURPLE = '\033[95m'
    OKBLUE = '\033[94m'
    BLUE = '\033[96m'
    GREEN = '\033[92m'
    WARNING = '\033[93m'
    FAIL = '\033[91m'
    END = '\033[0m'
    BOLD = '\033[1m'
    UNDERLINE = '\033[4m'

class Request:

    def __init__(self, url, domain, engine_name, silent=False):
        self.url = url
        self.domain = domain
        self.engine_name = engine_name
        self.silent = silent
        self.subdomains = []
        self.timeout = 25

        self.session = requests.Session()
        self.headers = {
            'User-Agent': 'Mozilla/5.0 (iPad; CPU OS 8_4_1 like Mac OS X) AppleWebKit/600.1.4 (KHTML, like Gecko) Version/8.0 Mobile/12H321 Safari/600.1.4',
            'Accept-Encoding': 'gzip, deflate',
        }
        self.session.headers.update(self.headers)

    def _requester(self):
        try:
            resp = self.session.get(self.url.format(domain=self.domain), headers=self.headers, timeout=self.timeout)

            if (resp.status_code == 200):
               return resp.text

            else:
                self.exceptions_requester(silent=self.silent)
        except Exception:
            pass

    def _requesters(self, apikey):
        try:
            resp = self.session.get(self.url.format(domain=self.domain or apikey, apikey=apikey or self.domain), headers=self.headers, timeout=self.timeout)

            if (resp.status_code == 200):
               return resp.text

            else:
                self.exceptions_requester(silent=self.silent)
        except Exception:
            pass

    def exceptions_requester(self, silent=False):
        error = f"{bcolors.FAIL}{bcolors.BOLD}Error: {engine_name} probably now is blocking our requests{bcolors.END}"
        if not silent:
            print(error)
        else:
            pass