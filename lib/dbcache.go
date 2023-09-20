package lib

import (
	"errors"
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
)

// DBCache is a write through caching layer for the registrar system to
// allow objects to be cached on the server side for each request rather than
// having to contact the database for each request.
type DBCache struct {
	DB *gorm.DB

	CacheHits   int64
	CacheMisses int64

	Approvers         map[int64]*Approver
	ApproverRevisions map[int64]*ApproverRevision

	ApproverSets         map[int64]*ApproverSet
	ApproverSetRevisions map[int64]*ApproverSetRevision

	APIUsers         map[int64]*APIUser
	APIUserRevisions map[int64]*APIUserRevision

	ChangeRequests map[int64]*ChangeRequest
	Approvals      map[int64]*Approval

	Domains         map[int64]*Domain
	DomainRevisions map[int64]*DomainRevision

	Hosts         map[int64]*Host
	HostRevisions map[int64]*HostRevision

	Contacts         map[int64]*Contact
	ContactRevisions map[int64]*ContactRevision
}

// NewDBCache will create a new DBCache object from the provided db object.
func NewDBCache(db *gorm.DB) DBCache {
	dbc := DBCache{}
	dbc.DB = db
	dbc.WipeCache()

	return dbc
}

// WipeCache is used to initialize or clear the DBCache object. Cases where you
// would want to include this call include: new dbcaches or clearing the cache
// after a save.
func (dbc *DBCache) WipeCache() {
	dbc.Approvers = make(map[int64]*Approver)
	dbc.ApproverRevisions = make(map[int64]*ApproverRevision)
	dbc.ApproverSets = make(map[int64]*ApproverSet)
	dbc.ApproverSetRevisions = make(map[int64]*ApproverSetRevision)
	dbc.APIUsers = make(map[int64]*APIUser)
	dbc.APIUserRevisions = make(map[int64]*APIUserRevision)
	dbc.ChangeRequests = make(map[int64]*ChangeRequest)
	dbc.Approvals = make(map[int64]*Approval)
	dbc.Domains = make(map[int64]*Domain)
	dbc.DomainRevisions = make(map[int64]*DomainRevision)
	dbc.Hosts = make(map[int64]*Host)
	dbc.HostRevisions = make(map[int64]*HostRevision)
	dbc.Contacts = make(map[int64]*Contact)
	dbc.ContactRevisions = make(map[int64]*ContactRevision)
}

// GetRevisionAtTime will look up the revision at a the time provided with the
// with the parentID matching. If an error occurs, it will be returned.
func (dbc *DBCache) GetRevisionAtTime(object RegistrarObject, parentID int64, timestamp int64) (err error) {
	timestampOfRequest := time.Unix(timestamp, 0)

	startTime := time.Unix(0, 0)

	switch object.(type) {
	case *APIUserRevision:
		err = dbc.DB.Where("api_user_id = ? and promoted_time <= ? and promoted_time > ?", parentID, timestampOfRequest, startTime).Order("promoted_time desc").First(object).Error
	case *ApproverRevision:
		err = dbc.DB.Where("approver_id = ? and promoted_time <= ? and promoted_time > ?", parentID, timestampOfRequest, startTime).Order("promoted_time desc").First(object).Error
	case *ApproverSetRevision:
		err = dbc.DB.Where("approver_set_id = ? and promoted_time <= ? and promoted_time > ?", parentID, timestampOfRequest, startTime).Order("promoted_time desc").First(object).Error
	case *ContactRevision:
		err = dbc.DB.Where("contact_id = ? and promoted_time <= ? and promoted_time > ?", parentID, timestampOfRequest, startTime).Order("promoted_time desc").First(object).Error
	case *DomainRevision:
		err = dbc.DB.Where("domain_id = ? and promoted_time <= ? and promoted_time > ?", parentID, timestampOfRequest, startTime).Order("promoted_time desc").First(object).Error
	case *HostRevision:
		err = dbc.DB.Where("host_id = ? and promoted_time <= ? and promoted_time > ?", parentID, timestampOfRequest, startTime).Order("promoted_time desc").First(object).Error
	}

	return err
}

// GetNewAndPendingRevisions will query for the first revision for the object
// that is in the new or pending approval state.
func (dbc *DBCache) GetNewAndPendingRevisions(object RegistrarParent) (err error) {
	switch object.(type) {
	case *APIUser:
		err = dbc.DB.Where("api_user_id = ? and revision_state in (?, ?)", object.GetID(), StateNew, StatePendingApproval).First(object.GetPendingRevision()).Error
	case *Approver:
		err = dbc.DB.Where("approver_id = ? and revision_state in (?, ?)", object.GetID(), StateNew, StatePendingApproval).First(object.GetPendingRevision()).Error
	case *ApproverSet:
		err = dbc.DB.Where("approver_set_id = ? and revision_state in (?, ?)", object.GetID(), StateNew, StatePendingApproval).First(object.GetPendingRevision()).Error
	case *Contact:
		err = dbc.DB.Where("contact_id = ? and revision_state in (?, ?)", object.GetID(), StateNew, StatePendingApproval).First(object.GetPendingRevision()).Error
	case *Domain:
		err = dbc.DB.Where("domain_id = ? and revision_state in (?, ?)", object.GetID(), StateNew, StatePendingApproval).First(object.GetPendingRevision()).Error
	case *Host:
		err = dbc.DB.Where("host_id = ? and revision_state in (?, ?)", object.GetID(), StateNew, StatePendingApproval).First(object.GetPendingRevision()).Error
	}

	return err
}

// FindAll will query for all objects for a given type.
func (dbc *DBCache) FindAll(value interface{}) (err error) {
	err = dbc.DB.Find(value).Order("id").Error
	if err != nil {
		logger.Error(err)
	}

	return err
}

// Save will attempt to save the provided object at the same time as pushing the
// object into the cache.
func (dbc *DBCache) Save(rawObject interface{}) error {
	dbc.WipeCache()

	// Save will just wipe the cache rathern than worry about save issues, leaving
	// around in the event that more problems are found with throughput
	//
	// switch rawObject.(type) {
	// case *Approver:
	// 	object := rawObject.(*Approver)
	// 	dbc.Approvers[object.GetID()] = object
	// case *ApproverRevision:
	// 	object := rawObject.(*ApproverRevision)
	// 	dbc.ApproverRevisions[object.GetID()] = object
	// case *ApproverSet:
	// 	object := rawObject.(*ApproverSet)
	// 	dbc.ApproverSets[object.GetID()] = object
	// case *ApproverSetRevision:
	// 	object := rawObject.(*ApproverSetRevision)
	// 	dbc.ApproverSetRevisions[object.GetID()] = object
	// case *APIUser:
	// 	object := rawObject.(*APIUser)
	// 	dbc.APIUsers[object.GetID()] = object
	// case *APIUserRevision:
	// 	object := rawObject.(*APIUserRevision)
	// 	dbc.APIUserRevisions[object.GetID()] = object
	// case *ChangeRequest:
	// 	object := rawObject.(*ChangeRequest)
	// 	dbc.ChangeRequests[object.GetID()] = object
	// case *Approval:
	// 	object := rawObject.(*Approval)
	// 	dbc.Approvals[object.GetID()] = object
	// case *Domain:
	// 	object := rawObject.(*Domain)
	// 	dbc.Domains[object.GetID()] = object
	// case *DomainRevision:
	// 	object := rawObject.(*DomainRevision)
	// 	dbc.DomainRevisions[object.GetID()] = object
	// case *Host:
	// 	object := rawObject.(*Host)
	// 	dbc.Hosts[object.GetID()] = object
	// case *HostRevision:
	// 	object := rawObject.(*HostRevision)
	// 	dbc.HostRevisions[object.GetID()] = object
	// case *Contact:
	// 	object := rawObject.(*Contact)
	// 	dbc.Contacts[object.GetID()] = object
	// case *ContactRevision:
	// 	object := rawObject.(*ContactRevision)
	// 	dbc.ContactRevisions[object.GetID()] = object
	// default:
	// 	panic(errors.New("Unknown object type"))
	// }

	err := Save(dbc.DB, rawObject).Error

	return err
}

// Update will update columns of the target object with the vales in the update
// fields object and then remove the object from the cache to indciate that the
// cache is not valid for that object.
func (dbc *DBCache) Update(targetObject interface{}, updateFields interface{}) error {
	if err := dbc.DB.Model(targetObject).UpdateColumn(updateFields).Error; err != nil {
		return err
	}

	dbc.InvalidateObject(targetObject)
	dbc.InvalidateObject(updateFields)

	return nil
}

// Related will attempt to locate the related objects for a provided target
// object.
func (dbc *DBCache) Related(targetObject interface{}, relatedObject interface{}) error {
	err := dbc.DB.Model(targetObject).Related(relatedObject).Error

	return err
}

// InvalidateObject will remove the given object from the cache if it exists.
func (dbc *DBCache) InvalidateObject(_ interface{}) {
	dbc.WipeCache()
}

// AutoMigrate is used to ensure that the data types are available in the
// selected storage mechanism.
func (dbc *DBCache) AutoMigrate(value interface{}) {
	dbc.DB.AutoMigrate(value)
}

// FindByID will try and find the object type provided with the given id and
// return it. If there is an error setting the ID or finding the object, the
// error will be returned.
func (dbc *DBCache) FindByID(inobj interface{}, objID int64) error {
	regObject, ok := inobj.(RegistrarObject)
	if !ok {
		return errors.New("FindByID is only supported for objects that implement RegistrarObject")
	}

	if err := regObject.SetID(objID); err != nil {
		return fmt.Errorf("error setting object id: %w", err)
	}

	return dbc.Find(inobj)
}

// Purge will ensure that any object with a matching type and ID is purged
// from the cache.
func (dbc *DBCache) Purge(inobj interface{}) error {
	switch typedObject := inobj.(type) {
	case *Approver:
		delete(dbc.Approvers, typedObject.GetID())
	case *ApproverRevision:
		delete(dbc.ApproverRevisions, typedObject.GetID())
	case *ApproverSet:
		delete(dbc.ApproverSets, typedObject.GetID())
	case *ApproverSetRevision:
		delete(dbc.ApproverSetRevisions, typedObject.GetID())
	case *APIUser:
		delete(dbc.APIUsers, typedObject.GetID())
	case *APIUserRevision:
		delete(dbc.APIUserRevisions, typedObject.GetID())
	case *ChangeRequest:
		delete(dbc.ChangeRequests, typedObject.GetID())
	case *Approval:
		delete(dbc.Approvals, typedObject.GetID())
	case *Domain:
		delete(dbc.Domains, typedObject.GetID())
	case *DomainRevision:
		delete(dbc.DomainRevisions, typedObject.GetID())
	case *Host:
		delete(dbc.Hosts, typedObject.GetID())
	case *HostRevision:
		delete(dbc.HostRevisions, typedObject.GetID())
	case *Contact:
		delete(dbc.Contacts, typedObject.GetID())
	case *ContactRevision:
		delete(dbc.ContactRevisions, typedObject.GetID())
	default:
		return fmt.Errorf("unsupported Type: %s", typedObject)
	}

	return nil
}

// Find will attempt to locate the object of the type passed with the id that is
// set. If the object is valid in cache, it will be returned from cache,
// otherwise it will be retrieved from the database, stored in the cache and
// returned.
func (dbc *DBCache) Find(inobj interface{}) error {
	switch typedObject := inobj.(type) {
	case *Approver:
		if pt, ok := dbc.Approvers[typedObject.GetID()]; ok {
			dbc.CacheHits++

			*typedObject = *pt

			return nil
		}
	case *ApproverRevision:
		if pt, ok := dbc.ApproverRevisions[typedObject.GetID()]; ok {
			dbc.CacheHits++

			*typedObject = *pt

			return nil
		}
	case *ApproverSet:
		if pt, ok := dbc.ApproverSets[typedObject.GetID()]; ok {
			dbc.CacheHits++

			*typedObject = *pt

			return nil
		}
	case *ApproverSetRevision:
		if pt, ok := dbc.ApproverSetRevisions[typedObject.GetID()]; ok {
			dbc.CacheHits++

			*typedObject = *pt

			return nil
		}
	case *APIUser:
		if pt, ok := dbc.APIUsers[typedObject.GetID()]; ok {
			dbc.CacheHits++

			*typedObject = *pt

			return nil
		}
	case *APIUserRevision:
		if pt, ok := dbc.APIUserRevisions[typedObject.GetID()]; ok {
			dbc.CacheHits++

			*typedObject = *pt

			return nil
		}
	case *ChangeRequest:
		if pt, ok := dbc.ChangeRequests[typedObject.GetID()]; ok {
			dbc.CacheHits++

			*typedObject = *pt

			return nil
		}
	case *Approval:
		if pt, ok := dbc.Approvals[typedObject.GetID()]; ok {
			dbc.CacheHits++

			*typedObject = *pt

			return nil
		}
	case *Domain:
		if pt, ok := dbc.Domains[typedObject.GetID()]; ok {
			dbc.CacheHits++

			*typedObject = *pt

			return nil
		}
	case *DomainRevision:
		if pt, ok := dbc.DomainRevisions[typedObject.GetID()]; ok {
			dbc.CacheHits++

			*typedObject = *pt

			return nil
		}
	case *Host:
		if pt, ok := dbc.Hosts[typedObject.GetID()]; ok {
			dbc.CacheHits++

			*typedObject = *pt

			return nil
		}
	case *HostRevision:
		if pt, ok := dbc.HostRevisions[typedObject.GetID()]; ok {
			dbc.CacheHits++

			*typedObject = *pt

			return nil
		}
	case *Contact:
		if pt, ok := dbc.Contacts[typedObject.GetID()]; ok {
			dbc.CacheHits++

			*typedObject = *pt

			return nil
		}
	case *ContactRevision:
		if pt, ok := dbc.ContactRevisions[typedObject.GetID()]; ok {
			dbc.CacheHits++

			*typedObject = *pt

			return nil
		}
	default:
		return errors.New("unsupported Type")
	}

	err := dbc.DB.First(inobj).Error
	if err != nil {
		return err
	}
	dbc.CacheMisses++

	if obj, ok := inobj.(RegistrarObject); ok {
		if err = obj.Prepare(dbc); err != nil {
			return fmt.Errorf("error preparing object: %w", err)
		}
	} else {
		return errors.New("unable to prepare object, not an RegistrarObject")
	}

	switch typedObject := inobj.(type) {
	case *Approver:
		var toSave Approver
		toSave = *typedObject
		dbc.Approvers[typedObject.GetID()] = &toSave
	case *ApproverRevision:
		var toSave ApproverRevision
		toSave = *typedObject
		dbc.ApproverRevisions[toSave.GetID()] = &toSave
	case *ApproverSet:
		var toSave ApproverSet
		toSave = *typedObject
		dbc.ApproverSets[typedObject.GetID()] = &toSave
	case *ApproverSetRevision:
		var toSave ApproverSetRevision
		toSave = *typedObject
		dbc.ApproverSetRevisions[typedObject.GetID()] = &toSave
	case *APIUser:
		var toSave APIUser
		toSave = *typedObject
		dbc.APIUsers[typedObject.GetID()] = &toSave
	case *APIUserRevision:
		var toSave APIUserRevision
		toSave = *typedObject
		dbc.APIUserRevisions[typedObject.GetID()] = &toSave
	case *ChangeRequest:
		var toSave ChangeRequest
		toSave = *typedObject
		dbc.ChangeRequests[typedObject.GetID()] = &toSave
	case *Approval:
		var toSave Approval
		toSave = *typedObject
		dbc.Approvals[typedObject.GetID()] = &toSave
	case *Domain:
		var toSave Domain
		toSave = *typedObject
		dbc.Domains[typedObject.GetID()] = &toSave
	case *DomainRevision:
		var toSave DomainRevision
		toSave = *typedObject
		dbc.DomainRevisions[typedObject.GetID()] = &toSave
	case *Host:
		var toSave Host
		toSave = *typedObject
		dbc.Hosts[typedObject.GetID()] = &toSave
	case *HostRevision:
		var toSave HostRevision
		toSave = *typedObject
		dbc.HostRevisions[typedObject.GetID()] = &toSave
	case *Contact:
		var toSave Contact
		toSave = *typedObject
		dbc.Contacts[typedObject.GetID()] = &toSave
	case *ContactRevision:
		var toSave ContactRevision
		toSave = *typedObject
		dbc.ContactRevisions[typedObject.GetID()] = &toSave
	}

	return nil
}

// GetCacheStatsLog returns a message that indicates the status of the database
// cache that can be used in logging.
func (dbc *DBCache) GetCacheStatsLog() string {
	return fmt.Sprintf("H:%d M:%d", dbc.CacheHits, dbc.CacheMisses)
}

// DBCacheFactory is used to generate new DBCache objects for requests.
type DBCacheFactory struct {
	db *gorm.DB
}

// NewDBCacheFactory will generate and return a new DBCacheFactory using the
// db objedct passed to initialize the factory.
func NewDBCacheFactory(db *gorm.DB) *DBCacheFactory {
	factory := &DBCacheFactory{}
	factory.db = db

	return factory
}

// GetNewDBCache is used to generate and return a new DBCache object.
func (f *DBCacheFactory) GetNewDBCache() *DBCache {
	cache := NewDBCache(f.db)

	return &cache
}
