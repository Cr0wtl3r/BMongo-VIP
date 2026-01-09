import { useState } from 'react';
import { SelectDirectory, BackupDatabase } from '../../../wailsjs/go/main/App';

interface BackupModalProps {
  show: boolean;
  onClose: () => void;
  showSuccess: (msg: string) => void;
  showError: (msg: string) => void;
}

export function BackupModal({ show, onClose, showSuccess, showError }: BackupModalProps) {
  const [backupDir, setBackupDir] = useState('');

  if (!show) return null;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal modal-wide" onClick={e => e.stopPropagation()}>
        <h3>💾 Fazer Backup</h3>
        <p>Cria um backup do banco de dados usando mongodump.</p>

        <div className="form-field">
          <label>Pasta de destino:</label>
          <div className="file-picker-row">
            <input
              type="text"
              value={backupDir}
              onChange={e => setBackupDir(e.target.value)}
              placeholder="Clique em Selecionar..."
              className="form-input"
            />
            <button className="file-picker-btn" onClick={() => {
              SelectDirectory("Selecione pasta para backup").then((path: string) => {
                if (path) setBackupDir(path);
              });
            }}>Selecionar</button>
          </div>
        </div>

        <div className="modal-actions">
          <button onClick={onClose}>Cancelar</button>
          <button className="primary" onClick={async () => {
            if (!backupDir) return showError('Selecione uma pasta!');
            try {
              onClose();
              const result = await BackupDatabase(backupDir);
              showSuccess(`✅ Backup criado: ${(result as any)?.path || 'OK'}`);
            } catch (err: any) {
              showError(err?.message || 'Erro no backup');
            }
          }}>Fazer Backup</button>
        </div>
      </div>
    </div>
  );
}
