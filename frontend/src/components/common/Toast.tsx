interface ToastProps {
  show: boolean;
  message: string;
  type: 'success' | 'error';
  onClose: () => void;
}

export function Toast({ show, message, type, onClose }: ToastProps) {
  if (!show) return null;

  return (
    <div className={`toast ${type}`} onClick={onClose}>
      {message}
    </div>
  );
}
