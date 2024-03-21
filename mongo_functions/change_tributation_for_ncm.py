from config.config import running_operations, running_operations_lock, cancel_event
from database_validator.database_validator import DatabaseValidator
from bson import ObjectId
import threading
import re
import customtkinter as ctk

class ChangeTributationForNCM:

    def __init__(self, db_connection, log):
        self.db = db_connection.db
        self.log = log
        self.database_validator = DatabaseValidator(db_connection, log)

    def run_change_tributation_for_ncm(self, ncms, tributation_id):
        with running_operations_lock:
            if not running_operations:
                self.log.insert(ctk.END, "Operação cancelada.\n")
                self.log.see(ctk.END)
                return

        tributation_id_obj = ObjectId(tributation_id)

        for ncm in ncms:

            result = self.db.ProdutosServicosEmpresa.update_many(
                {"NcmNbs.Codigo": {"$regex": f"^{ncm}.*", "$options": "i"}},
                {"$set": {"TributacaoEstadualReferencia": tributation_id_obj}}
            )
            if result.modified_count > 0:
                self.log.insert(ctk.END,
                                f"Atualizado NCMs que começam com {ncm} com TributacaoEstadualReferencia {tributation_id}\n")
                self.log.see(ctk.END)
            else:
                self.log.insert(ctk.END, f"Nenhum documento encontrado para NCMs que começam com {ncm}.\n")
                self.log.see(ctk.END)

    def cancel_operation(self):
        with running_operations_lock:
            global running_operations
            running_operations = False

    def run_thread_change_tributation_for_ncm(self, ncms, tributation_id):
        thread = threading.Thread(target=self.run_change_tributation_for_ncm, args=(ncms, tributation_id))
        thread.start()
