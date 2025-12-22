package operations

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FindObjectIdInDatabase searches for an ObjectId in all collections
func (m *Manager) FindObjectIdInDatabase(searchID string, log LogFunc) ([]map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Try to parse as ObjectId first
	var searchOID primitive.ObjectID
	var searchAsOID bool
	if oid, err := primitive.ObjectIDFromHex(searchID); err == nil {
		searchOID = oid
		searchAsOID = true
	}

	// Get all collection names
	collections, err := m.conn.Database.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("erro ao listar coleções: %w", err)
	}

	var results []map[string]string

	for _, colName := range collections {
		if m.state.ShouldStop() {
			log("Operação cancelada pelo usuário")
			return results, nil
		}

		col := m.conn.Database.Collection(colName)
		cursor, err := col.Find(ctx, bson.M{})
		if err != nil {
			continue
		}

		for cursor.Next(ctx) {
			if m.state.ShouldStop() {
				cursor.Close(ctx)
				return results, nil
			}

			var doc bson.M
			if err := cursor.Decode(&doc); err != nil {
				continue
			}

			// Search through document fields
			matches := searchDocument(doc, searchID, searchOID, searchAsOID)
			for _, field := range matches {
				result := map[string]string{
					"collection": colName,
					"field":      field,
				}
				results = append(results, result)
				log(fmt.Sprintf("Encontrado na coleção %s, campo %s", colName, field))
			}
		}
		cursor.Close(ctx)
	}

	return results, nil
}

// searchDocument recursively searches a document for a matching ObjectId or string
func searchDocument(doc bson.M, searchStr string, searchOID primitive.ObjectID, searchAsOID bool) []string {
	var matches []string

	for key, value := range doc {
		switch v := value.(type) {
		case primitive.ObjectID:
			if searchAsOID && v == searchOID {
				matches = append(matches, key)
			} else if v.Hex() == searchStr {
				matches = append(matches, key)
			}
		case string:
			if v == searchStr {
				matches = append(matches, key)
			}
		case bson.M:
			// Recurse into nested documents
			nested := searchDocument(v, searchStr, searchOID, searchAsOID)
			for _, n := range nested {
				matches = append(matches, key+"."+n)
			}
		case bson.A:
			// Handle arrays
			for i, item := range v {
				if m, ok := item.(bson.M); ok {
					nested := searchDocument(m, searchStr, searchOID, searchAsOID)
					for _, n := range nested {
						matches = append(matches, fmt.Sprintf("%s[%d].%s", key, i, n))
					}
				} else if oid, ok := item.(primitive.ObjectID); ok {
					if searchAsOID && oid == searchOID {
						matches = append(matches, fmt.Sprintf("%s[%d]", key, i))
					}
				}
			}
		}
	}

	return matches
}
