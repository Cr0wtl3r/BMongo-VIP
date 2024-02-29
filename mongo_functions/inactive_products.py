from config.config import running_operations, running_operations_lock, cancel_event
from database_validator.database_validator import DatabaseValidator
import threading
import customtkinter as ctk

class InactiveProducts:

    def __init__(self, db_connection, log):
        self.db = db_connection.db
        self.log = log
        self.running = True
        self.database_validator = DatabaseValidator(db_connection, log)

    def run_inactive_products(self):
        if not self.database_validator.connect_to_db():
            self.log.see(ctk.END)
            return

        with running_operations_lock:
            if not running_operations:
                self.log.insert(ctk.END, "Operação cancelada.\n")
                self.log.see(ctk.END)
                return

        self.log.insert(ctk.END, "Buscando estoques...\n")
        estoques = self.db.Estoques.find(
            {"$or": [{"Quantidades.0.Quantidade": {"$lt": 1.0}},
                     {"Quantidades": []}]})
        self.log.insert(ctk.END, "Busca concluída.\n")

        self.log.insert(ctk.END, "Iterando sobre os estoques...\n")
        for estoque in estoques:
            if cancel_event.is_set():
                self.log.insert(ctk.END)
                self.log.see(ctk.END)
                return

            referencia = estoque['_id']
            produto_servico = self.db.ProdutosServicosEmpresa.find_one(
                {'EstoqueReferencia': referencia})
            self.log.see(ctk.END)

            if produto_servico:
                self.log.insert(ctk.END, f"Atualizando produto_servico com id {produto_servico['ProdutoServicoReferencia']}...\n")
                self.db.ProdutosServicos.update_one(
                    {'_id': produto_servico['ProdutoServicoReferencia']},
                    {'$set': {'Ativo': False}})
                if not self.running:
                    self.log.insert(ctk.END, "Operação cancelada durante a atualização. \n")
                    self.log.see(ctk.END)
                    return
        self.log.insert(ctk.END, "Iteração concluída.\n")
        self.log.see(ctk.END)

    def cancel_operation(self):
        with running_operations_lock:
            global running_operations
            running_operations = False

    def run_thread_inactive_products(self):
        thread = threading.Thread(target=self.run_inactive_products)
        thread.start()

