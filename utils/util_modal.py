from tkinter import StringVar, Tk
import customtkinter as ctk
import sys
import os

class Modal:
    def __init__(self, title, callback, operation_type, show_second_entry=False):
        if getattr(sys, 'frozen', False):
            ico_path = Modal.resource_path('BMongo-VIP\\src\\logo.ico')
        else:
            ico_path = Modal.resource_path('src\\logo.ico')

        self.modal = ctk.CTk()
        self.modal.wm_iconbitmap(ico_path)
        self.modal.title(title)
        self.modal.geometry("300x150")
        self.callback = callback
        self.show_second_entry = show_second_entry

        if operation_type == "run_change_tributation_for_ncm":
            placeholder_ncms = "Digite os NCM's a serem alterados"
            placeholder_tributation_id = "Digite o ID da tributação a ser aplicada"
        elif operation_type == "run_find_ids":
            placeholder_ncms = "Digite o ID a buscar"
            placeholder_tributation_id = ""
        else:
            placeholder_ncms = "Digite os NCM's a serem alterados"
            placeholder_tributation_id = "Digite o ID da tributação a ser aplicada"

        self.ncms_var = StringVar()
        self.ncms_var.set(placeholder_ncms)

        self.label_ncms = ctk.CTkLabel(self.modal, text=placeholder_ncms)
        self.label_ncms.pack(pady=10)
        self.entry_ncms = ctk.CTkTextbox(self.modal, width=200, height=1, wrap="word", padx=5, pady=5,
                                         font=("Arial", 12),
                                         activate_scrollbars=False)
        self.entry_ncms.pack(pady=10)
        self.entry_ncms.bind("<FocusIn>", lambda event: self.label_ncms.pack_forget() if self.ncms_var.get() == "" else None)
        self.entry_ncms.bind("<FocusOut>", lambda event: self.label_ncms.pack() if self.ncms_var.get() == "" else None)

        if self.show_second_entry and operation_type == "run_change_tributation_for_ncm":
            self.modal.geometry("400x300")
            self.label_tributation_id = ctk.CTkLabel(self.modal, text=placeholder_tributation_id)
            self.label_tributation_id.pack(pady=10)
            self.entry_tributation_id = ctk.CTkTextbox(self.modal, width=200, height=1, wrap="word", padx=5, pady=5,
                                                       font=("Arial", 12), activate_scrollbars=False)
            self.entry_tributation_id.pack(pady=10)
            self.entry_tributation_id.bind("<FocusIn>", lambda event: self.label_tributation_id.pack_forget() if self.ncms_var.get() == "" else None)
            self.entry_tributation_id.bind("<FocusOut>", lambda event: self.label_tributation_id.pack() if self.ncms_var.get() == "" else None)

        button_action = ctk.CTkButton(
            self.modal, text="Executar",
            command=self.run_callback, fg_color='#f6882d', hover_color='#c86e24',
            text_color='white',
            border_color='#123f8c')
        button_action.pack(pady=10)

        self.modal.mainloop()

    def run_callback(self):
        ncms_string = self.entry_ncms.get("1.0", "end-1c")
        if ncms_string == "":
            ncms_string = "Digite os NCM's a serem alterados"
        if self.show_second_entry and self.callback.__name__ == 'run_change_tributation_for_ncm':
            tributation_id = self.entry_tributation_id.get("1.0", "end-1c")
            if tributation_id == "":
                tributation_id = "Digite o ID da tributação a ser aplicada"
            ncms_list = ncms_string.split(',')
            self.callback(ncms_list, tributation_id)
        else:
            self.callback(ncms_string)

    @classmethod
    def resource_path(cls, relative_path):
        try:
            base_path = sys._MEIPASS
        except Exception:
            base_path = os.path.abspath(".")
        return os.path.join(base_path, relative_path)