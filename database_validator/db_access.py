from urllib.parse import quote_plus
import pymongo

class DBConnection:
    def __init__(self, username, password, host, port):
        uri = f"mongodb://{quote_plus(username)}:{quote_plus(password)}@{host}:{port}/?serverSelectionTimeoutMS=5000"
        self.client = pymongo.MongoClient(uri, maxPoolSize=50)
        self.db = self.client['DigisatServer']
