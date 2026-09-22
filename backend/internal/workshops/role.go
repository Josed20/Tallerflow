package workshops

type Role string

const (
	RoleOwner      Role = "OWNER"
	RoleAdmin      Role = "ADMIN"
	RoleSupervisor Role = "SUPERVISOR"
	RoleOperator   Role = "OPERATOR"
)

func (r Role) Valid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleSupervisor, RoleOperator:
		return true
	default:
		return false
	}
}
