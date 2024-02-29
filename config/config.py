import threading

running_operations = True
cancel_event = threading.Event()
running_operations_lock = threading.Lock()