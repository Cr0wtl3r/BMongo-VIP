interface HeaderProps {
  connected: boolean;
  menuOpen: boolean;
  setMenuOpen: (open: boolean) => void;
}

export function Header({ connected, menuOpen, setMenuOpen }: HeaderProps) {
  return (
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
  );
}
