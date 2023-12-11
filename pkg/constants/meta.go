package constants

// MetaPrefix is the MetaPrefix for labels, annotations, and finalizers of Accurate.
const MetaPrefix = "accurate.cybozu.com/"

// Finalizer is the finalizer ID of Accurate.
const Finalizer = MetaPrefix + "finalizer"

// Labels
const (
	LabelType      = MetaPrefix + "type"
	LabelTemplate  = MetaPrefix + "template"
	LabelParent    = MetaPrefix + "parent"
	LabelCreatedBy = "app.kubernetes.io/created-by"
)

// Annotations
const (
	AnnFrom      = MetaPrefix + "from"
	AnnPropagate = MetaPrefix + "propagate"
	// Deprecated: Part of the deprecated propagate-generated feature subject for
	// removal soon.
	AnnPropagateGenerated = MetaPrefix + "propagate-generated"
	// Deprecated: Part of the deprecated propagate-generated feature subject for
	// removal soon.
	AnnGenerated = MetaPrefix + "generated"
)

// Label or annotation values
const (
	CreatedBy       = "accurate"
	NSTypeTemplate  = "template"
	NSTypeRoot      = "root"
	PropagateCreate = "create"
	PropagateUpdate = "update"
	PropagateAny    = "any" // defined as an in-memory index value
)

// InternalMetaPrefix is the MetaPrefix for internal (not user-facing) annotations of Accurate.
const InternalMetaPrefix = "internal.accurate.cybozu.com/"

const (
	AnnObservedGeneration = InternalMetaPrefix + "observed-generation"
	AnnConditions         = InternalMetaPrefix + "conditions"
)
