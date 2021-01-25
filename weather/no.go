package weather

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type WeatherDatum struct {
	TemperatureC    float64
	WindSpeedKM     float64
	PrecipitationMM float64
	HumidityPerc    float64
}

// WeatherClient is a generic interface which is currently implemented by using the met.no API
// The further away you get from Norway, the less accurate the results
// AFAICT the London (UK) data, which I am interested in, is fine
type WeatherClient interface {
	Latest(lat, long float64) (*WeatherDatum, error)
}

type weatherClient struct {
	url string
}

type tsdatum struct {
	Data struct {
		Instant struct {
			Details struct {
				AirTemp          float64 `json:"air_temperature"`
				RelativeHumidity float64 `json:"relative_humidity"`
				WindSpeed        float64 `json:"wind_speed"`
			} `json:"details"`
		} `json:"instant"`

		Next1Hour struct {
			Details struct {
				PrecipitationAmt float64 `json:"precipitation_amount"`
			} `json:"details"`
		} `json:"next_1_hours"`
	} `json:"data"`
}

func (c *weatherClient) Latest(lat, lon float64) (*WeatherDatum, error) {
	u := fmt.Sprintf("%s?lat=%f&lon=%f", c.url, lat, lon)

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "weather-exporter")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status code %d", res.StatusCode)
	}

	var data struct {
		Properties struct {
			Timeseries []tsdatum `json:"timeseries"`
		} `json:"properties"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	return &WeatherDatum{
		HumidityPerc:    data.Properties.Timeseries[0].Data.Instant.Details.RelativeHumidity,
		PrecipitationMM: data.Properties.Timeseries[0].Data.Next1Hour.Details.PrecipitationAmt,
		TemperatureC:    data.Properties.Timeseries[0].Data.Instant.Details.AirTemp,
		WindSpeedKM:     data.Properties.Timeseries[0].Data.Instant.Details.WindSpeed,
	}, nil
}

func New(url string) WeatherClient {
	return &weatherClient{
		url: url,
	}
}
