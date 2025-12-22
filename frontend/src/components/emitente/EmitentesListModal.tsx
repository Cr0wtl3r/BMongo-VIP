import { useState } from 'react';

interface EmitentesListModalProps {
  show: boolean;
  onClose: () => void;
  emitentesList: any[];
  onDelete: (id: string) => void;
}

export function EmitentesListModal({ show, onClose, emitentesList, onDelete }: EmitentesListModalProps) {
  const [selectedEmitenteId, setSelectedEmitenteId] = useState('');

  if (!show) return null;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h3>🗑️ Apagar Emitente</h3>
        <p className="modal-danger-text">⚠️ CUIDADO: Esta ação removerá o emitente e seus dados associados!</p>
        
        {emitentesList.length === 0 ? (
          <p>Nenhum emitente encontrado.</p>
        ) : (
          <div className="emitentes-list">
            {emitentesList.map((e: any) => (
              <div 
                key={e.id} 
                className={`emitente-item ${selectedEmitenteId === e.id ? 'selected' : ''}`}
                onClick={() => setSelectedEmitenteId(e.id)}
              >
                <div className="emitente-info">
                  <span className="emitente-nome">{e.nome}</span>
                  <span className="emitente-cnpj">{e.cnpj}</span>
                </div>
              </div>
            ))}
          </div>
        )}

        <div className="modal-actions">
          <button onClick={onClose}>Cancelar</button>
          <button 
            className="primary danger" 
            disabled={!selectedEmitenteId}
            onClick={() => {
                onDelete(selectedEmitenteId);
                // onClose(); // Let parent handle closing if needed, or close here?
                // Probably better to let parent confirm then close. 
                // But typically UI pattern is: click delete -> confirmation modal appears -> success -> list closes.
                // We'll simplisticly close this modal here if the confirm handles it?
                // No, if user cancels confirm, this modal should stay open.
                // But ConfirmModal is global in App.tsx. App.tsx logic usually closes active modals or handles success.
                // We will keep this open until success? 
                // Let's assume onConfirm will trigger success toast.
                // We can close this modal immediately or let the user close it.
                // Let's close it to avoid clutter, as the action is handed off to the global confirmation.
                onClose();
            }}
          >
            🗑️ Apagar
          </button>
        </div>
      </div>
    </div>
  );
}
