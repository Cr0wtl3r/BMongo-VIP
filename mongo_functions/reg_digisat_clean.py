import os
import subprocess
import threading
import ctypes
import sys
import customtkinter as ctk
import shutil
import winreg
from config.config import running_operations, running_operations_lock, cancel_event
from database_validator.database_validator import DatabaseValidator

def is_admin():
    try:
        return ctypes.windll.shell32.IsUserAnAdmin()
    except:
        return False

def run_as_admin():
    ctypes.windll.shell32.ShellExecuteW(None, "runas", sys.executable, " ".join(sys.argv), None, 1)

if not is_admin():
    run_as_admin()
    sys.exit()

class RegDigisatClean:

    def __init__(self, db_connection, log):
        self.db = db_connection.db
        self.log = log
        self.running = True
        self.database_validator = DatabaseValidator(db_connection, log)

    def run_reg_digisat_clean(self):
        with running_operations_lock:
            if not running_operations:
                self.log.insert(ctk.END, "Operação cancelada.\n")
                self.log.see(ctk.END)
                return

        self.log.insert(ctk.END, "Executando a remoção do Digisat dos Registros do Windows...\n")

        if self.is_process_running("ServidorG6.exe"):
            self.execute_command("taskkill /f /im ServidorG6.exe")
        else:
            self.log.insert(ctk.END, "Processo 'ServidorG6.exe' não encontrado.\n")

        if self.is_service_running("MongoDBDigisat"):
            self.execute_command("net stop MongoDBDigisat")
        else:
            self.log.insert(ctk.END, "Serviço 'MongoDBDigisat' não encontrado.\n")

        if self.is_service_running("SincronizadorDigisat"):
            self.execute_command("net stop SincronizadorDigisat")
        else:
            self.log.insert(ctk.END, "Serviço 'SincronizadorDigisat' não encontrado.\n")

        reg_file_path = os.path.join("regs_windows", "reg_digisat.reg")
        self.execute_command(f'regedit /s "{reg_file_path}"')

    def execute_command(self, command):
        try:
            subprocess.run(command, shell=True, check=True, stdout=subprocess.DEVNULL, stderr=subprocess.STDOUT)
            self.log.insert(ctk.END, f"Comando '{command}' executado com sucesso.\n")
        except subprocess.CalledProcessError as e:
            self.log.insert(ctk.END, f"Erro ao executar o comando '{command}': {e}\n")

    def is_process_running(self, process_name):
        try:
            output = subprocess.check_output('tasklist', shell=True).decode()
            return process_name in output
        except Exception as e:
            return False

    def is_service_running(self, service_name):
        try:
            output = subprocess.check_output(f'sc query "{service_name}"', shell=True).decode()
            return "RUNNING" in output
        except Exception as e:
            return False

    def cancel_operation(self):
        with running_operations_lock:
            global running_operations
            running_operations = False

    def run_thread_reg_digisat_clean(self):
        thread = threading.Thread(target=self.run_reg_digisat_clean)
        thread.start()
