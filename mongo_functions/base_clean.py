import os
import subprocess
import threading
import ctypes
import sys
import customtkinter as ctk
import shutil
from config.config import running_operations, running_operations_lock, cancel_event
from database_validator.database_validator import DatabaseValidator

def is_admin():
    try:
        return ctypes.windll.shell32.IsUserAnAdmin()
    except:
        return False

if not is_admin():
    ctypes.windll.shell32.ShellExecuteW(None, "runas", sys.executable, " ".join(sys.argv), None, 1)
    sys.exit()

class BaseClean:

    def __init__(self, db_connection, log):
        self.db = db_connection.db
        self.log = log
        self.running = True
        self.database_validator = DatabaseValidator(db_connection, log)

    def run_base_clean(self):
        with running_operations_lock:
            if not running_operations:
                self.log.insert(ctk.END, "Operação cancelada.\n")
                self.log.see(ctk.END)
                return

        self.log.insert(ctk.END, "Executando a Limpeza da base...\n")

        self.execute_command("taskkill /f /im ServidorG6.exe")

        self.execute_command("net stop MongoDBDigisat")
        self.execute_command("net stop SincronizadorDigisat")

        self.delete_file("C:\\DigiSat\\SuiteG6\\Servidor\\ConfiguracaoServer.xml")
        self.delete_file("C:\\DigiSat\\SuiteG6\\Sistema\\ConfiguracaoClient.xml")

        self.remove_directory("C:\\DigiSat\\SuiteG6\\Dados")
        self.create_directory("C:\\DigiSat\\SuiteG6\\Dados")

        self.log.insert(ctk.END, f"Conluída a limpeza da Base! Pode restaurar outra. \n")

    def execute_command(self, command):
        try:
            subprocess.run(command, shell=True, check=True)
            self.log.insert(ctk.END, f"Comando '{command}' executado com sucesso.\n")
        except subprocess.CalledProcessError as e:
            self.log.insert(ctk.END, f"Erro ao executar o comando '{command}': {e}\n")

    def delete_file(self, file_path):
        try:
            os.remove(file_path)
            self.log.insert(ctk.END, f"Arquivo '{file_path}' excluído com sucesso.\n")
        except FileNotFoundError:
            self.log.insert(ctk.END, f"Arquivo '{file_path}' não encontrado.\n")
        except PermissionError:
            self.log.insert(ctk.END, f"Permissão negada para excluir o arquivo '{file_path}'.\n")

    def remove_directory(self, directory_path):
        try:
            shutil.rmtree(directory_path)
            self.log.insert(ctk.END, f"Diretório '{directory_path}' removido com sucesso.\n")
        except FileNotFoundError:
            self.log.insert(ctk.END, f"Diretório '{directory_path}' não encontrado.\n")
        except PermissionError:
            self.log.insert(ctk.END, f"Permissão negada para remover o diretório '{directory_path}'.\n")
        except Exception as e:
            self.log.insert(ctk.END, f"Erro ao remover o diretório '{directory_path}': {e}\n")

    def create_directory(self, directory_path):
        try:
            os.makedirs(directory_path, exist_ok=True)
            self.log.insert(ctk.END, f"Diretório '{directory_path}' criado com sucesso.\n")
        except PermissionError:
            self.log.insert(ctk.END, f"Permissão negada para criar o diretório '{directory_path}'.\n")

    def cancel_operation(self):
        with running_operations_lock:
            global running_operations
            running_operations = False

    def run_thread_base_clean(self):
        thread = threading.Thread(target=self.run_base_clean)
        thread.start()
