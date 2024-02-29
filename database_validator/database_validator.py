import customtkinter as ctk
import pymongo

class DatabaseValidator:
    def __init__(self, db_connection, log):
        self.db = db_connection.db
        self.log = log
        self.is_db_empty = None

    def connect_to_db(self):
        if self.is_db_empty is not None:
            return self.is_db_empty

        try:
            self.db.client.server_info()
            self.is_db_empty = self.db.Estoques.count_documents({}) ==  0
            if self.is_db_empty:
                self.log.insert(ctk.END, "O banco de dados está vazio. Por favor, inicie o servidor para adicionar informações.\n")
            else:
                self.log.insert(ctk.END, "Conexão com o banco de dados estabelecida com sucesso. \\o/ \n")
            return not self.is_db_empty
        except pymongo.errors.ServerSelectionTimeoutError:
            self.log.insert(ctk.END, "Erro ao conectar ao banco de dados. Verifique se você restaurou alguma base.\n")
            return False
