package doh

type Provider struct {
	Name        string
	URL         string
	Description string
	IsCustom    bool
}

var BuiltinProviders = []Provider{
	{Name: "Cloudflare", URL: "https://cloudflare-dns.com/dns-query", Description: "Fast and privacy-focused"},
	{Name: "Google", URL: "https://dns.google/dns-query", Description: "Reliable and widely used"},
	{Name: "Quad9", URL: "https://dns.quad9.net/dns-query", Description: "Blocks malicious domains"},
	{Name: "AdGuard", URL: "https://dns.adguard-dns.com/dns-query", Description: "Blocks ads and trackers"},
	{Name: "Mullvad", URL: "https://dns.mullvad.net/dns-query", Description: "No logging, privacy first"},
	{Name: "NextDNS", URL: "https://dns.nextdns.io/dns-query", Description: "Customizable, use your own endpoint"},
}

func NewCustomProvider(name, url string) Provider {
	return Provider{Name: name, URL: url, Description: "Custom DoH provider", IsCustom: true}
}

func FindProvider(name string) (Provider, bool) {
	for _, p := range BuiltinProviders {
		if p.Name == name {
			return p, true
		}
	}
	return Provider{}, false
}

func ProviderNames() []string {
	names := make([]string, len(BuiltinProviders))
	for i, p := range BuiltinProviders {
		names[i] = p.Name
	}
	return names
}
