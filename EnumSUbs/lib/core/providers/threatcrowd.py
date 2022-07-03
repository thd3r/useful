#!/usr/bin/env python3

import requests
import threading
import json
import sys


from lib.core.connection.requester import Request

class ThreatcrowdEnums:

    def __init__(self, domain, silent=False):
        base_url = "https://www.threatcrowd.org/searchApi/v2/domain/report/?domain={domain}"
        engine_name = "Threatcrowd"
        self.domain = domain
        self.subdomains = []

        self.requester = Request(base_url, domain, engine_name, silent)

        return

    def send_request(self):
        response = self.requester._requester()
        if response:
            self.extract_domain(response)

    def extract_domain(self, response):
        out = json.loads(response)
        for subdomain in out["subdomains"]:
            if not subdomain.endswith(self.domain):
                continue
            if subdomain not in self.subdomains and subdomain != self.domain:
                self.subdomains.append(subdomain.strip())

    def enumerate(self):
        self.send_request()

        if len(self.subdomains) > 0:
            return self.subdomains
        
        return self.subdomains
