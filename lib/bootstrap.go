// Package lib provides the objects required to operate registrar
package lib

import "database/sql"

// BootstrapRegistrar is used to preform the initial bootstrapping of
// the registrar system. The bootstrap process will ensure that the
// database is configured properly and then initialize the required
// approver and approver sets to start the system correctly.
func BootstrapRegistrar(dbCache *DBCache, conf Config) (err error) {
	// Used if there are issues bootstrapping
	// testApprover := Approver{}
	// testApprover.SetID(1)
	// testErr := db.Find(&testApprover)
	//
	// testApproverSet := ApproverSet{}
	// testApproverSet.SetID(1)
	// testErr2 := db.Find(&testApproverSet)
	//
	// if testErr == nil && testErr2 == nil {
	// 	if testApprover.CurrentRevision.RevisionState == StateBootstrap || testApproverSet.CurrentRevision.RevisionState == StateBootstrap {
	// 		log.Debug("Resetting DB")
	// 		db.DB.Exec("drop table api_user_revisions")
	// 		db.DB.Exec("drop table api_users")
	// 		db.DB.Exec("drop table approvals")
	// 		db.DB.Exec("drop table approver_revisions")
	// 		db.DB.Exec("drop table approver_set_revisions")
	// 		db.DB.Exec("drop table approver_sets")
	// 		db.DB.Exec("drop table approver_to_revision_set")
	// 		db.DB.Exec("drop table approvers")
	// 		db.DB.Exec("drop table change_requests")
	// 		db.DB.Exec("drop table contact_revisions")
	// 		db.DB.Exec("drop table contacts")
	// 		db.DB.Exec("drop table d_s_data_entries")
	// 		db.DB.Exec("drop table d_s_data_entry_epps")
	// 		db.DB.Exec("drop table domain_revisions")
	// 		db.DB.Exec("drop table domains")
	// 		db.DB.Exec("drop table host_address_epps")
	// 		db.DB.Exec("drop table host_addresses")
	// 		db.DB.Exec("drop table host_revisions")
	// 		db.DB.Exec("drop table host_to_domainrevision")
	// 		db.DB.Exec("drop table hosts")
	// 		db.DB.Exec("drop table informed_approverset_to_apiuserrevision")
	// 		db.DB.Exec("drop table informed_approverset_to_approverrevision")
	// 		db.DB.Exec("drop table informed_approverset_to_approversetrevision")
	// 		db.DB.Exec("drop table informed_approverset_to_contactrevision")
	// 		db.DB.Exec("drop table informed_approverset_to_domainrevision")
	// 		db.DB.Exec("drop table informed_approverset_to_hostrevision")
	// 		db.DB.Exec("drop table ip_whitelist_revisions")
	// 		db.DB.Exec("drop table liveness_checks")
	// 		db.DB.Exec("drop table protected_hostname_list_revisions")
	// 		db.DB.Exec("drop table required_approverset_to_apiuserrevision")
	// 		db.DB.Exec("drop table required_approverset_to_approverrevision")
	// 		db.DB.Exec("drop table required_approverset_to_approversetrevision")
	// 		db.DB.Exec("drop table required_approverset_to_contactrevision")
	// 		db.DB.Exec("drop table required_approverset_to_domainrevision")
	// 		db.DB.Exec("drop table required_approverset_to_hostrevision")
	// 		db.DB.Exec("drop table e_p_p_actions")
	// 		db.DB.Exec("drop table e_p_p_encrypted_passphrases")
	// 		db.DB.Exec("drop table e_p_p_runs")
	// 	}
	// }
	// .
	MigrateDBApprover(dbCache)
	MigrateDBApproverRevision(dbCache)
	MigrateDBApproverSet(dbCache)
	MigrateDBApproverSetRevision(dbCache)
	MigrateDBChangeRequest(dbCache)
	MigrateDBApproval(dbCache)
	MigrateDBContact(dbCache)
	MigrateDBContactRevision(dbCache)
	MigrateDBHost(dbCache)
	MigrateDBHostRevision(dbCache)
	MigrateDBDomain(dbCache)
	MigrateDBDomainRevision(dbCache)
	MigrateDBAPIUser(dbCache)
	MigrateDBAPIUserRevision(dbCache)
	MigrateDBControls(dbCache)
	MigrateDBLivenessCheck(dbCache)
	MigrateEPPActionLog(dbCache)

	var count int64

	dbCache.DB.Model(ApproverRevision{}).Count(&count)
	// db.Model(ApproverSetRevision{}).Where("id=1").Count(&count)

	if count == 0 {
		// Create the default approver
		rootApprover := Approver{
			State:     StateActive,
			CreatedBy: conf.Bootstrap.Username,
			CreatedAt: TimeNow(),
			UpdatedBy: conf.Bootstrap.Username,
			UpdatedAt: TimeNow(),
		}
		// Save the default approver (some fields are missing but they will
		// be populated as the fields come into existence)
		if err = dbCache.Save(&rootApprover); err != nil {
			return err
		}

		// Create a revision that will be pushed into the active state but
		// approved after the fact
		rootApproverRev := ApproverRevision{
			ApproverID:    rootApprover.ID,
			RevisionState: StateBootstrap,
			DesiredState:  StateBootstrap,
			Name:          conf.Bootstrap.Name,
			EmailAddress:  conf.Bootstrap.EmailAddress,
			Role:          conf.Bootstrap.Role,
			Username:      conf.Bootstrap.Username,
			EmployeeID:    conf.Bootstrap.EmployeeID,
			Department:    conf.Bootstrap.Department,
			Fingerprint:   conf.Bootstrap.Fingerprint,
			PublicKey:     string(conf.Bootstrap.PubkeyContents),
			CreatedBy:     conf.Bootstrap.Username,
			CreatedAt:     TimeNow(),
			UpdatedBy:     conf.Bootstrap.Username,
			UpdatedAt:     TimeNow(),
		}
		rootApproverRevNew := ApproverRevision{
			ApproverID:    rootApprover.ID,
			RevisionState: StateNew,
			DesiredState:  StateActive,
			Name:          conf.Bootstrap.Name,
			EmailAddress:  conf.Bootstrap.EmailAddress,
			Role:          conf.Bootstrap.Role,
			Username:      conf.Bootstrap.Username,
			EmployeeID:    conf.Bootstrap.EmployeeID,
			Department:    conf.Bootstrap.Department,
			Fingerprint:   conf.Bootstrap.Fingerprint,
			PublicKey:     string(conf.Bootstrap.PubkeyContents),
			CreatedBy:     conf.Bootstrap.Username,
			CreatedAt:     TimeNow(),
			UpdatedBy:     conf.Bootstrap.Username,
			UpdatedAt:     TimeNow(),
		}

		// Save the first revision (some fields are missing but they will
		// be populated as the fields come into existence)
		if err = dbCache.Save(&rootApproverRev); err != nil {
			return err
		}

		if err = dbCache.Save(&rootApproverRevNew); err != nil {
			return err
		}

		// Add the approver revision to the default approver now that it is
		// created
		rootApprover.CurrentRevision = rootApproverRev
		rootApprover.CurrentRevisionID = sql.NullInt64{
			Valid: true,
			Int64: rootApproverRev.ID,
		}

		if err = dbCache.Save(&rootApprover); err != nil {
			return err
		}

		// create the default approver set
		rootApproverSet := ApproverSet{
			State:     StateActive,
			CreatedBy: conf.Bootstrap.Username,
			CreatedAt: TimeNow(),
		}
		// Save the first revision (some fields are missing but they will
		// be populated as the fields come into existence)
		if err = dbCache.Save(&rootApproverSet); err != nil {
			return err
		}

		// create the default approver set's first revision. The revision
		// will be pushed into the active state but approved after the fact
		rootApproverSetRev := ApproverSetRevision{
			ApproverSetID: rootApproverSet.ID,
			RevisionState: StateBootstrap,
			DesiredState:  StateBootstrap,
			Title:         conf.Bootstrap.DefaultSetTitle,
			Description:   conf.Bootstrap.DefaultSetDescription,
			CreatedBy:     conf.Bootstrap.Username,
			CreatedAt:     TimeNow(),
		}
		rootApproverSetRevNew := ApproverSetRevision{
			ApproverSetID: rootApproverSet.ID,
			RevisionState: StateNew,
			DesiredState:  StateActive,
			Title:         conf.Bootstrap.DefaultSetTitle,
			Description:   conf.Bootstrap.DefaultSetDescription,
			CreatedBy:     conf.Bootstrap.Username,
			CreatedAt:     TimeNow(),
		}
		// Add the default approver to the set of approvers
		rootApproverSetRev.Approvers = append(rootApproverSetRev.Approvers,
			rootApprover)
		rootApproverSetRevNew.Approvers = append(rootApproverSetRev.Approvers,
			rootApprover)
		// Add itself to the required approver sets
		rootApproverSetRev.RequiredApproverSets = append(rootApproverSetRev.RequiredApproverSets,
			rootApproverSet)
		rootApproverSetRevNew.RequiredApproverSets = append(rootApproverSetRev.RequiredApproverSets,
			rootApproverSet)

		// Save the first revision (some fields are missing but they will
		// be populated as the fields come into existence)
		if err = dbCache.Save(&rootApproverSetRev); err != nil {
			return err
		}

		if err = dbCache.Save(&rootApproverSetRevNew); err != nil {
			return err
		}

		// Add the approver set revision to the default approver now that it
		// is created
		rootApproverSet.CurrentRevision = rootApproverSetRev
		rootApproverSet.CurrentRevisionID = sql.NullInt64{
			Valid: true,
			Int64: rootApproverSetRev.ID,
		}

		for appSet := range rootApproverSetRev.RequiredApproverSets {
			rootApproverSetRev.RequiredApproverSets[appSet].CurrentRevisionID = sql.NullInt64{
				Valid: true,
				Int64: rootApproverSetRev.ID,
			}
		}

		for appSet := range rootApproverSetRevNew.RequiredApproverSets {
			rootApproverSetRevNew.RequiredApproverSets[appSet].CurrentRevisionID = sql.NullInt64{
				Valid: true,
				Int64: rootApproverSetRev.ID,
			}
		}

		if err = dbCache.Save(&rootApproverSet); err != nil {
			return err
		}

		// Add the approver set to the approver revision now that it is
		// created
		rootApproverRev.RequiredApproverSets = append(rootApproverRev.RequiredApproverSets,
			rootApproverSet)

		if err = dbCache.Save(&rootApproverRev); err != nil {
			return err
		}

		rootApproverRevNew.RequiredApproverSets = append(rootApproverRev.RequiredApproverSets,
			rootApproverSet)

		err = dbCache.Save(&rootApproverRevNew)

		return err
	}

	return nil
}
