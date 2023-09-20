# Registrar Overview

This is a proof of concept Domain Name Registrar developed in response
to an increasing risk of domain name hijacking. The server and lib
subpackages implement a cryptographic signature (gpg) based audit trail 
for all changes to a Registrar's sponsored domains.

## History

The original source code for this project was developed at 
[Akamai Technologies, Inc.](https://akamai.com) and managed its domain
name portfolio for approximately five years. After the domains were
migrated to another registrar, Akamai allowed the project's primary
developer to release the software as open source.

# Packages

The registrar is broken down into a collection of sub packages listed
below.

# EPP

The EPP sub package implements the data structures used to communicate
with an EPP endpoint as defined in RFC5730.

# Lib

The lib sub package implements the signature based audit trail of
change requests used to operate the registrar.

# Client

The client lib is used to interact with the registrar in order to
create, update and approve objects within the registrar.

# Approver Client

In collaboration with the client package, the approver client
interacts with the registrar to approve object changes.

# Provision

The provision package is used to verify and communicate registrar
changes to the registry.

# Escrow Generator

The escrow generator is used to explore the current registrar
database in a RAA2013 compliant way for communicating to the
Registrar Data Provider.

# Server

The server package provides the user interface for the registrar,
both for the web UI an the API used to communicate with the
various clients.

# Handler

The handler package manages the various handlers required for the
server package.

# WHOIS

Whois is used to operate a WHOIS server for the registrar, basing
its information off the registrar database.

# WHOIS Generate

Generates an IRRd WHOIS data file used to load WHOIS information
into an IRRd server.

# WHOIS Client

Implements a whois client to gather information via the WHOIS
protocol.

# WHOIS Parse

Used to parse WHOIS query responses into a uniform data structure.

# EPP Pass Rotate

Can be used to rotate the passphrase used to authenticate to
the EPP server.

# Helper

A collection of helper tools which can be used to verify proposed
changes to registrar objects (Hosts, Domains and Contacts).
