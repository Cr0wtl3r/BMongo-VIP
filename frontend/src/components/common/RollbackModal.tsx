import { UndoOperation, GetUndoableOperations } from '../../../wailsjs/go/main/App';

interface RollbackModalProps {
  show: boolean;
  onClose: () => void;
  undoableOps: any[];
  setUndoableOps: (ops: any[]) => void;
}

export function RollbackModal({ show, onClose, undoableOps, setUndoableOps }: RollbackModalProps) {
  
  const handleUndo = async (opId: string) => {
    try {
      await UndoOperation(opId);
      const ops = await GetUndoableOperations();
      setUndoableOps(ops || []);
      if (ops.length === 0) {
        onClose();
      }
    } catch(err) {
      console.error(err);
    }
  };

  if (!show) return null;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h3>🆘 Deu Merda? Reverta Aqui!</h3>
        {undoableOps.length === 0 ? (
          <p className="modal-desc">Nenhuma operação reversível no histórico.</p>
        ) : (
          <div className="undo-list">
            {undoableOps.map((op: any) => (
              <div key={op.id} className="undo-item">
                <div className="undo-info">
                  <span className="undo-label">{op.label}</span>
                  <span className="undo-time">{op.timestamp}</span>
                </div>
                <button className="undo-btn" onClick={() => handleUndo(op.id)}>
                  Reverter
                </button>
              </div>
            ))}
          </div>
        )}
        <div className="modal-actions">
          <button onClick={onClose}>Fechar</button>
          <span></span>
        </div>
      </div>
    </div>
  );
}
