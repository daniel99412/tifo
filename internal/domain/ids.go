package domain

type TIFOID string

type ExternalID struct {
	Provider string `json:"provider"`
	ID       string `json:"id"`
}

type ExternalIDs []ExternalID

func (e ExternalIDs) Get(provider string) (string, bool) {
	for _, id := range e {
		if id.Provider == provider {
			return id.ID, true
		}
	}
	return "", false
}

func (e ExternalIDs) Set(provider, id string) ExternalIDs {
	for i, existing := range e {
		if existing.Provider == provider {
			e[i].ID = id
			return e
		}
	}
	return append(e, ExternalID{Provider: provider, ID: id})
}
