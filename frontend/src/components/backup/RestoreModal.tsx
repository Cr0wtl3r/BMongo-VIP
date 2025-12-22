import { useState } from 'react';
import { SelectDirectory, SelectBackupFile, RestoreDatabase } from '../../../wailsjs/go/main/App';

interface RestoreModalProps {
  show: boolean;
  onClose: () => void;
  showSuccess: (msg: string) => void;
  showError: (msg: string) => void;
}

export function RestoreModal({ show, onClose, showSuccess, showError }: RestoreModalProps) {
  const [restorePath, setRestorePath] = useState('');
  const [restoreDropExisting, setRestoreDropExisting] = useState(false);
  const [sourceType, setSourceType] = useState<'folder' | 'zip'>('folder');

  if (!show) return null;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal modal-wide" onClick={e => e.stopPropagation()}>
        <h3>📥 Restaurar Backup</h3>
        <p className="modal-desc">Recupere dados de um backup anterior. Escolha a fonte e opções.</p>

        {}
        <div className="modal-tabs">
          <button 
            className={`tab ${sourceType === 'folder' ? 'active' : ''}`}
            onClick={() => setSourceType('folder')}
          >
            📁 Pasta (Diretório)
          </button>
          <button 
            className={`tab ${sourceType === 'zip' ? 'active' : ''}`}
            onClick={() => setSourceType('zip')}
          >
            📦 Arquivo ZIP
          </button>
        </div>

        {}
        <div className="form-group">
          <label>Caminho de Origem:</label>
          <div className="file-picker-row">
            <input
              type="text"
              value={restorePath}
              onChange={e => setRestorePath(e.target.value)}
              placeholder={sourceType === 'folder' ? "C:/Backups/2023..." : "C:/Backups/backup.zip..."}
              className="form-input"
            />
            <button className="file-picker-btn" onClick={() => {
              const picker = sourceType === 'folder' ? SelectDirectory : SelectBackupFile;
              const msg = sourceType === 'folder' ? "Selecione pasta do backup" : "Selecione arquivo ZIP do backup";
              picker(msg).then((path: string) => {
                if (path) setRestorePath(path);
              });
            }}>Selecionar</button>
          </div>
        </div>

        {}
        <div className="form-group">
          <label className="checkbox-row">
            <input
              type="checkbox"
              checked={restoreDropExisting}
              onChange={e => setRestoreDropExisting(e.target.checked)}
            />
            <span>⚠️ Sobrescrever dados existentes (--drop)</span>
          </label>
          <p className="modal-desc" style={{ marginTop: '0.5rem', fontSize: '0.8rem', color: '#f59e0b' }}>
             Isso apaga as coleções atuais antes de restaurar para evitar duplicações. Recomenda-se manter ativado para restauração completa.
          </p>
        </div>

        <div className="modal-actions">
          <button onClick={onClose}>Cancelar</button>
          <button className="primary" onClick={async () => {
             if (!restorePath) return showError('Selecione uma ' + (sourceType === 'folder' ? 'pasta' : 'arquivo ZIP') + '!');
             try {
               onClose();
               await RestoreDatabase(restorePath, restoreDropExisting);
               showSuccess('✅ Restauração concluída com sucesso!');
             } catch (err: any) {
               showError(err?.message || 'Erro na restauração');
             }
          }}>Restaurar</button>
        </div>
      </div>
    </div>
  );
}
