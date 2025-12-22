import { useState } from 'react';
import { FindObjectIdInDatabase } from '../../../wailsjs/go/main/App';

interface SearchModalProps {
  show: boolean;
  onClose: () => void;
}

export function SearchModal({ show, onClose }: SearchModalProps) {
  const [searchId, setSearchId] = useState('');

  const handleFindId = async () => {
    if (!searchId.trim()) return;
    onClose();
    try {
      await FindObjectIdInDatabase(searchId);
      setSearchId('');
    } catch (err) {
      console.error(err);
    }
  };

  if (!show) return null;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h3>🔍 Buscar ObjectID</h3>
        <p className="modal-desc">Digite o ObjectID para buscar em todas as coleções.</p>
        <div className="form-group">
          <input
            type="text"
            className="form-input"
            placeholder="Ex: 507f1f77bcf86cd799439011"
            value={searchId}
            onChange={(e) => setSearchId(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleFindId()}
            autoFocus
          />
        </div>
        <div className="modal-actions">
          <button onClick={onClose}>Cancelar</button>
          <button className="primary" onClick={handleFindId}>Buscar</button>
        </div>
      </div>
    </div>
  );
}
