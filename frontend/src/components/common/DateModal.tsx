import { useState } from 'react';
import { CleanDatabaseByDate } from '../../../wailsjs/go/main/App';

interface DateModalProps {
  show: boolean;
  onClose: () => void;
  showSuccess: (msg: string) => void;
  showError: (msg: string) => void;
}

export function DateModal({ show, onClose, showSuccess, showError }: DateModalProps) {
  const [dateInput, setDateInput] = useState('');

  if (!show) return null;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h3>🗓️ Limpar por Data</h3>
        <p>Remove movimentações anteriores à data selecionada.</p>
        
        <div className="form-group">
          <label>Data de Corte:</label>
          <input 
            type="date" 
            value={dateInput}
            onChange={(e) => setDateInput(e.target.value)}
          />
        </div>

        <div className="modal-actions">
          <button onClick={onClose}>Cancelar</button>
          <button className="primary" onClick={async () => {
            if (!dateInput) return showError('Selecione uma data!');
            if (window.confirm(`Confirma limpeza de dados anteriores a ${dateInput}?`)) {
               try {
                 onClose();
                 await CleanDatabaseByDate(dateInput);
                 showSuccess('✅ Limpeza concluída!');
               } catch(err: any) {
                 showError(err);
               }
            }
          }}>Limpar</button>
        </div>
      </div>
    </div>
  );
}
