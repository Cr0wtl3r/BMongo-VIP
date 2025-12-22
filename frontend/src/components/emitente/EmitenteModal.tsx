import { useState } from 'react';
import { SelectInfoDatFile, UpdateEmitenteFromFile } from '../../../wailsjs/go/main/App';

interface EmitenteModalProps {
  show: boolean;
  onClose: () => void;
  showSuccess: (msg: string) => void;
  showError: (msg: string) => void;
}

export function EmitenteModal({ show, onClose, showSuccess, showError }: EmitenteModalProps) {
  const [emitenteFilePath, setEmitenteFilePath] = useState('');

  if (!show) return null;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h3>👤 Atualizar Emitente</h3>
        <p>Selecione o arquivo info.dat para atualizar os dados.</p>
        
        <div className="form-group">
          <div className="file-picker-row">
            <input 
              readOnly 
              value={emitenteFilePath}
              placeholder="Caminho do info.dat..."
            />
            <button className="file-picker-btn" onClick={() => {
              SelectInfoDatFile().then((path: string) => {
                if(path) setEmitenteFilePath(path);
              });
            }}>📁</button>
          </div>
        </div>

        <div className="modal-actions">
          <button onClick={onClose}>Cancelar</button>
          <button className="primary" onClick={async () => {
             if(!emitenteFilePath) return showError('Selecione o arquivo!');
             try {
               await UpdateEmitenteFromFile(emitenteFilePath);
               showSuccess('✅ Emitente atualizado com sucesso!');
               onClose();
             } catch(err: any) {
               showError('Erro: ' + err);
             }
          }}>Atualizar</button>
        </div>
      </div>
    </div>
  );
}
