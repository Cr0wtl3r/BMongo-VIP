import { modules } from '../../config/modules';

interface SidebarProps {
  menuOpen: boolean;
  expandedModule: string | null;
  toggleModule: (moduleName: string) => void;
  handleMenuAction: (itemId: string) => void;
  pinnedActions: string[];
  togglePin: (actionId: string) => void;
}

export function Sidebar({ 
  menuOpen, 
  expandedModule, 
  toggleModule, 
  handleMenuAction,
  pinnedActions,
  togglePin 
}: SidebarProps) {
  return (
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
  );
}
