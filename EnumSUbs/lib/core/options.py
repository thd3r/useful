#!/usr/bin/env python3

import argparse, sys

class OptionsArgs:

    def __init__(self):
        self.parser = argparse.ArgumentParser(
            prog="TrEnum", 
            usage=f"{sys.argv[0]} [ -m options mode [default arguments] ] [ -d domain [default arguments] ] [ arguments ]", 
            epilog="See \033[4mgithub.com/thd3rBoy/TrEnum\033[0m for more info"
        )
        self.parser._optionals.title = "Options"
        self.parser.add_argument(
            "-v",
            "--version",
            action="version",
            version="%(prog)s 1.0"
        )
        self.parser.add_argument(
            "-m",
            "--options-mode",
            dest="mode",
            action="store",
            required=True,
            help="Mode options can be set by choice (ex: subs, wayback)"
        )
        self.parser.add_argument(
            "-d",
            "--domain",
            required=True,
            help="Domain name to enumerate"
        )
        self.parser.add_argument(
            "-o",
            "--output",
            help="file to write output results"
        )
        self.parser.add_argument(
            "-s",
            "--silent",
            action="store_true",
            default=False,
            help="Silent mode, Only print the result and ignore the error"
        )
        self.parser.add_argument(
            "--no-subs",
            action="store_false",
            default=True,
            help="Don\'t include subdomains. This only applies to the wayback mode option"
        )

        return

    def _options(self):
        args = self.parser.parse_args()
        return args
