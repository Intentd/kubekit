// Package serviceaccounts builds and writes ServiceAccounts with Build,
// Create, Delete and DeleteIfOwned.
//
// There is no Apply: a ServiceAccount has no spec fields that drift after it
// is created. Call Create again to bring one back if it was deleted; an
// existing one is left as it is.
package serviceaccounts
