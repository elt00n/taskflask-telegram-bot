package domain

// TaskParticipantRole описывает отношение назначенного пользователя к задаче.
type TaskParticipantRole string

const (
	TaskParticipantRoleOwner    TaskParticipantRole = "owner"
	TaskParticipantRoleAssignee TaskParticipantRole = "assignee"
	TaskParticipantRoleEditor   TaskParticipantRole = "editor"
)

// TaskParticipant связывает задачу с назначенным на неё пользователем.
// Сама видимость задачи этой связью не определяется: её видят все участники чата.
type TaskParticipant struct {
	TaskID TaskID
	UserID UserID
	Role   TaskParticipantRole
}

// CanEdit сообщает, разрешает ли роль изменять задачу.
func (participant TaskParticipant) CanEdit() bool {
	switch participant.Role {
	case TaskParticipantRoleOwner,
		TaskParticipantRoleAssignee,
		TaskParticipantRoleEditor:
		return true
	default:
		return false
	}
}
