from pymongo import MongoClient
from config.config import running_operations, running_operations_lock, cancel_event
from database_validator.database_validator import DatabaseValidator
import threading
import customtkinter as ctk

class MovimentationsClean:

    def __init__(self, db_connection, log):
        self.db = db_connection.db
        self.log = log
        self.running = True
        self.database_validator = DatabaseValidator(db_connection, log)

    def run_movimentations_clean(self):
        with running_operations_lock:
            if not running_operations:
                self.log.insert(ctk.END, "Operação cancelada.\n")
                self.log.see(ctk.END)
                return

        self.log.insert(ctk.END, "Iniciando operações de atualização...\n")
        self.update_movimentacoes()
        self.update_recebimentos()
        self.update_turnos_lancamentos()
        self.log.insert(ctk.END, "Operações de atualização concluídas.\n")
        self.log.see(ctk.END)

    def update_movimentacoes(self):
        if not self.running:
            self.log.insert(ctk.END, "Operação cancelada durante a atualização. \n")
            self.log.see(ctk.END)
            return

        self.log.insert(ctk.END, "Atualizando Movimentacoes...\n")
        for i in range(3):
            self.db.Movimentacoes.update_many(
                {"PagamentoRecebimento.Parcelas.0.Historico.{}.EspeciePagamento.Descricao".format(i): {"$regex": ".*Cart.*", "$options": "i"}},
                {"$unset": {"PagamentoRecebimento.Parcelas.0.Historico.{}.EspeciePagamento.Pessoa.Imagem".format(i): ""}}
            )
            if not self.running:
                self.log.insert(ctk.END, "Operação cancelada durante a atualização. \n")
                self.log.see(ctk.END)
                return

        self.db.Movimentacoes.update_many(
            {"PagamentoRecebimento.Parcelas.0.Historico.0.EspeciePagamento.Descricao": {"$regex": ".*Cart.*", "$options": "i"}},
            {"$unset": {"PagamentoRecebimento.Parcelas.0.Pessoa.Imagem": ""}}
        )
        if not self.running:
            self.log.insert(ctk.END, "Operação cancelada durante a atualização. \n")
            self.log.see(ctk.END)
            return

    def update_recebimentos(self):
        if not self.running:
            self.log.insert(ctk.END, "Operação cancelada durante a atualização. \n")
            self.log.see(ctk.END)
            return

        self.log.insert(ctk.END, "Atualizando Recebimentos...\n")
        for i in range(3):
            self.db.Recebimentos.update_many(
                {"Historico.{}.EspeciePagamento.Descricao".format(i): {"$regex": ".*Cart.*", "$options": "i"}},
                {"$unset": {"Historico.{}.EspeciePagamento.Pessoa.Imagem".format(i): ""}}
            )
            if not self.running:
                self.log.insert(ctk.END, "Operação cancelada durante a atualização. \n")
                self.log.see(ctk.END)
                return

        self.db.Recebimentos.update_many(
            {"Historico.0.EspeciePagamento.Descricao": {"$regex": ".*Cart.*", "$options": "i"}},
            {"$unset": {"Pessoa.Imagem": ""}}
        )
        if not self.running:
            self.log.insert(ctk.END, "Operação cancelada durante a atualização. \n")
            self.log.see(ctk.END)
            return

    def update_turnos_lancamentos(self):
        if not self.running:
            self.log.insert(ctk.END, "Operação cancelada durante a atualização. \n")
            self.log.see(ctk.END)
            return

        self.log.insert(ctk.END, "Atualizando TurnosLancamentos...\n")
        self.db.TurnosLancamentos.update_many(
            {"EspeciePagamento.Descricao": {"$regex": ".*Cart.*", "$options": "i"}},
            {"$unset": {"EspeciePagamento.Pessoa.Imagem": ""}}
        )
        if not self.running:
            self.log.insert(ctk.END, "Operação cancelada durante a atualização. \n")
            self.log.see(ctk.END)
            return

    def cancel_operation(self):
        with running_operations_lock:
            global running_operations
            running_operations = False

    def run_thread_movimentations_clean(self):
        thread = threading.Thread(target=self.run_movimentations_clean)
        thread.start()
