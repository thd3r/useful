#!/usr/bin/env python3

import json
import random
import time
import sys
import os
import requests
import threading
from getpass import getpass
from colorama import Fore

if os.name == 'nt':
    os.system('cls')

else:
    os.system('clear')

threadLock = threading.Lock()
threads = []

author = '@__BytexThunder__'
banner = f'''{Fore.LIGHTBLUE_EX}
 _   _               ____                      
| | | |___  ___ _ __|  _ \ ___  ___ ___  _ __  
| | | / __|/ _ \ '__| |_) / _ \/ __/ _ \| '_ \ 
| |_| \__ \  __/ |  |  _ <  __/ (_| (_) | | | |
 \___/|___/\___|_|  |_| \_\___|\___\___/|_| |_|

                                               
                    {Fore.WHITE}             {author}
'''

def Start():
    start = getpass(prompt=Fore.LIGHTYELLOW_EX + "Code To Unlock The Tool: " + Fore.WHITE)
    code = '999666'
    if start != code:
        print(Fore.RED + '[X] Wrong Code' + Fore.WHITE)
        try:
            ask = input("\nDo you want to get the code? (N/y) ")
            if ask == 'y' or ask == 'Y' or ask == 'yes' or ask == 'Yes':
                import xyz

            elif ask == 'n' or ask == 'N' or ask == 'no' or ask == 'No':
                sys.exit()

            else:
                print(f"{Fore.RED}Invalid Input!\n{Fore.WHITE}")
                Start()

        except KeyboardInterrupt:
            print("")
            sys.exit()

    else:
        if os.name == 'nt':
            os.system('cls')

        else:
            os.system('clear')

        for x in banner + '\n':
            sys.stdout.write(x)
            sys.stdout.flush()
            time.sleep(random.random() * 0.1)

def get_headers():
    headers = {
        'User-Agent': 'Mozilla/5.0 (X11; Linux x86_64; rv:91.0) Gecko/20100101 Firefox/91.0'
    }
    return headers

class Instagram(threading.Thread):
    def __init__(self, username):
        threading.Thread.__init__(self)
        self.username = username

    def GetUserInformation(self):
        headers = get_headers()
        req = requests.get(f'https://www.instagram.com/{self.username}/?__a=1', headers=headers)
        try:
            threadLock.acquire()
            assert req.status_code != 404
            threadLock.release()
        except AssertionError:
            raise BaseException("Error: User Not Found!")
        except requests.ConnectionError:
            raise ConnectionError("Error: Connection Error!")
        finally:
            data = req

        output = json.loads(data.content)
        path = 'Instagram/{}'.format(self.username)

        if not os.path.exists(path):
            os.makedirs(path)

        info = {
            'User ID': output['graphql']['user']['id'],
            'Full Name': output['graphql']['user']['full_name'],
            'Username': output['graphql']['user']['username'],
            'Biography': str(output['graphql']['user']['biography'].replace("\n", " | ")),
            'Followers': output['graphql']['user']['edge_followed_by']['count'],
            'Followings': output['graphql']['user']['edge_follow']['count'],
            'Total Media': output['graphql']['user']['edge_owner_to_timeline_media']['count'],
            'Highlight Story': output['graphql']['user']['highlight_reel_count'],
            'Verified': output['graphql']['user']['is_verified'],
            'Private Account': output['graphql']['user']['is_private'],
            'External Url': output['graphql']['user']['external_url'],
            'Professional Account': output['graphql']['user']['is_professional_account'],
            'Business Account': output['graphql']['user']['is_business_account'],
            'Business Address': output['graphql']['user']['business_address_json'],
            'Business Contact': output['graphql']['user']['business_contact_method'],
            'Business Category': output['graphql']['user']['business_category_name'],
            'Connect with Facebook': output['graphql']['user']['connected_fb_page']
        }

        with open(path + "/info.json", "w") as file:
            try:
                file.write(json.dumps(info, indent=4) + "\n")
            finally:
                file.close()

        threadLock.acquire()
        print("")
        print("-" * 53)
        print(f'\t\t{Fore.LIGHTGREEN_EX}__User Information__{Fore.WHITE}')
        print("-" * 53)
        print(f"{Fore.LIGHTBLUE_EX}Instagram: {Fore.WHITE}{req.url}")
        print("\nUser ID\t\t\t: " + str(output['graphql']['user']['id']))
        print("Full Name\t\t: " + str(output['graphql']['user']['full_name']))
        print("Username\t\t: " + str(output['graphql']['user']['username']))
        print("Biography\t\t: " + str(output['graphql']['user']['biography']))
        print("Followers\t\t: " + str(output['graphql']['user']['edge_followed_by']['count']))
        print("Followings\t\t: " + str(output['graphql']['user']['edge_follow']['count']))
        print("Total Media\t\t: " + str(output['graphql']['user']['edge_owner_to_timeline_media']['count']))
        print("Highlight Story\t\t: " + str(output['graphql']['user']['highlight_reel_count']))
        print("Verified\t\t: " + str(output['graphql']['user']['is_verified']))
        print("Private Account\t\t: " + str(output['graphql']['user']['is_private']))
        print("External Url\t\t: " + str(output['graphql']['user']['external_url']))
        print("Professional Account\t: " + str(output['graphql']['user']['is_professional_account']))
        print("Business Account\t: " + str(output['graphql']['user']['is_business_account']))
        print("Business Address\t: " + str(output['graphql']['user']['business_address_json']))
        print("Business Contact\t: " + str(output['graphql']['user']['business_contact_method']))
        print("Business Category\t: " + str(output['graphql']['user']['business_category_name']))
        print("Connect with Facebook\t: " + str(output['graphql']['user']['connected_fb_page']))
        print(f"\nOutput File: {path}/info.json")
        threadLock.release()

if __name__ == '__main__':
    user = input(f'{Fore.LIGHTYELLOW_EX}Username: {Fore.WHITE}')
    Start()
    exp = Instagram(user)
    t = threading.Thread(target=exp.GetUserInformation)
    threads.append(t)
    t.start()

    for i in threads:
        i.join()
