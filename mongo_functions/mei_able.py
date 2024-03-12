from config.config import running_operations, running_operations_lock, cancel_event
from database_validator.database_validator import DatabaseValidator
import threading
import customtkinter as ctk


class MeiAble:

    def __init__(self, db_connection, log):
        self.db = db_connection.db
        self.log = log
        self.running = True
        self.database_validator = DatabaseValidator(db_connection, log)

    def run_mei_able(self):

        with running_operations_lock:
            if not running_operations:
                self.log.insert(ctk.END, "Operação cancelada.\n")
                self.log.see(ctk.END)
                return

        self.log.insert(ctk.END, "Realizando a alteração do Enquandramento temporáriamente...\n")

        result = self.db.Pessoas.update_many(
            {"_t.2": "Emitente"},
            {"$set": {"MicroempreendedorIndividual.Habilitado": True}}
        )

        if (result.modified_count == 1):
            return self.log.insert(ctk.END, f"Foi encontrada e alterada {result.modified_count} referência \n")
        elif (result.modified_count > 1):
            return self.log.insert(ctk.END, f"Foram encontradas e alteradas {result.modified_count} referências \n")
        elif (result.modified_count == 0):
            return self.log.insert(ctk.END, f"Verifique a base, pois não foi possível proceder! \n "
                                            f"Você tem certeza que restaurou alguma base?\n"
                                            f"Ou já não executou esse script antes? \n"
                                            f"Na duvida, reinicie o Servidor!\n"
                                   )

    def cancel_operation(self):
        with running_operations_lock:
            global running_operations
            running_operations = False

    def run_thread_mei_able(self):
        thread = threading.Thread(target=self.run_mei_able)
        thread.start()
