package constants

// MetaPrefix is the MetaPrefix for labels, annotations, and finalizers of Innu.
const MetaPrefix = "innu.cybozu.com/"

// Finalizer is the finalizer ID of Innu.
const Finalizer = MetaPrefix + "finalizer"

// Labels
const (
	LabelTemplate  = MetaPrefix + "template"
	LabelRoot      = MetaPrefix + "root"
	LabelParent    = MetaPrefix + "parent"
	LabelCreatedBy = "app.kubernetes.io/created-by"
)

// Annotations
const (
	AnnFrom               = MetaPrefix + "from"
	AnnPropagate          = MetaPrefix + "propagate"
	AnnPropagateGenerated = MetaPrefix + "propagate-generated"
	AnnIsTemplate         = MetaPrefix + "is-template"
)

// Label or annotation values
const (
	CreatedBy       = "innu"
	PropagateCreate = "create"
	PropagateUpdate = "update"
	PropagateAny    = "any" // defined as an in-memory index value
)
