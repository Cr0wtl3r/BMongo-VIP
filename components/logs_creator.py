import customtkinter as ctk

class LogCreator:
    def __init__(self, parent, width, heigth, fg_color='#f6882d'):
        self.button = ctk.CTkButton(
            parent, text=text,
            fg_color=fg_color,
            hover_color='#c86e24',
            text_color='white',
            border_color='#123f8c',
            font=('arial', 15, 'bold'),
            width=350,
            command=command
        )

    def grid(self, row, column):
        self.button.grid(row=row, column=column, ipadx=10, ipady=10, pady=10, padx=10)

    def place(self, relx, rely, anchor):
        self.button.place(relx=relx, rely=rely, anchor=anchor)

    def get_button(self):
        return self.button
