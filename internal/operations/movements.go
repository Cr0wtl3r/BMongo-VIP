package operations

import (
	"context"
	"fmt"
	"time"

	"BMongo-VIP/internal/database"

	"go.mongodb.org/mongo-driver/bson"
)


func (m *Manager) CleanMovements(log LogFunc) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	log("Atualizando Movimentacoes...")
	err := m.updateMovimentacoes(ctx, log)
	if err != nil {
		return err
	}

	if m.state.ShouldStop() {
		return nil
	}

	log("Atualizando Recebimentos...")
	err = m.updateRecebimentos(ctx, log)
	if err != nil {
		return err
	}

	if m.state.ShouldStop() {
		return nil
	}

	log("Atualizando TurnosLancamentos...")
	err = m.updateTurnosLancamentos(ctx, log)
	if err != nil {
		return err
	}

	if m.state.ShouldStop() {
		return nil
	}

	log("Removendo AdministradoraCartao de Emitentes...")
	err = m.removeAdministradoraCartao(ctx, log)
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) updateMovimentacoes(ctx context.Context, log LogFunc) error {
	col := m.conn.GetCollection(database.CollectionMovimentacoes)


	for i := 0; i < 3; i++ {
		if m.state.ShouldStop() {
			return nil
		}

		filterField := fmt.Sprintf("PagamentoRecebimento.Parcelas.0.Historico.%d.EspeciePagamento.Descricao", i)
		unsetField := fmt.Sprintf("PagamentoRecebimento.Parcelas.0.Historico.%d.EspeciePagamento.Pessoa.Imagem", i)

		_, err := col.UpdateMany(ctx,
			bson.M{filterField: bson.M{"$regex": ".*Cart.*", "$options": "i"}},
			bson.M{"$unset": bson.M{unsetField: ""}},
		)
		if err != nil {
			log(fmt.Sprintf("Erro ao atualizar movimentacoes (índice %d): %s", i, err.Error()))
		}
	}


	_, err := col.UpdateMany(ctx,
		bson.M{"PagamentoRecebimento.Parcelas.0.Historico.0.EspeciePagamento.Descricao": bson.M{"$regex": ".*Cart.*", "$options": "i"}},
		bson.M{"$unset": bson.M{"PagamentoRecebimento.Parcelas.0.Pessoa.Imagem": ""}},
	)

	return err
}

func (m *Manager) updateRecebimentos(ctx context.Context, log LogFunc) error {
	col := m.conn.GetCollection(database.CollectionRecebimentos)


	for i := 0; i < 3; i++ {
		if m.state.ShouldStop() {
			return nil
		}

		filterField := fmt.Sprintf("Historico.%d.EspeciePagamento.Descricao", i)
		unsetField := fmt.Sprintf("Historico.%d.EspeciePagamento.Pessoa.Imagem", i)

		_, err := col.UpdateMany(ctx,
			bson.M{filterField: bson.M{"$regex": ".*Cart.*", "$options": "i"}},
			bson.M{"$unset": bson.M{unsetField: ""}},
		)
		if err != nil {
			log(fmt.Sprintf("Erro ao atualizar recebimentos (índice %d): %s", i, err.Error()))
		}
	}


	_, err := col.UpdateMany(ctx,
		bson.M{"Historico.0.EspeciePagamento.Descricao": bson.M{"$regex": ".*Cart.*", "$options": "i"}},
		bson.M{"$unset": bson.M{"Pessoa.Imagem": ""}},
	)

	return err
}

func (m *Manager) updateTurnosLancamentos(ctx context.Context, log LogFunc) error {
	col := m.conn.GetCollection(database.CollectionTurnosLancamentos)

	_, err := col.UpdateMany(ctx,
		bson.M{"EspeciePagamento.Descricao": bson.M{"$regex": ".*Cart.*", "$options": "i"}},
		bson.M{"$unset": bson.M{"EspeciePagamento.Pessoa.Imagem": ""}},
	)

	return err
}

func (m *Manager) removeAdministradoraCartao(ctx context.Context, log LogFunc) error {
	col := m.conn.GetCollection(database.CollectionPessoas)

	result, err := col.UpdateMany(ctx,
		bson.M{"_t": "Emitente"},
		bson.M{"$unset": bson.M{"AdministradoraCartao": ""}},
	)
	if err != nil {
		return err
	}

	log(fmt.Sprintf("Removido AdministradoraCartao de %d registros", result.ModifiedCount))
	return nil
}
