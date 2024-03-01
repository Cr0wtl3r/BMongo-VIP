from urllib.parse import quote_plus
import pymongo


class DBConnection:
    def __init__(self, username, password, host, port):
        if not username or not password or not host:
            raise ValueError("As variáveis de ambiente DB_USER, DB_PASS e DB_HOST devem estar definidas.")

        username_bytes = username.encode('utf-8')
        password_bytes = password.encode('utf-8')
        host_bytes = host.encode('utf-8')

        uri = f"mongodb://{quote_plus(username_bytes)}:{quote_plus(password_bytes)}@{quote_plus(host_bytes)}:{port}/?serverSelectionTimeoutMS=5000"
        self.client = pymongo.MongoClient(uri, maxPoolSize=50)
        self.db = self.client['DigisatServer']