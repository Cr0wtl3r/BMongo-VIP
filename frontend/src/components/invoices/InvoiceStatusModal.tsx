import { useState } from 'react';
import { GetInvoiceByNumber, ChangeInvoiceStatus } from '../../../wailsjs/go/main/App';

interface InvoiceStatusModalProps {
  show: boolean;
  onClose: () => void;
  availableTypes: string[];
  availableStatuses: string[];
  initialType?: string;
  initialStatus?: string;
  showSuccess: (msg: string) => void;
  showError: (msg: string) => void;
}

export function InvoiceStatusModal({ 
  show, onClose, availableTypes, availableStatuses, 
  initialType = 'NFe', initialStatus = 'Concluído',
  showSuccess, showError 
}: InvoiceStatusModalProps) {
  const [invoiceType, setInvoiceType] = useState(initialType);
  const [invoiceStatus, setInvoiceStatus] = useState(initialStatus);
  const [invoiceSeries, setInvoiceSeries] = useState('');
  const [invoiceNumber, setInvoiceNumber] = useState('');
  const [portalResult, setPortalResult] = useState<any>(null);

  if (!show) return null;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" style={{ minWidth: '600px' }} onClick={(e) => e.stopPropagation()}>
        <h3>📊 Alterar Situação</h3>
        <p className="modal-desc">Define a situação manualmente e adiciona histórico.</p>
        <div className="filter-grid">
           <div className="filter-row">
              <label>Tipo:</label>
              <select value={invoiceType} onChange={(e) => setInvoiceType(e.target.value)}>
                {availableTypes.map(t => <option key={t} value={t}>{t}</option>)}
              </select>
           </div>
           <div className="filter-row">
              <label>Situação:</label>
              <select value={invoiceStatus} onChange={(e) => setInvoiceStatus(e.target.value)}>
                {availableStatuses.map(s => <option key={s} value={s}>{s}</option>)}
              </select>
           </div>
        </div>
        <div className="filter-grid" style={{ alignItems: 'flex-end' }}>
           <div className="filter-row">
              <label>Série:</label>
              <input 
                value={invoiceSeries}
                onChange={(e) => setInvoiceSeries(e.target.value)}
                placeholder="Ex: 1"
                disabled={invoiceType === 'DAV' || invoiceType === 'DAV-OS'}
                title={invoiceType.includes('DAV') ? 'DAV não usa série' : ''}
              />
           </div>
           <div className="filter-row">
              <label>Número:</label>
              <div className="file-picker-row">
                <input 
                  className="form-input"
                  value={invoiceNumber}
                  onChange={(e) => setInvoiceNumber(e.target.value)}
                  placeholder="Ex: 12345"
                />
                <button className="file-picker-btn" onClick={() => {
                    if (!invoiceNumber) return showError('Informe o número para buscar!');

                    
                    GetInvoiceByNumber(invoiceType, invoiceSeries, invoiceNumber)
                       .then((inv: any) => {
                          if (inv) {
                            setPortalResult(inv); 
                            showSuccess('Documento encontrado!');
                          } else {
                             setPortalResult(null);
                             showError('Documento não encontrado.');
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
        </div>

        {}
         {portalResult && (
          <div className="invoice-preview-card">
              <div className="preview-item">
                 <span className="preview-label">Chave Acesso</span>
                 <span className="preview-value" style={{fontSize: '0.75rem', wordBreak: 'break-all'}}>{portalResult.chave || '---'}</span>
              </div>
              <div className="preview-item">
                 <span className="preview-label">Valor</span>
                 <span className="preview-value highlight">R$ {portalResult.valor?.toFixed(2)}</span>
              </div>
              <div className="preview-item">
                 <span className="preview-label">Data Emissão</span>
                 <span className="preview-value">{portalResult.data}</span>
              </div>
              <div className="preview-item">
                 <span className="preview-label">Situação Atual</span>
                 <div><span className="preview-value status">{portalResult.situacao || 'Não identificado'}</span></div>
              </div>
              <div className="preview-item" style={{ gridColumn: 'span 2' }}>
                 <span className="preview-label">Cliente</span>
                 <span className="preview-value">{portalResult.cliente || 'Consumidor Final'}</span>
              </div>
          </div>
        )}
        
        <div className="modal-actions">
          <button onClick={() => {
             onClose();
             setPortalResult(null);
          }}>Cancelar</button>
          <button className="primary" onClick={async () => {
            if (!invoiceNumber) return showError('Informe o número!');

            
            try {
              await ChangeInvoiceStatus(invoiceType, invoiceSeries, invoiceNumber, invoiceStatus);
              onClose();
              showSuccess(`✅ Situação alterada para ${invoiceStatus}!`);
              setInvoiceSeries(''); setInvoiceNumber('');
              setPortalResult(null);
            } catch(err: any) {
              console.error(err);
              showError('Erro: ' + err);
            }
          }}>Salvar</button>
        </div>
      </div>
    </div>
  );
}
