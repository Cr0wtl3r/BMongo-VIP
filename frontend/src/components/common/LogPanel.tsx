import { useRef, useEffect } from 'react';

interface LogPanelProps {
  logs: string[];
  setLogs: (logs: string[]) => void;
  onCancel: () => void;
}

export function LogPanel({ logs, setLogs, onCancel }: LogPanelProps) {
  const logsEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    logsEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [logs]);

  return (
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
        <button className="cancel-btn" onClick={onCancel}>
          ⏹️ Cancelar Operação
        </button>
      </div>
    </div>
  );
}
