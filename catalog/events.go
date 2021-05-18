package catalog

// EventListener interface that listens to TDD events.
type EventListener interface {
	CreateHandler(new ThingDescription) error
	UpdateHandler(old ThingDescription, new ThingDescription) error
	DeleteHandler(old ThingDescription) error
}

// eventHandler implements sequential fav-out/fan-in of events from registry
type eventHandler []EventListener

func (h eventHandler) created(new ThingDescription) error {
	for i := range h {
		err := h[i].CreateHandler(new)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h eventHandler) updated(old ThingDescription, new ThingDescription) error {
	for i := range h {
		err := h[i].UpdateHandler(old, new)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h eventHandler) deleted(old ThingDescription) error {
	for i := range h {
		err := h[i].DeleteHandler(old)
		if err != nil {
			return err
		}
	}
	return nil
}
