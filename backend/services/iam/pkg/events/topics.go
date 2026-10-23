package events

const (
	UserTopic         = "iam.user.v1"
	SessionTopic      = "iam.session.v1"
	OrganizationTopic = "iam.organization.v1"
	InvitationTopic   = "iam.invitation.v1"
)

var Topics = []string{UserTopic, SessionTopic, OrganizationTopic, InvitationTopic}
