import { useEffect, useState, useRef } from 'react';
import './App.css';
import {
  InactivateZeroProducts,
  CleanMovements,
  EnableMEI,
  CheckConnection,
  RetryConnection,
  GetLogs,
  CancelOperation,
  CleanDatabase,
  CreateNewDatabase,
  CleanDigisatRegistry,
  GetUndoableOperations,
  ZeroAllStock,
  ZeroNegativeStock,
  ZeroAllPrices,
  GetTotalProductCount,
  ListEmitentes,
  GetInvoiceTypes,
  GetInvoiceStatuses,
  StopDigisatServices,
  StartDigisatServices,
  KillDigisatProcesses,
  DeleteEmitente,
  RepairMongoDBOffline,
  RepairMongoDBOnline,
  ReleaseFirewallPorts,
  AllowSecurityExclusions,
} from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';

import { Header } from './components/layout/Header';
import { Sidebar } from './components/layout/Sidebar';
import { LogPanel } from './components/common/LogPanel';
import { Toast } from './components/common/Toast';
import { ConfirmModal } from './components/common/ConfirmModal';
import { RollbackModal } from './components/common/RollbackModal';
import { SearchModal } from './components/common/SearchModal';
import { DateModal } from './components/common/DateModal';

import { InventoryModal } from './components/common/InventoryModal';
import { InventoryReportModal } from './components/common/InventoryReportModal';

import { ProductFilterModal } from './components/products/ProductFilterModal';
import { NcmModal } from './components/products/NcmModal';

import { InvoiceKeyModal } from './components/invoices/InvoiceKeyModal';
import { InvoiceStatusModal } from './components/invoices/InvoiceStatusModal';

import { BackupModal } from './components/backup/BackupModal';
import { RestoreModal } from './components/backup/RestoreModal';

import { EmitenteModal } from './components/emitente/EmitenteModal';
import { EmitentesListModal } from './components/emitente/EmitentesListModal';

import { Login } from './components/Login';

import { modules } from './config/modules';

function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [logs, setLogs] = useState<string[]>([]);
  const [connected, setConnected] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const [expandedModule, setExpandedModule] = useState<string | null>(null);


  const [showSearchModal, setShowSearchModal] = useState(false);
  const [showNcmModal, setShowNcmModal] = useState(false);
  const [showFilterModal, setShowFilterModal] = useState(false);
  const [showDateModal, setShowDateModal] = useState(false);
  const [showEmitenteModal, setShowEmitenteModal] = useState(false);
  const [showEmitentesListModal, setShowEmitentesListModal] = useState(false);
  const [showInvoiceKeyModal, setShowInvoiceKeyModal] = useState(false);
  const [showInvoiceStatusModal, setShowInvoiceStatusModal] = useState(false);
  const [showBackupModal, setShowBackupModal] = useState(false);
  const [showRestoreModal, setShowRestoreModal] = useState(false);
  const [showRollbackModal, setShowRollbackModal] = useState(false);
  const [showInventoryModal, setShowInventoryModal] = useState(false);
  const [showInventoryReportModal, setShowInventoryReportModal] = useState(false);


  const [totalInDatabase, setTotalInDatabase] = useState(0);
  const [emitentesList, setEmitentesList] = useState<any[]>([]);
  const [availableInvoiceTypes, setAvailableInvoiceTypes] = useState<string[]>([]);
  const [availableInvoiceStatuses, setAvailableInvoiceStatuses] = useState<string[]>([]);
  const [undoableOps, setUndoableOps] = useState<any[]>([]);


  const [showConfirmModal, setShowConfirmModal] = useState<{show: boolean, title: string, desc: string, action: () => Promise<any>}>({
      show: false, title: '', desc: '', action: async () => {}
  });


  const [pinnedActions, setPinnedActions] = useState<string[]>(() => {
    const saved = localStorage.getItem('pinnedActions');
    return saved ? JSON.parse(saved) : ['gerenciador', 'inativar', 'tributacao', 'buscar_id'];
  });


  const [successToast, setSuccessToast] = useState<{show: boolean, message: string}>({show: false, message: ''});
  const [errorToast, setErrorToast] = useState<{show: boolean, message: string}>({show: false, message: ''});

  const showSuccess = (message: string) => {
    setSuccessToast({show: true, message});
    setTimeout(() => setSuccessToast({show: false, message: ''}), 3000);
  };

  const showError = (message: string) => {
    setErrorToast({show: true, message});
    setTimeout(() => setErrorToast({show: false, message: ''}), 4000);
  };

  const subscribed = useRef(false);

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
      setShowConfirmModal({ show: true, title, desc, action });
  };

  const handleActionConfirm = async () => {
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

  const openRollbackModal = async () => {
    try {
      const ops = await GetUndoableOperations();
      setUndoableOps(ops || []);
      setShowRollbackModal(true);
    } catch(err) {
      console.error(err);
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
        setShowNcmModal(true);
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
        break;
      case 'deu_merda':
        openRollbackModal();
        break;

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
      case 'ajustar_inventario':
        setShowInventoryModal(true);
        break;
      case 'gerar_inventario':
        setShowInventoryReportModal(true);
        break;

      case 'ajustar_emitente':
        setShowEmitenteModal(true);
        break;
      case 'apagar_emitente':
        ListEmitentes().then((list: any) => {
          setEmitentesList(list || []);
          setShowEmitentesListModal(true);
        }).catch(console.error);
        break;

      case 'alterar_chave':
        GetInvoiceTypes().then((types: any) => {
           setAvailableInvoiceTypes(types || []);
           setShowInvoiceKeyModal(true);
        }).catch(console.error);
        break;
      case 'alterar_situacao':
        Promise.all([GetInvoiceTypes(), GetInvoiceStatuses()]).then(([types, statuses]: any[]) => {
            setAvailableInvoiceTypes(types || []);
            setAvailableInvoiceStatuses(statuses || []);
            setShowInvoiceStatusModal(true);
        }).catch(console.error);
        break;

      case 'backup':
        setShowBackupModal(true);
        break;
      case 'restore':
        setShowRestoreModal(true);
        break;

      case 'stop_services':
        confirmAction("Parar Serviços Digisat", "Isso para todos os serviços Digisat do Windows.", StopDigisatServices);
        break;
      case 'start_services':
        confirmAction("Iniciar Serviços Digisat", "Isso inicia todos os serviços Digisat do Windows.", StartDigisatServices);
        break;
      case 'kill_processes':
        confirmAction("⚠️ Encerrar Processos", "Isso força o encerramento de todos os processos Digisat!", KillDigisatProcesses);
        break;

      case 'repair_offline':
        confirmAction("⚠️ Reparar MongoDB (Offline)", "Isso PARA o serviço MongoDB, executa reparo completo e reinicia. A aplicação pode ficar indisponível por vários minutos.", RepairMongoDBOffline);
        break;
      case 'repair_online':
        confirmAction("Reparar MongoDB (Ativo)", "Executa comando de reparo com o MongoDB rodando. Mais seguro mas menos efetivo.", RepairMongoDBOnline);
        break;
      case 'liberar_portas':
        confirmAction("Liberar Portas Firewall", "Adiciona regras no Windows Firewall para liberar portas usadas pelo Digisat.", ReleaseFirewallPorts);
        break;
      case 'permitir_seguranca':
        confirmAction("Permitir Segurança", "Adiciona exclusões no Windows Defender e configura permissões de pasta.", AllowSecurityExclusions);
        break;
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

  const handleRetryConnection = async () => {
    try {
      await RetryConnection();
      const isConnected = await CheckConnection();
      setConnected(isConnected);
      if (isConnected) {
        showSuccess('✅ Reconectado ao banco com sucesso!');
      }
    } catch (err) {
      showError('❌ Falha ao reconectar ao banco de dados');
      console.error(err);
    }
  };

  if (!isAuthenticated) {
    return <Login onLoginSuccess={() => setIsAuthenticated(true)} />;
  }

  return (
    <div className="app-container">
      <Header
        connected={connected}
        menuOpen={menuOpen}
        setMenuOpen={setMenuOpen}
        onRetryConnection={handleRetryConnection}
      />

      <main className={`main-content ${menuOpen ? 'sidebar-open' : ''}`}>
        {}
        {pinnedActions.length > 0 && (
          <div className="quick-access">
            {getPinnedItems().map(item => (
              <button key={item.id} className="quick-btn" onClick={() => handleMenuAction(item.id)}>
                {item.label}
              </button>
            ))}
          </div>
        )}

        <LogPanel
          logs={logs}
          setLogs={setLogs}
          onCancel={CancelOperation}
        />
      </main>

      <Sidebar
        menuOpen={menuOpen}
        expandedModule={expandedModule}
        toggleModule={toggleModule}
        handleMenuAction={handleMenuAction}
        pinnedActions={pinnedActions}
        togglePin={togglePin}
      />

      {}

      <SearchModal
        show={showSearchModal}
        onClose={() => setShowSearchModal(false)}
      />

      <ProductFilterModal
        show={showFilterModal}
        onClose={() => setShowFilterModal(false)}
        totalInDatabase={totalInDatabase}
      />

      <NcmModal
        show={showNcmModal}
        onClose={() => setShowNcmModal(false)}
      />

      <DateModal
        show={showDateModal}
        onClose={() => setShowDateModal(false)}
        showSuccess={showSuccess}
        showError={showError}
      />

      <InventoryModal
        show={showInventoryModal}
        onClose={() => setShowInventoryModal(false)}
        showSuccess={showSuccess}
        showError={showError}
      />

      <InventoryReportModal
        show={showInventoryReportModal}
        onClose={() => setShowInventoryReportModal(false)}
        showSuccess={showSuccess}
        showError={showError}
      />

      <EmitenteModal
        show={showEmitenteModal}
        onClose={() => setShowEmitenteModal(false)}
        showSuccess={showSuccess}
        showError={showError}
      />

      <EmitentesListModal
        show={showEmitentesListModal}
        onClose={() => setShowEmitentesListModal(false)}
        emitentesList={emitentesList}
        onDelete={(id) => {
           confirmAction(
             "⚠️ Excluir Emitente",
             "TEM CERTEZA? Essa ação apaga TUDO (Movimentações, Estoques, Financeiro etc) vinculado a este CNPJ e remove o info.dat do servidor se necessário. É irreversível!",
             async () => await DeleteEmitente(id)
           );
        }}
      />

      <InvoiceKeyModal
        show={showInvoiceKeyModal}
        onClose={() => setShowInvoiceKeyModal(false)}
        availableTypes={availableInvoiceTypes}
        showSuccess={showSuccess}
        showError={showError}
      />

      <InvoiceStatusModal
        show={showInvoiceStatusModal}
        onClose={() => setShowInvoiceStatusModal(false)}
        availableTypes={availableInvoiceTypes}
        availableStatuses={availableInvoiceStatuses}
        showSuccess={showSuccess}
        showError={showError}
      />

      <BackupModal
        show={showBackupModal}
        onClose={() => setShowBackupModal(false)}
        showSuccess={showSuccess}
        showError={showError}
      />

      <RestoreModal
        show={showRestoreModal}
        onClose={() => setShowRestoreModal(false)}
        showSuccess={showSuccess}
        showError={showError}
      />

      <RollbackModal
        show={showRollbackModal}
        onClose={() => setShowRollbackModal(false)}
        undoableOps={undoableOps}
        setUndoableOps={setUndoableOps}
      />

      <ConfirmModal
        show={showConfirmModal.show}
        title={showConfirmModal.title}
        desc={showConfirmModal.desc}
        onConfirm={handleActionConfirm}
        onCancel={() => setShowConfirmModal({...showConfirmModal, show: false})}
      />

      <Toast
        show={successToast.show}
        message={successToast.message}
        type="success"
        onClose={() => setSuccessToast({...successToast, show: false})}
      />
      <Toast
        show={errorToast.show}
        message={errorToast.message}
        type="error"
        onClose={() => setErrorToast({...errorToast, show: false})}
      />

    </div>
  );
}

export default App;
