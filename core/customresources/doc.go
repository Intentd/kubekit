// Package customresources builds and writes custom resources, or any kind
// without a typed package, with the same verbs as the typed packages: Build,
// Create, Apply and DeleteIfOwned. It works through a dynamic.Interface and
// a meta.RESTMapper, and each write takes an optional function that changes
// the object before it is written.
package customresources
