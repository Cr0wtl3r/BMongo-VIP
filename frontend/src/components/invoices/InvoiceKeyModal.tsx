import { useState } from 'react';
import { GetInvoiceByKey, ChangeInvoiceKey } from '../../../wailsjs/go/main/App';

interface InvoiceKeyModalProps {
  show: boolean;
  onClose: () => void;
  availableTypes: string[];
  initialType?: string;
  showSuccess: (msg: string) => void;
  showError: (msg: string) => void;
}

export function InvoiceKeyModal({ show, onClose, availableTypes, initialType = 'NFe', showSuccess, showError }: InvoiceKeyModalProps) {
  const [invoiceType, setInvoiceType] = useState(initialType);
  const [oldKey, setOldKey] = useState('');
  const [newKey, setNewKey] = useState('');
  const [portalResult, setPortalResult] = useState<any>(null);

  if (!show) return null;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" style={{ minWidth: '500px' }} onClick={(e) => e.stopPropagation()}>
        <h3>🔑 Alterar Chave de Acesso</h3>
        <p className="modal-desc">Corrige a chave de acesso (ID) de um documento fiscal.</p>
        
        <div className="form-group">
          <label>Tipo de Documento:</label>
          <select value={invoiceType} onChange={(e) => setInvoiceType(e.target.value)}>
            {availableTypes.map(t => <option key={t} value={t}>{t}</option>)}
          </select>
        </div>

        <div className="form-group">
          <label>Chave Atual:</label>
          <div className="file-picker-row">
             <input 
               className="form-input"
               value={oldKey}
               onChange={(e) => setOldKey(e.target.value)}
               placeholder="Chave antiga..."
             />
             <button className="file-picker-btn" onClick={() => {
                if (!oldKey) return showError('Digite a chave para buscar!');
                GetInvoiceByKey(invoiceType, oldKey)
                  .then((inv: any) => {
                      if (inv) {
                        setPortalResult(inv);
                        showSuccess('Documento encontrado!');
                      } else {
                         showError('Documento não encontrado.');
                         setPortalResult(null);
                      }
                  })
                  .catch((err: any) => {
                    console.error(err);
                    showError('Erro ao buscar: ' + err);
                    setPortalResult(null);
                  });
             }}>🔍</button>
          </div>
        </div>

        {portalResult && (
          <div className="invoice-preview-card">
              <div className="preview-item">
                 <span className="preview-label">Número / Série</span>
                 <span className="preview-value highlight">{portalResult.numero} <span style={{color: '#64748b'}}>/</span> {portalResult.serie}</span>
              </div>
              <div className="preview-item">
                 <span className="preview-label">Data Emissão</span>
                 <span className="preview-value">{portalResult.data}</span>
              </div>
              <div className="preview-item">
                 <span className="preview-label">Valor Total</span>
                 <span className="preview-value">R$ {portalResult.valor?.toFixed(2)}</span>
              </div>
              <div className="preview-item">
                 <span className="preview-label">Situação Atual</span>
                 <div><span className="preview-value status">{portalResult.situacao || 'Não identificado'}</span></div>
              </div>
              <div className="preview-item" style={{ gridColumn: 'span 2' }}>
                 <span className="preview-label">Cliente / Destinatário</span>
                 <span className="preview-value">{portalResult.cliente || 'Consumidor Final'}</span>
              </div>
          </div>
        )}

        <div className="form-group">
          <label>Nova Chave:</label>
          <input 
            value={newKey}
            onChange={(e) => setNewKey(e.target.value)}
            placeholder="Nova chave..."
          />
        </div>

        <div className="modal-actions">
          <button onClick={() => {
            onClose();
            setPortalResult(null);
          }}>Cancelar</button>
          <button className="primary" onClick={async () => {
            if (!oldKey || !newKey) return showError('Preencha todos os campos!');
            try {
              const count = await ChangeInvoiceKey(invoiceType, oldKey, newKey);
              onClose();
              showSuccess(`✅ Atualizado(s) ${count} documento(s)!`);
              setOldKey(''); setNewKey('');
              setPortalResult(null);
            } catch(err: any) {
              console.error(err);
              showError('Erro ao atualizar: ' + err);
            }
          }}>Salvar</button>
        </div>
      </div>
    </div>
  );
}
