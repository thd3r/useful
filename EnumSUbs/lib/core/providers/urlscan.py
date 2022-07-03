#!/usr/bin/env python3

import json

from lib.core.connection.requester import Request

class Urlscan:

    def __init__(self, domain, silent=False):
        base_url = "https://urlscan.io/api/v1/search/?q=domain:{domain}"
        engine_name = "Urlscan"
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

        for i in range(len(out["results"])):
            subdomain = out["results"][i]["page"]["domain"]
            try:
                if out["results"][i]["page"]["apexDomain"] != self.domain:
                    continue
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