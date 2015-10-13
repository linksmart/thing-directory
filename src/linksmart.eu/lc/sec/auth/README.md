#	Package auth provides interfaces to obtain and validate service tickets.

## USAGE

### Obtainer package
```
	// Setup ticket obtainer
	to, err := obtainer.Setup("cas", c.ServerAddr)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// Get Ticket Granting Ticket
	TGT, err := to.Login(c.Username, c.Password)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// Get Service Ticket
	serviceTicket, err := to.RequestTicket(TGT, c.ServiceID)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
			
	// Logout
	err := to.Logout(TGT)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
```
#### Using obtainer client
```
	// Setup ticket obtainer
	to, err := obtainer.Setup("cas", c.ServerAddr)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// Ticker obtainer client
	ticket := obtainer.NewClient(*to, c.Username, c.Password, c.ServiceID)

	// Obtain a new ticket
	t, err := ticket.Obtain()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// Renew the ticket
	t, err = ticket.Renew()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// Delete the ticket
	err = ticket.Delete()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
```

### Validator package
```
	// Setup ticket validator
	tv, err := validator.Setup("cas", c.ServerAddr, c.ServiceID, &c.Authz)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// Validate Service Ticket
	valid, attr, err := tv.Validate(serviceTicket)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
```
#### Using the handler
```
	// Setup ticket validator
	tv, err := validator.Setup("cas", c.ServerAddr, c.ServiceID, &c.Authz)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// Appending to an alice chain of HTTP middleware functions
	existingChain = existingChain.Append(tv.Handler)
```