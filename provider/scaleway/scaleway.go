/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scaleway

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	domain "github.com/scaleway/scaleway-sdk-go/api/domain/v2alpha2"
	"github.com/scaleway/scaleway-sdk-go/scw"
	log "github.com/sirupsen/logrus"

	"github.com/yangkailc/bigtree-dns/endpoint"
	"github.com/yangkailc/bigtree-dns/pkg/apis/externaldns"
	"github.com/yangkailc/bigtree-dns/plan"
	"github.com/yangkailc/bigtree-dns/provider"
)

const (
	scalewyRecordTTL        uint32 = 300
	scalewayDefaultPriority uint32 = 0
	scalewayPriorityKey     string = "scw/priority"
)

// ScalewayProvider implements the DNS provider for Scaleway DNS
type ScalewayProvider struct {
	provider.BaseProvider
	domainAPI DomainAPI
	dryRun    bool
	// only consider hosted zones managing domains ending in this suffix
	domainFilter endpoint.DomainFilter
}

// ScalewayChange differentiates between ChangActions
type ScalewayChange struct {
	Action string
	Record []domain.Record
}

// NewScalewayProvider initializes a new Scaleway DNS provider
func NewScalewayProvider(ctx context.Context, domainFilter endpoint.DomainFilter, dryRun bool) (*ScalewayProvider, error) {
	scwClient, err := scw.NewClient(
		scw.WithEnv(),
		scw.WithUserAgent("ExternalDNS/"+externaldns.Version),
	)
	if err != nil {
		return nil, err
	}

	if _, ok := scwClient.GetDefaultOrganizationID(); !ok {
		return nil, fmt.Errorf("default organization is not set")
	}

	if _, ok := scwClient.GetAccessKey(); !ok {
		return nil, fmt.Errorf("access key no set")
	}

	if _, ok := scwClient.GetSecretKey(); !ok {
		return nil, fmt.Errorf("secret key no set")
	}

	domainAPI := domain.NewAPI(scwClient)

	return &ScalewayProvider{
		domainAPI:    domainAPI,
		dryRun:       dryRun,
		domainFilter: domainFilter,
	}, nil
}

// Zones returns the list of hosted zones.
func (p *ScalewayProvider) Zones(ctx context.Context) ([]*domain.DNSZone, error) {
	res := []*domain.DNSZone{}

	dnsZones, err := p.domainAPI.ListDNSZones(&domain.ListDNSZonesRequest{}, scw.WithAllPages(), scw.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	for _, dnsZone := range dnsZones.DNSZones {
		if p.domainFilter.Match(getCompleteZoneName(dnsZone)) {
			res = append(res, dnsZone)
		}
	}

	return res, nil
}

// Records returns the list of records in a given zone.
func (p *ScalewayProvider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	endpoints := map[string]*endpoint.Endpoint{}
	dnsZones, err := p.Zones(ctx)
	if err != nil {
		return nil, err
	}

	for _, zone := range dnsZones {
		recordsResp, err := p.domainAPI.ListDNSZoneRecords(&domain.ListDNSZoneRecordsRequest{
			DNSZone: getCompleteZoneName(zone),
		}, scw.WithAllPages())
		if err != nil {
			return nil, err
		}

		for _, record := range recordsResp.Records {
			name := record.Name + "."

			// trim any leading or ending dot
			fullRecordName := strings.Trim(name+getCompleteZoneName(zone), ".")

			if !provider.SupportedRecordType(record.Type.String()) {
				log.Infof("Skipping record %s because type %s is not supported", fullRecordName, record.Type.String())
				continue
			}

			// in external DNS, same endpoint have the same ttl and same priority
			// it's not the case in Scaleway DNS. It should never happen, but if
			// the record is modified without going through ExternalDNS, we could have
			// different priorities of ttls for a same name.
			// In this case, we juste take the first one.
			if existingEndpoint, ok := endpoints[record.Type.String()+"/"+fullRecordName]; ok {
				existingEndpoint.Targets = append(existingEndpoint.Targets, record.Data)
				log.Infof("Appending target %s to record %s, using TTL and priotiry of target %s", record.Data, fullRecordName, existingEndpoint.Targets[0])
			} else {
				ep := endpoint.NewEndpointWithTTL(fullRecordName, record.Type.String(), endpoint.TTL(record.TTL), record.Data)
				ep = ep.WithProviderSpecific(scalewayPriorityKey, fmt.Sprintf("%d", record.Priority))
				endpoints[record.Type.String()+"/"+fullRecordName] = ep
			}
		}
	}
	returnedEndpoints := []*endpoint.Endpoint{}
	for _, ep := range endpoints {
		returnedEndpoints = append(returnedEndpoints, ep)
	}

	return returnedEndpoints, nil
}

// ApplyChanges applies a set of changes in a zone.
func (p *ScalewayProvider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	requests, err := p.generateApplyRequests(ctx, changes)
	if err != nil {
		return err
	}
	for _, req := range requests {
		logChanges(req)
		if p.dryRun {
			log.Info("Running in dry run mode")
			continue
		}
		_, err := p.domainAPI.UpdateDNSZoneRecords(req, scw.WithContext(ctx))
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *ScalewayProvider) generateApplyRequests(ctx context.Context, changes *plan.Changes) ([]*domain.UpdateDNSZoneRecordsRequest, error) {
	returnedRequests := []*domain.UpdateDNSZoneRecordsRequest{}
	recordsToAdd := map[string]*domain.RecordChangeAdd{}
	recordsToDelete := map[string][]*domain.RecordChange{}

	dnsZones, err := p.Zones(ctx)
	if err != nil {
		return nil, err
	}

	zoneNameMapper := provider.ZoneIDName{}
	for _, zone := range dnsZones {
		zoneName := getCompleteZoneName(zone)
		zoneNameMapper.Add(zoneName, zoneName)
		recordsToAdd[zoneName] = &domain.RecordChangeAdd{
			Records: []*domain.Record{},
		}
		recordsToDelete[zoneName] = []*domain.RecordChange{}
	}

	for _, c := range changes.UpdateOld {
		zone, _ := zoneNameMapper.FindZone(c.DNSName)
		if zone == "" {
			log.Infof("Ignore record %s since it's not handled by ExternalDNS", c.DNSName)
			continue
		}
		recordsToDelete[zone] = append(recordsToDelete[zone], endpointToScalewayRecordsChangeDelete(zone, c)...)
	}

	for _, c := range changes.Delete {
		zone, _ := zoneNameMapper.FindZone(c.DNSName)
		if zone == "" {
			log.Infof("Ignore record %s since it's not handled by ExternalDNS", c.DNSName)
			continue
		}
		recordsToDelete[zone] = append(recordsToDelete[zone], endpointToScalewayRecordsChangeDelete(zone, c)...)
	}

	for _, c := range changes.Create {
		zone, _ := zoneNameMapper.FindZone(c.DNSName)
		if zone == "" {
			log.Infof("Ignore record %s since it's not handled by ExternalDNS", c.DNSName)
			continue
		}
		recordsToAdd[zone].Records = append(recordsToAdd[zone].Records, endpointToScalewayRecords(zone, c)...)
	}
	for _, c := range changes.UpdateNew {
		zone, _ := zoneNameMapper.FindZone(c.DNSName)
		if zone == "" {
			log.Infof("Ignore record %s since it's not handled by ExternalDNS", c.DNSName)
			continue
		}
		recordsToAdd[zone].Records = append(recordsToAdd[zone].Records, endpointToScalewayRecords(zone, c)...)
	}

	for _, zone := range dnsZones {
		zoneName := getCompleteZoneName(zone)
		req := &domain.UpdateDNSZoneRecordsRequest{
			DNSZone: zoneName,
			Changes: recordsToDelete[zoneName],
		}
		req.Changes = append(req.Changes, &domain.RecordChange{
			Add: recordsToAdd[zoneName],
		})
		returnedRequests = append(returnedRequests, req)
	}

	return returnedRequests, nil
}

func getCompleteZoneName(zone *domain.DNSZone) string {
	subdomain := zone.Subdomain + "."
	if zone.Subdomain == "" {
		subdomain = ""
	}
	return subdomain + zone.Domain
}

func endpointToScalewayRecords(zoneName string, ep *endpoint.Endpoint) []*domain.Record {
	// no annotation results in a TTL of 0, default to 300 for consistency with other providers
	var ttl = scalewyRecordTTL
	if ep.RecordTTL.IsConfigured() {
		ttl = uint32(ep.RecordTTL)
	}
	var priority = scalewayDefaultPriority
	if prop, ok := ep.GetProviderSpecificProperty(scalewayPriorityKey); ok {
		prio, err := strconv.ParseUint(prop.Value, 10, 64)
		if err != nil {
			log.Errorf("Failed parsing value of %s: %s: %v; using priority of %d", scalewayPriorityKey, prop.Value, err, scalewayDefaultPriority)
		} else {
			priority = uint32(prio)
		}
	}

	records := []*domain.Record{}

	for _, target := range ep.Targets {
		records = append(records, &domain.Record{
			Data:     target,
			Name:     strings.Trim(strings.TrimSuffix(ep.DNSName, zoneName), ". "),
			Priority: priority,
			TTL:      ttl,
			Type:     domain.RecordType(ep.RecordType),
		})
	}

	return records
}

func endpointToScalewayRecordsChangeDelete(zoneName string, ep *endpoint.Endpoint) []*domain.RecordChange {
	records := []*domain.RecordChange{}

	for _, target := range ep.Targets {
		records = append(records, &domain.RecordChange{
			Delete: &domain.RecordChangeDelete{
				Data: target,
				Name: strings.Trim(strings.TrimSuffix(ep.DNSName, zoneName), ". "),
				Type: domain.RecordType(ep.RecordType),
			},
		})
	}

	return records
}

func logChanges(req *domain.UpdateDNSZoneRecordsRequest) {
	log.Infof("Updating zone %s", req.DNSZone)
	if !log.IsLevelEnabled(log.InfoLevel) {
		return
	}
	for _, change := range req.Changes {
		if change.Add != nil {
			for _, add := range change.Add.Records {
				name := add.Name + "."
				if add.Name == "" {
					name = ""
				}

				logFields := log.Fields{
					"record":   name + req.DNSZone,
					"type":     add.Type.String(),
					"ttl":      add.TTL,
					"priority": add.Priority,
					"data":     add.Data,
				}
				log.WithFields(logFields).Info("Adding record")
			}
		} else if change.Delete != nil {
			name := change.Delete.Name + "."
			if change.Delete.Name == "" {
				name = ""
			}

			logFields := log.Fields{
				"record": name + req.DNSZone,
				"type":   change.Delete.Type.String(),
				"data":   change.Delete.Data,
			}

			log.WithFields(logFields).Info("Deleting record")
		}
	}
}
