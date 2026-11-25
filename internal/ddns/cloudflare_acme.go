package ddns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// UpsertTXTRecord creates or updates a TXT record for ACME DNS-01.
func (c *Cloudflare) UpsertTXTRecord(ctx context.Context, cred Credentials, fqdn, value string) error {
	zone, host, err := ResolveZoneForFQDN(ctx, func(candidate string) (bool, error) {
		return c.HasZone(ctx, cred, candidate)
	}, fqdn)
	if err != nil {
		return err
	}
	zoneID, err := c.zoneID(ctx, cred.Token, zone)
	if err != nil {
		return err
	}
	recordFQDN := fqdnForRecord(zone, host)
	resp,err:=c.listACMETXT(ctx,cred.Token,zoneID,recordFQDN)
	if err!=nil{return err}
	var existing []string
	for _, rec := range resp.Result {
		existing = append(existing, rec.Content)
	}
	if txtValuePresent(existing, value) {
		return nil
	}
	_, err = c.request(ctx, cred.Token, http.MethodPost, fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", zoneID), map[string]any{
		"type":    "TXT",
		"name":    recordFQDN,
		"content": strings.TrimSpace(value),
		"ttl":     120,
		"proxied": false,
	})
	return err
}

// DeleteTXTRecord removes TXT records matching value at fqdn (ACME cleanup).
func (c *Cloudflare) DeleteTXTRecord(ctx context.Context, cred Credentials, fqdn, value string) error {
	if strings.TrimSpace(value)=="" {return fmt.Errorf("empty ACME TXT value")}
	zone, host, err := ResolveZoneForFQDN(ctx, func(candidate string) (bool, error) {
		return c.HasZone(ctx, cred, candidate)
	}, fqdn)
	if err != nil {
		return err
	}
	zoneID, err := c.zoneID(ctx, cred.Token, zone)
	if err != nil {
		return err
	}
	recordFQDN := fqdnForRecord(zone, host)
	resp,err:=c.listACMETXT(ctx,cred.Token,zoneID,recordFQDN)
	if err!=nil{return err}
	want := strings.TrimSpace(value)
	for _, rec := range resp.Result {
		if strings.TrimSpace(rec.Content) != want {
			continue
		}
		if rec.ID == "" {
			continue
		}
		if _, err := c.request(ctx, cred.Token, http.MethodDelete, fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zoneID, rec.ID), nil); err != nil {
			return err
		}
	}
	return nil
}

func (c *Cloudflare) listACMETXT(ctx context.Context,token,zoneID,fqdn string)(cfRecordsResponse,error){
	var out cfRecordsResponse
	for page:=1;page<=10000;page++ {
		body,err:=c.request(ctx,token,http.MethodGet,fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?type=TXT&name=%s&per_page=100&page=%d",zoneID,url.QueryEscape(fqdn),page),nil)
		if err!=nil{return out,err}
		var result cfRecordsResponse
		if err:=json.Unmarshal(body,&result);err!=nil{return out,err}
		out.Result=append(out.Result,result.Result...)
		if len(result.Result)==0||page>=result.ResultInfo.TotalPages{return out,nil}
	}
	return out,fmt.Errorf("Cloudflare TXT pagination exceeded limit")
}
