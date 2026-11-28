package ddns

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// alidnsACMETTL is the TTL used for ACME DNS-01 TXT records.
// Aliyun DNS rejects TTLs below 600 ("TTL 必须在 600-86400 之间"), so the
// ACME challenge record must not use the low TTL that DNSPod accepts.
const alidnsACMETTL = "600"

// UpsertTXTRecord creates or updates a TXT record for ACME DNS-01.
func (a *AliDNS) UpsertTXTRecord(ctx context.Context, cred Credentials, fqdn, value string) error {
	zone, host, err := ResolveZoneForFQDN(ctx, func(candidate string) (bool, error) {
		return a.HasZone(ctx, cred, candidate)
	}, fqdn)
	if err != nil {
		return err
	}
	subDomain := aliRR(host)
	resp,err:=a.listACMETXT(ctx,cred,zone,subDomain)
	if err!=nil{return err}
	var existing []string
	for _, rec := range resp.DomainRecords.Record {
		if strings.EqualFold(rec.Type, "TXT") && rec.RR == subDomain {
			existing = append(existing, rec.Value)
		}
	}
	if txtValuePresent(existing, value) {
		return nil
	}
	_, err = a.request(ctx, cred, map[string]string{
		"Action":     "AddDomainRecord",
		"DomainName": zone,
		"RR":         subDomain,
		"Type":       "TXT",
		"Value":      strings.TrimSpace(value),
		"TTL":        alidnsACMETTL,
	})
	return err
}

// DeleteTXTRecord removes TXT records matching value at fqdn (ACME cleanup).
func (a *AliDNS) DeleteTXTRecord(ctx context.Context, cred Credentials, fqdn, value string) error {
	if strings.TrimSpace(value)=="" {return fmt.Errorf("empty ACME TXT value")}
	zone, host, err := ResolveZoneForFQDN(ctx, func(candidate string) (bool, error) {
		return a.HasZone(ctx, cred, candidate)
	}, fqdn)
	if err != nil {
		return err
	}
	subDomain := aliRR(host)
	resp,err:=a.listACMETXT(ctx,cred,zone,subDomain)
	if err!=nil{return err}
	want := strings.TrimSpace(value)
	for _, rec := range resp.DomainRecords.Record {
		if !strings.EqualFold(rec.Type, "TXT") || rec.RR != subDomain || strings.TrimSpace(rec.Value) != want {
			continue
		}
		if rec.RecordID == "" {
			continue
		}
		if _, err := a.request(ctx, cred, map[string]string{
			"Action":   "DeleteDomainRecord",
			"RecordId": rec.RecordID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (a *AliDNS) listACMETXT(ctx context.Context,cred Credentials,zone,host string)(alidnsRecordsResponse,error){
	var out alidnsRecordsResponse
	for page:=1;page<=10000;page++ {
		body,err:=a.request(ctx,cred,map[string]string{"Action":"DescribeDomainRecords","DomainName":zone,"RRKeyWord":host,"Type":"TXT","PageNumber":strconv.Itoa(page),"PageSize":"50"})
		if err!=nil{return out,err}
		var result alidnsRecordsResponse
		if err:=json.Unmarshal(body,&result);err!=nil{return out,err}
		out.DomainRecords.Record=append(out.DomainRecords.Record,result.DomainRecords.Record...)
		if len(result.DomainRecords.Record)==0||len(out.DomainRecords.Record)>=result.TotalCount{return out,nil}
	}
	return out,fmt.Errorf("AliDNS TXT pagination exceeded limit")
}
