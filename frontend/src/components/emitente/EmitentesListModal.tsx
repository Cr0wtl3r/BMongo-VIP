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
