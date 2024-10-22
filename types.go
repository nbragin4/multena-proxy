package main

type LabelKey = string
type LabelVal = string
type Identifier = string

// type HashableACL = map[Identifier]HashableFilter
// type ACL = map[Identifier]Filter

type ACLs map[Identifier]map[LabelKey]map[LabelVal]bool

type LabelType = map[LabelVal]bool
type Filter = map[LabelKey]LabelType
