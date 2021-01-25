# weather-exporter

An basic exporter for the weather, which produces the following metrics:

* `weather_temp`
* `weather_wind_speed`
* `weather_precipitation`
* `weather_humidity`

Where each metric gets a `lat` and a `lon` label

## Usage

```
go build
export WEATHER_LATLONS="51.5,0.1 59.9,10.7" # London and Oslo
./weather-exporter
curl http://localhost:8080/metrics
```
