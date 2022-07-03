#!/usr/bin/env python3

import multiprocessing
import threading
import json
import sys
import os

from urllib.parse import urlparse
from datetime import datetime

from lib.core.connection.requester import Request
from lib.core.output import write_file


class BaseEnumerateUrls:

    def __init__(self, domain, filename, no_subs=False, silent=False):
        self.domain = domain
        self.filename = filename
        self.no_subs = no_subs
        self.silent = silent
        self.urls = []

    def WebArchiveEnums(self):
        if not self.no_subs:
            base_url = "https://web.archive.org/cdx/search/cdx?url={domain}/*&output=json&collapse=urlkey&fl=original"
        else:
            base_url = "https://web.archive.org/cdx/search/cdx?url=*.{domain}/*&output=json&collapse=urlkey&fl=original"
        engine_name = "Web archive"
        requester = Request(base_url, self.domain, engine_name, self.silent)

        response = requester._requester()
        if response:
            out = json.loads(response)

            for i in range(len(out)):
                for url in out[i]:
                    if 'original' in url:
                        url = url.replace("original", "")
                    
                    else:
                        self.urls.append(url)

    def VirustotalEnums(self):
        base_url = "https://www.virustotal.com/vtapi/v2/domain/report?domain={domain}&apikey={apikey}"
        engine_name = "Virustotal"
        requester = Request(base_url, self.domain, engine_name, self.silent)

        apikey = self.vt_apikey()
        response = requester._requesters(apikey=apikey)
        if response:
            self.extract_urls(response)

    def extract_urls(self, response):
        out = json.loads(response)
        try:
            for i in range(len(out["detected_urls"])):
                url = out["detected_urls"][i]["url"]
            for i in range(len(out["undetected_urls"])):
                url = out["undetected_urls"][i][0]
                self.urls.append(url)
        except KeyError:
            pass

    def vt_apikey(self):
        apikey = "/home/thd3r/Documents/projects/python/TrEnum/EnumSUbs/lib/core/config/"
        try:
            if os.path.exists(apikey):
                with open(apikey + "apikey.json", 'r') as key:
                    out = json.load(key)
                    Vtapikey = out["config"][0]["key"]["vt_apikey"]

                return Vtapikey

            else:
                return False

        except FileNotFoundError:
            print("Error: The apikey.json file is not in the path")
            exit(1)

    def enumerate(self):
        self.VirustotalEnums()
        self.WebArchiveEnums()

        for url in self.urls:
            print(url.strip())

        if not self.filename:
            write_file(filename=self.domain, content=self.urls)
        else:
            write_file(filename=self.filename, content=self.urls)

    def run(self):
        t = threading.Thread(target=self.enumerate())
        t.daemon = True
        t.start()
        t.join()