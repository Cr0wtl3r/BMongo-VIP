## Arquivo util_modal.py:
from tkinter import StringVar, Tk, messagebox
import customtkinter as ctk
import sys
import os


class Modal:
    def __init__(self, title, callback, operation_type, show_second_entry=False, tributation_ids=None):
        if getattr(sys, 'frozen', False):
            ico_path = Modal.resource_path('BMongo-VIP\\src\\logo.ico')
        else:
            ico_path = Modal.resource_path('src\\logo.ico')

        # Configurações iniciais do modal
        self.modal = ctk.CTk()
        self.modal.wm_iconbitmap(ico_path)
        self.modal.title(title)
        self.modal.geometry("300x150")
        self.callback = callback
        self.show_second_entry = show_second_entry
        self.tributation_ids = tributation_ids or []
        self.operation_type = operation_type

        # Define os placeholders para cada operação
        if operation_type == "run_change_tributation_for_ncm":
            placeholder_ncms = "Digite os NCM's a serem alterados"
            placeholder_tributation_id = "Selecione o ID da tributação"
        elif operation_type == "run_find_ids":
            placeholder_ncms = "Digite o ID a buscar"
            placeholder_tributation_id = ""
        else:
            placeholder_ncms = "Digite os NCM's a serem alterados"
            placeholder_tributation_id = "Digite o ID da tributação a ser aplicada"

        # Entrada para NCMs
        self.ncms_var = StringVar()
        self.ncms_var.set(placeholder_ncms)

        self.label_ncms = ctk.CTkLabel(self.modal, text=placeholder_ncms)
        self.label_ncms.pack(pady=10)
        self.entry_ncms = ctk.CTkTextbox(
            self.modal, width=200, height=1, wrap="word", padx=5, pady=5,
            font=("Arial", 12), activate_scrollbars=False
        )
        self.entry_ncms.pack(pady=10)
        self.entry_ncms.bind("<FocusIn>", self.clear_placeholder_ncms)
        self.entry_ncms.bind("<FocusOut>", self.restore_placeholder_ncms)

        # Exibe um Combobox para seleção de ID de tributação, caso aplicável
        if self.show_second_entry and operation_type == "run_change_tributation_for_ncm":
            self.modal.geometry("400x300")
            self.tributation_id_var = StringVar()
            self.label_tributation_id = ctk.CTkLabel(self.modal, text=placeholder_tributation_id)
            self.label_tributation_id.pack(pady=10)

            # Preparando as descrições e IDs para o Combobox
            self.tributation_descriptions = [item["Descricao"] for item in self.tributation_ids]
            self.tributation_descriptions.insert(0, "Selecione uma tributação")

            self.combobox_tributation_id = ctk.CTkComboBox(
                self.modal,
                values=self.tributation_descriptions,
                command=self.on_combobox_select  # Adicionando callback para seleção
            )
            self.combobox_tributation_id.set("Selecione uma tributação")
            self.combobox_tributation_id.pack(pady=10)

        # Botão de ação
        button_action = ctk.CTkButton(
            self.modal, text="Executar",
            command=self.validate_and_run_callback, fg_color='#f6882d', hover_color='#c86e24',
            text_color='white', border_color='#123f8c'
        )
        button_action.pack(pady=10)

        self.modal.mainloop()

    def on_combobox_select(self, choice):
        """Callback para quando uma opção é selecionada no Combobox"""
        self.selected_tributation = choice

    def clear_placeholder_ncms(self, event):
        if self.ncms_var.get() == "Digite os NCM's a serem alterados":
            self.entry_ncms.delete("1.0", "end")

    def restore_placeholder_ncms(self, event):
        if self.entry_ncms.get("1.0", "end-1c").strip() == "":
            self.entry_ncms.insert("1.0", "Digite os NCM's a serem alterados")

    def validate_and_run_callback(self):
        """Valida os campos antes de executar o callback"""
        # Obtém os valores dos campos
        ncms_string = self.entry_ncms.get("1.0", "end-1c").strip()

        # Valida entrada de NCMs
        if not ncms_string or ncms_string == "Digite os NCM's a serem alterados":
            messagebox.showerror("Erro", "Por favor, insira os NCMs a serem alterados.")
            return

        # Se for operação de tributação, valida a seleção do ComboBox
        if self.show_second_entry and self.operation_type == "run_change_tributation_for_ncm":
            selected_description = self.combobox_tributation_id.get()

            if not selected_description or selected_description == "Selecione uma tributação":
                messagebox.showerror("Erro", "Por favor, selecione uma tributação.")
                return

            # Encontra o ID correspondente à descrição selecionada
            selected_tributation = next(
                (item for item in self.tributation_ids if item["Descricao"] == selected_description),
                None
            )

            if not selected_tributation:
                messagebox.showerror("Erro", "Tributação selecionada não encontrada.")
                return

            tributation_id = selected_tributation["id"]

            # Executa o callback com os parâmetros validados
            ncms_list = [ncm.strip() for ncm in ncms_string.split(',') if ncm.strip()]
            self.callback(ncms_list, tributation_id)
        else:
            # Para outras operações, executa normalmente
            self.callback(ncms_string)

        # Fecha o modal após execução bem-sucedida
        self.modal.destroy()

    @staticmethod
    def resource_path(relative_path):
        try:
            base_path = sys._MEIPASS
        except Exception:
            base_path = os.path.abspath(".")
        return os.path.join(base_path, relative_path)