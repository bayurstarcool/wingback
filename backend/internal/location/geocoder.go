package location

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Resolver turns a coordinate into a human-readable city label. It must never
// return coordinates or an address to the caller.
type Resolver interface {
	ResolveCity(ctx context.Context, lat, lng float64) (City, error)
}

// City is a coarse city-center point. It is safe for private map display:
// it represents a city area, never the user's source coordinate.
type City struct {
	Name string
	Lat  float64
	Lng  float64
}

// NominatimResolver uses OpenStreetMap's Nominatim reverse-geocoding service.
// Calls happen server-side so private tracking responses never expose the
// original coordinates to the browser.
type NominatimResolver struct {
	BaseURL   string
	Client    *http.Client
	UserAgent string
}

func NewNominatimResolver(baseURL string) *NominatimResolver {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://nominatim.openstreetmap.org/reverse"
	}
	return &NominatimResolver{
		BaseURL:   strings.TrimRight(baseURL, "/"),
		Client:    &http.Client{Timeout: 4 * time.Second},
		UserAgent: "Wingback/1.0 (city labels; contact admin)",
	}
}

type nominatimResponse struct {
	Lat     string `json:"lat"`
	Lon     string `json:"lon"`
	Address struct {
		City         string `json:"city"`
		Town         string `json:"town"`
		Municipality string `json:"municipality"`
		Village      string `json:"village"`
		County       string `json:"county"`
		State        string `json:"state"`
	} `json:"address"`
}

func (r *NominatimResolver) ResolveCity(ctx context.Context, lat, lng float64) (City, error) {
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return City{}, fmt.Errorf("coordinate out of range")
	}
	endpoint, err := url.Parse(r.BaseURL)
	if err != nil {
		return City{}, fmt.Errorf("parse geocoder URL: %w", err)
	}
	query := endpoint.Query()
	query.Set("lat", fmt.Sprintf("%.7f", lat))
	query.Set("lon", fmt.Sprintf("%.7f", lng))
	query.Set("format", "jsonv2")
	query.Set("zoom", "10")
	query.Set("addressdetails", "1")
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return City{}, fmt.Errorf("create geocoder request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", r.UserAgent)
	res, err := r.Client.Do(req)
	if err != nil {
		return City{}, fmt.Errorf("geocoder request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return City{}, fmt.Errorf("geocoder returned HTTP %d", res.StatusCode)
	}

	var payload nominatimResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return City{}, fmt.Errorf("decode geocoder response: %w", err)
	}
	mapLat, err := strconv.ParseFloat(payload.Lat, 64)
	if err != nil {
		return City{}, fmt.Errorf("geocoder returned invalid latitude")
	}
	mapLng, err := strconv.ParseFloat(payload.Lon, 64)
	if err != nil {
		return City{}, fmt.Errorf("geocoder returned invalid longitude")
	}
	for _, candidate := range []string{
		payload.Address.City,
		payload.Address.Town,
		payload.Address.Municipality,
		payload.Address.Village,
		payload.Address.County,
		payload.Address.State,
	} {
		if city := strings.TrimSpace(candidate); city != "" {
			return City{Name: city, Lat: mapLat, Lng: mapLng}, nil
		}
	}
	return City{}, fmt.Errorf("geocoder returned no city")
}

// NormalizeCity makes equality checks stable without changing display text.
func NormalizeCity(city string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(city)), " "))
}

func SameCity(from, to string) bool {
	return NormalizeCity(from) != "" && NormalizeCity(from) == NormalizeCity(to)
}
