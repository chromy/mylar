package schemas

import (
	"fmt"
	"github.com/chromy/viz/internal/constants"
	"github.com/hypersequent/zen"
	"sort"
	"sync"
)

type Schema struct {
	Id    string
	Value interface{}
}

type State struct {
	Schemas map[string]Schema
}

var (
	mu    sync.RWMutex
	state State
)

func Register(id string, structValue interface{}) {
	mu.Lock()
	defer mu.Unlock()

	schema := Schema{
		Id:    id,
		Value: structValue,
	}

	if _, found := state.Schemas[schema.Id]; found {
		panic(fmt.Sprintf("schema already registered %s", schema.Id))
	}

	state.Schemas[schema.Id] = schema
}

func Get(id string) (Schema, bool) {
	mu.RLock()
	defer mu.RUnlock()

	schema, found := state.Schemas[id]
	return schema, found
}

func List() []string {
	mu.RLock()
	defer mu.RUnlock()

	list := make([]string, 0, len(state.Schemas))
	for id := range state.Schemas {
		list = append(list, id)
	}
	return list
}

func ToZodSchema() string {
	mu.RLock()
	defer mu.RUnlock()

	var ids []string
	for id := range state.Schemas {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	c := zen.NewConverterWithOpts()

	for _, id := range ids {
		c.AddType(state.Schemas[id].Value)
	}

	var text string
	text += fmt.Sprintf("import { z } from \"zod\";\n\n")
	text += fmt.Sprintf("export const TILE_SIZE = %d;\n\n", constants.TileSize)
	text += c.Export()

	return text
}

func init() {
	mu.Lock()
	defer mu.Unlock()
	state = State{}
	state.Schemas = make(map[string]Schema)
}
