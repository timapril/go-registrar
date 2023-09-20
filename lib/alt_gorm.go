package lib

import (
	"reflect"

	"github.com/jinzhu/gorm"
)

// All of the functions below we derived from the gorm package https://github.com/jinzhu/gorm

/*The MIT License (MIT)

Copyright (c) 2013-NOW  Jinzhu <wosmvp@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.*/

// changeableField
// from gorm/scope.go.
func changeableField(scope *gorm.Scope, field *gorm.Field) bool {
	selectAttrs := scope.SelectAttrs()
	omitAttrs := scope.OmitAttrs()

	if len(selectAttrs) > 0 {
		for _, attr := range selectAttrs {
			if field.Name == attr || field.DBName == attr {
				return true
			}
		}

		return false
	}

	for _, attr := range omitAttrs {
		if field.Name == attr || field.DBName == attr {
			return false
		}
	}

	return !field.IsIgnored
}

// shouldSaveAssociations from gorm/scope.go.
func shouldSaveAssociations(scope *gorm.Scope) bool {
	saveAssociations, ok := scope.Get("gorm:save_associations")

	if ok {
		saveAssoc, typeAssertOK := saveAssociations.(bool)
		if typeAssertOK && !saveAssoc {
			return false
		}
	}

	return true && !scope.HasError()
}

// SaveBeforeAssociations from gorm/callback_shared.go, altered use our cycle handling Save function that only saves modified structs,
// (and trivially, our copies of their unexported shouldSaveAssociations and changeableField functions)
// Apart from that, a literal copy of the code gorm/callback_shared.go.
func SaveBeforeAssociations(scope *gorm.Scope) {
	if !shouldSaveAssociations(scope) { // altered to use our local function
		return
	}

	for _, field := range scope.Fields() {
		if changeableField(scope, field) && !field.IsBlank && !field.IsIgnored { // altered to use our local function
			if relationship := field.Relationship; relationship != nil && relationship.Kind == "belongs_to" {
				value := field.Field

				err := scope.Err(Save(scope.NewDB(), value.Addr().Interface()).Error) // altered to use our smarter Save function
				if err != nil {
					logger.Errorf("gorm_alt: %s", err.Error())
				}

				if len(relationship.ForeignFieldNames) != 0 {
					for idx, fieldName := range relationship.ForeignFieldNames {
						associationForeignName := relationship.AssociationForeignDBNames[idx]
						if f, ok := scope.New(value.Addr().Interface()).FieldByName(associationForeignName); ok {
							err := scope.Err(scope.SetColumn(fieldName, f.Field.Interface()))
							if err != nil {
								logger.Errorf("gorm_alt: %s", err.Error())
							}
						}
					}
				}
			}
		}
	}
}

// SaveAfterAssociations from gorm/callback_shared.go, altered use our cycle handling Save function that only saves modified structs,
// (and trivially, our copies of their unexported shouldSaveAssociations and changeableField functions)
// Apart from that, a literal copy of the code gorm/callback_shared.go.
func SaveAfterAssociations(scope *gorm.Scope) {
	if !shouldSaveAssociations(scope) { // altered to use our local function
		return
	}

	for _, field := range scope.Fields() {
		if changeableField(scope, field) && !field.IsBlank && !field.IsIgnored { // altered to use our local function
			if relationship := field.Relationship; relationship != nil &&
				(relationship.Kind == "has_one" || relationship.Kind == "has_many" || relationship.Kind == "many_to_many") {
				value := field.Field

				switch value.Kind() {
				case reflect.Slice:
					for i := 0; i < value.Len(); i++ {
						newDB := scope.NewDB()
						elem := value.Index(i).Addr().Interface()
						newScope := newDB.NewScope(elem)

						if relationship.JoinTableHandler == nil && len(relationship.ForeignFieldNames) != 0 {
							for idx, fieldName := range relationship.ForeignFieldNames {
								associationForeignName := relationship.AssociationForeignDBNames[idx]
								if f, ok := scope.FieldByName(associationForeignName); ok {
									err := scope.Err(newScope.SetColumn(fieldName, f.Field.Interface()))
									if err != nil {
										logger.Errorf("gorm_alt: %s", err.Error())
									}
								}
							}
						}

						if relationship.PolymorphicType != "" {
							err := scope.Err(newScope.SetColumn(relationship.PolymorphicType, scope.TableName()))
							if err != nil {
								logger.Errorf("gorm_alt: %s", err.Error())
							}
						}

						err := scope.Err(Save(newDB, elem).Error) // altered to use our smarter Save function
						if err != nil {
							logger.Errorf("gorm_alt: %s", err.Error())
						}

						if joinTableHandler := relationship.JoinTableHandler; joinTableHandler != nil {
							err := scope.Err(joinTableHandler.Add(joinTableHandler, scope.NewDB(), scope.Value, newScope.Value))
							if err != nil {
								logger.Errorf("gorm_alt: %s", err.Error())
							}
						}
					}
				default:
					elem := value.Addr().Interface()
					newScope := scope.New(elem)

					if len(relationship.ForeignFieldNames) != 0 {
						for idx, fieldName := range relationship.ForeignFieldNames {
							associationForeignName := relationship.AssociationForeignDBNames[idx]
							if f, ok := scope.FieldByName(associationForeignName); ok {
								err := scope.Err(newScope.SetColumn(fieldName, f.Field.Interface()))
								if err != nil {
									logger.Errorf("gorm_alt: %s", err.Error())
								}
							}
						}
					}

					if relationship.PolymorphicType != "" {
						err := scope.Err(newScope.SetColumn(relationship.PolymorphicType, scope.TableName()))
						if err != nil {
							logger.Errorf("gorm_alt: %s", err.Error())
						}
					}

					err := scope.Err(Save(scope.NewDB(), elem).Error) // altered to use our smarter Save function
					if err != nil {
						logger.Errorf("gorm_alt: %s", err.Error())
					}
				}
			}
		}
	}
}
