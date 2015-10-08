package obtainer

// Interface methods to login, obtain Service Ticket, and logout
type Obtainer interface {
	// Given valid username and password,
	// 	Login must return a Ticket Granting Ticket (TGT).
	Login(username, password string) (string, error)
	// Given valid TGT and serviceID,
	//	RequestTicket must return a Service Ticket.
	RequestTicket(TGT, serviceID string) (string, error)
	// Given a valid TGT,
	// 	Logout must expire it.
	Logout(TGT string) error
}
