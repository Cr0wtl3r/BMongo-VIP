import React, { useState } from "react";
import { Login as GoLogin } from "../../wailsjs/go/main/App";
import "./Login.css";

interface LoginProps {
  onLoginSuccess: () => void;
}

export const Login: React.FC<LoginProps> = ({ onLoginSuccess }) => {
  const [senha, setSenha] = useState("");
  const [mensagemErro, setMensagemErro] = useState("");
  const [loading, setLoading] = useState(false);

  const fazerLogin = async () => {
    setMensagemErro("");
    if (senha === "") {
      setMensagemErro("Por favor, digite a senha.");
      return;
    }

    setLoading(true);
    try {
      const sucesso = await GoLogin(senha);
      if (sucesso) {
        onLoginSuccess();
      } else {
        setMensagemErro("Senha incorreta!");
        setSenha("");
      }
    } catch (err) {
      setMensagemErro(`Erro inesperado: ${err}`);
    } finally {
      setLoading(false);
    }
  };

  const handleKeydown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter") {
      fazerLogin();
    }
  };

  return (
    <div className="login-container">
      <div className="login-card">
        <div className="login-avatar">
          <img 
            src="/profile.png" 
            alt="Profile" 
            className="avatar-image"
          />
        </div>
        <h1 className="login-title">BMongo-VIP</h1>
        <p className="login-subtitle">
          Por favor, insira a senha para continuar.
        </p>
        <div className="login-input-group">
          <input
            id="senha"
            type="password"
            className="login-input"
            value={senha}
            onChange={(e) => setSenha(e.target.value)}
            onKeyDown={handleKeydown}
            placeholder="Senha"
            autoComplete="current-password"
            disabled={loading}
            autoFocus
          />
          <button
            className="login-button"
            onClick={fazerLogin}
            disabled={loading}
          >
            {loading ? "..." : "Entrar"}
          </button>
        </div>
        {mensagemErro && <p className="login-error">{mensagemErro}</p>}
      </div>
    </div>
  );
};
