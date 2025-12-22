import { useState, useEffect } from 'react';
import { GetTributations, GetFederalTributations, ChangeTributationByNCM, ChangeFederalTributationByNCM } from '../../../wailsjs/go/main/App';

interface NcmModalProps {
  show: boolean;
  onClose: () => void;

}

export function NcmModal({ show, onClose }: NcmModalProps) {
  const [ncmInput, setNcmInput] = useState('');
  const [selectedTrib, setSelectedTrib] = useState('');
  const [tributations, setTributations] = useState<Record<string, any>[]>([]);
  const [ncmScope, setNcmScope] = useState<'estadual' | 'federal'>('estadual');

  useEffect(() => {
    if (!show) return;
    const fetchTribs = async () => {
        try {
            const tribs = ncmScope === 'estadual' 
              ? await GetTributations() 
              : await GetFederalTributations();
            setTributations(tribs || []);
            setSelectedTrib('');
        } catch(err) {
            console.error(err);
        }
    }
    fetchTribs();
  }, [ncmScope, show]);

  const handleChangeTributation = async () => {
    if (!ncmInput.trim() || !selectedTrib) return;
    const ncms = ncmInput.split(',').map((n: string) => n.trim()).filter((n: string) => n);
    onClose();
    
    try {
      if (ncmScope === 'estadual') {
          await ChangeTributationByNCM(ncms, selectedTrib);
      } else {
          await ChangeFederalTributationByNCM(ncms, selectedTrib);
      }
      setNcmInput('');
      setSelectedTrib('');
    } catch (err) {
      console.error(err);
    }
  };

  if (!show) return null;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h3>🏷️ Alterar Tributação</h3>
        
        <div className="modal-tabs">
          <button 
            className={`tab ${ncmScope === 'estadual' ? 'active' : ''}`}
            onClick={() => setNcmScope('estadual')}
          >
            Estadual (ICMS)
          </button>
          <button 
            className={`tab ${ncmScope === 'federal' ? 'active' : ''}`}
            onClick={() => setNcmScope('federal')}
          >
            Federal (PIS/COFINS)
          </button>
        </div>

        <div className="form-group">
          <label>NCMs (separados por vírgula):</label>
          <textarea
            value={ncmInput}
            onChange={(e) => setNcmInput(e.target.value)}
            placeholder="Ex: 8471, 8473, 9504"
            rows={3}
          />
        </div>
        <div className="form-group">
          <label>Nova Tributação:</label>
          <select 
            value={selectedTrib} 
            onChange={(e) => setSelectedTrib(e.target.value)}
          >
            <option value="">Selecione...</option>
            {tributations.map((t: any) => (
              <option key={t.id} value={t.id}>{t.Descricao}</option>
            ))}
          </select>
        </div>
        <div className="modal-actions">
          <button onClick={onClose}>Cancelar</button>
          <button className="primary" onClick={handleChangeTributation}>Executar</button>
        </div>
      </div>
    </div>
  );
}
