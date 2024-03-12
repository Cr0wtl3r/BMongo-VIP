from mongo_functions.base_clean import BaseClean
from mongo_functions.find_ids import FindIds
from mongo_functions.inactive_products import InactiveProducts
from mongo_functions.mei_able import MeiAble
from mongo_functions.movimentations_clean import MovimentationsClean
from database_validator.db_access import DBConnection
from database_validator.database_validator import DatabaseValidator
from config.config import running_operations_lock, cancel_event
from PIL import Image
from dotenv import load_dotenv
import os
import sys
import customtkinter as ctk
import threading



extDataDir = os.getcwd()
if getattr(sys, 'frozen', False):
    extDataDir = sys._MEIPASS

dotenv_path = os.path.join(extDataDir, '.env')
load_dotenv(dotenv_path)


class UserInterface:
    db_user = os.getenv("DB_USER")
    db_pass = os.getenv("DB_PASS")
    db_host = os.getenv("DB_HOST")

    db_connection = DBConnection(db_user, db_pass, db_host, 12220)

    def __init__(self):
        if getattr(sys, 'frozen', False):
            # Variáveis de compilação
            ico_path = UserInterface.resource_path('BMongo-VIP\\src\\logo.ico')
            background_path = UserInterface.resource_path('BMongo-VIP\\src\\background.png')
        else:
            # Vairáveis de desenvolvimento
            ico_path = UserInterface.resource_path('src\\logo.ico')
            background_path = UserInterface.resource_path('src\\background.png')

        ctk.set_appearance_mode("dark")
        self.app = ctk.CTk()
        self.app.title("BMongo - VIP")
        self.app.wm_iconbitmap(ico_path)
        self.app.geometry('800x600')
        self.app.config(takefocus=True)
        image_background = ctk.CTkImage(dark_image=Image.open(
            background_path), size=(800, 600))
        background_label = ctk.CTkLabel(self.app, image=image_background, text='')
        background_label.place(x=0, y=0, relwidth=1, relheight=1)
        self.log = ctk.CTkTextbox(self.app, width=500)
        self.log.pack(pady=50)
        self.inactive_products = InactiveProducts(self.db_connection, self.log)
        self.mei_able = MeiAble(self.db_connection, self.log)
        self.find_ids = FindIds(self.db_connection, self.log)
        self.movimentations_clean = MovimentationsClean(self.db_connection, self.log)
        self.base_clean = BaseClean(self.db_connection, self.log)
        self.database_validator = DatabaseValidator(self.db_connection, self.log)

        thread = threading.Thread(target=self.check_database_connection)
        thread.start()


        button_inactive_products = ctk.CTkButton(
            self.app, text="Inativar produtos Zerados ou Negativos",
            command=self.inactive_products.run_thread_inactive_products, fg_color='#f6882d', hover_color='#c86e24',
            text_color='white',
            border_color='#123f8c')
        button_inactive_products.pack(pady=10)

        button_mei_able = ctk.CTkButton(
            self.app, text="Permitir o ajuste de Estoque",
            command=self.mei_able.run_thread_mei_able, fg_color='#f6882d', hover_color='#c86e24',
            text_color='white',
            border_color='#123f8c')
        button_mei_able.pack(pady=10)

        button_movimentations_clean = ctk.CTkButton(
            self.app, text="Limpa movimentações da Base",
            command=self.movimentations_clean.run_thread_movimentations_clean, fg_color='#f6882d', hover_color='#c86e24',
            text_color='white',
            border_color='#123f8c')
        button_movimentations_clean.pack(pady=10)

        button_base_clean = ctk.CTkButton(
            self.app, text="Zera a base atual",
            command=self.base_clean.run_thread_base_clean, fg_color='#f6882d', hover_color='#c86e24',
            text_color='white',
            border_color='#123f8c')
        button_base_clean.pack(pady=10)

        button_find_ids = ctk.CTkButton(
            self.app, text="Localiza ID's no banco",
            command=self.open_search_modal, fg_color='#f6882d', hover_color='#c86e24',
            text_color='white',
            border_color='#123f8c')
        button_find_ids.pack(pady=10)

        cancel_button = ctk.CTkButton(
            self.app, text="Cancelar Operação", command=self.cancel_operation, fg_color='#031229',
            hover_color='#010c1c',
            text_color='white', border_color='#123f8c')
        cancel_button.pack(pady=25)



    def cancel_operation(self):
        with running_operations_lock:
            global running_operations
            running_operations = False
        cancel_event.set()
        self.log.insert(ctk.END, "Todas as operações foram canceladas.\n")
        self.log.see(ctk.END)
        self.app.after(1000, self.reset_operation_state)

    @staticmethod
    def reset_operation_state():
        with running_operations_lock:
            global running_operations
            running_operations = True
        cancel_event.clear()

    @classmethod
    def resource_path(cls, relative_path):
        try:
            base_path = sys._MEIPASS
        except Exception:
            base_path = os.path.abspath(".")
        return os.path.join(base_path, relative_path)

    def open_search_modal(self):
        self.search_modal = ctk.CTk()
        self.search_modal.title("Buscar ObjectId")
        if getattr(sys, 'frozen', False):
            # Variáveis de compilação
            self.search_modal.wm_iconbitmap(UserInterface.resource_path('BMongo-VIP\\src\\logo.ico'))
        else:
            # Vairáveis de desenvolvimento
            self.search_modal.wm_iconbitmap(UserInterface.resource_path('src\\logo.ico'))

        self.search_modal.geometry("300x150")
        self.object_id_entry = ctk.CTkEntry(self.search_modal, width=150)
        self.object_id_entry.pack(pady=10)

        button_search = ctk.CTkButton(
            self.search_modal, text="Pesquisar",
            command=self.run_find_ids, fg_color='#f6882d', hover_color='#c86e24',
            text_color='white',
            border_color='#123f8c')
        button_search.pack(pady=10)

        modal_width = 300
        modal_height = 150
        window_width = self.app.winfo_width()
        window_height = self.app.winfo_height()
        x = (window_width // 2) - (modal_width // 2)
        y = (window_height // 2) - (modal_height // 2)

        self.search_modal.geometry(f"{modal_width}x{modal_height}+{x}+{y}")

        self.search_modal.mainloop()

    def run_find_ids(self):
        search_id = self.object_id_entry.get()
        self.find_ids.run_thread_find_ids(search_id)
        self.search_modal.destroy()

    def check_database_connection(self):
        try:
            if not self.database_validator.connect_to_db():
                return
        except Exception as e:
            self.log.insert(ctk.END, str(e) + "\n")
            return

    def run(self):
        self.app.mainloop()


if __name__ == "__main__":
    user_interface = UserInterface()
    user_interface.run()