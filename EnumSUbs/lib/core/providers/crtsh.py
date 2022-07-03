#!/usr/bin/env python3

import json

from lib.core.connection.requester import Request

class CrtshSearch:

    def __init__(self, domain, silent=False):
        base_url = "https://crt.sh/?q=%25.{domain}&output=json"
        engine_name = "Crtsh"
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
        for subdomain in out:
            subdomain = subdomain["name_value"]
            if not subdomain.endswith(self.domain) or '*' in subdomain:
                continue
            if '@' in subdomain:
                subdomain = subdomain[subdomain.find('@')+1:]
            if subdomain not in self.subdomains and subdomain != self.domain:
                self.subdomains.append(subdomain.strip())

    def enumerate(self):
        self.send_request()

        if len(self.subdomains) > 0:
            return self.subdomains
        
        return self.subdomains