package schemas

import (
	"fmt"
	"github.com/hypersequent/zen"
	"reflect"
	"sync"
)

type Schema struct {
	Id     string
	Type   reflect.Type
	Schema string
}

type State struct {
	Schemas map[string]Schema
}

var (
	mu    sync.RWMutex
	state State
)

func Register(schema Schema) {
	mu.Lock()
	defer mu.Unlock()

	if _, found := state.Schemas[schema.Id]; found {
		panic(fmt.Sprintf("schema already registered %s", schema.Id))
	}
	state.Schemas[schema.Id] = schema
}

func RegisterStruct(id string, structValue interface{}) {
	structType := reflect.TypeOf(structValue)
	zodSchema := zen.StructToZodSchema(structValue)

	Register(Schema{
		Id:     id,
		Type:   structType,
		Schema: zodSchema,
	})
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

func GetAllSchemas() map[string]string {
	mu.RLock()
	defer mu.RUnlock()

	result := make(map[string]string)
	for id, schema := range state.Schemas {
		result[id] = schema.Schema
	}
	return result
}

func init() {
	mu.Lock()
	defer mu.Unlock()
	state = State{}
	state.Schemas = make(map[string]Schema)
}
