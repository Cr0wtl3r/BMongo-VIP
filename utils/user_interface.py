## Arquivo user_interface.py:
from mongo_functions.base_clean import BaseClean
from mongo_functions.base_create import BaseCreate
from mongo_functions.change_tributation_for_ncm import *
from mongo_functions.find_ids import FindIds
from mongo_functions.inactive_products import InactiveProducts
from mongo_functions.mei_able import MeiAble
from mongo_functions.movimentations_clean import MovimentationsClean
from mongo_functions.reg_digisat_clean import RegDigisatClean
from database_validator.db_access import DBConnection
from database_validator.database_validator import DatabaseValidator
from components.buttons_creator import ButtonCreator
from config.config import running_operations_lock, cancel_event
from utils.util_modal import Modal
from PIL import Image
from dotenv import load_dotenv
import os
import sys
import threading
import customtkinter as ctk

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
            ico_path = UserInterface.resource_path('BMongo-VIP\\src\\logo.ico')
            background_path = UserInterface.resource_path('BMongo-VIP\\src\\background.png')
        else:
            ico_path = UserInterface.resource_path('src\\logo.ico')
            background_path = UserInterface.resource_path('src\\background.png')

        ctk.set_appearance_mode("dark")
        self.app = ctk.CTk(fg_color=None)
        self.app.title("BMongo - VIP")
        self.app.wm_iconbitmap(ico_path)
        self.app.config(takefocus=True)
        screen_width = self.app.winfo_screenwidth()
        screen_height = self.app.winfo_screenheight()
        window_width = 900
        window_height = 850
        position_top = int(screen_height / 2 - window_height / 2)
        position_right = int(screen_width / 2 - window_width / 2)
        self.app.geometry(f"{window_width}x{window_height}+{position_right}+{position_top}")

        self.image_background = ctk.CTkImage(dark_image=Image.open(background_path))
        self.background_label = ctk.CTkLabel(self.app, image=self.image_background, text='')
        self.background_label.place(x=0, y=0, relwidth=1, relheight=1)

        def update_background(*args):
            width = self.app.winfo_width()
            height = self.app.winfo_height()
            self.image_background.configure(size=(width, height))
            self.background_label.configure(image=self.image_background)
            self.background_label.place(x=0, y=0, relwidth=1, relheight=1)

        self.app.bind("<Configure>", update_background)
        self.log = ctk.CTkTextbox(self.app, width=700, height=250)
        self.log.grid(row=0, column=0, columnspan=6)
        self.log.place(relx=0.5, rely=0.18, anchor='center')

        self.inactive_products = InactiveProducts(self.db_connection, self.log)
        self.mei_able = MeiAble(self.db_connection, self.log)
        self.find_ids = FindIds(self.db_connection, self.log)
        self.movimentations_clean = MovimentationsClean(self.db_connection, self.log)
        self.change_tributation_for_ncm = ChangeTributationForNCM(self.db_connection, self.log, self.open_modal)
        self.base_clean = BaseClean(self.db_connection, self.log)
        self.base_create = BaseCreate(self.db_connection, self.log)
        self.reg_digisat_clean = RegDigisatClean(self.db_connection, self.log)
        self.database_validator = DatabaseValidator(self.db_connection, self.log)


        thread = threading.Thread(target=self.check_database_connection)
        thread.start()

        buttons_frame = ctk.CTkFrame(self.app, border_width=0, corner_radius=8, bg_color='#0f1623', fg_color='#0f1623')
        buttons_frame.grid(row=1, column=0, columnspan=6)
        buttons_frame.grid_rowconfigure(0, weight=1)
        buttons_frame.grid_columnconfigure(0, weight=1)
        buttons_frame.grid_columnconfigure(1, weight=1)
        buttons_frame.place(relx=0.5, rely=0.55, anchor='center')

        button_change_tributation_for_ncm = (
            ButtonCreator(buttons_frame, "Alterar a tributação de Itens por NCM",
                          lambda: self.open_modal("Digite os NCM's e o ID da Tributação",
                                            self.run_change_tributation_for_ncm,
                                            operation_type="run_change_tributation_for_ncm",
                                            show_second_entry=True)))
        button_change_tributation_for_ncm.grid(row=0,column=0)

        button_reg_digisat_clean = ButtonCreator(buttons_frame,"Elimina os registro do Digisat do Windows",
                                                 self.reg_digisat_clean.run_thread_reg_digisat_clean)
        button_reg_digisat_clean.grid(row=0,column=1)

        button_inactive_products = ButtonCreator(buttons_frame, "Inativar produtos Zerados ou Negativos",
                                                 self.inactive_products.run_thread_inactive_products)
        button_inactive_products.grid(row=1, column=0)

        button_movimentations_clean = ButtonCreator(buttons_frame,"Limpa movimentações da Base",
                                                    self.movimentations_clean.run_thread_movimentations_clean)
        button_movimentations_clean.grid(row=1, column=1)

        button_mei_able = ButtonCreator(buttons_frame, 'Permitir o ajuste de Estoque',
                                        self.mei_able.run_thread_mei_able)
        button_mei_able.grid(row=2, column=0)

        button_base_create = ButtonCreator(buttons_frame, "Criar base nova zerada",
                                           self.base_create.run_thread_base_creator)
        button_base_create.grid(row=2, column=1)

        button_find_ids = ButtonCreator(
            buttons_frame, "Localiza ID's no banco",
            lambda: self.open_modal("Digite o ID a buscar", self.run_find_ids,
                                            operation_type="run_find_ids",
                                            show_second_entry=False),

        )
        button_find_ids.grid(row=3, column=0)

        button_base_clean = ButtonCreator(
            buttons_frame, "Zera a base atual",
            self.base_clean.run_thread_base_clean
        )
        button_base_clean.grid(row=3, column=1)

        button_void = ButtonCreator(
            buttons_frame, "Botão que ainda não faz nada!",
            self.click_void()
        )
        button_void.grid(row=4, column=0)

        cancel_button = ButtonCreator(
            self.app, "Cancelar Operação", self.cancel_operation, fg_color='#031229'
        )
        cancel_button.grid(row=2, column=0)
        # cancel_button.grid_rowconfigure(0, weight=1)
        cancel_button.place(relx=0.5, rely=0.81, anchor='center')

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

    def open_modal(self, title, callback, operation_type=None, show_second_entry=False):
        tributation_ids = []
        if operation_type == "run_change_tributation_for_ncm":
            tributation_ids = self.change_tributation_for_ncm.fetch_tributation_ids()
        modal = Modal(title, callback, operation_type=operation_type, show_second_entry=show_second_entry,
                      tributation_ids=tributation_ids)

    def run_find_ids(self, search_id):
        self.find_ids.run_thread_find_ids(search_id)

    def run_change_tributation_for_ncm(self, ncms, tributation_id):
        self.change_tributation_for_ncm.run_thread_change_tributation_for_ncm(ncms, tributation_id)

    def check_database_connection(self):
        try:
            if not self.database_validator.connect_to_db():
                return
        except Exception as e:
            self.log.insert(ctk.END, str(e) + "\n")
            return

    def click_void(self):
        pass

    def run(self):
        self.app.mainloop()


if __name__ == "__main__":
    user_interface = UserInterface()
    user_interface.run()

