package clob

// GetAPIKey returns the L2 API Key (may trigger derivation if not yet available)
func (c *ClobClient) GetAPIKey() string {
	if c.creds == nil {
		return ""
	}
	return c.creds.APIKey
}

// GetAPISecret returns the L2 API Secret
func (c *ClobClient) GetAPISecret() string {
	if c.creds == nil {
		return ""
	}
	return c.creds.APISecret
}

// GetAPIPassphrase returns the L2 API Passphrase
func (c *ClobClient) GetAPIPassphrase() string {
	if c.creds == nil {
		return ""
	}
	return c.creds.APIPassphrase
}

// GetFunderAddress returns the L1 Funder Address
func (c *ClobClient) GetFunderAddress() string {
	if c.creds == nil {
		return ""
	}
	return c.creds.FunderAddress
}
