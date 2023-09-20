package handler

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/timapril/go-registrar/lib"
)

const (
	// ObjTypeParam is the key used to represent the type of an object
	// being requested to in a response
	ObjTypeParam = "objecttype"

	// ObjIDParam is the key used to represent the ID of an object being
	// requested or in a response
	ObjIDParam = "id"

	// ObjTimeStampParam is the key used to represent the timestamp of
	// a request or a response
	ObjTimeStampParam = "ts"
)

// getRouteVar retrieves a route variable from the context,
// forming an error if it's not found.
func getRouteVar(ctx Context, name string) (string, error) {
	routes := ctx.GetRouteVars()

	value := routes[name]
	if value == "" {
		return "", fmt.Errorf("%s: %s", RouteVarMissing, name)
	}

	return value, nil
}

// ToForm will take a map of keys to a valies and generate a url.Values
// object that can be used to represent a form
func ToForm(values map[string]string) url.Values {
	form := make(url.Values)
	for k, v := range values {
		form.Set(k, v)
	}

	return form
}

func blankObjFromForm(form url.Values) (lib.RegistrarObject, error) {
	objType := form.Get(ObjTypeParam)
	if objType == "" {
		return nil, fmt.Errorf("%s: %s", ValueMissingError, ObjTypeParam)
	}

	obj, err := lib.NewRegistrarObject(objType)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", ObjectNotSupportedError, err)
	}

	return obj, nil
}

func blankObjFromRoute(ctx Context) (lib.RegistrarObject, error) {
	return blankObjFromForm(ToForm(ctx.GetRouteVars()))
}

func objFromForm(db *lib.DBCache, values url.Values) (obj lib.RegistrarObject, err error) {
	obj, err = blankObjFromForm(values)
	if err != nil {
		return nil, err
	}

	idInput := values.Get(ObjIDParam)
	if idInput == "" {
		return nil, fmt.Errorf("%s: %s", ValueMissingError, ObjIDParam)
	}

	idParsed, err := strconv.ParseInt(idInput, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%s: (value of %s) %s", ParsingError, ObjIDParam, err)
	}

	err = db.FindByID(obj, idParsed)

	return obj, err
}

func objFromRoute(ctx Context) (lib.RegistrarObject, error) {
	return objFromForm(ctx.GetDB(), ToForm(ctx.GetRouteVars()))
}
