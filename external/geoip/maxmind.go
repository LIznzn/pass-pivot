package geoip

type MaxmindGeoipProvider struct{}

func NewMaxmindGeoipProvider() *MaxmindGeoipProvider {
	geoip := &MaxmindGeoipProvider{}
	return geoip
}

func (captcha *MaxmindGeoipProvider) LookupLocation(ipAddress string) (string, error) {
	return "", nil
}
