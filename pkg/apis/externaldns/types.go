package externaldns

import (
	"fmt"
	"reflect"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"

	"github.com/yangkailc/bigtree-dns/source"
)

const (
	passwordMask = "******"
)

var (
	// Version is the current version of the app, generated at build time
	Version = "unknown"
)

// Config is a project-wide configuration
type Config struct {
	APIServerURL                      string
	KubeConfig                        string
	RequestTimeout                    time.Duration
	ContourLoadBalancerService        string
	SkipperRouteGroupVersion          string
	Sources                           []string
	Namespace                         string
	AnnotationFilter                  string
	FQDNTemplate                      string
	CombineFQDNAndAnnotation          bool
	IgnoreHostnameAnnotation          bool
	Compatibility                     string
	PublishInternal                   bool
	PublishHostIP                     bool
	AlwaysPublishNotReadyAddresses    bool
	ConnectorSourceServer             string
	Provider                          string
	GoogleProject                     string
	GoogleBatchChangeSize             int
	GoogleBatchChangeInterval         time.Duration
	DomainFilter                      []string
	ExcludeDomains                    []string
	ZoneIDFilter                      []string
	AlibabaCloudConfigFile            string
	AlibabaCloudZoneType              string
	AWSZoneType                       string
	AWSZoneTagFilter                  []string
	AWSAssumeRole                     string
	AWSBatchChangeSize                int
	AWSBatchChangeInterval            time.Duration
	AWSEvaluateTargetHealth           bool
	AWSAPIRetries                     int
	AWSPreferCNAME                    bool
	AzureConfigFile                   string
	AzureResourceGroup                string
	AzureSubscriptionID               string
	AzureUserAssignedIdentityClientID string
	CloudflareProxied                 bool
	CloudflareZonesPerPage            int
	CoreDNSPrefix                     string
	RcodezeroTXTEncrypt               bool
	AkamaiServiceConsumerDomain       string
	AkamaiClientToken                 string
	AkamaiClientSecret                string
	AkamaiAccessToken                 string
	InfobloxGridHost                  string
	InfobloxWapiPort                  int
	InfobloxWapiUsername              string
	InfobloxWapiPassword              string `secure:"yes"`
	InfobloxWapiVersion               string
	InfobloxSSLVerify                 bool
	InfobloxView                      string
	InfobloxMaxResults                int
	DynCustomerName                   string
	DynUsername                       string
	DynPassword                       string `secure:"yes"`
	DynMinTTLSeconds                  int
	OCIConfigFile                     string
	InMemoryZones                     []string
	OVHEndpoint                       string
	OVHApiRateLimit                   int
	PDNSServer                        string
	PDNSAPIKey                        string `secure:"yes"`
	PDNSTLSEnabled                    bool
	TLSCA                             string
	TLSClientCert                     string
	TLSClientCertKey                  string
	Policy                            string
	Registry                          string
	TXTOwnerID                        string
	TXTPrefix                         string
	TXTSuffix                         string
	Interval                          time.Duration
	Once                              bool
	DryRun                            bool
	UpdateEvents                      bool
	LogFormat                         string
	MetricsAddress                    string
	LogLevel                          string
	TXTCacheInterval                  time.Duration
	ExoscaleEndpoint                  string
	ExoscaleAPIKey                    string `secure:"yes"`
	ExoscaleAPISecret                 string `secure:"yes"`
	CRDSourceAPIVersion               string
	CRDSourceKind                     string
	ServiceTypeFilter                 []string
	CFAPIEndpoint                     string
	CFUsername                        string
	CFPassword                        string
	RFC2136Host                       string
	RFC2136Port                       int
	RFC2136Zone                       string
	RFC2136Insecure                   bool
	RFC2136TSIGKeyName                string
	RFC2136TSIGSecret                 string `secure:"yes"`
	RFC2136TSIGSecretAlg              string
	RFC2136TAXFR                      bool
	RFC2136MinTTL                     time.Duration
	NS1Endpoint                       string
	NS1IgnoreSSL                      bool
	TransIPAccountName                string
	TransIPPrivateKeyFile             string
	DigitalOceanAPIPageSize           int
}

var defaultConfig = &Config{
	APIServerURL:                "",
	KubeConfig:                  "",
	RequestTimeout:              time.Second * 30,
	ContourLoadBalancerService:  "heptio-contour/contour",
	SkipperRouteGroupVersion:    "zalando.org/v1",
	Sources:                     nil,
	Namespace:                   "",
	AnnotationFilter:            "",
	FQDNTemplate:                "",
	CombineFQDNAndAnnotation:    false,
	IgnoreHostnameAnnotation:    false,
	Compatibility:               "",
	PublishInternal:             false,
	PublishHostIP:               false,
	ConnectorSourceServer:       "localhost:8080",
	Provider:                    "",
	GoogleProject:               "",
	GoogleBatchChangeSize:       1000,
	GoogleBatchChangeInterval:   time.Second,
	DomainFilter:                []string{},
	ExcludeDomains:              []string{},
	AlibabaCloudConfigFile:      "/etc/kubernetes/alibaba-cloud.json",
	AWSZoneType:                 "",
	AWSZoneTagFilter:            []string{},
	AWSAssumeRole:               "",
	AWSBatchChangeSize:          1000,
	AWSBatchChangeInterval:      time.Second,
	AWSEvaluateTargetHealth:     true,
	AWSAPIRetries:               3,
	AWSPreferCNAME:              false,
	AzureConfigFile:             "/etc/kubernetes/azure.json",
	AzureResourceGroup:          "",
	AzureSubscriptionID:         "",
	CloudflareProxied:           false,
	CloudflareZonesPerPage:      50,
	CoreDNSPrefix:               "/skydns/",
	RcodezeroTXTEncrypt:         false,
	AkamaiServiceConsumerDomain: "",
	AkamaiClientToken:           "",
	AkamaiClientSecret:          "",
	AkamaiAccessToken:           "",
	InfobloxGridHost:            "",
	InfobloxWapiPort:            443,
	InfobloxWapiUsername:        "admin",
	InfobloxWapiPassword:        "",
	InfobloxWapiVersion:         "2.3.1",
	InfobloxSSLVerify:           true,
	InfobloxView:                "",
	InfobloxMaxResults:          0,
	OCIConfigFile:               "/etc/kubernetes/oci.yaml",
	InMemoryZones:               []string{},
	OVHEndpoint:                 "ovh-eu",
	OVHApiRateLimit:             20,
	PDNSServer:                  "http://localhost:8081",
	PDNSAPIKey:                  "",
	PDNSTLSEnabled:              false,
	TLSCA:                       "",
	TLSClientCert:               "",
	TLSClientCertKey:            "",
	Policy:                      "sync",
	Registry:                    "txt",
	TXTOwnerID:                  "default",
	TXTPrefix:                   "",
	TXTSuffix:                   "",
	TXTCacheInterval:            0,
	Interval:                    time.Minute,
	Once:                        false,
	DryRun:                      false,
	UpdateEvents:                false,
	LogFormat:                   "text",
	MetricsAddress:              ":7979",
	LogLevel:                    logrus.InfoLevel.String(),
	ExoscaleEndpoint:            "https://api.exoscale.ch/dns",
	ExoscaleAPIKey:              "",
	ExoscaleAPISecret:           "",
	CRDSourceAPIVersion:         "externaldns.k8s.io/v1alpha1",
	CRDSourceKind:               "DNSEndpoint",
	ServiceTypeFilter:           []string{},
	CFAPIEndpoint:               "",
	CFUsername:                  "",
	CFPassword:                  "",
	RFC2136Host:                 "",
	RFC2136Port:                 0,
	RFC2136Zone:                 "",
	RFC2136Insecure:             false,
	RFC2136TSIGKeyName:          "",
	RFC2136TSIGSecret:           "",
	RFC2136TSIGSecretAlg:        "",
	RFC2136TAXFR:                true,
	RFC2136MinTTL:               0,
	NS1Endpoint:                 "",
	NS1IgnoreSSL:                false,
	TransIPAccountName:          "",
	TransIPPrivateKeyFile:       "",
	DigitalOceanAPIPageSize:     50,
}

// NewConfig returns new Config object
func NewConfig() *Config {
	return &Config{}
}

func (cfg *Config) String() string {
	// prevent logging of sensitive information
	temp := *cfg

	t := reflect.TypeOf(temp)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if val, ok := f.Tag.Lookup("secure"); ok && val == "yes" {
			if f.Type.Kind() != reflect.String {
				continue
			}
			v := reflect.ValueOf(&temp).Elem().Field(i)
			if v.String() != "" {
				v.SetString(passwordMask)
			}
		}
	}

	return fmt.Sprintf("%+v", temp)
}

// allLogLevelsAsStrings returns all logrus levels as a list of strings
func allLogLevelsAsStrings() []string {
	var levels []string
	for _, level := range logrus.AllLevels {
		levels = append(levels, level.String())
	}
	return levels
}

// ParseFlags adds and parses flags from command line
func (cfg *Config) ParseFlags(args []string) error {
	app := kingpin.New("bigtree-dns", "BigtreeDNS synchronizes exposed Kubernetes Services and Ingresses with DNS \n")
	app.Version(Version)
	app.DefaultEnvars()

	// Flags related to Kubernetes
	app.Flag("server", "The Kubernetes API server to connect to (default: auto-detect)").Default(defaultConfig.APIServerURL).StringVar(&cfg.APIServerURL)
	app.Flag("kubeconfig", "Retrieve target cluster configuration from a Kubernetes configuration file (default: auto-detect)").Default(defaultConfig.KubeConfig).StringVar(&cfg.KubeConfig)
	app.Flag("request-timeout", "Request timeout when calling Kubernetes APIs. 0s means no timeout").Default(defaultConfig.RequestTimeout.String()).DurationVar(&cfg.RequestTimeout)

        app.Flag("skipper-routegroup-groupversion", "The resource version for skipper routegroup").Default(source.DefaultRoutegroupVersion).StringVar(&cfg.SkipperRouteGroupVersion)
	// Flags related to processing sources
	app.Flag("source", "The resource types that are queried for endpoints; specify multiple times for multiple sources (required, options: service, ingress, node, fake)").Required().PlaceHolder("source").EnumsVar(&cfg.Sources, "service", "ingress", "node", "fake")

	app.Flag("namespace", "Limit sources of endpoints to a specific namespace (default: all namespaces)").Default(defaultConfig.Namespace).StringVar(&cfg.Namespace)
	app.Flag("annotation-filter", "Filter sources managed by external-dns via annotation using label selector semantics (default: all sources)").Default(defaultConfig.AnnotationFilter).StringVar(&cfg.AnnotationFilter)
	app.Flag("fqdn-template", "A templated string that's used to generate DNS names from sources that don't define a hostname themselves, or to add a hostname suffix when paired with the fake source (optional). Accepts comma separated list for multiple global FQDN.").Default(defaultConfig.FQDNTemplate).StringVar(&cfg.FQDNTemplate)
	app.Flag("combine-fqdn-annotation", "Combine FQDN template and Annotations instead of overwriting").BoolVar(&cfg.CombineFQDNAndAnnotation)
	app.Flag("ignore-hostname-annotation", "Ignore hostname annotation when generating DNS names, valid only when using fqdn-template is set (optional, default: false)").BoolVar(&cfg.IgnoreHostnameAnnotation)
	app.Flag("compatibility", "Process annotation semantics from legacy implementations (optional, options: mate, molecule)").Default(defaultConfig.Compatibility).EnumVar(&cfg.Compatibility, "", "mate", "molecule")
	app.Flag("publish-internal-services", "Allow external-dns to publish DNS records for ClusterIP services (optional)").BoolVar(&cfg.PublishInternal)
	app.Flag("publish-host-ip", "Allow external-dns to publish host-ip for headless services (optional)").BoolVar(&cfg.PublishHostIP)
	app.Flag("always-publish-not-ready-addresses", "Always publish also not ready addresses for headless services (optional)").BoolVar(&cfg.AlwaysPublishNotReadyAddresses)
	app.Flag("connector-source-server", "The server to connect for connector source, valid only when using connector source").Default(defaultConfig.ConnectorSourceServer).StringVar(&cfg.ConnectorSourceServer)
	app.Flag("crd-source-apiversion", "API version of the CRD for crd source, e.g. `externaldns.k8s.io/v1alpha1`, valid only when using crd source").Default(defaultConfig.CRDSourceAPIVersion).StringVar(&cfg.CRDSourceAPIVersion)
	app.Flag("crd-source-kind", "Kind of the CRD for the crd source in API group and version specified by crd-source-apiversion").Default(defaultConfig.CRDSourceKind).StringVar(&cfg.CRDSourceKind)
	app.Flag("service-type-filter", "The service types to take care about (default: all, expected: ClusterIP, NodePort, LoadBalancer or ExternalName)").StringsVar(&cfg.ServiceTypeFilter)

	// Flags related to providers
	app.Flag("provider", "The DNS provider where the DNS records will be created (required, options: coredns, skydns, inmemory)").Required().PlaceHolder("provider").EnumVar(&cfg.Provider, "coredns", "skydns", "inmemory")
	app.Flag("domain-filter", "Limit possible target zones by a domain suffix; specify multiple times for multiple domains (optional)").Default("").StringsVar(&cfg.DomainFilter)
	app.Flag("exclude-domains", "Exclude subdomains (optional)").Default("").StringsVar(&cfg.ExcludeDomains)
	app.Flag("zone-id-filter", "Filter target zones by hosted zone id; specify multiple times for multiple zones (optional)").Default("").StringsVar(&cfg.ZoneIDFilter)
	app.Flag("coredns-prefix", "When using the CoreDNS provider, specify the prefix name").Default(defaultConfig.CoreDNSPrefix).StringVar(&cfg.CoreDNSPrefix)
	app.Flag("inmemory-zone", "Provide a list of pre-configured zones for the inmemory provider; specify multiple times for multiple zones (optional)").Default("").StringsVar(&cfg.InMemoryZones)

	// Flags related to policies
	app.Flag("policy", "Modify how DNS records are synchronized between sources and providers (default: sync, options: sync, upsert-only, create-only)").Default(defaultConfig.Policy).EnumVar(&cfg.Policy, "sync", "upsert-only", "create-only")

	// Flags related to the registry
	app.Flag("registry", "The registry implementation to use to keep track of DNS record ownership (default: txt, options: txt, noop, aws-sd)").Default(defaultConfig.Registry).EnumVar(&cfg.Registry, "txt", "noop", "aws-sd")
	app.Flag("txt-owner-id", "When using the TXT registry, a name that identifies this instance of ExternalDNS (default: default)").Default(defaultConfig.TXTOwnerID).StringVar(&cfg.TXTOwnerID)
	app.Flag("txt-prefix", "When using the TXT registry, a custom string that's prefixed to each ownership DNS record (optional). Mutual exclusive with txt-suffix!").Default(defaultConfig.TXTPrefix).StringVar(&cfg.TXTPrefix)
	app.Flag("txt-suffix", "When using the TXT registry, a custom string that's suffixed to the host portion of each ownership DNS record (optional). Mutual exclusive with txt-prefix!").Default(defaultConfig.TXTSuffix).StringVar(&cfg.TXTSuffix)

	// Flags related to the main control loop
	app.Flag("txt-cache-interval", "The interval between cache synchronizations in duration format (default: disabled)").Default(defaultConfig.TXTCacheInterval.String()).DurationVar(&cfg.TXTCacheInterval)
	app.Flag("interval", "The interval between two consecutive synchronizations in duration format (default: 1m)").Default(defaultConfig.Interval.String()).DurationVar(&cfg.Interval)
	app.Flag("once", "When enabled, exits the synchronization loop after the first iteration (default: disabled)").BoolVar(&cfg.Once)
	app.Flag("dry-run", "When enabled, prints DNS record changes rather than actually performing them (default: disabled)").BoolVar(&cfg.DryRun)
	app.Flag("events", "When enabled, in addition to running every interval, the reconciliation loop will get triggered when supported sources change (default: disabled)").BoolVar(&cfg.UpdateEvents)

	// Miscellaneous flags
	app.Flag("log-format", "The format in which log messages are printed (default: text, options: text, json)").Default(defaultConfig.LogFormat).EnumVar(&cfg.LogFormat, "text", "json")
	app.Flag("metrics-address", "Specify where to serve the metrics and health check endpoint (default: :7979)").Default(defaultConfig.MetricsAddress).StringVar(&cfg.MetricsAddress)
	app.Flag("log-level", "Set the level of logging. (default: info, options: panic, debug, info, warning, error, fatal").Default(defaultConfig.LogLevel).EnumVar(&cfg.LogLevel, allLogLevelsAsStrings()...)

	_, err := app.Parse(args)
	if err != nil {
		return err
	}

	return nil
}
