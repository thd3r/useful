#!/usr/bin/env python3

from __future__ import print_function

from dataclasses import dataclass
import mysql.connector

@dataclass
class db:
    host:   str = "localhost"
    user:   str = "amer"
    password:   str = "amer"
    database:   str = "flaskz"

Mydb = (
    lambda host, user, password, database: mysql.connector.connect(
        host = host,
        user = user,
        password = password,
        database = database or None
    )
    )(db.host, db.user, db.password, db.database)
