package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
)

// IPAllowListRevision represents a list of IPs that correspond to NameServers
// in control of the registrar.
type IPAllowListRevision struct {
	Model

	IPsJSON []byte `sql:"type:text"`

	IsActive bool

	RefreshedAt time.Time
	RefreshedBy string

	CreatedAt time.Time `json:"CreatedAt"`
	CreatedBy string    `json:"CreatedBy"`
}

// GetIPAllowList will try to retrieve the current Host IP allow list from
// the database and return it. If an error occurs, it will be returned
//
// TODO: Consider moving the db query into the dbcache object.
func GetIPAllowList(dbCache *DBCache) (ips []string, err error) {
	rev := IPAllowListRevision{}

	err = dbCache.DB.Where(&IPAllowListRevision{IsActive: true}).Order("created_at desc").First(&rev).Error

	if err != nil {
		logger.Error(err)

		return ips, err
	}

	err = json.Unmarshal(rev.IPsJSON, &ips)
	if err != nil {
		return
	}

	return ips, nil
}

// CreateNewIPAllowListRevision is used to create a new Host IP allow list
// in the database and if an error occurs during creation, it will be returned.
func CreateNewIPAllowListRevision(dbCache *DBCache, ips []string, username string) (err error) {
	newRev := IPAllowListRevision{}
	newRev.IsActive = true
	newRev.CreatedBy = username
	newRev.CreatedAt = time.Now()
	newRev.RefreshedBy = username
	newRev.RefreshedAt = time.Now()
	newRev.IPsJSON, err = json.Marshal(ips)

	if err != nil {
		return fmt.Errorf("unarshal err: %w", err)
	}

	return dbCache.Save(&newRev)
}

// SetIPAllowList will attempt to set the host ip allow list in the
// database and if the list is not set it will be created.
//
// TODO: Consider moving the db query into the dbcache object.
func SetIPAllowList(dbCache *DBCache, ips []string, username string) (err error) {
	rev := IPAllowListRevision{}

	currErr := dbCache.DB.Where(&IPAllowListRevision{IsActive: false}).Order("created_at desc").First(&rev).Error

	if currErr != nil && errors.Is(err, gorm.RecordNotFound) {
		// record not found, try to create
		logger.Info("Creating Host IP Allow List")

		return CreateNewIPAllowListRevision(dbCache, ips, username)
	} else if currErr != nil {
		return currErr
	}

	currentIPs := []string{}

	unmarshalErr := json.Unmarshal(rev.IPsJSON, &currentIPs)
	if unmarshalErr != nil {
		return fmt.Errorf("unarshal err: %w", unmarshalErr)
	}

	IPsMap := make(map[string]int64)

	for _, cidr := range currentIPs {
		if val, exist := IPsMap[cidr]; exist {
			IPsMap[cidr] = val + 1
		} else {
			IPsMap[cidr] = 1
		}
	}

	for _, cidr := range ips {
		if val, exist := IPsMap[cidr]; exist {
			IPsMap[cidr] = val + -1
		} else {
			IPsMap[cidr] = -1
		}
	}

	isSame := true

	for _, val := range IPsMap {
		if val != 0 {
			isSame = false

			break
		}
	}

	if isSame {
		logger.Info("Uploaded IP Allow List is the same")

		rev.RefreshedAt = time.Now()
		rev.RefreshedBy = username
		rev.IsActive = true

		return dbCache.Save(&rev)
	}

	logger.Info("New IP Allow List Uploaded")

	rev.IsActive = false

	saveErr := dbCache.Save(&rev)
	if saveErr != nil {
		return saveErr
	}

	return CreateNewIPAllowListRevision(dbCache, ips, username)
}

// ProtectedHostnameListRevision is a list of domains that encourage extra
// attention when changes are being made.
type ProtectedHostnameListRevision struct {
	Model

	DomainsJSON []byte `sql:"type:text"`

	IsActive bool

	RefreshedAt time.Time
	RefreshedBy string

	CreatedAt time.Time `json:"CreatedAt"`
	CreatedBy string    `json:"CreatedBy"`
}

// GetProtectedDomainList will try to retrieve the current Protected
// Domain list from the database and return it. If an error occurs, it will be
// returned.
//
// TODO: Consider moving the db query into the dbcache object.
func GetProtectedDomainList(dbCache *DBCache) (domains []string, err error) {
	rev := ProtectedHostnameListRevision{}

	err = dbCache.DB.Where(&ProtectedHostnameListRevision{IsActive: true}).Order("created_at desc").First(&rev).Error

	if err != nil {
		logger.Error(err)

		return domains, err
	}

	err = json.Unmarshal(rev.DomainsJSON, &domains)
	if err != nil {
		return
	}

	return domains, nil
}

// CreateNewProtectedDomainList is used to create a new Protected domain list
// in the database and if an error occurs during creation, it will be returned.
func CreateNewProtectedDomainList(dbCache *DBCache, domains []string, username string) (err error) {
	newRev := ProtectedHostnameListRevision{}
	newRev.IsActive = true
	newRev.CreatedBy = username
	newRev.CreatedAt = time.Now()
	newRev.RefreshedBy = username
	newRev.RefreshedAt = time.Now()
	newRev.DomainsJSON, err = json.Marshal(domains)

	if err != nil {
		return fmt.Errorf("unmarshal error: %w", err)
	}

	return dbCache.Save(&newRev)
}

// SetProtectedDomainList will attempt to set the protected domain list
// in the database and if the list is not set it will be created
//
// TODO: Consider moving the db query into the dbcache object.
func SetProtectedDomainList(dbCache *DBCache, domains []string, username string) (err error) {
	rev := ProtectedHostnameListRevision{}

	currErr := dbCache.DB.Where(&ProtectedHostnameListRevision{IsActive: false}).Order("created_at desc").First(&rev).Error

	if currErr != nil && errors.Is(err, gorm.RecordNotFound) {
		// record not found, try to create
		logger.Info("Creating Protected Domain List")

		return CreateNewProtectedDomainList(dbCache, domains, username)
	} else if currErr != nil {
		return currErr
	}

	currentDomains := []string{}

	unmarshalErr := json.Unmarshal(rev.DomainsJSON, &currentDomains)
	if unmarshalErr != nil {
		return fmt.Errorf("unarshal err: %w", unmarshalErr)
	}

	DomainMap := make(map[string]int64)

	for _, domain := range currentDomains {
		if val, exist := DomainMap[domain]; exist {
			DomainMap[domain] = val + 1
		} else {
			DomainMap[domain] = 1
		}
	}

	for _, domain := range domains {
		if val, exist := DomainMap[domain]; exist {
			DomainMap[domain] = val + -1
		} else {
			DomainMap[domain] = -1
		}
	}

	isSame := true

	for _, val := range DomainMap {
		if val != 0 {
			isSame = false

			break
		}
	}

	if isSame {
		logger.Info("Uploaded Portected Domain List is the same")

		rev.RefreshedAt = time.Now()
		rev.RefreshedBy = username
		rev.IsActive = true

		return dbCache.Save(&rev)
	}

	logger.Info("New Portected Domain List Uploaded")

	rev.IsActive = false
	saveErr := dbCache.Save(&rev)

	if saveErr != nil {
		return saveErr
	}

	return CreateNewProtectedDomainList(dbCache, domains, username)
}

// MigrateDBControls will run the automigrate function for the separate controls
// that have been added.
func MigrateDBControls(dbCache *DBCache) {
	dbCache.AutoMigrate(&IPAllowListRevision{})
	dbCache.AutoMigrate(&ProtectedHostnameListRevision{})
}
