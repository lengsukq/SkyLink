package cloudflare

import "encoding/json"

// Zone 域名 Zone
type Zone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// zonesResponse API 返回结构
type zonesResponse struct {
	Result []Zone `json:"result"`
}

// ListZones 列出当前账号下的所有 Zone
func (c *Client) ListZones() ([]Zone, error) {
	data, err := c.do("GET", "/zones?per_page=50", nil)
	if err != nil {
		return nil, err
	}
	var res zonesResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return res.Result, nil
}
