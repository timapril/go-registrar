package handler

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/op/go-logging"

	"github.com/timapril/go-registrar/lib"
)

// GetWorkHandlerAPI will generate a list of items that are pending work
// for the object type provided. The lists of items will be serialized
// into JSON before returning the data to the user.
func GetWorkHandlerAPI(w ResponseWriter, request *http.Request, ctx apiContext) (err error) {
	objType, err := getRouteVar(ctx, ObjTypeParam)
	if err != nil {
		return err
	}

	db := ctx.GetDB()

	var idList []int64
	var revisions []lib.APIRevisionHint
	switch objType {
	case lib.DomainType:
		idList, revisions, err = lib.GetWorkDomains(db)
	case lib.HostType:
		idList, revisions, err = lib.GetWorkHosts(db)
	case lib.ContactType:
		idList, revisions, err = lib.GetWorkContacts(db)
	default:
		return errors.New(ObjectNotSupportedError)
	}

	if err != nil {
		return err
	}

	ctx.LogRequest(logging.INFO, request.URL.String(), "GetWorkHandlerAPI", ctx.db.GetCacheStatsLog())

	return w.SendAPIResponse(lib.GenerateIDList(objType, idList, revisions))
}

// GetServerLocksWorkWeb will generate a list of domains that need to be locked
// or unlocked in order to operate on the domains.
func GetServerLocksWorkWeb(w ResponseWriter, request *http.Request, ctx webContext) (err error) {
	changeRequested := false
	db := ctx.GetDB()

	unlocks, locks, err := lib.GetServerLockChanges(db)
	if err != nil {
		return err
	}

	b := &bytes.Buffer{}
	writer := bufio.NewWriter(b)

	outCSV := csv.NewWriter(writer)

	err = outCSV.Write([]string{"AddOrRemove", "Domain", "Lock"})
	if err != nil {
		return err
	}

	for dom, locList := range locks {
		for _, lock := range locList {
			row := []string{}
			row = append(row, "add", dom, lock)
			err := outCSV.Write(row)
			if err != nil {
				return err
			}
			changeRequested = true
		}
	}
	for dom, locList := range unlocks {
		for _, lock := range locList {
			row := []string{}
			row = append(row, "remove", dom, lock)
			err := outCSV.Write(row)
			if err != nil {
				return err
			}
			changeRequested = true
		}
	}
	outCSV.Flush()

	if changeRequested {
		filename := fmt.Sprintf("serverLockChanges-%d.csv", time.Now().Unix())
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		w.Header().Set("Content-Type", "text/csv")
		_, writeErr := w.Write(b.Bytes())
		return writeErr
	}
	_, writeErr := fmt.Fprintf(w, "No server lock changes needed")
	return writeErr

}
