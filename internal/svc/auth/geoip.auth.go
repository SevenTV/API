package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

func (a *authorizer) LocateIP(ctx context.Context, addr string) (GeoIPResult, error) {
	result := GeoIPResult{}

	// http api request
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.iplocation.net/?ip="+addr, nil)
	if err != nil {
		return result, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return result, errors.New("bad response from iplocation.net")
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return result, err
	}

	// Fix bad country names
	switch result.CountryCode {
	case "TW":
		result.CountryName = "Taiwan (Republic of China)"
	}

	return result, nil
}

type GeoIPResult struct {
	IP          string `json:"ip"`
	CountryName string `json:"country_name"`
	CountryCode string `json:"country_code2"`
}
