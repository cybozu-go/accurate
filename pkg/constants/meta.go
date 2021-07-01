package constants

const prefix = "innu.cybozu.com/"

// Finalizer is the finalizer ID of Innu.
const Finalizer = prefix + "finalizer"

// Labels
const (
	LabelTemplate  = prefix + "template"
	LabelRoot      = prefix + "root"
	LabelParent    = prefix + "parent"
	LabelCreatedBy = "app.kubernetes.io/created-by"
)

// Annotations
const (
	AnnFrom               = prefix + "from"
	AnnPropagate          = prefix + "propagate"
	AnnPropagateGenerated = prefix + "propagate-generated"
	AnnIsTemplate         = prefix + "is-template"
)

// Label or annotation values
const (
	CreatedBy       = "innu"
	PropagateCreate = "create"
	PropagateUpdate = "update"
)
