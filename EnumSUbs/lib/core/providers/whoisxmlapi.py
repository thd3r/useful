#!/usr/bin/env python3

import json
import os

from lib.core.connection.requester import Request

class Whoisxmlapi:

    def __init__(self, domain, silent=False):
        base_url = "https://subdomains.whoisxmlapi.com/api/v1?apiKey={apikey}&domainName={domain}&outputFormat=json"
        engine_name = "Subdomains whoisxmlapi"
        self.domain = domain
        self.subdomains = []

        self.requester = Request(base_url, domain, engine_name, silent)


    def send_request(self):
        apikey = self.handle_whoisxml_apikey()
        response = self.requester._requesters(apikey=apikey)
        if response:
            self.extract_domain(response)

    def extract_domain(self, response):
        out = json.loads(response)

        for i in range(len(out["result"]["records"])):
            try:
                subdomain = out["result"]["records"][i]["domain"]
                if not subdomain.endswith(self.domain):
                    continue
                if subdomain not in self.subdomains and subdomain != self.domain:
                    self.subdomains.append(subdomain.strip())

            except KeyError:
                pass

    def handle_whoisxml_apikey(self):
        apikey = "lib/config/apikey.json"
        try:
            if os.path.exists(apikey):
                with open(apikey, 'r') as key:
                    out = json.load(key)
                    whoisxmlKey = out["config"][0]["key"]["whoisxml_apikey"]

                return whoisxmlKey

            else:
                return False

        except FileNotFoundError:
            pass

    def enumerate(self):
        self.send_request()

        if len(self.subdomains) > 0:
            return self.subdomains
        
        return self.subdomains