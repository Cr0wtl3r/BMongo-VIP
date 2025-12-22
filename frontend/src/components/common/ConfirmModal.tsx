interface ConfirmModalProps {
  show: boolean;
  title: string;
  desc: string;
  onConfirm: () => void;
  onCancel: () => void;
}

export function ConfirmModal({ show, title, desc, onConfirm, onCancel }: ConfirmModalProps) {
  if (!show) return null;

  return (
    <div className="modal-overlay" onClick={onCancel}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h3>Confirmar Ação</h3>
        <p className="modal-title">{title}</p>
        <p className="modal-desc">{desc}</p>
        <div className="modal-actions">
          <button onClick={onCancel}>Cancelar</button>
          <button className="primary" onClick={onConfirm}>Confirmar</button>
        </div>
      </div>
    </div>
  );
}
