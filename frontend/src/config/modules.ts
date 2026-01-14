export const modules = [
  {
    name: "📦 Produtos",
    items: [
      {
        id: "gerenciador",
        label: "Gerenciador Avançado",
        desc: "Filtra e gerencia produtos em lote",
      },
      {
        id: "inativar",
        label: "Inativar Zerados",
        desc: "Inativa produtos com estoque zerado ou negativo",
      },
      {
        id: "tributacao",
        label: "Alterar Tributação",
        desc: "Altera tributação por NCM (Estadual/Federal)",
      },
      {
        id: "mei",
        label: "Habilitar MEI",
        desc: "Ativa controle de estoque para MEI",
      },
    ],
  },
  {
    name: "🗄️ Base de Dados",
    items: [
      {
        id: "limpar_mov",
        label: "Limpar Movimentações",
        desc: "Remove imagens de cartão e movimentações",
      },
      {
        id: "limpar_por_data",
        label: "Limpar por Data",
        desc: "Remove movimentações antes de uma data",
      },
      {
        id: "limpar_base",
        label: "Limpar Base (Parcial)",
        desc: "Mantém config e emitentes",
      },
      {
        id: "nova_base",
        label: "Nova Base (Zero)",
        desc: "⚠️ DESTRÓI TUDO - use com cuidado!",
        danger: true,
      },
    ],
  },
  {
    name: "⚙️ Sistema",
    items: [
      {
        id: "registro",
        label: "Limpar Registro Win",
        desc: "Remove chaves do Digisat no registro",
      },
      {
        id: "buscar_id",
        label: "Buscar ObjectID",
        desc: "Procura ID em todas as coleções",
      },
    ],
  },
  {
    name: "📈 Estoque / Preços",
    items: [
      {
        id: "gerar_inventario",
        label: "Gerar Inventário",
        desc: "Gera relatório P7 em XLSX/CSV com valor alvo",
      },
      {
        id: "zerar_estoque",
        label: "Zerar TODO Estoque",
        desc: "Zera quantidade de todos os produtos",
        danger: true,
      },
      {
        id: "zerar_negativo",
        label: "Zerar Estoque Negativo",
        desc: "Zera apenas estoques negativos",
      },
      {
        id: "zerar_precos",
        label: "Zerar Todos Preços",
        desc: "Zera custo e venda de todos produtos",
        danger: true,
      },
    ],
  },
  {
    name: "👤 Emitente",
    items: [
      {
        id: "ajustar_emitente",
        label: "Alterar Emitente",
        desc: "Altera dados do emitente via info.dat",
      },
      {
        id: "apagar_emitente",
        label: "Apagar Emitente",
        desc: "Remove emitente e dados associados",
        danger: true,
      },
    ],
  },
  {
    name: "📄 Notas Fiscais",
    items: [
      {
        id: "alterar_chave",
        label: "Alterar Chave",
        desc: "Corrige chave de acesso de NF",
      },
      {
        id: "alterar_situacao",
        label: "Alterar Situação",
        desc: "Define situação de NF manualmente",
      },
      {
        id: "notas_manuais",
        label: "Notas Manuais",
        desc: "Imprimir fatura fiscal manual",
      },
    ],
  },
  {
    name: "🆘 Emergência",
    items: [
      {
        id: "deu_merda",
        label: "Deu Merda!",
        desc: "Reverter operações recentes",
        danger: true,
      },
    ],
  },
  {
    name: "💾 Backup / Restore",
    items: [
      {
        id: "backup",
        label: "Fazer Backup",
        desc: "Cria backup do banco de dados",
      },
      {
        id: "restore",
        label: "Restaurar Backup",
        desc: "Restaura de uma pasta de backup",
      },
    ],
  },
  {
    name: "🖥️ Serviços Windows",
    items: [
      {
        id: "stop_services",
        label: "Parar Serviços",
        desc: "Para todos os serviços Digisat",
      },
      {
        id: "start_services",
        label: "Iniciar Serviços",
        desc: "Inicia todos os serviços Digisat",
      },
      {
        id: "kill_processes",
        label: "Encerrar Processos",
        desc: "Força encerramento de processos Digisat",
        danger: true,
      },
    ],
  },
  {
    name: "🔧 Manutenção",
    items: [
      {
        id: "repair_offline",
        label: "Reparar MongoDB (Offline)",
        desc: "Para o serviço e executa reparo completo",
        danger: true,
      },
      {
        id: "repair_online",
        label: "Reparar MongoDB (Ativo)",
        desc: "Repara banco com serviço rodando",
      },
      {
        id: "liberar_portas",
        label: "Liberar Portas Firewall",
        desc: "Adiciona regras para portas Digisat",
      },
      {
        id: "permitir_seguranca",
        label: "Permitir Segurança",
        desc: "Adiciona exclusões no Windows Defender",
      },
    ],
  },
];
