## Arquivo change_tributation_for_ncm.py:
from config.config import running_operations, running_operations_lock, cancel_event
from database_validator.database_validator import DatabaseValidator
from bson import ObjectId
from bson.errors import InvalidId
import threading
import customtkinter as ctk


class ChangeTributationForNCM:

    def __init__(self, db_connection, log, open_modal_callback):
        self.db = db_connection.db
        self.log = log
        self.database_validator = DatabaseValidator(db_connection, log)
        self.open_modal_callback = open_modal_callback

    def fetch_tributation_ids(self):
        """Função para buscar todos os IDs de tributação estadual."""
        try:
            tributations = self.db.TributacoesEstadual.find({"Ativo": True}, {"_id": 1, "Descricao": 1})
            tributation_list = [
                {
                    "id": str(t["_id"]),
                    "Descricao": t["Descricao"] if "Descricao" in t else "Sem descrição"
                }
                for t in tributations
            ]

            if not tributation_list:
                self.log.insert(ctk.END, "Aviso: Nenhuma tributação encontrada no banco de dados.\n")
                self.log.see(ctk.END)

            return tributation_list

        except Exception as e:
            self.log.insert(ctk.END, f"Erro ao buscar tributações: {str(e)}\n")
            self.log.see(ctk.END)
            return []

    def process_change_tributation(self, ncms, tributation_id):
        """Função que realiza a alteração da tributação para os NCMs especificados"""
        total_updates = 0

        try:
            with running_operations_lock:
                if not running_operations:
                    self.log.insert(ctk.END, "Operação cancelada.\n")
                    self.log.see(ctk.END)
                    return


            if not tributation_id or not isinstance(tributation_id, str):
                self.log.insert(ctk.END, "ID de tributação inválido.\n")
                self.log.see(ctk.END)
                return

            if not ncms or not isinstance(ncms, list) or not all(isinstance(n, str) for n in ncms):
                self.log.insert(ctk.END, "Lista de NCMs inválida.\n")
                self.log.see(ctk.END)
                return


            ncms = [ncm.strip() for ncm in ncms if ncm.strip()]
            if not ncms:
                self.log.insert(ctk.END, "Nenhum NCM válido fornecido.\n")
                self.log.see(ctk.END)
                return

            try:
                tributation_id_obj = ObjectId(tributation_id)
            except InvalidId:
                self.log.insert(ctk.END, f"ID de tributação inválido: {tributation_id}\n")
                self.log.see(ctk.END)
                return


            tributation_exists = self.db.TributacoesEstadual.find_one({"_id": tributation_id_obj})
            if not tributation_exists:
                self.log.insert(ctk.END, f"ID de tributação não encontrado no banco: {tributation_id}\n")
                self.log.see(ctk.END)
                return


            for ncm in ncms:
                try:
                    result = self.db.ProdutosServicosEmpresa.update_many(
                        {"NcmNbs.Codigo": {"$regex": f"^{ncm}.*", "$options": "i"}},
                        {"$set": {"TributacaoEstadualReferencia": tributation_id_obj}}
                    )

                    if result.modified_count > 0:
                        total_updates += result.modified_count
                        self.log.insert(ctk.END,
                                        f"Atualizado {result.modified_count} produtos com NCM que começam com {ncm}\n")
                    else:
                        self.log.insert(ctk.END, f"Nenhum produto encontrado para NCM que começa com {ncm}\n")
                    self.log.see(ctk.END)
                except Exception as e:
                    self.log.insert(ctk.END, f"Erro ao processar NCM {ncm}: {str(e)}\n")
                    self.log.see(ctk.END)


            self.log.insert(ctk.END, f"\n=== Operação concluída com sucesso! ===\n")
            self.log.insert(ctk.END, f"Total de produtos atualizados: {total_updates}\n")
            self.log.insert(ctk.END, f"NCMs processados: {len(ncms)}\n")
            self.log.insert(ctk.END, "=====================================\n\n")
            self.log.see(ctk.END)

        except Exception as e:
            self.log.insert(ctk.END, f"Erro durante o processamento: {str(e)}\n")
            self.log.see(ctk.END)

    def cancel_operation(self):
        with running_operations_lock:
            global running_operations
            running_operations = False

    def run_thread_change_tributation_for_ncm(self, ncms, tributation_id=None):
        """
        Função principal que gerencia a execução da alteração de tributação.
        Se tributation_id não for fornecido, abre o modal para seleção.
        """
        if tributation_id is None:
            print(f"NCMs recebidos: {ncms}")

            self.open_modal_callback(
                "Escolha o ID da Tributação",
                self.process_change_tributation,
                operation_type="run_change_tributation_for_ncm",
                show_second_entry=True
            )
        else:
            print(f"NCMs recebidos: {ncms}")
            print(f"Tributation ID recebido: {tributation_id}")

            thread = threading.Thread(
                target=self.process_change_tributation,
                args=(ncms, tributation_id)
            )
            thread.start()
