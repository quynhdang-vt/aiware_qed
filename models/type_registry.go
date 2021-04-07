package models

import (
	"fmt"
	controllerClient "github.com/veritone/realtime/modules/controller/client"
	"reflect"
)

/**
typeRegistry allows the registration of a type - via a pointer to an object so that it can be instantiate via reflection easily
 */

/*
ObjectTypeName return the string representation of an object.
 */
func ObjectTypeName(o interface{}) string {
	return reflect.TypeOf(o).String()
}

/*
RegisterMyType adds the type of the object to the global theTypeRegistrys
 */
func RegisterMyType(o interface{}) {
	theTypeRegistry.register(o)
}

/**
NewObjectForMyType looks up name which should be the type name of an object that must be Register with theTypeRegistry,
and returns a new object as created via reflection --  The returned interface is a Value (not pointer) of the type
 */
func NewObjectForMyType(name string) (interface{}, error) {
	return theTypeRegistry.instantiate(name)
}

/**
theTypeRegistry is a global object -
 */
var theTypeRegistry = typeRegistry{registry: make(map[string]reflect.Type)}

func init() {
	registerTypes()
}

/**
registerTypes provides default initialization.  New types can be registered via the Register method
*/
func registerTypes() {
	RegisterMyType(new(GetWorkRequest))
	RegisterMyType(new(GetWorkResponse))
	RegisterMyType(new(controllerClient.EngineInstanceWorkRequest))
	RegisterMyType(new(controllerClient.EngineInstanceWorkRequestResponse))
}

type typeRegistry struct {
	registry map[string]reflect.Type
}

func (tr *typeRegistry) getTypeNameOfObject(o interface{}) (string, reflect.Type) {
	t := reflect.TypeOf(o)
	return t.String(), t
}
func (tr *typeRegistry) register(o interface{}) {
	name, t := tr.getTypeNameOfObject(o)
	tr.registry[name] = t
}

func (tr *typeRegistry) instantiate(name string) (interface{}, error) {
	if myType, found := tr.registry[name]; found {
		p := reflect.New(myType.Elem()).Interface()
		return p, nil
	} else {
		// if not there we should register? need an object for discovery
		return nil, fmt.Errorf("%s is not registered", name)
	}
}
