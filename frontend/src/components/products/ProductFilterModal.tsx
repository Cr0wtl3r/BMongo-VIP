import { useState } from 'react';
import { FilterProducts, BulkActivateProducts } from '../../../wailsjs/go/main/App';
import { PriceAdjustmentModal } from './PriceAdjustmentModal';
import { NCMChangeModal } from './NCMChangeModal';

interface ProductFilterModalProps {
  show: boolean;
  onClose: () => void;
  totalInDatabase: number;
}

export function ProductFilterModal({ show, onClose, totalInDatabase }: ProductFilterModalProps) {
  const [filterResults, setFilterResults] = useState<any[]>([]);
  const [totalProducts, setTotalProducts] = useState(0);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [productFilter, setProductFilter] = useState({
    quantityOp: '', quantityValue: 0,
    brand: '', ncms: '',
    activeStatus: '', weighable: '',
    itemType: '', stateTribId: '', federalTribId: '',
    costPriceOp: '', costPriceVal: 0,
    salePriceOp: '', salePriceVal: 0
  });
  const [showPriceModal, setShowPriceModal] = useState(false);
  const [showNCMModal, setShowNCMModal] = useState(false);

  const toggleSelectAll = () => {
    if (selectedIds.length === filterResults.length) {
      setSelectedIds([]);
    } else {
      setSelectedIds(filterResults.map(p => p.id));
    }
  };

  const toggleSelect = (id: string) => {
    if (selectedIds.includes(id)) {
      setSelectedIds(selectedIds.filter(sid => sid !== id));
    } else {
      setSelectedIds([...selectedIds, id]);
    }
  };

  const handleBulkActivate = async (activate: boolean) => {
    if (selectedIds.length === 0) return;
    try {
      await BulkActivateProducts(selectedIds, activate);
      // Refresh results
      handleFilter();
      setSelectedIds([]);
    } catch (err) {
      console.error(err);
    }
  };

  const handleFilter = async () => {
    const filter: any = {};
    if (productFilter.ncms) filter.ncms = productFilter.ncms.split(',').map((n: string) => n.trim());
    if (productFilter.brand) filter.brand = productFilter.brand;
    if (productFilter.quantityOp) {
      filter.quantityOp = productFilter.quantityOp;
      filter.quantityValue = productFilter.quantityValue;
    }
    if (productFilter.activeStatus) filter.activeStatus = productFilter.activeStatus === 'true';
    
    try {
      const response = await FilterProducts(filter);
      setFilterResults((response as any)?.products || []);
      setTotalProducts((response as any)?.total || 0);
    } catch(err) {
      console.error(err);
    }
  };

  if (!show) return null;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal modal-large" onClick={(e) => e.stopPropagation()}>
        <h3>📦 Gerenciador Avançado de Produtos</h3>
        <p className="modal-subtitle">📊 Total na base: <strong>{totalInDatabase.toLocaleString()}</strong> produtos</p>
        
        {}
        <div className="filter-grid" onKeyDown={(e) => {
          if (e.key === 'Enter') handleFilter();
        }}>
          <div className="filter-row">
            <label>NCMs:</label>
            <input 
              type="text" 
              placeholder="Ex: 8471, 8473" 
              value={productFilter.ncms}
              onChange={e => setProductFilter({...productFilter, ncms: e.target.value})}
            />
          </div>
          <div className="filter-row">
            <label>Marca:</label>
            <input 
              type="text" 
              placeholder="Contém..."
              value={productFilter.brand}
              onChange={e => setProductFilter({...productFilter, brand: e.target.value})}
            />
          </div>
          <div className="filter-row">
            <label>Quantidade:</label>
            <div className="filter-inline">
              <select value={productFilter.quantityOp} onChange={e => setProductFilter({...productFilter, quantityOp: e.target.value})}>
                <option value="">Qualquer</option>
                <option value="lt">Menor que</option>
                <option value="lte">Menor ou igual</option>
                <option value="eq">Igual a</option>
                <option value="gte">Maior ou igual</option>
                <option value="gt">Maior que</option>
              </select>
              <input 
                type="text"
                inputMode="numeric"
                pattern="[0-9]*"
                placeholder="Valor"
                value={productFilter.quantityValue || ''}
                onChange={e => setProductFilter({...productFilter, quantityValue: parseFloat(e.target.value) || 0})}
              />
            </div>
          </div>
          <div className="filter-row">
            <label>Status:</label>
            <select value={productFilter.activeStatus} onChange={e => setProductFilter({...productFilter, activeStatus: e.target.value})}>
              <option value="">Todos</option>
              <option value="true">Ativos</option>
              <option value="false">Inativos</option>
            </select>
          </div>
        </div>

        <div className="filter-actions">
          <button className="primary" onClick={handleFilter}>🔍 Filtrar</button>
          {totalProducts > 0 && (
            <div className="filter-stats">
              <span className="stat-item">🎯 <strong>{totalProducts.toLocaleString()}</strong> filtrados</span>
              <span className="stat-separator">|</span>
              <span className="stat-item">📋 <strong>{Math.min(filterResults.length, 100)}</strong> exibidos</span>
            </div>
          )}
        </div>

        {}
        {filterResults.length > 0 && (
          <div className="results-table-container">
            {selectedIds.length > 0 && (
              <div className="bulk-actions" style={{ marginBottom: '10px', display: 'flex', gap: '10px', alignItems: 'center', background: 'rgba(255,255,255,0.05)', padding: '8px', borderRadius: '6px' }}>
                <span style={{ fontSize: '0.9rem', color: '#cbd5e1' }}>{selectedIds.length} selecionados</span>
                <button 
                  onClick={() => handleBulkActivate(true)}
                  style={{ padding: '4px 12px', borderRadius: '4px', border: 'none', background: '#22c55e', color: 'white', cursor: 'pointer', fontSize: '0.85rem' }}
                >
                  ✅ Ativar
                </button>
                <button 
                  onClick={() => handleBulkActivate(false)}
                  style={{ padding: '4px 12px', borderRadius: '4px', border: 'none', background: '#ef4444', color: 'white', cursor: 'pointer', fontSize: '0.85rem' }}
                >
                  ❌ Inativar
                </button>
              </div>
            )}
            
            <table className="results-table">
              <thead>
                <tr>
                   <th style={{ width: '40px', textAlign: 'center' }}>
                     <input 
                       type="checkbox" 
                       checked={filterResults.length > 0 && selectedIds.length === filterResults.length}
                       onChange={toggleSelectAll}
                       style={{ cursor: 'pointer' }}
                     />
                   </th>
                   <th>Descrição</th>
                   <th>Marca</th>
                   <th>NCM</th>
                   <th>Estoque</th>
                   <th>Ativo</th>
                </tr>
              </thead>
              <tbody>
                {filterResults.map((p, idx) => (
                    <tr key={idx} className={selectedIds.includes(p.id) ? 'selected-row' : ''} style={selectedIds.includes(p.id) ? { background: 'rgba(246, 136, 45, 0.1)' } : {}}>
                        <td style={{ textAlign: 'center' }}>
                          <input 
                            type="checkbox" 
                            checked={selectedIds.includes(p.id)}
                            onChange={() => toggleSelect(p.id)}
                            style={{ cursor: 'pointer' }}
                          />
                        </td>
                        <td>{p.name}</td>
                        <td>{p.brand}</td>
                        <td>{p.ncm}</td>
                        <td>{p.quantity}</td>
                        <td>{p.active ? '✅' : '❌'}</td>
                    </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Secondary Modals */}
        <PriceAdjustmentModal 
          show={showPriceModal} 
          onClose={() => setShowPriceModal(false)} 
        />
        <NCMChangeModal
          show={showNCMModal}
          onClose={() => setShowNCMModal(false)}
        />

        <div className="modal-actions">
          <button onClick={onClose}>Fechar</button>
          <div style={{ marginLeft: 'auto', display: 'flex', gap: '8px' }}>
            <button 
              className="secondary" 
              onClick={() => setShowNCMModal(true)}
            >
              📋 Alterar NCM
            </button>
            <button 
              className="secondary" 
              onClick={() => setShowPriceModal(true)}
            >
              💰 Ajuste de Preços
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
