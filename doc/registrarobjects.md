# Registrar Interface

The Registrar Interface is an interface that most of the main
objects within the registrar code base will implement. The
interface enforces methods that will allow for modularity of the
server and client software when handeling the process.

There are fields that should be present in all objects that implement
the registrar interface and they will be defined below.

## Fields for All Registrar Objects

### CreatedAt

The timestamp that the object was created

### CreatedBy

The user who created the object, this field is based on the user who
made no the change, not always the user who requested the change.

### UpdatedAt

The timestamp that the object was updated

### UpdatedBy

The user who last updated the object, this field is based on the user
who made no the change, not always the user who requested the change.

## Fields for All Registrar Revisable Objects

### CurrentRevision

The current revision of the object if there is one available.

### Revisions

An array of all revisions that exist for the object

### PendingRevision

The current pending revision of the object if there is one available.

## Fields for All Registrar Revisions Objects

### RequiredApproverSets

The list of approver sets that will be required to submit an approval
for the parent object, if the revision is the current active revision.

### InformedApproverSets

The list of approvers that will be informed when a change request is
submitted for the parent object, if the revision is the current active
revision.

### CR

The Change Request that is associated with the change if there is one
that has been created.

## TODO
* Look into `gorm:"polymorphic:Owner;"` for approver set mappings
* Version export formats
