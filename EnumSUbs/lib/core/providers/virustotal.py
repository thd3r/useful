#!/usr/bin/env python3

import json
import os
import sys

from lib.core.connection.requester import Request

class VirustotalEnums:

    def __init__(self, domain, silent=False):
        base_url = "https://www.virustotal.com/vtapi/v2/domain/report?domain={domain}&apikey={apikey}"
        engine_name = "Virustotal"
        self.domain = domain
        self.subdomains = []

        self.requester = Request(base_url, domain, engine_name, silent)

        return

    def vt_apikey(self):
        apikey = "lib/core/config/apikey.json"
        try:
            if os.path.exists(apikey):
                with open(apikey, 'r') as key:
                    out = json.load(key)
                    Vtapikey = out["config"][0]["key"]["vt_apikey"]

                return Vtapikey

            else:
                return False

        except FileNotFoundError:
            pass

    def send_request(self):
        apikey = self.vt_apikey()
        response = self.requester._requesters(apikey=apikey)
        if response:
            self.extract_domain(response)

    def extract_domain(self, response):
        out = json.loads(response)
        try:
            if "Domain not found" in out["verbose_msg"]:
                print("Error: Domain not found, Please double check your target domain")
                sys.exit(1)
            else:
                for subdomain in out["subdomains"]:
                    if not subdomain.endswith(self.domain):
                        continue
                    if subdomain not in self.subdomains and subdomain != self.domain:
                        self.subdomains.append(subdomain.strip())
        except KeyError:
            pass

    def enumerate(self):
        self.send_request()

        if len(self.subdomains) > 0:
            return self.subdomains
        
        return self.subdomains