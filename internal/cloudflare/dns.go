package cloudflare

import (
	"encoding/json"
	"fmt"
)

// DNSRecord DNS 记录
type DNSRecord struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	Proxied  bool   `json:"proxied"`
	TTL      int    `json:"ttl"`
	ZoneID   string `json:"zone_id"`
	ZoneName string `json:"zone_name"`
}

// dnsListResponse 列表响应
type dnsListResponse struct {
	Result []DNSRecord `json:"result"`
}

// ListDNSRecords 列出 Zone 下的 DNS 记录
func (c *Client) ListDNSRecords(zoneID string) ([]DNSRecord, error) {
	data, err := c.do("GET", "/zones/"+zoneID+"/dns_records?per_page=100", nil)
	if err != nil {
		return nil, err
	}
	var res dnsListResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return res.Result, nil
}

// CreateDNSRecord 创建记录
func (c *Client) CreateDNSRecord(zoneID string, recordType, name, content string, ttl int, proxied bool) (*DNSRecord, error) {
	if ttl <= 0 {
		ttl = TTLAuto
	}
	body := map[string]interface{}{
		"type":    recordType,
		"name":    name,
		"content": content,
		"ttl":     ttl,
		"proxied": proxied,
	}
	data, err := c.do("POST", "/zones/"+zoneID+"/dns_records", body)
	if err != nil {
		return nil, err
	}
	var res struct {
		Result DNSRecord `json:"result"`
	}
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return &res.Result, nil
}

// UpdateDNSRecord 更新记录
func (c *Client) UpdateDNSRecord(zoneID, recordID string, recordType, name, content string, ttl int, proxied bool) (*DNSRecord, error) {
	if ttl <= 0 {
		ttl = TTLAuto
	}
	body := map[string]interface{}{
		"type":    recordType,
		"name":    name,
		"content": content,
		"ttl":     ttl,
		"proxied": proxied,
	}
	data, err := c.do("PUT", "/zones/"+zoneID+"/dns_records/"+recordID, body)
	if err != nil {
		return nil, err
	}
	var res struct {
		Result DNSRecord `json:"result"`
	}
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return &res.Result, nil
}

// DeleteDNSRecord 删除记录
func (c *Client) DeleteDNSRecord(zoneID, recordID string) error {
	_, err := c.do("DELETE", "/zones/"+zoneID+"/dns_records/"+recordID, nil)
	return err
}

// EnsureCNAME 若不存在则创建 CNAME name -> content，存在则更新；返回 record ID
func (c *Client) EnsureCNAME(zoneID, name, content string) (recordID string, err error) {
	records, err := c.ListDNSRecords(zoneID)
	if err != nil {
		return "", err
	}
	for _, r := range records {
		if r.Name == name || r.Name == name+"." {
			if r.Type == "CNAME" && r.Content == content {
				return r.ID, nil
			}
			_, err := c.UpdateDNSRecord(zoneID, r.ID, "CNAME", name, content, TTLAuto, true)
			if err != nil {
				return "", err
			}
			return r.ID, nil
		}
	}
	rec, err := c.CreateDNSRecord(zoneID, "CNAME", name, content, TTLAuto, true)
	if err != nil {
		return "", err
	}
	return rec.ID, nil
}

// EnsureA 若不存在则创建 A 记录，存在则更新；返回 record ID
func (c *Client) EnsureA(zoneID, name, content string) (recordID string, err error) {
	records, err := c.ListDNSRecords(zoneID)
	if err != nil {
		return "", err
	}
	for _, r := range records {
		if r.Name == name || r.Name == name+"." {
			if r.Type == "A" && r.Content == content {
				return r.ID, nil
			}
			_, err := c.UpdateDNSRecord(zoneID, r.ID, "A", name, content, TTLAuto, false)
			if err != nil {
				return "", err
			}
			return r.ID, nil
		}
	}
	rec, err := c.CreateDNSRecord(zoneID, "A", name, content, TTLAuto, false)
	if err != nil {
		return "", fmt.Errorf("create A: %w", err)
	}
	return rec.ID, nil
}
