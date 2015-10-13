package obtainer

type Client struct {
	obtainer  *Obtainer
	username  string
	password  string
	serviceID string
	tgt       string
	ticket    string
}

func NewClient(obtainer *Obtainer, username, password, serviceID string) *Client {
	return &Client{
		obtainer:  obtainer,
		username:  username,
		password:  password,
		serviceID: serviceID,
	}
}

func (c *Client) Ticket() string {
	return c.ticket
}

// Obtain a new ticket
func (c *Client) Obtain() (string, error) {
	// Get Ticket Granting Ticket
	TGT, err := c.obtainer.Login(c.username, c.password)
	if err != nil {
		return "", err
	}

	// Get Service Ticket
	ticket, err := c.obtainer.RequestTicket(TGT, c.serviceID)
	if err != nil {
		return "", err
	}
	c.ticket = ticket

	// Keep a copy for renewal references
	c.tgt = TGT

	return ticket, nil
}

// Renew the ticket
func (c *Client) Renew() (string, error) {
	// Renew Service Ticket using previous TGT
	ticket, err := c.obtainer.RequestTicket(c.tgt, c.serviceID)
	if err != nil {
		// Get a new Ticket Granting Ticket
		TGT, err := c.obtainer.Login(c.username, c.password)
		if err != nil {
			return "", err
		}

		// Get Service Ticket
		ticket, err := c.obtainer.RequestTicket(TGT, c.serviceID)
		if err != nil {
			return "", err
		}
		c.ticket = ticket
		// Keep a copy for future renewal references
		c.tgt = TGT

		return ticket, nil
	}
	return ticket, nil
}

// Delete the ticket granting ticket
func (c *Client) Delete() error {
	err := c.obtainer.Logout(c.tgt)
	if err != nil {
		return err
	}
	return nil
}
