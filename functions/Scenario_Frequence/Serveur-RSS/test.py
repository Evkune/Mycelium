from flask import Flask, Response, request
import requests
from datetime import datetime
import time
import json
from xml.etree import ElementTree

if __name__ == '__main__':
    print("running")
    with open("test.json", 'r') as file : 
        requests.post("http://127.0.0.1:80/add_post",json=json.load(file))
        response = requests.get("http://127.0.0.1:80/rss")
        xml_content = ElementTree.fromstring(response.content)
        xml_string = ElementTree.tostring(xml_content, encoding='utf-8', method='xml')
        print(xml_string.decode('utf-8'))
