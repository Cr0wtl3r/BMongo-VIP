import { useEffect, useState, useRef } from 'react';
import './App.css';
import { 
  InactivateZeroProducts, 
  CleanMovements, 
  EnableMEI,
  FindObjectIdInDatabase,
  CheckConnection,
  GetLogs,
  CancelOperation,
  GetTributations,
  GetFederalTributations,
  ChangeTributationByNCM,
  ChangeFederalTributationByNCM,
  CleanDatabase,
  CreateNewDatabase,
  CleanDigisatRegistry,
  GetUndoableOperations,
  UndoOperation,
  FilterProducts,
  BulkActivateProducts,
  BulkActivateByFilter,
  ZeroAllStock,
  ZeroNegativeStock,
  ZeroAllPrices,
  CleanDatabaseByDate,
  GetTotalProductCount,
  SelectInfoDatFile,
  UpdateEmitenteFromFile,
  ListEmitentes,
  DeleteEmitente,
  ChangeInvoiceKey,
  ChangeInvoiceStatus,
  GetInvoiceTypes,
  GetInvoiceStatuses,
  GetInvoiceByKey,
  GetInvoiceByNumber,
  BackupDatabase,
  RestoreDatabase,
  ListBackups,
  SelectDirectory,
  StopDigisatServices,
  StartDigisatServices,
  KillDigisatProcesses,
  GetDigisatServices,
  GetDigisatProcesses,
  SelectBackupFile
} from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';

// Module definitions
const modules = [
  {
    name: "📦 Produtos",
    items: [
      { id: "gerenciador", label: "Gerenciador Avançado", desc: "Filtra e gerencia produtos em lote" },
      { id: "inativar", label: "Inativar Zerados", desc: "Inativa produtos com estoque zerado ou negativo" },
      { id: "tributacao", label: "Alterar Tributação", desc: "Altera tributação por NCM (Estadual/Federal)" },
      { id: "mei", label: "Habilitar MEI", desc: "Ativa controle de estoque para MEI" },
    ]
  },
  {
    name: "🗄️ Base de Dados",
    items: [
      { id: "limpar_mov", label: "Limpar Movimentações", desc: "Remove imagens de cartão e movimentações" },
      { id: "limpar_por_data", label: "Limpar por Data", desc: "Remove movimentações antes de uma data" },
      { id: "limpar_base", label: "Limpar Base (Parcial)", desc: "Mantém config e emitentes" },
      { id: "nova_base", label: "Nova Base (Zero)", desc: "⚠️ DESTRÓI TUDO - use com cuidado!", danger: true },
    ]
  },
  {
    name: "⚙️ Sistema",
    items: [
      { id: "registro", label: "Limpar Registro Win", desc: "Remove chaves do Digisat no registro" },
      { id: "buscar_id", label: "Buscar ObjectID", desc: "Procura ID em todas as coleções" },
    ]
  },
  {
    name: "📈 Estoque / Preços",
    items: [
      { id: "zerar_estoque", label: "Zerar TODO Estoque", desc: "Zera quantidade de todos os produtos", danger: true },
      { id: "zerar_negativo", label: "Zerar Estoque Negativo", desc: "Zera apenas estoques negativos" },
      { id: "zerar_precos", label: "Zerar Todos Preços", desc: "Zera custo e venda de todos produtos", danger: true },
    ]
  },
  {
    name: "👤 Emitente",
    items: [
      { id: "ajustar_emitente", label: "Alterar Emitente", desc: "Altera dados do emitente via info.dat" },
      { id: "apagar_emitente", label: "Apagar Emitente", desc: "Remove emitente e dados associados", danger: true },
    ]
  },
  {
    name: "📄 Notas Fiscais",
    items: [
      { id: "alterar_chave", label: "Alterar Chave", desc: "Corrige chave de acesso de NF" },
      { id: "alterar_situacao", label: "Alterar Situação", desc: "Define situação de NF manualmente" },
    ]
  },
  {
    name: "🆘 Emergência",
    items: [
      { id: "deu_merda", label: "Deu Merda!", desc: "Reverter operações recentes", danger: true },
    ]
  },
  {
    name: "💾 Backup / Restore",
    items: [
      { id: "backup", label: "Fazer Backup", desc: "Cria backup do banco de dados" },
      { id: "restore", label: "Restaurar Backup", desc: "Restaura de uma pasta de backup" },
    ]
  },
  {
    name: "🖥️ Serviços Windows",
    items: [
      { id: "stop_services", label: "Parar Serviços", desc: "Para todos os serviços Digisat" },
      { id: "start_services", label: "Iniciar Serviços", desc: "Inicia todos os serviços Digisat" },
      { id: "kill_processes", label: "Encerrar Processos", desc: "Força encerramento de processos Digisat", danger: true },
    ]
  }
];

function App() {
  const [logs, setLogs] = useState<string[]>([]);
  const [connected, setConnected] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const [expandedModule, setExpandedModule] = useState<string | null>(null);
  
  // Search State
  const [searchId, setSearchId] = useState('');
  const [showSearchModal, setShowSearchModal] = useState(false);

  // NCM Modal State
  const [showNcmModal, setShowNcmModal] = useState(false);
  const [ncmInput, setNcmInput] = useState('');
  const [selectedTrib, setSelectedTrib] = useState('');
  const [tributations, setTributations] = useState<Record<string, any>[]>([]);
  const [ncmScope, setNcmScope] = useState<'estadual' | 'federal'>('estadual');

  // Confirmation Scope
  const [showConfirmModal, setShowConfirmModal] = useState<{show: boolean, title: string, desc: string, action: () => Promise<any>}>({
      show: false, title: '', desc: '', action: async () => {} 
  });

  // Rollback modal
  const [showRollbackModal, setShowRollbackModal] = useState(false);
  const [undoableOps, setUndoableOps] = useState<any[]>([]);

  // Pinned quick access (persisted in localStorage)
  const [pinnedActions, setPinnedActions] = useState<string[]>(() => {
    const saved = localStorage.getItem('pinnedActions');
    return saved ? JSON.parse(saved) : ['gerenciador', 'inativar', 'tributacao', 'buscar_id'];
  });

  // Product Filter Modal State
  const [showFilterModal, setShowFilterModal] = useState(false);
  const [filterResults, setFilterResults] = useState<any[]>([]);
  const [totalProducts, setTotalProducts] = useState(0);
  const [totalInDatabase, setTotalInDatabase] = useState(0);
  const [selectedProducts, setSelectedProducts] = useState<string[]>([]);
  const [productFilter, setProductFilter] = useState({
    quantityOp: '', quantityValue: 0,
    brand: '', ncms: '',
    activeStatus: '', weighable: '',
    itemType: '', stateTribId: '', federalTribId: '',
    costPriceOp: '', costPriceVal: 0,
    salePriceOp: '', salePriceVal: 0
  });

   // Date Modal State (for CleanDatabaseByDate)
  const [showDateModal, setShowDateModal] = useState(false);
  const [dateInput, setDateInput] = useState('');

  // Emitente Modal State (Fase 3)
  const [showEmitenteModal, setShowEmitenteModal] = useState(false);
  const [emitenteFilePath, setEmitenteFilePath] = useState('');
  const [showEmitentesListModal, setShowEmitentesListModal] = useState(false);
  const [emitentesList, setEmitentesList] = useState<any[]>([]);
  const [selectedEmitenteId, setSelectedEmitenteId] = useState('');

  // Invoice Operations State (Fase 5)
  const [showInvoiceKeyModal, setShowInvoiceKeyModal] = useState(false);
  const [showInvoiceStatusModal, setShowInvoiceStatusModal] = useState(false);
  
  const [invoiceType, setInvoiceType] = useState('NFe');
  const [oldKey, setOldKey] = useState('');
  const [newKey, setNewKey] = useState('');
  
  const [invoiceSeries, setInvoiceSeries] = useState('');
  const [invoiceNumber, setInvoiceNumber] = useState('');
  const [invoiceStatus, setInvoiceStatus] = useState('Concluído');

  const [availableInvoiceTypes, setAvailableInvoiceTypes] = useState<string[]>([]);
  const [availableInvoiceStatuses, setAvailableInvoiceStatuses] = useState<string[]>([]);

  // Invoice Preview State (Repurposed from Portal)
  const [portalResult, setPortalResult] = useState<any>(null);

  // Backup/Restore State
  const [showBackupModal, setShowBackupModal] = useState(false);
  const [showRestoreModal, setShowRestoreModal] = useState(false);
  const [backupDir, setBackupDir] = useState('');
  const [restorePath, setRestorePath] = useState('');
  const [restoreDropExisting, setRestoreDropExisting] = useState(false);

  // Success Toast State
  const [successToast, setSuccessToast] = useState<{show: boolean, message: string}>({show: false, message: ''});
  const showSuccess = (message: string) => {
    setSuccessToast({show: true, message});
    setTimeout(() => setSuccessToast({show: false, message: ''}), 3000);
  };

  // Error Toast State
  const [errorToast, setErrorToast] = useState<{show: boolean, message: string}>({show: false, message: ''});
  const showError = (message: string) => {
    setErrorToast({show: true, message});
    setTimeout(() => setErrorToast({show: false, message: ''}), 4000);
  };

  const logsEndRef = useRef<HTMLDivElement>(null);
  const subscribed = useRef(false);

  // Auto-scroll logs
  const scrollToBottom = () => {
    logsEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  useEffect(() => {
    scrollToBottom();
  }, [logs]);

  useEffect(() => {
    if (subscribed.current) return;
    subscribed.current = true;
    
    const handler = (message: string) => {
        setLogs(prev => [...prev, message]);
    };

    EventsOn('log', handler);
    CheckConnection().then(setConnected);
    GetLogs().then(msgs => setLogs(msgs || []));
  }, []);

  const confirmAction = (title: string, desc: string, action: () => Promise<any>) => {
      setMenuOpen(false);
      setShowConfirmModal({ show: true, title, desc, action });
  };

  const handleAction = async () => {
      const actionToRun = showConfirmModal.action;
      const actionTitle = showConfirmModal.title;
      setShowConfirmModal({...showConfirmModal, show: false});
      
      if (actionToRun) {
          try {
              await actionToRun();
              showSuccess(`✅ ${actionTitle} concluído com sucesso!`);
          } catch(err) {
              console.error(err);
          }
      }
  };

  const handleMenuAction = (itemId: string) => {
    switch(itemId) {
      case 'gerenciador':
        setShowFilterModal(true);
        GetTotalProductCount().then(count => setTotalInDatabase(Number(count) || 0));
        break;
      case 'inativar':
        confirmAction("Inativar Produtos Zerados", "Inativa produtos com estoque ≤ 0, exceto kits e serviços.", InactivateZeroProducts);
        break;
      case 'tributacao':
        openNcmModal();
        break;
      case 'mei':
        confirmAction("Habilitar MEI", "Ativa configuração de estoque para Microempreendedor Individual.", EnableMEI);
        break;
      case 'limpar_mov':
        confirmAction("Limpar Movimentações", "Remove imagens de cartão e tabelas de movimentação pesadas.", CleanMovements);
        break;
      case 'limpar_base':
        confirmAction("Limpar Base (Parcial)", "Remove coleções mantendo apenas configurações e emitentes.", CleanDatabase);
        break;
      case 'nova_base':
        confirmAction("⚠️ NOVA BASE (ZERO)", "ATENÇÃO: Isso DESTRÓI todos os dados! Use apenas para restore limpo.", CreateNewDatabase);
        break;
      case 'registro':
        confirmAction("Limpar Registro Windows", "Remove chaves HKCU\\Software\\Digisat do registro.", CleanDigisatRegistry);
        break;
      case 'buscar_id':
        setShowSearchModal(true);
        break;
      case 'cancelar':
        CancelOperation();
        setMenuOpen(false);
        break;
      case 'deu_merda':
        openRollbackModal();
        break;
      // Fase 4: Estoque/Preços
      case 'zerar_estoque':
        confirmAction("⚠️ Zerar TODO Estoque", "Isso zera quantidade de TODOS os produtos! Tem certeza?", ZeroAllStock);
        break;
      case 'zerar_negativo':
        confirmAction("Zerar Estoque Negativo", "Zera apenas estoques com quantidade negativa.", ZeroNegativeStock);
        break;
      case 'zerar_precos':
        confirmAction("⚠️ Zerar TODOS Preços", "Isso zera custo e venda de TODOS os produtos! Tem certeza?", ZeroAllPrices);
        break;
      case 'limpar_por_data':
        setShowDateModal(true);
        break;
      // Fase 3: Emitente
      case 'ajustar_emitente':
        setShowEmitenteModal(true);
        setMenuOpen(false);
        break;
      case 'apagar_emitente':
        ListEmitentes().then((list: any) => {
          setEmitentesList(list || []);
          setSelectedEmitenteId('');
          setShowEmitentesListModal(true);
        }).catch(console.error);
        setMenuOpen(false);
        break;
      // Fase 5: Notas Fiscais
      case 'alterar_chave':
        GetInvoiceTypes().then((types: any) => {
           setAvailableInvoiceTypes(types || []);
           setInvoiceType(types?.[0] || 'NFe');
           setShowInvoiceKeyModal(true);
        }).catch(console.error);
        setMenuOpen(false);
        break;
      case 'alterar_situacao':
        Promise.all([GetInvoiceTypes(), GetInvoiceStatuses()]).then(([types, statuses]: any[]) => {
            setAvailableInvoiceTypes(types || []);
            setAvailableInvoiceStatuses(statuses || []);
            setInvoiceType(types?.[0] || 'NFe');
            setInvoiceStatus(statuses?.[0] || 'Concluído');
            setShowInvoiceStatusModal(true);
        }).catch(console.error);
        setMenuOpen(false);
        break;
      // Backup / Restore
      case 'backup':
        setShowBackupModal(true);
        setMenuOpen(false);
        break;
      case 'restore':
        setShowRestoreModal(true);
        setMenuOpen(false);
        break;
      // Windows Services
      case 'stop_services':
        confirmAction("Parar Serviços Digisat", "Isso para todos os serviços Digisat do Windows.", StopDigisatServices);
        break;
      case 'start_services':
        confirmAction("Iniciar Serviços Digisat", "Isso inicia todos os serviços Digisat do Windows.", StartDigisatServices);
        break;
      case 'kill_processes':
        confirmAction("⚠️ Encerrar Processos", "Isso força o encerramento de todos os processos Digisat!", KillDigisatProcesses);
        break;
    }
  };

  const openRollbackModal = async () => {
    setMenuOpen(false);
    try {
      const ops = await GetUndoableOperations();
      setUndoableOps(ops || []);
      setShowRollbackModal(true);
    } catch(err) {
      console.error(err);
    }
  };

  const handleUndo = async (opId: string) => {
    try {
      await UndoOperation(opId);
      const ops = await GetUndoableOperations();
      setUndoableOps(ops || []);
      if (ops.length === 0) {
        setShowRollbackModal(false);
      }
    } catch(err) {
      console.error(err);
    }
  };

  const openNcmModal = async () => {
    try {
      setNcmScope('estadual');
      const tribs = await GetTributations();
      setTributations(tribs || []);
      setShowNcmModal(true);
    } catch (err) {
      console.error(err);
    }
  };

  useEffect(() => {
    if (!showNcmModal) return;
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
  }, [ncmScope, showNcmModal]);

  const handleChangeTributation = async () => {
    if (!ncmInput.trim() || !selectedTrib) return;
    const ncms = ncmInput.split(',').map((n: string) => n.trim()).filter((n: string) => n);
    setShowNcmModal(false);
    
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

  const handleFindId = async () => {
    if (!searchId.trim()) return;
    setShowSearchModal(false);
    try {
      await FindObjectIdInDatabase(searchId);
      setSearchId('');
    } catch (err) {
      console.error(err);
    }
  };

  const toggleModule = (moduleName: string) => {
    setExpandedModule(expandedModule === moduleName ? null : moduleName);
  };

  const togglePin = (actionId: string) => {
    setPinnedActions(prev => {
      const newPinned = prev.includes(actionId)
        ? prev.filter(id => id !== actionId)
        : [...prev, actionId];
      localStorage.setItem('pinnedActions', JSON.stringify(newPinned));
      return newPinned;
    });
  };

  const getPinnedItems = () => {
    const allItems = modules.flatMap(m => m.items);
    return pinnedActions.map(id => allItems.find(item => item.id === id)).filter(Boolean) as typeof modules[0]['items'];
  };

  return (
    <div className="app-container">
      {/* Header */}
      <header className="app-header">
        <div className="header-left">
          <h1>🔧 Digisat Tools</h1>
          <span className={`status-badge ${connected ? 'connected' : 'disconnected'}`}>
            {connected ? '● Conectado' : '○ Offline'}
          </span>
        </div>
        <button 
          className="hamburger-btn"
          onClick={() => setMenuOpen(!menuOpen)}
          aria-label="Menu"
        >
          <span></span>
          <span></span>
          <span></span>
        </button>
      </header>

      {/* Main Content */}
      <main className={`main-content ${menuOpen ? 'sidebar-open' : ''}`}>
        {/* Quick Access - Pinned Actions */}
        {pinnedActions.length > 0 && (
          <div className="quick-access">
            {getPinnedItems().map(item => (
              <button key={item.id} className="quick-btn" onClick={() => handleMenuAction(item.id)}>
                {item.label}
              </button>
            ))}
          </div>
        )}

        {/* Logs Panel */}
        <div className="logs-panel">
          <div className="logs-toolbar">
            <span>📋 Log de Execução</span>
            <button onClick={() => setLogs([])}>Limpar</button>
          </div>
          <div className="logs-body">
            {logs.length === 0 ? (
              <div className="logs-empty">Nenhum log ainda. Execute uma operação.</div>
            ) : (
              logs.map((log, i) => (
                <div key={i} className="log-entry">{log}</div>
              ))
            )}
            <div ref={logsEndRef} />
          </div>
          <div className="logs-footer">
            <button className="cancel-btn" onClick={() => CancelOperation()}>
              ⏹️ Cancelar Operação
            </button>
          </div>
        </div>
      </main>

      {/* Sidebar Menu */}
      <aside className={`sidebar ${menuOpen ? 'open' : ''}`}>
        <div className="sidebar-header">
          <h2>Módulos</h2>
        </div>
        <nav className="sidebar-nav">
          {modules.map(module => (
            <div key={module.name} className="module-group">
              <button 
                className={`module-header ${expandedModule === module.name ? 'expanded' : ''}`}
                onClick={() => toggleModule(module.name)}
              >
                {module.name}
                <span className="chevron">▼</span>
              </button>
              {expandedModule === module.name && (
                <ul className="module-items">
                  {module.items.map(item => (
                    <li key={item.id}>
                      <div className="module-item-row">
                        <button 
                          className={`module-item ${item.danger ? 'danger' : ''}`}
                          onClick={() => handleMenuAction(item.id)}
                        >
                          {item.label}
                        </button>
                        <button 
                          className={`pin-btn ${pinnedActions.includes(item.id) ? 'pinned' : ''}`}
                          onClick={(e) => { e.stopPropagation(); togglePin(item.id); }}
                          title={pinnedActions.includes(item.id) ? 'Remover dos atalhos' : 'Adicionar aos atalhos'}
                        >
                          {pinnedActions.includes(item.id) ? '★' : '☆'}
                        </button>
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          ))}
        </nav>
      </aside>

      {/* Search Modal */}
      {showSearchModal && (
        <div className="modal-overlay" onClick={() => setShowSearchModal(false)}>
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
              <button onClick={() => setShowSearchModal(false)}>Cancelar</button>
              <button className="primary" onClick={handleFindId}>Buscar</button>
            </div>
          </div>
        </div>
      )}

      {/* NCM Modal */}
      {showNcmModal && (
        <div className="modal-overlay" onClick={() => setShowNcmModal(false)}>
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
              <button onClick={() => setShowNcmModal(false)}>Cancelar</button>
              <button className="primary" onClick={handleChangeTributation}>Executar</button>
            </div>
          </div>
        </div>
      )}

      {/* Confirmation Modal */}
      {showConfirmModal.show && (
        <div className="modal-overlay" onClick={() => setShowConfirmModal({...showConfirmModal, show: false})}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h3>Confirmar Ação</h3>
            <p className="modal-title">{showConfirmModal.title}</p>
            <p className="modal-desc">{showConfirmModal.desc}</p>
            <div className="modal-actions">
              <button onClick={() => setShowConfirmModal({...showConfirmModal, show: false})}>Cancelar</button>
              <button className="primary" onClick={handleAction}>Confirmar</button>
            </div>
          </div>
        </div>
      )}

      {/* Rollback Modal */}
      {showRollbackModal && (
        <div className="modal-overlay" onClick={() => setShowRollbackModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h3>🆘 Deu Merda? Reverta Aqui!</h3>
            {undoableOps.length === 0 ? (
              <p className="modal-desc">Nenhuma operação reversível no histórico.</p>
            ) : (
              <div className="undo-list">
                {undoableOps.map((op: any) => (
                  <div key={op.id} className="undo-item">
                    <div className="undo-info">
                      <span className="undo-label">{op.label}</span>
                      <span className="undo-time">{op.timestamp}</span>
                    </div>
                    <button className="undo-btn" onClick={() => handleUndo(op.id)}>
                      Reverter
                    </button>
                  </div>
                ))}
              </div>
            )}
            <div className="modal-actions">
              <button onClick={() => setShowRollbackModal(false)}>Fechar</button>
              <span></span>
            </div>
          </div>
        </div>
      )}

      {/* Product Filter Modal */}
      {showFilterModal && (
        <div className="modal-overlay" onClick={() => setShowFilterModal(false)}>
          <div className="modal modal-large" onClick={(e) => e.stopPropagation()}>
            <h3>📦 Gerenciador Avançado de Produtos</h3>
            <p className="modal-subtitle">📊 Total na base: <strong>{totalInDatabase.toLocaleString()}</strong> produtos</p>
            
            {/* Filter Form */}
            <div className="filter-grid" onKeyDown={async (e) => {
              if (e.key === 'Enter') {
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
                  setSelectedProducts([]);
                } catch(err) {
                  console.error(err);
                }
              }
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
              <button className="primary" onClick={async () => {
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
                  setSelectedProducts([]);
                } catch(err) {
                  console.error(err);
                }
              }}>🔍 Filtrar</button>
              {totalProducts > 0 && (
                <div className="filter-stats">
                  <span className="stat-item">🎯 <strong>{totalProducts.toLocaleString()}</strong> filtrados</span>
                  <span className="stat-separator">|</span>
                  <span className="stat-item">📋 <strong>{Math.min(filterResults.length, 100)}</strong> exibidos</span>
                </div>
              )}
            </div>

            {/* Results Table */}
            {filterResults.length > 0 && (
              <div className="results-table-container">
                <table className="results-table">
                  <thead>
                    <tr>
                      <th><input type="checkbox" onChange={e => {
                        if (e.target.checked) {
                          setSelectedProducts(filterResults.map((p: any) => p.id));
                        } else {
                          setSelectedProducts([]);
                        }
                      }} checked={selectedProducts.length === filterResults.length && filterResults.length > 0} /></th>
                      <th>Nome</th>
                      <th>NCM</th>
                      <th>Marca</th>
                      <th>Qtd</th>
                      <th>Ativo</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filterResults.slice(0, 100).map((p: any) => (
                      <tr key={p.id}>
                        <td><input type="checkbox" checked={selectedProducts.includes(p.id)} onChange={e => {
                          if (e.target.checked) {
                            setSelectedProducts([...selectedProducts, p.id]);
                          } else {
                            setSelectedProducts(selectedProducts.filter(id => id !== p.id));
                          }
                        }} /></td>
                        <td title={p.name}>{p.name?.substring(0, 35)}{p.name?.length > 35 ? '...' : ''}</td>
                        <td>{p.ncm}</td>
                        <td>{p.brand}</td>
                        <td>{p.quantity}</td>
                        <td>{p.active ? '✅' : '❌'}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                {filterResults.length > 100 && (
                  <p className="result-note">
                    ⚠️ Tabela mostra primeiros 100 de {filterResults.length} carregados ({totalProducts} total no banco)
                  </p>
                )}
              </div>
            )}

            {/* Bulk Actions - Selected */}
            {selectedProducts.length > 0 && (
              <div className="bulk-actions">
                <span>{selectedProducts.length} selecionados</span>
                <button onClick={async () => {
                  await BulkActivateProducts(selectedProducts, true);
                  showSuccess(`✅ ${selectedProducts.length} produtos ativados!`);
                  setSelectedProducts([]);
                }}>✅ Ativar</button>
                <button className="danger-btn" onClick={async () => {
                  await BulkActivateProducts(selectedProducts, false);
                  showSuccess(`✅ ${selectedProducts.length} produtos inativados!`);
                  setSelectedProducts([]);
                }}>❌ Inativar</button>
              </div>
            )}

            {/* Bulk Actions - ALL Filtered (only show when nothing selected AND at least one filter active) */}
            {totalProducts > 0 && selectedProducts.length === 0 && 
             (productFilter.ncms || productFilter.brand || productFilter.activeStatus || productFilter.quantityOp) && (
              <div className="bulk-all-actions">
                <span>⚡ Aplicar a <strong>TODOS os {totalProducts.toLocaleString()}</strong> produtos que correspondem ao filtro:</span>
                <button onClick={async () => {
                  const filter: any = {};
                  if (productFilter.ncms) filter.ncms = productFilter.ncms.split(',').map((n: string) => n.trim());
                  if (productFilter.brand) filter.brand = productFilter.brand;
                  if (productFilter.activeStatus) filter.activeStatus = productFilter.activeStatus === 'true';
                  const result = await BulkActivateByFilter(filter, true);
                  showSuccess(`✅ ${result} produtos ativados!`);
                }}>✅ Ativar TODOS</button>
                <button className="danger-btn" onClick={async () => {
                  const filter: any = {};
                  if (productFilter.ncms) filter.ncms = productFilter.ncms.split(',').map((n: string) => n.trim());
                  if (productFilter.brand) filter.brand = productFilter.brand;
                  if (productFilter.activeStatus) filter.activeStatus = productFilter.activeStatus === 'true';
                  const result = await BulkActivateByFilter(filter, false);
                  showSuccess(`✅ ${result} produtos inativados!`);
                }}>❌ Inativar TODOS</button>
              </div>
            )}

            <div className="modal-actions">
              <button onClick={() => { setShowFilterModal(false); setFilterResults([]); setSelectedProducts([]); }}>Fechar</button>
              <span></span>
            </div>
          </div>
        </div>
      )}

      {/* Date Input Modal (for CleanDatabaseByDate) */}
      {showDateModal && (
        <div className="modal-overlay" onClick={() => setShowDateModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h3>📅 Limpar por Data</h3>
            <p className="modal-desc">Remove movimentações anteriores à data informada.</p>
            <div className="form-group">
              <label>Data limite (registros ANTES desta data serão removidos):</label>
              <input 
                type="date"
                value={dateInput}
                onChange={(e) => setDateInput(e.target.value)}
                style={{ width: '100%', padding: '0.75rem', marginTop: '0.5rem' }}
              />
            </div>
            <div className="modal-actions">
              <button onClick={() => setShowDateModal(false)}>Cancelar</button>
              <button className="primary" onClick={async () => {
                if (!dateInput) return;
                setShowDateModal(false);
                try {
                  await CleanDatabaseByDate(dateInput);
                  setDateInput('');
                } catch(err) {
                  console.error(err);
                }
              }}>🗑️ Limpar</button>
            </div>
          </div>
        </div>
      )}

      {/* Ajustar Emitente Modal - Native File Dialog */}
      {showEmitenteModal && (
        <div className="modal-overlay" onClick={() => setShowEmitenteModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h3>👤 Ajustar Emitente</h3>
            <p className="modal-desc">Altera dados do emitente Matriz a partir do arquivo info.dat</p>
            <div className="form-group">
              <label>Arquivo info.dat:</label>
              <div className="file-picker-row">
                <input 
                  type="text"
                  className="form-input"
                  placeholder="Clique em 'Selecionar' para escolher o arquivo"
                  value={emitenteFilePath}
                  readOnly
                />
                <button className="file-picker-btn" onClick={async () => {
                  const path = await SelectInfoDatFile();
                  if (path) setEmitenteFilePath(path);
                }}>📂 Selecionar</button>
              </div>
            </div>
            <div className="modal-actions">
              <button onClick={() => { setShowEmitenteModal(false); setEmitenteFilePath(''); }}>Cancelar</button>
              <button className="primary" onClick={async () => {
                if (!emitenteFilePath.trim()) {
                  alert('Selecione o arquivo info.dat');
                  return;
                }
                setShowEmitenteModal(false);
                try {
                  await UpdateEmitenteFromFile(emitenteFilePath);
                  showSuccess('✅ Emitente atualizado com sucesso!');
                } catch(err) {
                  console.error(err);
                }
                setEmitenteFilePath('');
              }}>Executar</button>
            </div>
          </div>
        </div>
      )}

      {/* Lista de Emitentes Modal */}
      {showEmitentesListModal && (
        <div className="modal-overlay" onClick={() => setShowEmitentesListModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h3>👤 Emitentes Cadastrados</h3>
            {emitentesList.length === 0 ? (
              <p className="modal-desc">Nenhum emitente encontrado.</p>
            ) : (
              <div className="emitentes-list">
                {emitentesList.map((e: any) => (
                  <div key={e.id} className="emitente-item">
                    <div className="emitente-info">
                      <span className="emitente-nome">{e.nome}</span>
                      <span className="emitente-cnpj">{e.cnpj}</span>
                    </div>
                    {emitentesList.length > 1 && (
                      <button 
                        className="danger-btn-small"
                        onClick={() => {
                          setShowEmitentesListModal(false);
                          setShowConfirmModal({
                            show: true,
                            title: `Remover ${e.nome}?`,
                            desc: `Isso removerá o emitente ${e.cnpj} e TODOS os dados associados (produtos, estoques, movimentações). Esta ação NÃO pode ser desfeita!`,
                            action: async () => {
                              await DeleteEmitente(e.id);
                              const updated = await ListEmitentes();
                              setEmitentesList(updated || []);
                            }
                          });
                        }}
                      >🗑️</button>
                    )}
                  </div>
                ))}
              </div>
            )}
            <div className="modal-actions">
              <button onClick={() => setShowEmitentesListModal(false)}>Fechar</button>
              <span></span>
            </div>
          </div>
        </div>
      )}

      {/* Alterar Chave Invoice Modal */}
      {showInvoiceKeyModal && (
        <div className="modal-overlay" onClick={() => setShowInvoiceKeyModal(false)}>
          <div className="modal" style={{ minWidth: '500px' }} onClick={(e) => e.stopPropagation()}>
            <h3>🔑 Alterar Chave de Acesso</h3>
            <p className="modal-desc">Corrige a chave de acesso (ID) de um documento fiscal.</p>
            
            <div className="form-group">
              <label>Tipo de Documento:</label>
              <select value={invoiceType} onChange={(e) => setInvoiceType(e.target.value)}>
                {availableInvoiceTypes.map(t => <option key={t} value={t}>{t}</option>)}
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

            {/* Invoce Preview Card */}
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
                setShowInvoiceKeyModal(false);
                setPortalResult(null);
              }}>Cancelar</button>
              <button className="primary" onClick={async () => {
                if (!oldKey || !newKey) return showError('Preencha todos os campos!');
                try {
                  const count = await ChangeInvoiceKey(invoiceType, oldKey, newKey);
                  setShowInvoiceKeyModal(false);
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
      )}

      {/* Alterar Situação Invoice Modal */}
      {showInvoiceStatusModal && (
        <div className="modal-overlay" onClick={() => setShowInvoiceStatusModal(false)}>
          <div className="modal" style={{ minWidth: '600px' }} onClick={(e) => e.stopPropagation()}>
            <h3>📊 Alterar Situação</h3>
            <p className="modal-desc">Define a situação manualmente e adiciona histórico.</p>
            <div className="filter-grid">
               <div className="filter-row">
                  <label>Tipo:</label>
                  <select value={invoiceType} onChange={(e) => setInvoiceType(e.target.value)}>
                    {availableInvoiceTypes.map(t => <option key={t} value={t}>{t}</option>)}
                  </select>
               </div>
               <div className="filter-row">
                  <label>Situação:</label>
                  <select value={invoiceStatus} onChange={(e) => setInvoiceStatus(e.target.value)}>
                    {availableInvoiceStatuses.map(s => <option key={s} value={s}>{s}</option>)}
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
                        // Backend defaults Series to "1" if empty
                        
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

            {/* Invoice Preview Card */}
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
                 setShowInvoiceStatusModal(false);
                 setPortalResult(null);
              }}>Cancelar</button>
              <button className="primary" onClick={async () => {
                if (!invoiceNumber) return showError('Informe o número!');
                // Backend defaults Series to "1"
                
                try {
                  await ChangeInvoiceStatus(invoiceType, invoiceSeries, invoiceNumber, invoiceStatus);
                  setShowInvoiceStatusModal(false);
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
      )}


      {/* Backup Modal */}
      {showBackupModal && (
        <div className="modal-overlay" onClick={() => setShowBackupModal(false)}>
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
                />
                <button className="file-picker-btn" onClick={() => {
                  SelectDirectory("Selecione pasta para backup").then((path: string) => {
                    if (path) setBackupDir(path);
                  });
                }}>Selecionar</button>
              </div>
            </div>

            <div className="modal-actions">
              <button onClick={() => setShowBackupModal(false)}>Cancelar</button>
              <button className="primary" onClick={async () => {
                if (!backupDir) return showError('Selecione uma pasta!');
                try {
                  setShowBackupModal(false);
                  const result = await BackupDatabase(backupDir);
                  showSuccess(`✅ Backup criado: ${(result as any)?.path || 'OK'}`);
                } catch (err: any) {
                  showError(err?.message || 'Erro no backup');
                }
              }}>Fazer Backup</button>
            </div>
          </div>
        </div>
      )}

      {/* Restore Modal */}
      {showRestoreModal && (
        <div className="modal-overlay" onClick={() => setShowRestoreModal(false)}>
          <div className="modal modal-wide" onClick={e => e.stopPropagation()}>
            <h3>📥 Restaurar Backup</h3>
            <p>Restaura o banco de dados de um backup anterior.</p>

            <div className="form-field">
              <label>Pasta ou arquivo ZIP do backup:</label>
              <div className="file-picker-row">
                <input
                  type="text"
                  value={restorePath}
                  onChange={e => setRestorePath(e.target.value)}
                  placeholder="Selecione pasta ou arquivo .zip..."
                />
                <button className="file-picker-btn" onClick={() => {
                  SelectDirectory("Selecione pasta do backup").then((path: string) => {
                    if (path) setRestorePath(path);
                  });
                }}>📁 Pasta</button>
                <button className="file-picker-btn" onClick={() => {
                  SelectBackupFile("Selecione arquivo ZIP do backup").then((path: string) => {
                    if (path) setRestorePath(path);
                  });
                }}>📦 ZIP</button>
              </div>
            </div>

            <div className="form-field">
              <label className="checkbox-row">
                <input
                  type="checkbox"
                  checked={restoreDropExisting}
                  onChange={e => setRestoreDropExisting(e.target.checked)}
                />
                <span>⚠️ Sobrescrever coleções existentes (--drop)</span>
              </label>
              <small style={{color: '#f59e0b', display: 'block', marginTop: '0.5rem'}}>
                ⚠️ <strong>Obrigatório para restauração completa!</strong> Sem esta opção, 
                documentos existentes não serão atualizados.
              </small>
            </div>

            <div className="modal-actions">
              <button onClick={() => setShowRestoreModal(false)}>Cancelar</button>
              <button className="primary danger" onClick={async () => {
                if (!restorePath) return showError('Selecione uma pasta!');
                try {
                  setShowRestoreModal(false);
                  await RestoreDatabase(restorePath, restoreDropExisting);
                  showSuccess('✅ Restauração concluída!');
                } catch (err: any) {
                  showError(err?.message || 'Erro na restauração');
                }
              }}>Restaurar</button>
            </div>
          </div>
        </div>
      )}

      {/* Success Toast */}
      {successToast.show && (
        <div className="toast success" onClick={() => setSuccessToast({show: false, message: ''})}>
          {successToast.message}
        </div>
      )}

      {/* Error Toast */}
      {errorToast.show && (
        <div className="toast error" onClick={() => setErrorToast({show: false, message: ''})}>
          {errorToast.message}
        </div>
      )}
    </div>
  );
}

export default App;
